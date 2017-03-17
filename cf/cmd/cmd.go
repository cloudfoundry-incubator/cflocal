package cmd

import (
	"io"

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
	Stage(config *local.StageConfig, color local.Colorizer) (droplet local.Stream, err error)
	Download(path string) (stream local.Stream, err error)
}

//go:generate mockgen -package mocks -destination mocks/runner.go github.com/sclevine/cflocal/cf/cmd Runner
type Runner interface {
	Run(config *local.RunConfig, color local.Colorizer) (status int, err error)
	Export(config *local.ExportConfig, reference string) (imageID string, err error)
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/sclevine/cflocal/cf/cmd FS
type FS interface {
	Tar(path string) (io.ReadCloser, error)
	ReadFile(path string) (io.ReadCloser, int64, error)
	WriteFile(path string) (io.WriteCloser, error)
	MakeDirAll(path string) error
	IsDirEmpty(path string) (bool, error)
	Abs(path string) (string, error)
}

//go:generate mockgen -package mocks -destination mocks/help.go github.com/sclevine/cflocal/cf/cmd Help
type Help interface {
	Show() error
}

//go:generate mockgen -package mocks -destination mocks/config.go github.com/sclevine/cflocal/cf/cmd Config
type Config interface {
	Load() (*local.LocalYML, error)
	Save(localYML *local.LocalYML) error
}

//go:generate mockgen -package mocks -destination mocks/closer.go io Closer

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
