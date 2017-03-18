package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"

	"github.com/fatih/color"

	"github.com/sclevine/cflocal/local"
)

type Run struct {
	UI     UI
	Stager Stager
	Runner Runner
	App    App
	FS     FS
	Help   Help
	Config Config
}

type runOptions struct {
	name, appDir           string
	serviceApp, forwardApp string
	port                   uint
}

func (r *Run) Match(args []string) bool {
	return len(args) > 0 && args[0] == "run"
}

func (r *Run) Run(args []string) error {
	options, err := r.options(args)
	if err != nil {
		r.Help.Short()
		return err
	}
	absAppDir, appDirEmpty := "", false
	if options.appDir != "" {
		if absAppDir, err = r.FS.Abs(options.appDir); err != nil {
			return err
		}
		if err := r.FS.MakeDirAll(absAppDir); err != nil {
			return err
		}
		if appDirEmpty, err = r.FS.IsDirEmpty(absAppDir); err != nil {
			return err
		}
	}

	localYML, err := r.Config.Load()
	if err != nil {
		return err
	}

	droplet, dropletSize, err := r.FS.ReadFile(fmt.Sprintf("./%s.droplet", options.name))
	if err != nil {
		return err
	}
	defer droplet.Close()
	launcher, err := r.Stager.Download("/tmp/lifecycle/launcher")
	if err != nil {
		return err
	}
	defer launcher.Close()

	appConfig := getAppConfig(options.name, localYML)
	remoteServices, forwardConfig, err := getRemoteServices(r.App, options.serviceApp, options.forwardApp)
	if err != nil {
		return err
	}
	if remoteServices != nil {
		appConfig.Services = remoteServices
	}

	forwarder := local.Forwarder{Config: forwardConfig}
	if forwarder.Config != nil {
		if forwarder.SSHPass, err = r.Stager.Download("/usr/bin/sshpass"); err != nil {
			return err
		}
		defer forwarder.SSHPass.Close()
	}

	r.UI.Output("Running %s on port %d...", options.name, options.port)
	_, err = r.Runner.Run(&local.RunConfig{
		Droplet:     local.Stream{droplet, dropletSize},
		Launcher:    launcher,
		Forwarder:   forwarder,
		Port:        options.port,
		AppDir:      absAppDir,
		AppDirEmpty: appDirEmpty,
		AppConfig:   appConfig,
	}, color.GreenString)
	return err
}

func (*Run) options(args []string) (*runOptions, error) {
	set := &flag.FlagSet{}
	set.SetOutput(ioutil.Discard)

	defaultPort, err := freePort()
	if err != nil {
		return nil, err
	}

	options := &runOptions{name: args[1]}
	set.UintVar(&options.port, "p", defaultPort, "")
	set.StringVar(&options.appDir, "d", "", "")
	set.StringVar(&options.serviceApp, "s", "", "")
	set.StringVar(&options.forwardApp, "f", "", "")
	if err := set.Parse(args[2:]); err != nil {
		return nil, err
	}

	if set.NArg() != 0 {
		return nil, errors.New("invalid arguments")
	}
	return options, nil
}

func freePort() (uint, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	address := listener.Addr().String()
	portStr := strings.SplitN(address, ":", 2)[1]
	port, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(port), nil
}
