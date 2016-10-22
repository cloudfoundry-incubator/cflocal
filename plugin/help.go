package plugin

import cfplugin "code.cloudfoundry.org/cli/plugin"

type Help struct {
	CLI cfplugin.CliConnection
}

//go:generate mockgen -package mocks -destination mocks/cli_connection.go code.cloudfoundry.org/cli/plugin CliConnection

func (h *Help) Show() error {
	_, err := h.CLI.CliCommand("help", "local")
	return err
}
