package cmd

import (
	"flag"
	"fmt"

	"github.com/sclevine/forge"
	"github.com/sclevine/forge/engine"
)

type Export struct {
	UI     UI
	Stager Stager
	Runner Runner
	FS     FS
	Help   Help
	Config Config
}

type exportOptions struct {
	name      string
	reference string
}

func (e *Export) Match(args []string) bool {
	return len(args) > 0 && args[0] == "export"
}

func (e *Export) Run(args []string) error {
	options, err := e.options(args)
	if err != nil {
		e.Help.Short()
		return err
	}
	localYML, err := e.Config.Load()
	if err != nil {
		return err
	}

	droplet, dropletSize, err := e.FS.ReadFile(fmt.Sprintf("./%s.droplet", options.name))
	if err != nil {
		return err
	}
	defer droplet.Close()
	launcher, err := e.Stager.Download("/tmp/lifecycle/launcher", LatestStack)
	if err != nil {
		return err
	}
	defer launcher.Close()

	id, err := e.Runner.Export(&forge.ExportConfig{
		Droplet:   engine.NewStream(droplet, dropletSize),
		Launcher:  launcher,
		Stack:     LatestStack,
		Ref:       options.reference,
		AppConfig: getAppConfig(options.name, localYML),
	})
	if err != nil {
		return err
	}
	if options.reference != "" {
		e.UI.Output("Exported %s as %s with ID: %s", options.name, options.reference, id)
	} else {
		e.UI.Output("Exported %s with ID: %s", options.name, id)
	}
	return nil
}

func (*Export) options(args []string) (*exportOptions, error) {
	options := &exportOptions{}

	return options, parseOptions(args, func(name string, set *flag.FlagSet) {
		options.name = name
		set.StringVar(&options.reference, "r", "", "")
	})
}
