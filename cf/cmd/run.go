package cmd

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/sclevine/cflocal/local"

	"github.com/fatih/color"
)

type Run struct {
	UI     UI
	Stager Stager
	Runner Runner
	FS     FS
	Help   Help
	Config Config
}

type runOptions struct {
	name string
	port uint
}

func (r *Run) Match(args []string) bool {
	return len(args) > 0 && args[0] == "run"
}

func (r *Run) Run(args []string) error {
	options, err := r.options(args)
	if err != nil {
		if err := r.Help.Show(); err != nil {
			r.UI.Error(err)
		}
		return err
	}
	droplet, dropletSize, err := r.FS.ReadFile(fmt.Sprintf("./%s.droplet", options.name))
	if err != nil {
		return err
	}
	defer droplet.Close()
	launcher, launcherSize, err := r.Stager.Launcher()
	if err != nil {
		return err
	}
	defer launcher.Close()
	localYML, err := r.Config.Load()
	if err != nil {
		return err
	}
	r.UI.Output("Running %s on port %d...", options.name, options.port)
	_, err = r.Runner.Run(&local.RunConfig{
		Droplet:      droplet,
		DropletSize:  dropletSize,
		Launcher:     launcher,
		LauncherSize: launcherSize,
		Port:         options.port,
		AppConfig:    getAppConfig(options.name, localYML),
	}, color.GreenString)
	return err
}

func (*Run) options(args []string) (*runOptions, error) {
	set := &flag.FlagSet{}
	defaultPort, err := freePort()
	if err != nil {
		return nil, err
	}
	options := &runOptions{}
	set.UintVar(&options.port, "p", defaultPort, "")
	if err := set.Parse(args[1:]); err != nil {
		return nil, err
	}
	if set.NArg() != 1 {
		return nil, errors.New("invalid arguments")
	}
	options.name = set.Arg(0)
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
