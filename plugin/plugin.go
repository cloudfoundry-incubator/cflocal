package plugin

import (
	"errors"
	"fmt"
	"io"

	"github.com/sclevine/cflocal/app"

	cfplugin "code.cloudfoundry.org/cli/plugin"
	"github.com/fatih/color"
)

type Plugin struct {
	UI      UserInterface
	Stager  Stager
	Runner  Runner
	FS      FS
	Version cfplugin.VersionType
}

type UserInterface interface {
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
func (p *Plugin) Run(cliConnection cfplugin.CliConnection, args []string) {
	if args[0] == "CLI-MESSAGE-UNINSTALL" {
		return
	}

	switch args[1] {
	case "help":
		p.help(cliConnection)
	case "version", "--version":
		p.version(cliConnection)
	case "stage":
		p.stage(cliConnection, args[2:])
	case "run":
		p.run(cliConnection, args[2:])
	}
}

func (p *Plugin) help(cliConnection cfplugin.CliConnection) {
	if _, err := cliConnection.CliCommand("help", "local"); err != nil {
		p.UI.Error(err)
	}
}

func (p *Plugin) version(cliConnection cfplugin.CliConnection) {
	p.UI.Output("CF Local version %d.%d.%d", p.Version.Major, p.Version.Minor, p.Version.Build)
}

//go:generate mockgen -package mocks -destination mocks/closer.go io Closer
func (p *Plugin) stage(cliConnection cfplugin.CliConnection, args []string) {
	if len(args) != 1 {
		p.help(cliConnection)
		p.UI.Error(errors.New("invalid arguments"))
		return
	}
	name := args[0]
	appTar, err := p.FS.Tar(".")
	if err != nil {
		p.UI.Error(err)
		return
	}
	defer appTar.Close()
	droplet, size, err := p.Stager.Stage(name, color.GreenString, &app.StageConfig{
		AppTar:     appTar,
		Buildpacks: Buildpacks,
	})
	if err != nil {
		p.UI.Error(err)
		return
	}
	defer droplet.Close()
	file, err := p.FS.WriteFile(fmt.Sprintf("./%s.droplet", name))
	if err != nil {
		p.UI.Error(err)
		return
	}
	defer file.Close()
	if _, err := io.CopyN(file, droplet, size); err != nil && err != io.EOF {
		p.UI.Error(err)
		return
	}
	p.UI.Output("Staging of %s successful.", name)
}

func (p *Plugin) run(cliConnection cfplugin.CliConnection, args []string) {
	if len(args) != 1 {
		p.help(cliConnection)
		p.UI.Error(errors.New("invalid arguments"))
		return
	}
	name := args[0]
	droplet, dropletSize, err := p.FS.ReadFile(fmt.Sprintf("./%s.droplet", name))
	if err != nil {
		p.UI.Error(err)
		return
	}
	defer droplet.Close()
	launcher, launcherSize, err := p.Stager.Launcher()
	if err != nil {
		p.UI.Error(err)
		return
	}
	defer launcher.Close()
	p.UI.Output("Running %s...", name)
	_, err = p.Runner.Run(name, color.GreenString, &app.RunConfig{
		Droplet:      droplet,
		DropletSize:  dropletSize,
		Launcher:     launcher,
		LauncherSize: launcherSize,
		Port:         3000,
	})
	if err != nil {
		p.UI.Error(err)
		return
	}
}

func (p *Plugin) GetMetadata() cfplugin.PluginMetadata {
	return cfplugin.PluginMetadata{
		Name:    "cflocal",
		Version: p.Version,
		Commands: []cfplugin.Command{
			cfplugin.Command{
				Name:     "local",
				HelpText: "Build and launch Cloud Foundry applications locally",
				UsageDetails: cfplugin.Usage{
					Usage: `cf local SUBCOMMAND

SUBCOMMANDS:
   stage <name>  Build a droplet from the app in the current directory.
   help          Output this help text.
   version       Output the CF Local version.`,
				},
			},
		},
	}
}
