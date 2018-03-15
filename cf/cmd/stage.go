package cmd

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io"

	"github.com/fatih/color"

	"github.com/sclevine/forge"
	"github.com/sclevine/forge/engine"
)

type Stage struct {
	UI        UI
	Stager    Stager
	RemoteApp RemoteApp
	TarApp    func(string) (io.ReadCloser, error)
	FS        FS
	Help      Help
	Config    Config
}

type stageOptions struct {
	name        string
	buildpacks  buildpacks
	app         string
	serviceApp  string
	forwardApp  string
	forceDetect bool
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

	appTar, err := s.TarApp(options.app)
	if err != nil {
		return err
	}
	defer appTar.Close()

	appConfig := getAppConfig(options.name, localYML)

	if len(options.buildpacks) > 0 {
		appConfig.Buildpacks = options.buildpacks
		appConfig.Buildpack = options.buildpacks[len(options.buildpacks)-1]
	}
	buildpackZips := map[string]engine.Stream{}
	for _, buildpack := range append([]string{appConfig.Buildpack}, appConfig.Buildpacks...) {
		checksum := fmt.Sprintf("%x", md5.Sum([]byte(buildpack)))
		if _, ok := buildpackZips[checksum]; ok {
			continue
		}
		// TODO: enforce starting with . or /
		zip, size, err := s.FS.ReadFile(buildpack)
		if err != nil {
			continue
		}
		buildpackZip := engine.NewStream(zip, size)
		defer buildpackZip.Close()
		buildpackZips[checksum] = buildpackZip
	}

	remoteServices, _, err := getRemoteServices(s.RemoteApp, options.serviceApp, options.forwardApp)
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

	droplet, err := s.Stager.Stage(&forge.StageConfig{
		AppTar:        appTar,
		Cache:         cache,
		CacheEmpty:    cacheSize == 0,
		BuildpackZips: buildpackZips,
		Stack:         BuildStack,
		ForceDetect:   options.forceDetect,
		Color:         color.GreenString,
		AppConfig:     appConfig,
	})
	if err != nil {
		return err
	}
	defer droplet.Close()

	if err := s.streamOut(droplet, dropletPath); err != nil {
		return err
	}

	s.UI.Output("Successfully staged: %s", options.name)
	return nil
}

func (s *Stage) streamOut(stream engine.Stream, path string) error {
	file, err := s.FS.WriteFile(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return stream.Out(file)
}

func (*Stage) options(args []string) (*stageOptions, error) {
	options := &stageOptions{}

	return options, parseOptions(args, func(name string, set *flag.FlagSet) {
		options.name = name
		set.StringVar(&options.app, "p", ".", "")
		set.Var(&options.buildpacks, "b", "")
		set.StringVar(&options.serviceApp, "s", "", "")
		set.StringVar(&options.forwardApp, "f", "", "")
		set.BoolVar(&options.forceDetect, "e", false, "")
	})
}

type buildpacks []string

func (*buildpacks) String() string {
	return ""
}

func (b *buildpacks) Set(value string) error {
	*b = append(*b, value)
	return nil
}
