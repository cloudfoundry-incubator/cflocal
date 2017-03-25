package main

import (
	"os"

	cfplugin "code.cloudfoundry.org/cli/plugin"
	"github.com/fatih/color"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/sclevine/cflocal/plugin"
	"github.com/sclevine/cflocal/ui"
)

var Version = "0.0.0"

func main() {
	ui := &ui.UI{
		Out:       color.Output,
		Err:       os.Stderr,
		In:        os.Stdin,
		ErrIsTerm: terminal.IsTerminal(int(os.Stderr.Fd())),
	}

	cflocal := &plugin.Plugin{
		UI:      ui,
		Version: Version,
		Exit:    make(chan struct{}),
	}

	if len(os.Args) > 1 && os.Args[1] != "" {
		switch os.Args[1] {
		case "help", "-h", "--help":
			cflocal.Help(os.Args[0])
		default:
			cfplugin.Start(cflocal)
		}
		select {
		case <-cflocal.Exit:
			os.Exit(128)
		default:
			if err := cflocal.RunErr; err != nil {
				ui.Error(err)
				os.Exit(1)
			}
		}

		os.Exit(0)
	}

	if err := cflocal.Install(); err != nil {
		ui.Error(err)
		os.Exit(1)
	}
}
