package local

import (
	"errors"
	"fmt"
	"io"

	"github.com/sclevine/cflocal/app"

	cfplugin "code.cloudfoundry.org/cli/plugin"
	"github.com/fatih/color"
)

type CF struct {
	UI      UI
	Stager  Stager
	Runner  Runner
	FS      FS
	CLI     cfplugin.CliConnection
	Version string
}

type UI interface {
	Prompt(prompt string) string
	Output(format string, a ...interface{})
	Error(err error)
}

//go:generate mockgen -package mocks -destination mocks/stager.go github.com/sclevine/cflocal/plugin Stager
type Stager interface {
	Stage(name string, color app.Colorizer, config *app.StageConfig) (droplet io.ReadCloser, size int64, err error)
	Launcher() (launcher io.ReadCloser, size int64, err error)
}

//go:generate mockgen -package mocks -destination mocks/runner.go github.com/sclevine/cflocal/plugin Runner
type Runner interface {
	Run(name string, color app.Colorizer, config *app.RunConfig) (status int, err error)
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/sclevine/cflocal/plugin FS
type FS interface {
	Tar(path string) (io.ReadCloser, error)
	ReadFile(path string) (io.ReadCloser, int64, error)
	WriteFile(path string) (io.WriteCloser, error)
}

//go:generate mockgen -package mocks -destination mocks/cli_connection.go code.cloudfoundry.org/cli/plugin CliConnection

func (c *CF) Run(args []string) error {
	var err error
	switch args[0] {
	case "help":
		err = c.help()
	case "version", "--version":
		c.version()
	case "stage":
		err = c.stage(args[1:])
	case "run":
		err = c.run(args[1:])
	default:
		return errors.New("invalid command")
	}
	return err
}

func (c *CF) help() error {
	// move cliCommand into plugin.UI, add UI.Help()
	// downloader should live in plugin package as plugin.DropletDownloader
	// if a nil downloader is passed, assume non-plugin version (and eventually allow URL to droplet)
	_, err := c.CLI.CliCommand("help", "local")
	return err
}

func (c *CF) version() {
	c.UI.Output("CF Local version %s", c.Version)
}

//go:generate mockgen -package mocks -destination mocks/closer.go io Closer
func (c *CF) stage(args []string) error {
	if len(args) != 1 {
		if err := c.help(); err != nil {
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
	droplet, size, err := c.Stager.Stage(name, color.GreenString, &app.StageConfig{
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
		if err := c.help(); err != nil {
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
	_, err = c.Runner.Run(name, color.GreenString, &app.RunConfig{
		Droplet:      droplet,
		DropletSize:  dropletSize,
		Launcher:     launcher,
		LauncherSize: launcherSize,
		Port:         3000,
	})
	return err
}
