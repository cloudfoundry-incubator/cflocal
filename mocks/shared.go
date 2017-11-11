package mocks

import "code.cloudfoundry.org/cli/plugin"

//go:generate mockgen -package mocks -destination cli_connection.go code.cloudfoundry.org/cflocal/mocks CliConnection
type CliConnection interface {
	plugin.CliConnection
}
