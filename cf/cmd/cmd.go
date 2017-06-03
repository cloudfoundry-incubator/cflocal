package cmd

import (
	"errors"
	"flag"
	"io"
	"io/ioutil"

	"github.com/sclevine/cflocal/engine"
	"github.com/sclevine/cflocal/fs"
	"github.com/sclevine/cflocal/local"
	"github.com/sclevine/cflocal/remote"
	"github.com/sclevine/cflocal/service"
)

type UI interface {
	Prompt(prompt string) string
	Output(format string, a ...interface{})
	Warn(format string, a ...interface{})
	Error(err error)
}

//go:generate mockgen -package mocks -destination mocks/app.go github.com/sclevine/cflocal/cf/cmd App
type App interface {
	Command(name string) (string, error)
	Droplet(name string) (droplet io.ReadCloser, size int64, err error)
	SetDroplet(name string, droplet io.Reader, size int64) error
	Env(name string) (*remote.AppEnv, error)
	SetEnv(name string, env map[string]string) error
	Restart(name string) error
	Services(name string) (service.Services, error)
	Forward(name string, services service.Services) (service.Services, *service.ForwardConfig, error)
}

//go:generate mockgen -package mocks -destination mocks/stager.go github.com/sclevine/cflocal/cf/cmd Stager
type Stager interface {
	Stage(config *local.StageConfig) (droplet engine.Stream, err error)
	Download(path string) (stream engine.Stream, err error)
}

//go:generate mockgen -package mocks -destination mocks/runner.go github.com/sclevine/cflocal/cf/cmd Runner
type Runner interface {
	Run(config *local.RunConfig) (status int64, err error)
	Export(config *local.ExportConfig) (imageID string, err error)
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/sclevine/cflocal/cf/cmd FS
type FS interface {
	TarApp(path string) (io.ReadCloser, error)
	ReadFile(path string) (io.ReadCloser, int64, error)
	WriteFile(path string) (io.WriteCloser, error)
	OpenFile(path string) (fs.ReadResetWriteCloser, int64, error)
	MakeDirAll(path string) error
	IsDir(path string) (bool, error)
	IsDirEmpty(path string) (bool, error)
	Abs(path string) (string, error)
}

//go:generate mockgen -package mocks -destination mocks/help.go github.com/sclevine/cflocal/cf/cmd Help
type Help interface {
	Short()
}

//go:generate mockgen -package mocks -destination mocks/config.go github.com/sclevine/cflocal/cf/cmd Config
type Config interface {
	Load() (*local.LocalYML, error)
	Save(localYML *local.LocalYML) error
}

//go:generate mockgen -package mocks -destination mocks/closer.go io Closer

func parseOptions(args []string, f func(name string, set *flag.FlagSet)) error {
	if len(args) < 2 {
		return errors.New("app name required")
	}
	set := &flag.FlagSet{}
	set.SetOutput(ioutil.Discard)
	f(args[1], set)
	if err := set.Parse(args[2:]); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("invalid arguments")
	}
	return nil
}

func getAppConfig(name string, localYML *local.LocalYML) *local.AppConfig {
	var app *local.AppConfig
	for _, appConfig := range localYML.Applications {
		if appConfig.Name == name {
			app = appConfig
		}
	}
	if app == nil {
		app = &local.AppConfig{Name: name}
		localYML.Applications = append(localYML.Applications, app)
	}
	return app
}

func getRemoteServices(app App, serviceApp, forwardApp string) (service.Services, *service.ForwardConfig, error) {
	var services service.Services

	if serviceApp == "" {
		serviceApp = forwardApp
	}
	if serviceApp != "" {
		var err error
		if services, err = app.Services(serviceApp); err != nil {
			return nil, nil, err
		}
	}
	if forwardApp != "" {
		return app.Forward(forwardApp, services)
	}
	return services, nil, nil
}
