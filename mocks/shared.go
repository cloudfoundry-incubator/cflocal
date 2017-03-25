package mocks

import "code.cloudfoundry.org/cli/plugin"

//go:generate mockgen -package mocks -destination cli_connection.go github.com/sclevine/cflocal/mocks CliConnection
type CliConnection interface {
	plugin.CliConnection
}
