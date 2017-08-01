package cmd

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/docker/docker/api/types"
	"github.com/sclevine/cflocal/engine"
	"github.com/sclevine/cflocal/local"
	"github.com/docker/go-connections/nat"
)

type Run struct {
	UI        UI
	Stager    Stager
	Runner    Runner
	Forwarder Forwarder
	App       App
	FS        FS
	Help      Help
	Config    Config
}

type runOptions struct {
	name, appDir           string
	serviceApp, forwardApp string
	ip                     string
	port                   uint
	rsync, watch           bool
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
	var (
		appDir  string
		restart <-chan time.Time
	)
	if options.appDir != "" {
		if appDir, err = r.FS.Abs(options.appDir); err != nil {
			return err
		}
		if err := r.FS.MakeDirAll(appDir); err != nil {
			return err
		}
		if options.watch {
			var done chan<- struct{}
			restart, done, err = r.FS.Watch(appDir, time.Second)
			if err != nil {
				return err
			}
			defer close(done)
		}
	} else if options.watch {
		return errors.New("-w is only valid with -d")
	} else if options.rsync {
		return errors.New("-r is only valid with -d")
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

	// TODO: create network config, don't leak Docker nat package
	var network string
	portBindings := nat.PortMap{
		"8080/tcp": {{HostIP: options.ip, HostPort: strconv.FormatUint(uint64(options.port), 10)}},
	}
	if forwardConfig != nil {
		sshpass, err := r.Stager.Download("/usr/bin/sshpass")
		if err != nil {
			return err
		}
		defer sshpass.Close()
		health, done, networkMode, err := r.Forwarder.Forward(&local.ForwardConfig{
			AppName:       appConfig.Name,
			SSHPass:       sshpass,
			Color:         color.GreenString,
			ForwardConfig: forwardConfig,
			PortBindings:  portBindings,
		})
		if err != nil {
			return err
		}
		defer done()
		if err := waitForHealthy(health); err != nil {
			return fmt.Errorf("error forwarding services: %s", err)
		}
		network = networkMode
		portBindings = nil
	}

	r.UI.Output("Running %s on port %d...", options.name, options.port)
	_, err = r.Runner.Run(&local.RunConfig{
		Droplet:     engine.NewStream(droplet, dropletSize),
		Launcher:    launcher,
		NetworkMode: network,
		AppDir:      appDir,
		RSync:       options.rsync,
		Restart:     restart,
		Color:       color.GreenString,
		AppConfig:   appConfig,
		PortBindngs: portBindings,
	})
	return err
}

func (*Run) options(args []string) (*runOptions, error) {
	options := &runOptions{}
	defaultPort, err := freePort()
	if err != nil {
		return nil, err
	}

	return options, parseOptions(args, func(name string, set *flag.FlagSet) {
		options.name = name
		set.UintVar(&options.port, "p", defaultPort, "")
		set.StringVar(&options.ip, "i", "127.0.0.1", "")
		set.StringVar(&options.appDir, "d", "", "")
		set.StringVar(&options.serviceApp, "s", "", "")
		set.StringVar(&options.forwardApp, "f", "", "")
		set.BoolVar(&options.rsync, "r", false, "")
		set.BoolVar(&options.watch, "w", false, "")
	})
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

func waitForHealthy(health <-chan string) error {
	timeout := time.NewTimer(30 * time.Second).C
	for {
		select {
		case status := <-health:
			if status == types.Healthy {
				return nil
			}
		case <-timeout:
			return errors.New("timeout")
		}
	}
}
