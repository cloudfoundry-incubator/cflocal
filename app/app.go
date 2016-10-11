package app

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

type App struct {
	DiegoVersion string
	GoVersion    string
	UpdateRootFS bool
	Docker       *docker.Client
	Logs         io.Writer
}

type Colorizer func(text string) string

type vcapApplication struct {
	ApplicationID      string          `json:"application_id"`
	ApplicationName    string          `json:"application_name"`
	ApplicationURIs    []string        `json:"application_uris"`
	ApplicationVersion string          `json:"application_version"`
	Host               string          `json:"host,omitempty"`
	InstanceID         string          `json:"instance_id,omitempty"`
	InstanceIndex      string          `json:"instance_index,omitempty"`
	Limits             map[string]uint `json:"limits"`
	Name               string          `json:"name"`
	Port               uint            `json:"port,omitempty"`
	SpaceID            string          `json:"space_id"`
	SpaceName          string          `json:"space_name"`
	URIs               []string        `json:"uris"`
	Version            string          `json:"version"`
}

type splitReadCloser struct {
	io.Reader
	io.Closer
}

func (a *App) Stage(name string, logColorizer Colorizer, appTar io.Reader, buildpacks []string) (droplet io.ReadCloser, err error) {
	if err := a.buildDockerfile(); err != nil {
		return nil, err
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
		return nil, err
	}
	cont := utils.Container{Docker: a.Docker, Err: &err}
	id := cont.Create(name+"-stage", &container.Config{
		User: "vcap",
		Env: []string{
			"CF_INSTANCE_ADDR=",
			"CF_INSTANCE_IP=0.0.0.0",
			"CF_INSTANCE_PORT=",
			"CF_INSTANCE_PORTS=[]",
			"CF_STACK=cflinuxfs2",
			"HOME=/home/vcap",
			"MEMORY_LIMIT=512m",
			fmt.Sprintf("VCAP_APPLICATION=%s", vcapApp),
			"VCAP_SERVICES={}",
		},
		Image:      "cflocal",
		WorkingDir: "/home/vcap",
		Entrypoint: strslice.StrSlice{
			"/tmp/lifecycle/builder",
			"-buildpackOrder", strings.Join(buildpacks, ","),
			fmt.Sprintf("-skipDetect=%t", len(buildpacks) == 1),
		},
	})
	if id == "" {
		return nil, err
	}
	defer cont.RemoveAfterCopy(id, &droplet)

	if err := a.Docker.CopyToContainer(context.Background(), id, "/tmp/app", appTar, types.CopyToContainerOptions{}); err != nil {
		return nil, err
	}
	if err := a.Docker.ContainerStart(context.Background(), id, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}
	logs, err := a.Docker.ContainerLogs(context.Background(), id, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
	})
	if err != nil {
		return nil, err
	}
	defer logs.Close()
	go utils.CopyStream(a.Logs, logs, logColorizer(fmt.Sprintf("[%s]", name))+" ")

	status, err := a.Docker.ContainerWait(context.Background(), id)
	if err != nil {
		return nil, err
	}
	if status != 0 {
		return nil, fmt.Errorf("container exited with status %d", status)
	}

	dropletCloser, _, err := a.Docker.CopyFromContainer(context.Background(), id, "/tmp/droplet")
	if err != nil {
		return nil, err
	}
	droplet = dropletCloser
	dropletReader, err := utils.FileFromTar("droplet", dropletCloser)
	if err != nil {
		return nil, err
	}
	return splitReadCloser{dropletReader, dropletCloser}, nil
}

func (a *App) Launcher() (launcher io.ReadCloser, err error) {
	if err := a.buildDockerfile(); err != nil {
		return nil, err
	}
	cont := utils.Container{Docker: a.Docker, Err: &err}
	id := cont.Create("launcher", &container.Config{
		Image: "cflocal",
	})
	if id == "" {
		return nil, err
	}
	defer cont.RemoveAfterCopy(id, &launcher)
	launcherCloser, _, err := a.Docker.CopyFromContainer(context.Background(), id, "/tmp/lifecycle/launcher")
	if err != nil {
		return nil, err
	}
	launcher = launcherCloser
	launcherReader, err := utils.FileFromTar("launcher", launcherCloser)
	if err != nil {
		return nil, err
	}

	return splitReadCloser{launcherReader, launcherCloser}, nil
}

func (a *App) buildDockerfile() error {
	dockerfileBuf := &bytes.Buffer{}
	dockerfileTmpl := template.Must(template.New("Dockerfile").Parse(dockerfile))
	if err := dockerfileTmpl.Execute(dockerfileBuf, a); err != nil {
		return err
	}
	dockerfileTar, err := utils.TarFile("Dockerfile", dockerfileBuf.Bytes())
	if err != nil {
		return err
	}
	response, err := a.Docker.ImageBuild(context.Background(), dockerfileTar, types.ImageBuildOptions{
		Tags:           []string{"cflocal"},
		SuppressOutput: true,
		PullParent:     a.UpdateRootFS,
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
