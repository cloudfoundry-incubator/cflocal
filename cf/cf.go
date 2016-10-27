package cf

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/sclevine/cflocal/local"
	"github.com/sclevine/cflocal/remote"

	"github.com/fatih/color"
)

type CF struct {
	UI      UI
	Stager  Stager
	Runner  Runner
	App     App
	FS      FS
	Help    Help
	Config  Config
	Version string
}

type UI interface {
	Prompt(prompt string) string
	Output(format string, a ...interface{})
	Error(err error)
}

//go:generate mockgen -package mocks -destination mocks/stager.go github.com/sclevine/cflocal/cf Stager
type Stager interface {
	Stage(config *local.StageConfig, color local.Colorizer) (droplet io.ReadCloser, size int64, err error)
	Launcher() (launcher io.ReadCloser, size int64, err error)
}

//go:generate mockgen -package mocks -destination mocks/runner.go github.com/sclevine/cflocal/cf Runner
type Runner interface {
	Run(config *local.RunConfig, color local.Colorizer) (status int, err error)
}

//go:generate mockgen -package mocks -destination mocks/app.go github.com/sclevine/cflocal/cf App
type App interface {
	Droplet(name string) (droplet io.ReadCloser, size int64, err error)
	Command(name string) (string, error)
	Env(name string) (*remote.AppEnv, error)
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/sclevine/cflocal/cf FS
type FS interface {
	Tar(path string) (io.ReadCloser, error)
	ReadFile(path string) (io.ReadCloser, int64, error)
	WriteFile(path string) (io.WriteCloser, error)
}

//go:generate mockgen -package mocks -destination mocks/help.go github.com/sclevine/cflocal/cf Help
type Help interface {
	Show() error
}

//go:generate mockgen -package mocks -destination mocks/config.go github.com/sclevine/cflocal/cf Config
type Config interface {
	Load() (*local.LocalYML, error)
	Save(localYML *local.LocalYML) error
}

func (c *CF) Run(args []string) error {
	var err error
	switch args[0] {
	case "help":
		err = c.Help.Show()
	case "version", "--version":
		c.version()
	case "stage":
		err = c.stage(args[1:])
	case "run":
		err = c.run(args[1:])
	case "pull":
		err = c.pull(args[1:])
	default:
		return errors.New("invalid command")
	}
	return err
}

func (c *CF) version() {
	c.UI.Output("CF Local version %s", c.Version)
}

//go:generate mockgen -package mocks -destination mocks/closer.go io Closer
func (c *CF) stage(args []string) error {
	name, buildpack, err := stageFlags(args)
	if err != nil {
		if err := c.Help.Show(); err != nil {
			c.UI.Error(err)
		}
		return err
	}
	appTar, err := c.FS.Tar(".")
	if err != nil {
		return err
	}
	defer appTar.Close()
	localYML, err := c.Config.Load()
	if err != nil {
		return err
	}
	buildpacks := Buildpacks
	if buildpack != "" {
		buildpacks = []string{buildpack}
	}
	droplet, size, err := c.Stager.Stage(&local.StageConfig{
		AppTar:     appTar,
		Buildpacks: buildpacks,
		AppConfig:  getAppConfig(name, localYML),
	}, color.GreenString)
	if err != nil {
		return err
	}
	defer droplet.Close()
	file, err := c.FS.WriteFile(fmt.Sprintf("./%s.droplet", name))
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.CopyN(file, droplet, size); err != nil && err != io.EOF {
		return err
	}
	c.UI.Output("Successfully staged: %s", name)
	return nil
}

func stageFlags(args []string) (name string, buildpack string, err error) {
	set := &flag.FlagSet{}
	set.StringVar(&buildpack, "b", "", "")
	if err := set.Parse(args); err != nil {
		return "", "", err
	}
	if set.NArg() != 1 {
		return "", "", errors.New("invalid arguments")
	}
	return set.Arg(0), buildpack, nil
}

func (c *CF) run(args []string) error {
	name, port, err := runFlags(args)
	if err != nil {
		if err := c.Help.Show(); err != nil {
			c.UI.Error(err)
		}
		return err
	}
	droplet, dropletSize, err := c.FS.ReadFile(fmt.Sprintf("./%s.droplet", name))
	if err != nil {
		return err
	}
	defer droplet.Close()
	launcher, launcherSize, err := c.Stager.Launcher()
	if err != nil {
		return err
	}
	defer launcher.Close()
	localYML, err := c.Config.Load()
	if err != nil {
		return err
	}
	c.UI.Output("Running %s on port %d...", name, port)
	_, err = c.Runner.Run(&local.RunConfig{
		Droplet:      droplet,
		DropletSize:  dropletSize,
		Launcher:     launcher,
		LauncherSize: launcherSize,
		Port:         port,
		AppConfig:    getAppConfig(name, localYML),
	}, color.GreenString)
	return err
}

func runFlags(args []string) (name string, port uint, err error) {
	set := &flag.FlagSet{}
	defaultPort, err := freePort()
	if err != nil {
		return "", 0, err
	}
	set.UintVar(&port, "p", defaultPort, "")
	if err := set.Parse(args); err != nil {
		return "", 0, err
	}
	if set.NArg() != 1 {
		return "", 0, errors.New("invalid arguments")
	}
	return set.Arg(0), port, nil
}

func freePort() (uint, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	address := listener.Addr().String()
	portStr := strings.SplitN(address, ":", 2)[1]
	port, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(port), nil
}

func (c *CF) pull(args []string) error {
	if len(args) != 1 {
		if err := c.Help.Show(); err != nil {
			c.UI.Error(err)
		}
		return errors.New("invalid arguments")
	}
	name := args[0]
	if err := c.saveDroplet(name); err != nil {
		return err
	}
	if err := c.saveLocalYML(name); err != nil {
		return err
	}
	c.UI.Output("Successfully downloaded: %s", name)
	return nil
}

func (c *CF) saveDroplet(name string) error {
	droplet, size, err := c.App.Droplet(name)
	if err != nil {
		return err
	}
	defer droplet.Close()
	file, err := c.FS.WriteFile(fmt.Sprintf("./%s.droplet", name))
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.CopyN(file, droplet, size); err != nil && err != io.EOF {
		return err
	}
	return nil
}

func (c *CF) saveLocalYML(name string) error {
	localYML, err := c.Config.Load()
	if err != nil {
		return err
	}
	app := getAppConfig(name, localYML)

	env, err := c.App.Env(name)
	if err != nil {
		return err
	}
	app.StagingEnv = env.Staging
	app.RunningEnv = env.Running
	app.Env = env.App

	command, err := c.App.Command(name)
	if err != nil {
		return err
	}
	app.Command = command

	if err := c.Config.Save(localYML); err != nil {
		return err
	}
	return nil
}

func getAppConfig(name string, localYML *local.LocalYML) *local.AppConfig {
	var app *local.AppConfig
	for _, appConfig := range localYML.Applications {
		if appConfig.Name == name {
			app = appConfig
		}
	}
	if app == nil {
		app = &local.AppConfig{Name: name}
		localYML.Applications = append(localYML.Applications, app)
	}
	return app
}
