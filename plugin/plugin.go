package plugin

import (
	"os"
	"os/signal"
	"syscall"

	docker "github.com/docker/docker/client"
	"github.com/sclevine/cflocal/app"
	"github.com/sclevine/cflocal/local"
	"github.com/sclevine/cflocal/utils"

	cfplugin "code.cloudfoundry.org/cli/plugin"
)

type Plugin struct {
	UI      UserInterface
	Version cfplugin.VersionType
}

type UserInterface interface {
	Prompt(prompt string) string
	Output(format string, a ...interface{})
	Error(err error)
}

func (p *Plugin) Run(cliConnection cfplugin.CliConnection, args []string) {
	if args[0] == "CLI-MESSAGE-UNINSTALL" {
		return
	}

	signal.Notify(make(chan os.Signal), syscall.SIGHUP)
	quitChan := make(chan os.Signal, 1)
	signal.Notify(quitChan, syscall.SIGINT)
	exitChan := make(chan struct{})
	go func() {
		<-quitChan
		close(exitChan)
	}()

	client, err := docker.NewEnvClient()
	if err != nil {
		p.UI.Error(err)
		os.Exit(1)
	}
	cf := &local.CF{
		UI: p.UI,
		Stager: &app.Stager{
			DiegoVersion: "0.1482.0",
			GoVersion:    "1.7",
			UpdateRootFS: true,
			Docker:       client,
			Logs:         os.Stdout,
		},
		Runner: &app.Runner{
			Docker:   client,
			Logs:     os.Stdout,
			ExitChan: exitChan,
		},
		FS:      &utils.FS{},
		CLI:     cliConnection,
		Version: p.Version,
	}
	cf.Run(args[1:])
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
