package local

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"text/template"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"

	"github.com/sclevine/cflocal/engine"
	"github.com/sclevine/cflocal/service"
)

const stagerScript = `
	set -e

	chown -R vcap:vcap /tmp/app

	exec su vcap -p -c "PATH=$PATH exec /tmp/lifecycle/builder -buildpackOrder $0 -skipDetect=$1"
`

type Stager struct {
	DiegoVersion string
	GoVersion    string
	StackVersion string
	Logs         io.Writer
	UI           UI
	Engine       Engine
	Image        Image
}

type StageConfig struct {
	AppTar     io.Reader
	Buildpacks []string
	AppConfig  *AppConfig
}

func (s *Stager) Stage(config *StageConfig, color Colorizer) (droplet engine.Stream, err error) {
	if err := s.buildDockerfile(); err != nil {
		return engine.Stream{}, err
	}
	vcapApp, err := json.Marshal(&vcapApplication{
		ApplicationID:      "01d31c12-d066-495e-aca2-8d3403165360",
		ApplicationName:    config.AppConfig.Name,
		ApplicationURIs:    []string{"localhost"},
		ApplicationVersion: "2b860df9-a0a1-474c-b02f-5985f53ea0bb",
		Limits:             map[string]uint{"fds": 16384, "mem": 512, "disk": 1024},
		Name:               config.AppConfig.Name,
		SpaceID:            "18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1",
		SpaceName:          "cflocal-space",
		URIs:               []string{"localhost"},
		Version:            "18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1",
	})
	if err != nil {
		return engine.Stream{}, err
	}

	services := config.AppConfig.Services
	if services == nil {
		services = service.Services{}
	}
	vcapServices, err := json.Marshal(services)
	if err != nil {
		return engine.Stream{}, err
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
	containerConfig := &container.Config{
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
	}

	contr, err := s.Engine.NewContainer(containerConfig, nil)
	if err != nil {
		return engine.Stream{}, err
	}
	defer contr.CloseAfterStream(&droplet)

	if err := contr.ExtractTo(config.AppTar, "/tmp/app"); err != nil {
		return engine.Stream{}, err
	}
	status, err := contr.Start(color("[%s] ", config.AppConfig.Name), s.Logs)
	if err != nil {
		return engine.Stream{}, err
	}
	if status != 0 {
		return engine.Stream{}, fmt.Errorf("container exited with status %d", status)
	}

	return contr.CopyFrom("/tmp/droplet")
}

func (s *Stager) Download(path string) (stream engine.Stream, err error) {
	if err := s.buildDockerfile(); err != nil {
		return engine.Stream{}, err
	}
	containerConfig := &container.Config{
		Hostname:   "cflocal",
		User:       "root",
		Image:      "cflocal",
		Entrypoint: strslice.StrSlice{"read"},
	}
	contr, err := s.Engine.NewContainer(containerConfig, nil)
	if err != nil {
		return engine.Stream{}, err
	}
	defer contr.CloseAfterStream(&stream)
	return contr.CopyFrom(path)
}

func (s *Stager) buildDockerfile() error {
	dockerfileBuf := &bytes.Buffer{}
	dockerfileTmpl := template.Must(template.New("Dockerfile").Parse(dockerfile))
	if err := dockerfileTmpl.Execute(dockerfileBuf, s); err != nil {
		return err
	}
	progress, done := s.Image.Build("cflocal", engine.NewStream(ioutil.NopCloser(dockerfileBuf), int64(dockerfileBuf.Len())))
	return s.UI.Loading("Image", progress, done)
}
