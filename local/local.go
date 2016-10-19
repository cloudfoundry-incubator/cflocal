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
	UI       UI
	Stager   Stager
	Runner   Runner
	FS       FS
	CLI      cfplugin.CliConnection
	Version  cfplugin.VersionType
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

func (c *CF) Run(args []string) {
	switch args[0] {
	case "help":
		c.help()
	case "version", "--version":
		c.version()
	case "stage":
		c.stage(args[1:])
	case "run":
		c.run(args[1:])
	default:
		c.UI.Error(errors.New("invalid command"))
	}
}

func (c *CF) help() {
	// move cliCommand into plugin.UI, add UI.Help()
	// downloader should live in plugin package as plugin.DropletDownloader
	// if a nil downloader is passed, assume non-plugin version (and eventually allow URL to droplet)
	if _, err := c.CLI.CliCommand("help", "local"); err != nil {
		c.UI.Error(err)
	}
}

func (c *CF) version() {
	c.UI.Output("CF Local version %d.%d.%d", c.Version.Major, c.Version.Minor, c.Version.Build)
}

//go:generate mockgen -package mocks -destination mocks/closer.go io Closer
func (c *CF) stage(args []string) {
	if len(args) != 1 {
		c.help()
		c.UI.Error(errors.New("invalid arguments"))
		return
	}
	name := args[0]
	appTar, err := c.FS.Tar(".")
	if err != nil {
		c.UI.Error(err)
		return
	}
	defer appTar.Close()
	droplet, size, err := c.Stager.Stage(name, color.GreenString, &app.StageConfig{
		AppTar:     appTar,
		Buildpacks: Buildpacks,
	})
	if err != nil {
		c.UI.Error(err)
		return
	}
	defer droplet.Close()
	file, err := c.FS.WriteFile(fmt.Sprintf("./%s.droplet", name))
	if err != nil {
		c.UI.Error(err)
		return
	}
	defer file.Close()
	if _, err := io.CopyN(file, droplet, size); err != nil && err != io.EOF {
		c.UI.Error(err)
		return
	}
	c.UI.Output("Staging of %s successful.", name)
}

func (c *CF) run(args []string) {
	if len(args) != 1 {
		c.help()
		c.UI.Error(errors.New("invalid arguments"))
		return
	}
	name := args[0]
	droplet, dropletSize, err := c.FS.ReadFile(fmt.Sprintf("./%s.droplet", name))
	if err != nil {
		c.UI.Error(err)
		return
	}
	defer droplet.Close()
	launcher, launcherSize, err := c.Stager.Launcher()
	if err != nil {
		c.UI.Error(err)
		return
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
	if err != nil {
		c.UI.Error(err)
		return
	}
}
