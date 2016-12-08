package cmd

import (
	"io"

	"github.com/sclevine/cflocal/local"
	"github.com/sclevine/cflocal/remote"
)

type UI interface {
	Prompt(prompt string) string
	Output(format string, a ...interface{})
	Error(err error)
}

//go:generate mockgen -package mocks -destination mocks/app.go github.com/sclevine/cflocal/cf/cmd App
type App interface {
	Droplet(name string) (droplet io.ReadCloser, size int64, err error)
	Command(name string) (string, error)
	Env(name string) (*remote.AppEnv, error)
}

//go:generate mockgen -package mocks -destination mocks/stager.go github.com/sclevine/cflocal/cf/cmd Stager
type Stager interface {
	Stage(config *local.StageConfig, color local.Colorizer) (droplet io.ReadCloser, size int64, err error)
	Launcher() (launcher io.ReadCloser, size int64, err error)
}

//go:generate mockgen -package mocks -destination mocks/runner.go github.com/sclevine/cflocal/cf/cmd Runner
type Runner interface {
	Run(config *local.RunConfig, color local.Colorizer) (status int, err error)
	Export(config *local.RunConfig, reference string) (imageID string, err error)
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
