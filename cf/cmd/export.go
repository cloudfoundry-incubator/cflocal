package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/sclevine/cflocal/local"
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
	name, reference string
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
	launcher, err := e.Stager.Download("/tmp/lifecycle/launcher")
	if err != nil {
		return err
	}
	defer launcher.Close()

	id, err := e.Runner.Export(&local.ExportConfig{
		Droplet:   local.NewStream(droplet, dropletSize),
		Launcher:  launcher,
		AppConfig: getAppConfig(options.name, localYML),
	}, options.reference)
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

// TODO: refactor to invert control
func (*Export) options(args []string) (*exportOptions, error) {
	if len(args) < 2 {
		return nil, errors.New("app name required")
	}
	options := &exportOptions{name: args[1]}
	set := &flag.FlagSet{}
	set.SetOutput(ioutil.Discard)
	set.StringVar(&options.reference, "r", "", "")
	if err := set.Parse(args[2:]); err != nil {
		return nil, err
	}
	if set.NArg() != 0 {
		return nil, errors.New("invalid arguments")
	}
	return options, nil
}
