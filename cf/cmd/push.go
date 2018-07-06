package cmd

import (
	"flag"
	"fmt"
	"os"
)

type Push struct {
	UI        UI
	RemoteApp RemoteApp
	FS        FS
	Help      Help
	Config    Config
}

type pushOptions struct {
	name      string
	keepState bool
	pushEnv   bool
}

func (p *Push) Match(args []string) bool {
	return len(args) > 0 && args[0] == "push"
}

func (p *Push) Run(args []string) error {
	options, err := p.options(args)
	if err != nil {
		p.Help.Short()
		return err
	}

	if err := p.pushDroplet(options.name); err != nil {
		return err
	}
	if options.pushEnv {
		if err := p.pushEnv(options.name); err != nil {
			return err
		}
	}
	if !options.keepState {
		if err := p.RemoteApp.Restart(options.name); err != nil {
			return err
		}
	}
	p.UI.Output("Successfully pushed: %s", options.name)
	return nil
}

func (p *Push) pushDroplet(name string) error {
	filename := fmt.Sprintf("./%s.droplet", name)
	droplet, size, err := p.FS.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s: Did you provide a filepath instead of an app name?", filename)
		}
		return err
	}
	defer droplet.Close()
	return p.RemoteApp.SetDroplet(name, droplet, size)
}

func (p *Push) pushEnv(name string) error {
	localYML, err := p.Config.Load()
	if err != nil {
		return err
	}
	return p.RemoteApp.SetEnv(name, getAppConfig(name, localYML).Env)
}

func (*Push) options(args []string) (*pushOptions, error) {
	options := &pushOptions{}

	return options, parseOptions(args, func(name string, set *flag.FlagSet) {
		options.name = name
		set.BoolVar(&options.keepState, "k", false, "")
		set.BoolVar(&options.pushEnv, "e", false, "")
	})
}
