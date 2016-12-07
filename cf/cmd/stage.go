package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/sclevine/cflocal/local"

	"github.com/fatih/color"
)

type Stage struct {
	UI     UI
	Stager Stager
	FS     FS
	Help   Help
	Config Config
}

type stageOptions struct {
	name, buildpack string
}

func (s *Stage) Match(args []string) bool {
	return len(args) > 0 && args[0] == "stage"
}

func (s *Stage) Run(args []string) error {
	options, err := s.options(args)
	if err != nil {
		if err := s.Help.Show(); err != nil {
			s.UI.Error(err)
		}
		return err
	}
	appTar, err := s.FS.Tar(".")
	if err != nil {
		return err
	}
	defer appTar.Close()
	localYML, err := s.Config.Load()
	if err != nil {
		return err
	}
	var buildpacks []string
	switch options.buildpack {
	case "":
		s.UI.Output("Downloading all buildpacks...")
		buildpacks = Buildpacks
	default:
		s.UI.Output("Downloading %s...", options.buildpack)
		buildpacks = []string{options.buildpack}
	}
	droplet, size, err := s.Stager.Stage(&local.StageConfig{
		AppTar:     appTar,
		Buildpacks: buildpacks,
		AppConfig:  getAppConfig(options.name, localYML),
	}, color.GreenString)
	if err != nil {
		return err
	}
	defer droplet.Close()
	file, err := s.FS.WriteFile(fmt.Sprintf("./%s.droplet", options.name))
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.CopyN(file, droplet, size); err != nil && err != io.EOF {
		return err
	}
	s.UI.Output("Successfully staged: %s", options.name)
	return nil
}

func (*Stage) options(args []string) (*stageOptions, error) {
	set := &flag.FlagSet{}
	options := &stageOptions{}
	set.StringVar(&options.buildpack, "b", "", "")
	if err := set.Parse(args[1:]); err != nil {
		return nil, err
	}
	if set.NArg() != 1 {
		return nil, errors.New("invalid arguments")
	}
	options.name = set.Arg(0)
	return options, nil
}
