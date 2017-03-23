package local

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	docker "github.com/docker/docker/client"
	"github.com/sclevine/cflocal/service"
	"github.com/sclevine/cflocal/utils"
)

const stagerScript = `
	set -e

	chown -R vcap:vcap /tmp/app

	exec su vcap -p -c "PATH=$PATH exec /tmp/lifecycle/builder -buildpackOrder $0 -skipDetect=$1"
`

type Stager struct {
	UI           UI
	DiegoVersion string
	GoVersion    string
	StackVersion string
	Docker       *docker.Client
	Logs         io.Writer
	ExitChan     <-chan struct{}
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

func (s *Stager) Stage(config *StageConfig, color Colorizer) (droplet Stream, err error) {
	name := config.AppConfig.Name
	if err := s.buildDockerfile(); err != nil {
		return Stream{}, err
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
		return Stream{}, err
	}

	services := config.AppConfig.Services
	if services == nil {
		services = service.Services{}
	}
	vcapServices, err := json.Marshal(services)
	if err != nil {
		return Stream{}, err
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
		"VCAP_SERVICES":     string(vcapServices),
	}
	cont := utils.Container{
		Name: name + "-stage",
		Config: &container.Config{
			Hostname:   "cflocal",
			User:       "root",
			Env:        mapToEnv(mergeMaps(env, config.AppConfig.StagingEnv, config.AppConfig.Env)),
			Image:      "cflocal",
			WorkingDir: "/home/vcap",
			Entrypoint: strslice.StrSlice{
				"/bin/bash", "-c", stagerScript,
				strings.Join(config.Buildpacks, ","),
				strconv.FormatBool(len(config.Buildpacks) == 1),
			},
		},
		Docker: s.Docker,
		Err:    &err,
	}

	cont.Create()
	id := cont.ID()
	if id == "" {
		return Stream{}, err
	}
	defer cont.RemoveAfterCopy(&droplet.ReadCloser)

	if err := s.Docker.CopyToContainer(context.Background(), id, "/tmp/app", config.AppTar, types.CopyToContainerOptions{}); err != nil {
		return Stream{}, err
	}
	if err := s.Docker.ContainerStart(context.Background(), id, types.ContainerStartOptions{}); err != nil {
		return Stream{}, err
	}
	logs, err := s.Docker.ContainerLogs(context.Background(), id, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
	})
	if err != nil {
		return Stream{}, err
	}
	defer logs.Close()
	go utils.CopyStream(s.Logs, logs, color("[%s] ", name))

	go func() {
		<-s.ExitChan
		cont.Remove()
	}()
	status, err := s.Docker.ContainerWait(context.Background(), id)
	if err != nil {
		return Stream{}, err
	}
	if status != 0 {
		return Stream{}, fmt.Errorf("container exited with status %d", status)
	}

	dropletTar, dropletStat, err := s.Docker.CopyFromContainer(context.Background(), id, "/tmp/droplet")
	if err != nil {
		return Stream{}, err
	}
	droplet.ReadCloser = dropletTar // allows removal in error case
	dropletReader, _, err := utils.FileFromTar("droplet", dropletTar)
	if err != nil {
		return Stream{}, err
	}
	return NewStream(splitReadCloser{dropletReader, dropletTar}, dropletStat.Size), nil
}

func (s *Stager) Download(path string) (stream Stream, err error) {
	if err := s.buildDockerfile(); err != nil {
		return Stream{}, err
	}
	filename := filepath.Base(path)
	cont := utils.Container{
		Name: filename,
		Config: &container.Config{
			Image:      "cflocal",
			Entrypoint: strslice.StrSlice{"bash"},
		},
		Docker: s.Docker,
		Err:    &err,
	}

	cont.Create()
	id := cont.ID()
	if id == "" {
		return Stream{}, err
	}
	defer cont.RemoveAfterCopy(&stream.ReadCloser)
	tar, stat, err := s.Docker.CopyFromContainer(context.Background(), id, path)
	if err != nil {
		return Stream{}, err
	}
	stream.ReadCloser = tar // allows deferred removal in error case
	reader, _, err := utils.FileFromTar(filename, tar)
	if err != nil {
		return Stream{}, err
	}

	return NewStream(splitReadCloser{reader, tar}, stat.Size), nil
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
	return s.UI.Loading("Building Diego-powered base image", func() error {
		response, err := s.Docker.ImageBuild(context.Background(), dockerfileTar, types.ImageBuildOptions{
			Tags:           []string{"cflocal"},
			SuppressOutput: true,
			PullParent:     true,
			Remove:         true,
			ForceRemove:    true,
		})
		if err != nil {
			return err
		}
		defer response.Body.Close()
		return checkBody(response.Body)
	})
}
