package plugin

import cfplugin "code.cloudfoundry.org/cli/plugin"

type Help struct {
	CLI cfplugin.CliConnection
	UI  UI
}

func (h *Help) Short() {
	h.UI.Output("Usage:%s\n", ShortUsage)
}

func (h *Help) Long() {
	if _, err := h.CLI.CliCommand("help", "local"); err != nil {
		h.UI.Error(err)
	}
}
