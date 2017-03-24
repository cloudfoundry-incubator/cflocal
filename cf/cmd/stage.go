package cmd

import (
	"flag"
	"fmt"

	"github.com/fatih/color"

	"github.com/sclevine/cflocal/local"
)

type Stage struct {
	UI     UI
	Stager Stager
	App    App
	FS     FS
	Help   Help
	Config Config
}

type stageOptions struct {
	name, buildpack        string
	serviceApp, forwardApp string
}

func (s *Stage) Match(args []string) bool {
	return len(args) > 0 && args[0] == "stage"
}

func (s *Stage) Run(args []string) error {
	options, err := s.options(args)
	if err != nil {
		s.Help.Short()
		return err
	}

	localYML, err := s.Config.Load()
	if err != nil {
		return err
	}
	appTar, err := s.FS.Tar(".")
	if err != nil {
		return err
	}
	defer appTar.Close()

	appConfig := getAppConfig(options.name, localYML)
	remoteServices, _, err := getRemoteServices(s.App, options.serviceApp, options.forwardApp)
	if err != nil {
		return err
	}
	if remoteServices != nil {
		appConfig.Services = remoteServices
	}
	if sApp, fApp := options.serviceApp, options.forwardApp; sApp != fApp && sApp != "" && fApp != "" {
		s.UI.Warn("'%s' app selected for service forwarding will not be used", fApp)
	}

	lookupURL := Buildpacks[options.buildpack]
	var buildpacks []string
	switch {
	case options.buildpack == "":
		s.UI.Output("Downloading all buildpacks...")
		buildpacks = valuesInOrder(Buildpacks, BuildpackOrder)
	case lookupURL != "":
		s.UI.Output("Downloading %s from %s...", options.buildpack, lookupURL)
		buildpacks = []string{lookupURL}
	default:
		s.UI.Output("Downloading %s...", options.buildpack)
		buildpacks = []string{options.buildpack}
	}
	droplet, err := s.Stager.Stage(&local.StageConfig{
		AppTar:     appTar,
		Buildpacks: buildpacks,
		AppConfig:  appConfig,
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
	if err := droplet.Write(file); err != nil {
		return err
	}
	s.UI.Output("Successfully staged: %s", options.name)
	return nil
}

func (*Stage) options(args []string) (*stageOptions, error) {
	options := &stageOptions{}

	return options, parseOptions(args, func(name string, set *flag.FlagSet) {
		options.name = name
		set.StringVar(&options.buildpack, "b", "", "")
		set.StringVar(&options.serviceApp, "s", "", "")
		set.StringVar(&options.forwardApp, "f", "", "")
	})
}

func valuesInOrder(m map[string]string, l []string) []string {
	var result []string
	for _, lv := range l {
		result = append(result, m[lv])
	}
	return result
}
