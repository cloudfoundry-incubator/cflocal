package cmd

import (
	"flag"
	"fmt"
	"io"
)

type Pull struct {
	UI        UI
	RemoteApp RemoteApp
	FS        FS
	Help      Help
	Config    Config
}

func (p *Pull) Match(args []string) bool {
	return len(args) > 0 && args[0] == "pull"
}

func (p *Pull) Run(args []string) error {
	name, err := p.options(args)
	if err != nil {
		p.Help.Short()
		return err
	}
	if err := p.saveDroplet(name); err != nil {
		return err
	}
	if err := p.updateLocalYML(name); err != nil {
		return err
	}
	p.UI.Output("Successfully downloaded: %s", name)
	return nil
}

func (p *Pull) saveDroplet(name string) error {
	droplet, size, err := p.RemoteApp.Droplet(name)
	if err != nil {
		return err
	}
	defer droplet.Close()
	file, err := p.FS.WriteFile(fmt.Sprintf("./%s.droplet", name))
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.CopyN(file, droplet, size); err != nil && err != io.EOF {
		return err
	}
	return nil
}

func (p *Pull) updateLocalYML(name string) error {
	localYML, err := p.Config.Load()
	if err != nil {
		return err
	}
	app := getAppConfig(name, localYML)

	env, err := p.RemoteApp.Env(name)
	if err != nil {
		return err
	}
	app.StagingEnv = env.Staging
	app.RunningEnv = env.Running
	app.Env = env.App

	command, err := p.RemoteApp.Command(name)
	if err != nil {
		return err
	}
	app.Command = command

	if err := p.Config.Save(localYML); err != nil {
		return err
	}
	return nil
}

func (*Pull) options(args []string) (appName string, err error) {
	return appName, parseOptions(args, func(name string, _ *flag.FlagSet) {
		appName = name
	})
}
