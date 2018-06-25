package cmd

import (
	"flag"
	"fmt"

	"github.com/buildpack/forge"
	"github.com/buildpack/forge/engine"
)

type Export struct {
	UI       UI
	Exporter Exporter
	Image    Image
	FS       FS
	Help     Help
	Config   Config
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

	dropletFile, dropletSize, err := e.FS.ReadFile(fmt.Sprintf("./%s.droplet", options.name))
	if err != nil {
		return err
	}
	droplet := engine.NewStream(dropletFile, dropletSize)
	defer droplet.Close()

	if err := e.UI.Loading("Image", e.Image.Pull(RunStack)); err != nil {
		return err
	}
	id, err := e.Exporter.Export(&forge.ExportConfig{
		Droplet:    droplet,
		Stack:      RunStack,
		Ref:        options.reference,
		OutputDir:  "/home/vcap",
		WorkingDir: "/home/vcap/app",
		AppConfig:  getAppConfig(options.name, localYML),
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
