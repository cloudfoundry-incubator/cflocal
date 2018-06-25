package plugin

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
	goversion "github.com/hashicorp/go-version"
	"github.com/kardianos/osext"
	"github.com/buildpack/forge"
	"github.com/buildpack/forge/app"
	"github.com/buildpack/forge/engine"
	"github.com/buildpack/forge/engine/docker"

	"code.cloudfoundry.org/cflocal/cf"
	"code.cloudfoundry.org/cflocal/cf/cmd"
	"code.cloudfoundry.org/cflocal/cfplugin"
	"code.cloudfoundry.org/cflocal/fs"
	"code.cloudfoundry.org/cflocal/remote"
)

type Plugin struct {
	UI      UI
	Version string
	RunErr  error
	Exit    <-chan struct{}
}

type UI interface {
	forge.Loader
	Prompt(prompt string) string
	Output(format string, a ...interface{})
	Warn(format string, a ...interface{})
	Error(err error)
}

func (p *Plugin) Run(cliConnection cfplugin.CliConnection, args []string) {
	if args[0] == "CLI-MESSAGE-UNINSTALL" {
		return
	}

	proxy := engine.ProxyConfig{
		HTTPProxy:  firstEnv("HTTP_PROXY", "http_proxy"),
		HTTPSProxy: firstEnv("HTTPS_PROXY", "https_proxy"),
		NoProxy:    firstEnv("NO_PROXY", "no_proxy"),
	}
	if useProxy, ok := boolEnv("CFL_USE_PROXY"); ok {
		if useProxy {
			proxy.UseRemotely = true
		} else {
			proxy = engine.ProxyConfig{}
		}
	}
	engine, err := docker.New(&engine.EngineConfig{
		Proxy: proxy,
		Exit:  p.Exit,
	})
	if err != nil {
		p.RunErr = err
		return
	}
	defer engine.Close()

	ccSkipSSLVerify, err := cliConnection.IsSSLDisabled()
	if err != nil {
		p.RunErr = err
		return
	}

	ccHTTPClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: ccSkipSSLVerify,
			},
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	stager := forge.NewStager(engine)
	stager.Logs = color.Output
	stager.Loader = p.UI

	runner := forge.NewRunner(engine)
	runner.Logs = color.Output
	runner.Loader = p.UI

	exporter := forge.NewExporter(engine)
	exporter.Loader = p.UI

	forwarder := forge.NewForwarder(engine)
	forwarder.Logs = color.Output
	forwarder.Loader = p.UI

	remoteApp := &remote.App{
		CLI:  cliConnection,
		UI:   p.UI,
		HTTP: ccHTTPClient,
	}
	sysFS := &fs.FS{}
	config := &app.Config{
		Path: "./local.yml",
	}
	help := &Help{
		CLI: cliConnection,
		UI:  p.UI,
	}
	cf := &cf.CF{
		UI:   p.UI,
		Help: help,
		Cmds: []cf.Cmd{
			&cmd.Export{
				UI:       p.UI,
				Exporter: exporter,
				FS:       sysFS,
				Help:     help,
				Config:   config,
			},
			&cmd.Pull{
				UI:        p.UI,
				RemoteApp: remoteApp,
				FS:        sysFS,
				Help:      help,
				Config:    config,
			},
			&cmd.Push{
				UI:        p.UI,
				RemoteApp: remoteApp,
				FS:        sysFS,
				Help:      help,
				Config:    config,
			},
			&cmd.Run{
				UI:        p.UI,
				Runner:    runner,
				Forwarder: forwarder,
				RemoteApp: remoteApp,
				FS:        sysFS,
				Help:      help,
				Config:    config,
			},
			&cmd.Stage{
				UI:        p.UI,
				Stager:    stager,
				RemoteApp: remoteApp,
				TarApp:    app.Tar,
				FS:        sysFS,
				Help:      help,
				Config:    config,
			},
		},
		Version: p.Version,
	}
	if err := cf.Run(args[1:]); err != nil {
		p.RunErr = err
		return
	}
}

func (p *Plugin) GetMetadata() cfplugin.PluginMetadata {
	version := goversion.Must(goversion.NewVersion(p.Version))
	return cfplugin.PluginMetadata{
		Name: "cflocal",
		Version: cfplugin.VersionType{
			Major: version.Segments()[0],
			Minor: version.Segments()[1],
			Build: version.Segments()[2],
		},
		Commands: []cfplugin.Command{{
			Name:         "local",
			HelpText:     "Stage, launch, push, pull, and export CF apps -- in Docker",
			UsageDetails: cfplugin.Usage{Usage: strings.TrimSpace(Usage)},
		}},
	}
}

func (p *Plugin) Help(name string) {
	p.UI.Output("Usage: %s", name)
	p.UI.Output("Running this binary directly will automatically install the CF Local cf CLI plugin.")
	p.UI.Output("You must have the latest version of the cf CLI and Docker installed to use CF Local.")
	p.UI.Output("After installing, run: cf local help")
}

func (p *Plugin) Install() error {
	plugin, err := osext.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine plugin path: %s", err)
	}

	operation := "upgraded"
	if err := exec.Command("cf", "uninstall-plugin", "cflocal").Run(); err != nil {
		operation = "installed"
	}

	cliVersion, err := cliVersion()
	if err != nil {
		return err
	}
	installOpts := []string{"install-plugin", plugin}
	if !cliVersion.LessThan(goversion.Must(goversion.NewVersion("6.13.0"))) {
		installOpts = append(installOpts, "-f")
	}
	if output, err := exec.Command("cf", installOpts...).CombinedOutput(); err != nil {
		return errors.New(strings.TrimSpace(string(output)))
	}

	p.UI.Output("Plugin successfully %s. Current version: %s", operation, p.Version)
	return nil
}

func cliVersion() (*goversion.Version, error) {
	versionLine, err := exec.Command("cf", "--version").Output()
	if err != nil {
		return nil, errors.New("failed to determine cf CLI version")
	}
	versionStr := strings.TrimPrefix(strings.TrimSpace(string(versionLine)), "cf version ")
	version, err := goversion.NewVersion(versionStr)
	if err != nil || version.LessThan(goversion.Must(goversion.NewVersion("6.7.0"))) {
		return nil, errors.New("cf CLI version too old")
	}
	return version, nil
}

func firstEnv(ks ...string) string {
	for _, k := range ks {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func boolEnv(k string) (v, ok bool) {
	switch strings.TrimSpace(strings.ToLower(os.Getenv(k))) {
	case "true", "yes", "1":
		return true, true
	case "false", "no", "0":
		return false, true
	}
	return false, false
}
