package local

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"text/template"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/sclevine/cflocal/service"
	"github.com/sclevine/cflocal/utils"
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
	Docker       *docker.Client
	Logs         io.Writer
	ExitChan     <-chan struct{}
}

type Stream struct {
	io.ReadCloser
	Size int64
}

type Forwarder struct {
	SSHPass Stream
	Config  *service.ForwardConfig
}

type RunConfig struct {
	Droplet     Stream
	Launcher    Stream
	Forwarder   Forwarder
	Port        uint
	AppDir      string
	AppDirEmpty bool
	AppConfig   *AppConfig
}

func (r *Runner) Run(config *RunConfig, color Colorizer) (status int, err error) {
	if err := r.pull(); err != nil {
		return 0, err
	}

	name := config.AppConfig.Name
	containerConfig, err := r.buildContainerConfig(config.AppConfig, config.Forwarder.Config, config.AppDir != "" && !config.AppDirEmpty)
	if err != nil {
		return 0, err
	}
	hostConfig := buildHostConfig(config.Port, config.AppDir)
	cont := utils.Container{
		Name:       name,
		Config:     containerConfig,
		HostConfig: hostConfig,
		Docker:     r.Docker,
		Err:        &err,
	}
	cont.Create()
	id := cont.ID()
	if id == "" {
		return 0, err
	}
	defer cont.Remove()

	if err := r.copy(id, config.Launcher, "/tmp/lifecycle/launcher"); err != nil {
		return 0, err
	}
	if err := r.copy(id, config.Droplet, "/tmp/droplet"); err != nil {
		return 0, err
	}
	if config.Forwarder != (Forwarder{}) {
		if err := r.copy(id, config.Forwarder.SSHPass, "/usr/bin/sshpass"); err != nil {
			return 0, err
		}
	}

	if err := r.Docker.ContainerStart(context.Background(), id, types.ContainerStartOptions{}); err != nil {
		return 0, err
	}
	logs, err := r.Docker.ContainerLogs(context.Background(), id, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
	})
	if err != nil {
		return 0, err
	}
	defer logs.Close()
	go utils.CopyStream(r.Logs, logs, color("[%s] ", name))

	go func() {
		<-r.ExitChan
		cont.Remove()
	}()
	status, err = r.Docker.ContainerWait(context.Background(), id)
	if err != nil {
		return 0, err
	}
	return status, nil
}

type ExportConfig struct {
	Droplet   Stream
	Launcher  Stream
	Port      uint
	AppConfig *AppConfig
}

func (r *Runner) Export(config *ExportConfig, reference string) (imageID string, err error) {
	if err := r.pull(); err != nil {
		return "", err
	}

	name := config.AppConfig.Name
	containerConfig, err := r.buildContainerConfig(config.AppConfig, nil, false)
	if err != nil {
		return "", err
	}
	hostConfig := buildHostConfig(config.Port, "")
	cont := utils.Container{
		Name:       name,
		Config:     containerConfig,
		HostConfig: hostConfig,
		Docker:     r.Docker,
		Err:        &err,
	}
	cont.Create()
	id := cont.ID()
	if id == "" {
		return "", err
	}
	defer cont.Remove()

	if err := r.copy(id, config.Launcher, "/tmp/lifecycle/launcher"); err != nil {
		return "", err
	}
	if err := r.copy(id, config.Droplet, "/tmp/droplet"); err != nil {
		return "", err
	}

	response, err := r.Docker.ContainerCommit(context.Background(), id, types.ContainerCommitOptions{
		Reference: reference,
		Author:    "CF Local",
		Pause:     true,
		Config:    containerConfig,
	})
	if err != nil {
		return "", err
	}
	return response.ID, nil
}

func (r *Runner) pull() error {
	body, err := r.Docker.ImagePull(context.Background(), "cloudfoundry/cflinuxfs2:" + r.StackVersion, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer body.Close()
	decoder := json.NewDecoder(body)
	for {
		var stream struct{ Error string }
		if err := decoder.Decode(&stream); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if stream.Error != "" {
			return fmt.Errorf("pull failure: %s", stream.Error)
		}
	}
	return nil
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
	err = template.Must(template.New("").Parse(runnerScript)).Execute(scriptBuffer, options)
	if err != nil {
		return nil, err
	}

	return &container.Config{
		Hostname:     "cflocal",
		User:         "vcap",
		ExposedPorts: map[nat.Port]struct{}{"8080/tcp": {}},
		Env:          mapToEnv(mergeMaps(env, config.RunningEnv, config.Env)),
		Image:        "cloudfoundry/cflinuxfs2:" + r.StackVersion,
		WorkingDir:   "/home/vcap/app",
		Entrypoint: strslice.StrSlice{
			"/bin/bash", "-c", scriptBuffer.String(), config.Command,
		},
	}, nil
}

func (r *Runner) copy(id string, stream Stream, path string) error {
	tar, err := utils.TarFile(path, stream, stream.Size, 0755)
	if err != nil {
		return err
	}
	if err := r.Docker.CopyToContainer(context.Background(), id, "/", tar, types.CopyToContainerOptions{}); err != nil {
		return err
	}
	if err := stream.Close(); err != nil {
		return err
	}
	return nil
}

func buildHostConfig(port uint, appDir string) *container.HostConfig {
	config := &container.HostConfig{
		PortBindings: nat.PortMap{
			"8080/tcp": {{HostIP: "127.0.0.1", HostPort: strconv.FormatUint(uint64(port), 10)}},
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
