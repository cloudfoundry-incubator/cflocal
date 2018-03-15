package plugin

import "code.cloudfoundry.org/cflocal/cfplugin"

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
