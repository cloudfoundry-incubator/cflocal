package cf

import "errors"

type CF struct {
	UI      UI
	Help    Help
	Cmds    []Cmd
	Version string
}

type UI interface {
	Prompt(prompt string) string
	Output(format string, a ...interface{})
	Error(err error)
}

//go:generate mockgen -package mocks -destination mocks/help.go github.com/sclevine/cflocal/cf Help
type Help interface {
	Short()
	Long()
}

//go:generate mockgen -package mocks -destination mocks/cmd.go github.com/sclevine/cflocal/cf Cmd
type Cmd interface {
	Match(args []string) bool
	Run(args []string) error
}

func (c *CF) Run(args []string) error {
	switch args[0] {
	case "help":
		c.Help.Long()
		return nil
	case "version", "--version":
		c.UI.Output("CF Local version %s", c.Version)
		return nil
	}
	for _, cmd := range c.Cmds {
		if cmd.Match(args) {
			return cmd.Run(args)
		}
	}
	c.Help.Short()
	return errors.New("invalid command")
}
