package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	goversion "github.com/hashicorp/go-version"
	"github.com/sclevine/cflocal/plugin"

	cfplugin "code.cloudfoundry.org/cli/plugin"
	"github.com/kardianos/osext"
)

var Version = "0.0.0"

func main() {
	ui := &plugin.UI{
		Out:     os.Stdout,
		Err:     os.Stderr,
		In:      os.Stdin,
		ErrTerm: terminal.IsTerminal(int(os.Stderr.Fd())),
	}

	version, err := goversion.NewVersion(Version)
	if err != nil {
		ui.Error(err)
		os.Exit(1)
	}

	confirmInstalled(ui)

	cfplugin.Start(&plugin.Plugin{
		UI: ui,
		Version: cfplugin.VersionType{
			Major: version.Segments()[0],
			Minor: version.Segments()[1],
			Build: version.Segments()[2],
		},
	})
}

func confirmInstalled(ui *plugin.UI) {
	var firstArg string
	if len(os.Args) > 1 {
		firstArg = os.Args[1]
	}

	switch firstArg {
	case "":
		plugin, err := osext.Executable()
		if err != nil {
			ui.Error(fmt.Errorf("failed to determine plugin path: %s", err))
			os.Exit(1)
		}

		operation := "upgraded"
		if err := exec.Command("cf", "uninstall-plugin", "cflocal").Run(); err != nil {
			operation = "installed"
		}

		installOpts := []string{"install-plugin", plugin}
		if needsConfirm := checkCLIVersion(ui); needsConfirm {
			installOpts = append(installOpts, "-f")
		}
		if output, err := exec.Command("cf", installOpts...).CombinedOutput(); err != nil {
			ui.Error(errors.New(strings.TrimSpace(string(output))))
			os.Exit(1)
		}

		ui.Output("Plugin successfully %s. Current version: %s", operation, Version)
		os.Exit(0)
	case "help", "-h", "--help":
		ui.Output("Usage: %s", os.Args[0])
		ui.Output("Running this binary directly will automatically install the CF Local cf CLI plugin.")
		ui.Output("You must have the latest version of the cf CLI and Docker installed to use CF Local.")
		ui.Output("After installing, run: cf local help")
		os.Exit(0)
	}
}

func checkCLIVersion(ui *plugin.UI) (installNeedsConfirm bool) {
	cfVersion, err := exec.Command("cf", "--version").Output()
	versionParts := strings.SplitN(strings.TrimPrefix(string(cfVersion), "cf version "), ".", 3)
	if err != nil || len(versionParts) < 3 {
		ui.Error(errors.New("failed to determine cf CLI version"))
		os.Exit(1)
	}
	majorVersion, errMajor := strconv.Atoi(versionParts[0])
	minorVersion, errMinor := strconv.Atoi(versionParts[1])
	if errMajor != nil || errMinor != nil || majorVersion < 6 || (majorVersion == 6 && minorVersion < 7) {
		ui.Error(errors.New("cf CLI version too old"))
		os.Exit(1)
	}
	if majorVersion == 6 && minorVersion < 13 {
		return false
	}
	return true
}
