package plugin

import cfplugin "code.cloudfoundry.org/cli/plugin"

type Plugin struct {
	UI      UI
	Version cfplugin.VersionType
}

type UI interface {
	Failed(message string, args ...interface{})
	Say(message string, args ...interface{})
	Ask(prompt string) (answer string)
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
	case "build":
		p.UI.Say("Built.")
	}
}

func (p *Plugin) help(cliConnection cfplugin.CliConnection) {
	if _, err := cliConnection.CliCommand("help", "local"); err != nil {
		p.UI.Failed("Error: %s.", err)
	}
}

func (p *Plugin) version(cliConnection cfplugin.CliConnection) {
	p.UI.Say("CF Local version %d.%d.%d", p.Version.Major, p.Version.Minor, p.Version.Build)
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
   build    Build app.
   help     Output this help text.
   version  Output the CF Local version.`,
				},
			},
		},
	}
}
