package local

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	docker "github.com/docker/docker/client"

	"github.com/sclevine/cflocal/utils"
)

type Stager struct {
	DiegoVersion string
	GoVersion    string
	StackVersion string
	UpdateRootFS bool
	Docker       *docker.Client
	Logs         io.Writer
	ExitChan     <-chan struct{}
}

type Colorizer func(string, ...interface{}) string

type vcapApplication struct {
	ApplicationID      string          `json:"application_id"`
	ApplicationName    string          `json:"application_name"`
	ApplicationURIs    []string        `json:"application_uris"`
	ApplicationVersion string          `json:"application_version"`
	Host               string          `json:"host,omitempty"`
	InstanceID         string          `json:"instance_id,omitempty"`
	InstanceIndex      *uint           `json:"instance_index,omitempty"`
	Limits             map[string]uint `json:"limits"`
	Name               string          `json:"name"`
	Port               *uint           `json:"port,omitempty"`
	SpaceID            string          `json:"space_id"`
	SpaceName          string          `json:"space_name"`
	URIs               []string        `json:"uris"`
	Version            string          `json:"version"`
}

type splitReadCloser struct {
	io.Reader
	io.Closer
}

type StageConfig struct {
	AppTar     io.Reader
	Buildpacks []string
	AppConfig  *AppConfig
}

func (s *Stager) Stage(config *StageConfig, color Colorizer) (droplet io.ReadCloser, size int64, err error) {
	name := config.AppConfig.Name
	if err := s.buildDockerfile(); err != nil {
		return nil, 0, err
	}
	vcapApp, err := json.Marshal(&vcapApplication{
		ApplicationID:      "01d31c12-d066-495e-aca2-8d3403165360",
		ApplicationName:    name,
		ApplicationURIs:    []string{"localhost"},
		ApplicationVersion: "2b860df9-a0a1-474c-b02f-5985f53ea0bb",
		Limits:             map[string]uint{"fds": 16384, "mem": 512, "disk": 1024},
		Name:               name,
		SpaceID:            "18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1",
		SpaceName:          "cflocal-space",
		URIs:               []string{"localhost"},
		Version:            "18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1",
	})
	if err != nil {
		return nil, 0, err
	}
	env := map[string]string{
		"CF_INSTANCE_ADDR":  "",
		"CF_INSTANCE_IP":    "0.0.0.0",
		"CF_INSTANCE_PORT":  "",
		"CF_INSTANCE_PORTS": "[]",
		"CF_STACK":          "cflinuxfs2",
		"HOME":              "/home/vcap",
		"LANG":              "en_US.UTF-8",
		"MEMORY_LIMIT":      "512m",
		"PATH":              "/usr/local/bin:/usr/bin:/bin",
		"USER":              "vcap",
		"VCAP_APPLICATION":  string(vcapApp),
		"VCAP_SERVICES":     "{}",
	}
	cont := utils.Container{Docker: s.Docker, Err: &err}
	id := cont.Create(name+"-stage", 0, &container.Config{
		Hostname:   "cflocal",
		User:       "vcap",
		Env:        mapToEnv(mergeMaps(env, config.AppConfig.StagingEnv, config.AppConfig.Env)),
		Image:      "cflocal",
		WorkingDir: "/home/vcap",
		Entrypoint: strslice.StrSlice{
			"/tmp/lifecycle/builder",
			"-buildpackOrder", strings.Join(config.Buildpacks, ","),
			fmt.Sprintf("-skipDetect=%t", len(config.Buildpacks) == 1),
		},
	})
	if id == "" {
		return nil, 0, err
	}
	defer cont.RemoveAfterCopy(id, &droplet)

	if err := s.Docker.CopyToContainer(context.Background(), id, "/tmp/app", config.AppTar, types.CopyToContainerOptions{}); err != nil {
		return nil, 0, err
	}
	if err := s.Docker.ContainerStart(context.Background(), id, types.ContainerStartOptions{}); err != nil {
		return nil, 0, err
	}
	logs, err := s.Docker.ContainerLogs(context.Background(), id, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
	})
	if err != nil {
		return nil, 0, err
	}
	defer logs.Close()
	go utils.CopyStream(s.Logs, logs, color("[%s] ", name))

	go func() {
		<-s.ExitChan
		cont.Remove(id)
	}()
	status, err := s.Docker.ContainerWait(context.Background(), id)
	if err != nil {
		return nil, 0, err
	}
	if status != 0 {
		return nil, 0, fmt.Errorf("container exited with status %d", status)
	}

	dropletCloser, dropletStat, err := s.Docker.CopyFromContainer(context.Background(), id, "/tmp/droplet")
	if err != nil {
		return nil, 0, err
	}
	droplet = dropletCloser
	dropletReader, err := utils.FileFromTar("droplet", dropletCloser)
	if err != nil {
		return nil, 0, err
	}
	return splitReadCloser{dropletReader, dropletCloser}, dropletStat.Size, nil
}

func (s *Stager) Launcher() (launcher io.ReadCloser, size int64, err error) {
	if err := s.buildDockerfile(); err != nil {
		return nil, 0, err
	}
	cont := utils.Container{Docker: s.Docker, Err: &err}
	id := cont.Create("launcher", 0, &container.Config{
		Image:      "cflocal",
		Entrypoint: strslice.StrSlice{"bash"},
	})
	if id == "" {
		return nil, 0, err
	}
	defer cont.RemoveAfterCopy(id, &launcher)
	launcherCloser, launcherStat, err := s.Docker.CopyFromContainer(context.Background(), id, "/tmp/lifecycle/launcher")
	if err != nil {
		return nil, 0, err
	}
	launcher = launcherCloser
	launcherReader, err := utils.FileFromTar("launcher", launcherCloser)
	if err != nil {
		return nil, 0, err
	}

	return splitReadCloser{launcherReader, launcherCloser}, launcherStat.Size, nil
}

func (s *Stager) buildDockerfile() error {
	dockerfileBuf := &bytes.Buffer{}
	dockerfileTmpl := template.Must(template.New("Dockerfile").Parse(dockerfile))
	if err := dockerfileTmpl.Execute(dockerfileBuf, s); err != nil {
		return err
	}
	dockerfileTar, err := utils.TarFile("Dockerfile", dockerfileBuf, int64(dockerfileBuf.Len()), 0644)
	if err != nil {
		return err
	}
	response, err := s.Docker.ImageBuild(context.Background(), dockerfileTar, types.ImageBuildOptions{
		Tags:           []string{"cflocal"},
		SuppressOutput: true,
		PullParent:     s.UpdateRootFS,
		Remove:         true,
		ForceRemove:    true,
	})
	if err != nil {
		return err
	}
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	for {
		var stream struct{ Error string }
		if err := decoder.Decode(&stream); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if stream.Error != "" {
			return fmt.Errorf("build failure: %s", stream.Error)
		}
	}
	return nil
}
