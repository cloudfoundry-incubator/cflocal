package plugin

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	cfplugin "code.cloudfoundry.org/cli/plugin"
	docker "github.com/docker/docker/client"
	goversion "github.com/hashicorp/go-version"
	"github.com/kardianos/osext"

	"github.com/sclevine/cflocal/cf"
	"github.com/sclevine/cflocal/cf/cmd"
	"github.com/sclevine/cflocal/local"
	"github.com/sclevine/cflocal/remote"
	"github.com/sclevine/cflocal/utils"
)

type Plugin struct {
	UI       PluginUI
	Version  string
	RunErr   error
	ExitChan chan struct{}
}

type PluginUI interface {
	Prompt(prompt string) string
	Output(format string, a ...interface{})
	Warn(format string, a ...interface{})
	Error(err error)
}

func (p *Plugin) Run(cliConnection cfplugin.CliConnection, args []string) {
	if args[0] == "CLI-MESSAGE-UNINSTALL" {
		return
	}

	signal.Notify(make(chan os.Signal), syscall.SIGHUP)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)
	signal.Notify(signalChan, syscall.SIGTERM)
	go func() {
		<-signalChan
		close(p.ExitChan)
	}()

	client, err := docker.NewEnvClient()
	if err != nil {
		p.RunErr = err
		return
	}
	client.UpdateClientVersion("")
	stager := &local.Stager{
		DiegoVersion: "0.1482.0",
		GoVersion:    "1.7",
		StackVersion: "latest",
		Docker:       client,
		Logs:         os.Stdout,
		ExitChan:     p.ExitChan,
	}
	runner := &local.Runner{
		StackVersion: "latest",
		Docker:       client,
		Logs:         os.Stdout,
		ExitChan:     p.ExitChan,
	}
	app := &remote.App{
		CLI: cliConnection,
		UI:  p.UI,
	}
	fs := &utils.FS{}
	config := &local.Config{Path: "./local.yml"}
	help := &Help{
		CLI: cliConnection,
		UI:  p.UI,
	}
	cf := &cf.CF{
		UI:   p.UI,
		Help: help,
		Cmds: []cf.Cmd{
			&cmd.Export{
				UI:     p.UI,
				Stager: stager,
				Runner: runner,
				FS:     fs,
				Help:   help,
				Config: config,
			},
			&cmd.Pull{
				UI:     p.UI,
				App:    app,
				FS:     fs,
				Help:   help,
				Config: config,
			},

			&cmd.Push{
				UI:     p.UI,
				App:    app,
				FS:     fs,
				Help:   help,
				Config: config,
			},
			&cmd.Run{
				UI:     p.UI,
				Stager: stager,
				Runner: runner,
				App:    app,
				FS:     fs,
				Help:   help,
				Config: config,
			},
			&cmd.Stage{
				UI:     p.UI,
				Stager: stager,
				App:    app,
				FS:     fs,
				Help:   help,
				Config: config,
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
