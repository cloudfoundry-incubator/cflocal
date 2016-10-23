package plugin

import cfplugin "code.cloudfoundry.org/cli/plugin"

type Help struct {
	CLI cfplugin.CliConnection
}

func (h *Help) Show() error {
	_, err := h.CLI.CliCommand("help", "local")
	return err
}
