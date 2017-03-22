package cmd

import (
	"errors"
	"fmt"
	"io"
)

type Pull struct {
	UI     UI
	App    App
	FS     FS
	Help   Help
	Config Config
}

func (p *Pull) Match(args []string) bool {
	return len(args) > 0 && args[0] == "pull"
}

func (p *Pull) Run(args []string) error {
	if len(args) != 2 {
		p.Help.Short()
		if len(args) < 2 {
			return errors.New("app name required")
		}
		return errors.New("invalid arguments")
	}
	name := args[1]
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
	droplet, size, err := p.App.Droplet(name)
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

	env, err := p.App.Env(name)
	if err != nil {
		return err
	}
	app.StagingEnv = env.Staging
	app.RunningEnv = env.Running
	app.Env = env.App

	command, err := p.App.Command(name)
	if err != nil {
		return err
	}
	app.Command = command

	if err := p.Config.Save(localYML); err != nil {
		return err
	}
	return nil
}
