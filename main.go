package main

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	goversion "github.com/hashicorp/go-version"
	"github.com/sclevine/cflocal/plugin"

	"code.cloudfoundry.org/cli/cf/terminal"
	"code.cloudfoundry.org/cli/cf/trace"
	cfplugin "code.cloudfoundry.org/cli/plugin"
	"github.com/kardianos/osext"
)

var Version string

func main() {
	ui := terminal.NewUI(
		os.Stdin,
		os.Stdout,
		terminal.NewTeePrinter(os.Stdout),
		trace.NewWriterPrinter(os.Stdout, true),
	)

	confirmInstalled(ui)

	version, err := goversion.NewVersion(Version)
	if err != nil {
		ui.Say("Error: %s", err)
		os.Exit(1)
	}
	cfplugin.Start(&plugin.Plugin{
		UI: &plugin.NonTranslatingUI{ui},
		Version: cfplugin.VersionType{
			Major: version.Segments()[0],
			Minor: version.Segments()[1],
			Build: version.Segments()[2],
		},
	})
}

func confirmInstalled(ui terminal.UI) {
	var firstArg string
	if len(os.Args) > 1 {
		firstArg = os.Args[1]
	}

	switch firstArg {
	case "":
		plugin, err := osext.Executable()
		if err != nil {
			ui.Say("Failed to determine plugin path: %s", err)
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
			ui.Say(strings.TrimSpace(string(output)))
			os.Exit(1)
		}

		ui.Say("Plugin successfully %s. Current version: %s", operation, Version)
		os.Exit(0)
	case "help", "-h", "--help":
		ui.Say("Usage: %s", os.Args[0])
		ui.Say("Running this binary directly will automatically install the CF Local cf CLI plugin.")
		ui.Say("You must have the latest version of the cf CLI and Docker installed to use CF Local.")
		ui.Say("After installing, run: cf local help")
		os.Exit(0)
	}
}

func checkCLIVersion(ui terminal.UI) (installNeedsConfirm bool) {
	cfVersion, err := exec.Command("cf", "--version").Output()
	versionParts := strings.SplitN(strings.TrimPrefix(string(cfVersion), "cf version "), ".", 3)
	if err != nil || len(versionParts) < 3 {
		ui.Say("Failed to determine cf CLI version.")
		os.Exit(1)
	}
	majorVersion, errMajor := strconv.Atoi(versionParts[0])
	minorVersion, errMinor := strconv.Atoi(versionParts[1])
	if errMajor != nil || errMinor != nil || majorVersion < 6 || (majorVersion == 6 && minorVersion < 7) {
		ui.Say("Your cf CLI version is too old. Please install the latest cf CLI.")
		os.Exit(1)
	}
	if majorVersion == 6 && minorVersion < 13 {
		return false
	}
	return true
}
