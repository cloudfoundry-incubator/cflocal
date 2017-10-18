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
	"github.com/sclevine/forge/engine"
	"github.com/sclevine/forge"
	"github.com/sclevine/forge/wait"
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
	name       string
	appDir     string
	serviceApp string
	forwardApp string
	ip         string
	port       uint
	rsync      bool
	watch      bool
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

	netConfig := &forge.NetworkConfig{
		HostIP:   options.ip,
		HostPort: strconv.FormatUint(uint64(options.port), 10),
	}
	if forwardConfig != nil {
		sshpass, err := r.Stager.Download("/usr/bin/sshpass")
		if err != nil {
			return err
		}
		defer sshpass.Close()
		waiter, waiterDone := wait.New(5 * time.Second)
		defer waiterDone()
		health, done, id, err := r.Forwarder.Forward(&forge.ForwardConfig{
			AppName:       appConfig.Name,
			SSHPass:       sshpass,
			Color:         color.GreenString,
			ForwardConfig: forwardConfig,
			HostIP:        netConfig.HostIP,
			HostPort:      netConfig.HostPort,
			Wait:          waiter,
		})
		if err != nil {
			return err
		}
		defer done()
		if err := waitForHealthy(health); err != nil {
			return fmt.Errorf("error forwarding services: %s", err)
		}
		netConfig.ContainerID = id
	}

	r.UI.Output("Running %s on port %d...", options.name, options.port)
	_, err = r.Runner.Run(&forge.RunConfig{
		Droplet:       engine.NewStream(droplet, dropletSize),
		Launcher:      launcher,
		AppDir:        appDir,
		RSync:         options.rsync,
		Restart:       restart,
		Color:         color.GreenString,
		AppConfig:     appConfig,
		NetworkConfig: netConfig,
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

// TODO: replace with contr.WaitFor("healthy")
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
