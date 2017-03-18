package plugin

import cfplugin "code.cloudfoundry.org/cli/plugin"

type Help struct {
	CLI cfplugin.CliConnection
	UI  PluginUI
}

type HelpUI interface {
	Output(format string, a ...interface{})
	Error(err error)
}

func (h *Help) Short() {
	h.UI.Output("Usage:%s\n", ShortUsage)
}

func (h *Help) Long() {
	if _, err := h.CLI.CliCommand("help", "local"); err != nil {
		h.UI.Error(err)
	}
}
