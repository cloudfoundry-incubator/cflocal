package local

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"text/template"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"

	"github.com/sclevine/cflocal/engine"
	"github.com/sclevine/cflocal/service"
)

const runnerScript = `
	set -e

	{{with .ForwardConfig}}
	{{if .Forwards}}
		echo 'Forwarding:{{range .Forwards}} {{.Name}}{{end}}'
		sshpass -p '{{.Code}}' ssh -f -N \
			-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no \
			-o LogLevel=ERROR -o ExitOnForwardFailure=yes \
			-o ServerAliveInterval=10 -o ServerAliveCountMax=60 \
			-p '{{.Port}}' '{{.User}}@{{.Host}}' \
			{{range .Forwards}} -L '{{.From}}:{{.To}}' \
			{{end}}
	{{end}}
	{{end}}

	tar --exclude={{.Exclude}} -C /home/vcap -xzf /tmp/droplet
	chown -R vcap:vcap /home/vcap

	command=$1
	if [[ -z $command ]]; then
		command=$(jq -r .start_command /home/vcap/staging_info.yml)
	fi

	exec /tmp/lifecycle/launcher /home/vcap/app "$command" ''
`

type Runner struct {
	StackVersion string
	Logs         io.Writer
	UI           UI
	Engine       Engine
	Image        Image
}

type RunConfig struct {
	Droplet       engine.Stream
	Launcher      engine.Stream
	SSHPass       engine.Stream
	IP            string
	Port          uint
	AppDir        string
	AppDirEmpty   bool
	AppConfig     *AppConfig
	ForwardConfig *service.ForwardConfig
}

func (r *Runner) Run(config *RunConfig, color Colorizer) (status int64, err error) {
	if err := r.pull(); err != nil {
		return 0, err
	}

	excludeApp := config.AppDir != "" && !config.AppDirEmpty
	containerConfig, err := r.buildContainerConfig(config.AppConfig, config.ForwardConfig, excludeApp)
	if err != nil {
		return 0, err
	}
	hostConfig := buildHostConfig(config.IP, config.Port, config.AppDir)

	contr, err := r.Engine.NewContainer(containerConfig, hostConfig)
	if err != nil {
		return 0, err
	}
	defer contr.Close()

	if err := contr.CopyTo(config.Launcher, "/tmp/lifecycle/launcher"); err != nil {
		return 0, err
	}
	if err := contr.CopyTo(config.Droplet, "/tmp/droplet"); err != nil {
		return 0, err
	}
	if config.SSHPass.Size > 0 {
		if err := contr.CopyTo(config.SSHPass, "/usr/bin/sshpass"); err != nil {
			return 0, err
		}
	}
	return contr.Start(color("[%s] ", config.AppConfig.Name), r.Logs)

}

type ExportConfig struct {
	Droplet   engine.Stream
	Launcher  engine.Stream
	AppConfig *AppConfig
}

func (r *Runner) Export(config *ExportConfig, ref string) (imageID string, err error) {
	if err := r.pull(); err != nil {
		return "", err
	}

	containerConfig, err := r.buildContainerConfig(config.AppConfig, nil, false)
	if err != nil {
		return "", err
	}
	contr, err := r.Engine.NewContainer(containerConfig, nil)
	if err != nil {
		return "", err
	}
	defer contr.Close()

	if err := contr.CopyTo(config.Launcher, "/tmp/lifecycle/launcher"); err != nil {
		return "", err
	}
	if err := contr.CopyTo(config.Droplet, "/tmp/droplet"); err != nil {
		return "", err
	}

	return contr.Commit(ref)
}

func (r *Runner) pull() error {
	return r.UI.Loading("Image", r.Image.Pull("cloudfoundry/cflinuxfs2:" + r.StackVersion))
}

func (r *Runner) buildContainerConfig(config *AppConfig, forwardConfig *service.ForwardConfig, excludeApp bool) (*container.Config, error) {
	name := config.Name
	vcapApp, err := json.Marshal(&vcapApplication{
		ApplicationID:      "01d31c12-d066-495e-aca2-8d3403165360",
		ApplicationName:    name,
		ApplicationURIs:    []string{"localhost"},
		ApplicationVersion: "2b860df9-a0a1-474c-b02f-5985f53ea0bb",
		Host:               "0.0.0.0",
		InstanceID:         "999db41a-508b-46eb-74d8-6f9c06c006da",
		InstanceIndex:      uintPtr(0),
		Limits:             map[string]uint{"fds": 16384, "mem": 512, "disk": 1024},
		Name:               name,
		Port:               uintPtr(8080),
		SpaceID:            "18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1",
		SpaceName:          "cflocal-space",
		URIs:               []string{"localhost"},
		Version:            "18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1",
	})
	if err != nil {
		return nil, err
	}

	services := config.Services
	if services == nil {
		services = service.Services{}
	}
	vcapServices, err := json.Marshal(services)
	if err != nil {
		return nil, err
	}

	env := map[string]string{
		"CF_INSTANCE_ADDR":  "0.0.0.0:8080",
		"CF_INSTANCE_GUID":  "999db41a-508b-46eb-74d8-6f9c06c006da",
		"CF_INSTANCE_INDEX": "0",
		"CF_INSTANCE_IP":    "0.0.0.0",
		"CF_INSTANCE_PORT":  "8080",
		"CF_INSTANCE_PORTS": `[{"external":8080,"internal":8080}]`,
		"HOME":              "/home/vcap",
		"INSTANCE_GUID":     "999db41a-508b-46eb-74d8-6f9c06c006da",
		"INSTANCE_INDEX":    "0",
		"LANG":              "en_US.UTF-8",
		"MEMORY_LIMIT":      "512m",
		"PATH":              "/usr/local/bin:/usr/bin:/bin",
		"PORT":              "8080",
		"TMPDIR":            "/home/vcap/tmp",
		"USER":              "vcap",
		"VCAP_APPLICATION":  string(vcapApp),
		"VCAP_SERVICES":     string(vcapServices),
	}

	options := struct {
		Exclude       string
		ForwardConfig *service.ForwardConfig
	}{"", forwardConfig}
	if excludeApp {
		options.Exclude = "./app"
	}

	scriptBuffer := &bytes.Buffer{}
	if err := template.Must(template.New("").Parse(runnerScript)).Execute(scriptBuffer, options); err != nil {
		return nil, err
	}

	return &container.Config{
		Hostname:     "cflocal",
		User:         "vcap",
		ExposedPorts: nat.PortSet(map[nat.Port]struct{}{"8080/tcp": {}}),
		Env:          mapToEnv(mergeMaps(env, config.RunningEnv, config.Env)),
		Image:        "cloudfoundry/cflinuxfs2:" + r.StackVersion,
		WorkingDir:   "/home/vcap/app",
		Entrypoint: strslice.StrSlice{
			"/bin/bash", "-c", scriptBuffer.String(), config.Command,
		},
	}, nil
}

func buildHostConfig(ip string, port uint, appDir string) *container.HostConfig {
	config := &container.HostConfig{
		PortBindings: nat.PortMap{
			"8080/tcp": {{HostIP: ip, HostPort: strconv.FormatUint(uint64(port), 10)}},
		},
	}
	if appDir != "" {
		config.Binds = []string{appDir + ":/home/vcap/app"}
	}
	return config
}

func mergeMaps(maps ...map[string]string) map[string]string {
	merged := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}

func mapToEnv(env map[string]string) []string {
	var out []string
	for k, v := range env {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}

func uintPtr(i uint) *uint {
	return &i
}
