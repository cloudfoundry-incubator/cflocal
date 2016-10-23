package cf

import (
	"errors"
	"fmt"
	"io"

	"github.com/sclevine/cflocal/local"
	"github.com/sclevine/cflocal/remote"

	"github.com/fatih/color"
)

type CF struct {
	UI      UI
	Stager  Stager
	Runner  Runner
	App     App
	FS      FS
	Help    Help
	Version string
}

type UI interface {
	Prompt(prompt string) string
	Output(format string, a ...interface{})
	Error(err error)
}

//go:generate mockgen -package mocks -destination mocks/stager.go github.com/sclevine/cflocal/cf Stager
type Stager interface {
	Stage(name string, color local.Colorizer, config *local.StageConfig) (droplet io.ReadCloser, size int64, err error)
	Launcher() (launcher io.ReadCloser, size int64, err error)
}

//go:generate mockgen -package mocks -destination mocks/runner.go github.com/sclevine/cflocal/cf Runner
type Runner interface {
	Run(name string, color local.Colorizer, config *local.RunConfig) (status int, err error)
}

//go:generate mockgen -package mocks -destination mocks/app.go github.com/sclevine/cflocal/cf App
type App interface {
	Droplet(name string) (io.ReadCloser, error)
	Env(name string) (*remote.AppEnv, error)
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/sclevine/cflocal/cf FS
type FS interface {
	Tar(path string) (io.ReadCloser, error)
	ReadFile(path string) (io.ReadCloser, int64, error)
	WriteFile(path string) (io.WriteCloser, error)
}

//go:generate mockgen -package mocks -destination mocks/help.go github.com/sclevine/cflocal/cf Help
type Help interface {
	Show() error
}

func (c *CF) Run(args []string) error {
	var err error
	switch args[0] {
	case "help":
		err = c.Help.Show()
	case "version", "--version":
		c.version()
	case "stage":
		err = c.stage(args[1:])
	case "run":
		err = c.run(args[1:])
	case "pull":
		err = c.pull(args[1:])
	default:
		return errors.New("invalid command")
	}
	return err
}

func (c *CF) version() {
	c.UI.Output("CF Local version %s", c.Version)
}

//go:generate mockgen -package mocks -destination mocks/closer.go io Closer
func (c *CF) stage(args []string) error {
	if len(args) != 1 {
		if err := c.Help.Show(); err != nil {
			c.UI.Error(err)
		}
		return errors.New("invalid arguments")
	}
	name := args[0]
	appTar, err := c.FS.Tar(".")
	if err != nil {
		return err
	}
	defer appTar.Close()
	droplet, size, err := c.Stager.Stage(name, color.GreenString, &local.StageConfig{
		AppTar:     appTar,
		Buildpacks: Buildpacks,
	})
	if err != nil {
		return err
	}
	defer droplet.Close()
	file, err := c.FS.WriteFile(fmt.Sprintf("./%s.droplet", name))
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.CopyN(file, droplet, size); err != nil && err != io.EOF {
		return err
	}
	c.UI.Output("Staging of %s successful.", name)
	return nil
}

func (c *CF) run(args []string) error {
	if len(args) != 1 {
		if err := c.Help.Show(); err != nil {
			c.UI.Error(err)
		}
		return errors.New("invalid arguments")
	}
	name := args[0]
	droplet, dropletSize, err := c.FS.ReadFile(fmt.Sprintf("./%s.droplet", name))
	if err != nil {
		return err
	}
	defer droplet.Close()
	launcher, launcherSize, err := c.Stager.Launcher()
	if err != nil {
		return err
	}
	defer launcher.Close()
	c.UI.Output("Running %s...", name)
	_, err = c.Runner.Run(name, color.GreenString, &local.RunConfig{
		Droplet:      droplet,
		DropletSize:  dropletSize,
		Launcher:     launcher,
		LauncherSize: launcherSize,
		Port:         3000,
	})
	return err
}

func (c *CF) pull(args []string) error {
	//c.App.Pull()
	return nil
}
