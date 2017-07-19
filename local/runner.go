package local

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"text/template"
	"time"

	"code.cloudfoundry.org/cli/cf/formatters"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"

	"github.com/sclevine/cflocal/engine"
	"github.com/sclevine/cflocal/service"
)

const RunnerScript = `
	set -e
	{{if .RSync -}}
	rsync -a /tmp/local/ /home/vcap/app/
	{{end -}}
	if [[ ! -z $(ls -A /home/vcap/app) ]]; then
		exclude='--exclude=./app'
	fi
	tar $exclude -C /home/vcap -xzf /tmp/droplet
	chown -R vcap:vcap /home/vcap
	{{if .RSync -}}
	if [[ -z $(ls -A /tmp/local) ]]; then
		rsync -a /home/vcap/app/ /tmp/local/
	fi
	{{end -}}
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
	Droplet   engine.Stream
	Launcher  engine.Stream
	IP        string
	Port      uint
	AppDir    string
	RSync     bool
	Restart   <-chan time.Time
	Color     Colorizer
	AppConfig *AppConfig
}

func (r *Runner) Run(config *RunConfig) (status int64, err error) {
	if err := r.pull(); err != nil {
		return 0, err
	}

	r.setDefaults(config.AppConfig)
	containerConfig, err := r.buildContainerConfig(config.AppConfig, config.RSync)
	if err != nil {
		return 0, err
	}
	remoteDir := "/home/vcap/app"
	if config.RSync {
		remoteDir = "/tmp/local"
	}
	memory, err := formatters.ToMegabytes(config.AppConfig.Memory)
	if err != nil {
		return 0, err
	}
	hostConfig := r.buildHostConfig(config.IP, config.Port, memory, config.AppDir, remoteDir)
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
	return contr.Start(config.Color("[%s] ", config.AppConfig.Name), r.Logs, config.Restart)
}

type ExportConfig struct {
	Droplet   engine.Stream
	Launcher  engine.Stream
	Ref       string
	AppConfig *AppConfig
}

func (r *Runner) Export(config *ExportConfig) (imageID string, err error) {
	if err := r.pull(); err != nil {
		return "", err
	}

	r.setDefaults(config.AppConfig)
	containerConfig, err := r.buildContainerConfig(config.AppConfig, false)
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

	return contr.Commit(config.Ref)
}

func (r *Runner) pull() error {
	return r.UI.Loading("Image", r.Image.Pull(fmt.Sprintf("cloudfoundry/cflinuxfs2:%s", r.StackVersion)))
}

func (r *Runner) setDefaults(config *AppConfig) {
	if config.Memory == "" {
		config.Memory = "1024m"
	}
	if config.DiskQuota == "" {
		config.DiskQuota = "1024m"
	}
}

func (r *Runner) buildContainerConfig(config *AppConfig, rsync bool) (*container.Config, error) {
	name := config.Name
	memory, err := formatters.ToMegabytes(config.Memory)
	if err != nil {
		return nil, err
	}
	disk, err := formatters.ToMegabytes(config.DiskQuota)
	if err != nil {
		return nil, err
	}
	vcapApp, err := json.Marshal(&vcapApplication{
		ApplicationID:      "01d31c12-d066-495e-aca2-8d3403165360",
		ApplicationName:    name,
		ApplicationURIs:    []string{"localhost"},
		ApplicationVersion: "2b860df9-a0a1-474c-b02f-5985f53ea0bb",
		Host:               "0.0.0.0",
		InstanceID:         "999db41a-508b-46eb-74d8-6f9c06c006da",
		InstanceIndex:      uintPtr(0),
		Limits:             map[string]int64{"fds": 16384, "mem": memory, "disk": disk},
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
		"INSTANCE_GUID":     "999db41a-508b-46eb-74d8-6f9c06c006da",
		"INSTANCE_INDEX":    "0",
		"LANG":              "en_US.UTF-8",
		"MEMORY_LIMIT":      fmt.Sprintf("%dm", memory),
		"PATH":              "/usr/local/bin:/usr/bin:/bin",
		"PORT":              "8080",
		"TMPDIR":            "/home/vcap/tmp",
		"USER":              "vcap",
		"VCAP_APPLICATION":  string(vcapApp),
		"VCAP_SERVICES":     string(vcapServices),
	}

	options := struct{ RSync bool }{rsync}
	scriptBuf := &bytes.Buffer{}
	tmpl := template.Must(template.New("").Parse(RunnerScript))
	if err := tmpl.Execute(scriptBuf, options); err != nil {
		return nil, err
	}

	return &container.Config{
		Hostname:     "cflocal",
		User:         "vcap",
		ExposedPorts: nat.PortSet{"8080/tcp": {}},
		Env:          mapToEnv(mergeMaps(env, config.RunningEnv, config.Env)),
		Image:        "cloudfoundry/cflinuxfs2:" + r.StackVersion,
		WorkingDir:   "/home/vcap/app",
		Entrypoint: strslice.StrSlice{
			"/bin/bash", "-c", scriptBuf.String(), config.Command,
		},
	}, nil
}

func (*Runner) buildHostConfig(ip string, port uint, memory int64, appDir, remoteDir string) *container.HostConfig {
	config := &container.HostConfig{
		PortBindings: nat.PortMap{
			"8080/tcp": {{HostIP: ip, HostPort: strconv.FormatUint(uint64(port), 10)}},
		},
		Resources: container.Resources{
			Memory: memory * 1024 * 1024,
		},
	}
	if appDir != "" && remoteDir != "" {
		config.Binds = []string{appDir + ":" + remoteDir}
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
