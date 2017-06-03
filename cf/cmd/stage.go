package cmd

import (
	"flag"
	"fmt"

	"github.com/fatih/color"

	"github.com/sclevine/cflocal/engine"
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
	name, buildpack, app   string
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

	dropletPath := fmt.Sprintf("./%s.droplet", options.name)
	cachePath := fmt.Sprintf("./.%s.cache", options.name)

	localYML, err := s.Config.Load()
	if err != nil {
		return err
	}
	appTar, err := s.FS.TarApp(options.app)
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

	cache, cacheSize, err := s.FS.OpenFile(cachePath)
	if err != nil {
		return err
	}
	defer cache.Close()

	droplet, err := s.Stager.Stage(&local.StageConfig{
		AppTar:     appTar,
		Cache:      cache,
		CacheEmpty: cacheSize == 0,
		Buildpack:  options.buildpack,
		Color:      color.GreenString,
		AppConfig:  appConfig,
	})
	if err != nil {
		return err
	}

	if err := s.streamOut(droplet, dropletPath); err != nil {
		return err
	}

	s.UI.Output("Successfully staged: %s", options.name)
	return nil
}

func (*Stage) options(args []string) (*stageOptions, error) {
	options := &stageOptions{}

	return options, parseOptions(args, func(name string, set *flag.FlagSet) {
		options.name = name
		set.StringVar(&options.app, "p", ".", "")
		set.StringVar(&options.buildpack, "b", "", "")
		set.StringVar(&options.serviceApp, "s", "", "")
		set.StringVar(&options.forwardApp, "f", "", "")
	})
}

func (s *Stage) streamOut(stream engine.Stream, path string) error {
	file, err := s.FS.WriteFile(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return stream.Out(file)
}
