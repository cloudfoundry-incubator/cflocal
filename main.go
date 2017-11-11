package main

import (
	"os"
	"os/signal"
	"syscall"

	cfplugin "code.cloudfoundry.org/cli/plugin"
	"github.com/fatih/color"
	"golang.org/x/crypto/ssh/terminal"

	"code.cloudfoundry.org/cflocal/plugin"
	"code.cloudfoundry.org/cflocal/ui"
)

var Version = "0.0.0"

func main() {
	exitChan := make(chan struct{})
	signalChan := make(chan os.Signal, 1)

	signal.Notify(make(chan os.Signal), syscall.SIGHUP)
	signal.Notify(signalChan, syscall.SIGINT)
	signal.Notify(signalChan, syscall.SIGTERM)
	go func() {
		<-signalChan
		close(exitChan)
	}()

	// TODO: cut off ui.Out, etc. on sigint
	ui := &ui.UI{
		Out:       color.Output,
		Err:       os.Stderr,
		In:        os.Stdin,
		ErrIsTerm: terminal.IsTerminal(int(os.Stderr.Fd())),
	}

	cflocal := &plugin.Plugin{
		UI:      ui,
		Version: Version,
		Exit:    exitChan,
	}

	if len(os.Args) > 1 && os.Args[1] != "" {
		switch os.Args[1] {
		case "help", "-h", "--help":
			cflocal.Help(os.Args[0])
		default:
			cfplugin.Start(cflocal)
		}
		select {
		case <-exitChan:
			os.Exit(128)
		default:
			if err := cflocal.RunErr; err != nil {
				ui.Error(err)
				os.Exit(1)
			}
		}
	} else if err := cflocal.Install(); err != nil {
		ui.Error(err)
		os.Exit(1)
	}
}
