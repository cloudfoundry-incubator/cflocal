package local

import (
	"bytes"
	"crypto/md5"
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
	"github.com/sclevine/cflocal/local/version"
	"github.com/sclevine/cflocal/service"
)

const StagerScript = `
	set -e
	chown -R vcap:vcap /tmp/app /tmp/cache
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
	Versioner    Versioner
}

type StageConfig struct {
	AppTar     io.Reader
	Cache      ReadResetWriter
	CacheEmpty bool
	Buildpack  string
	AppDir     string
	Color      Colorizer
	AppConfig  *AppConfig
}

type ReadResetWriter interface {
	io.ReadWriter
	Reset() error
}

func (s *Stager) Stage(config *StageConfig) (droplet engine.Stream, err error) {
	if err := s.buildDockerfile(); err != nil {
		return engine.Stream{}, err
	}

	var buildpacks []string
	if config.Buildpack == "" {
		s.UI.Output("Buildpack: will detect")
		buildpacks = Buildpacks.names()
	} else {
		s.UI.Output("Buildpack: %s", config.Buildpack)
		buildpacks = []string{config.Buildpack}
	}

	containerConfig, err := s.buildContainerConfig(config.AppConfig, buildpacks)
	if err != nil {
		return engine.Stream{}, err
	}
	hostConfig := s.buildHostConfig(config.AppDir)
	contr, err := s.Engine.NewContainer(containerConfig, hostConfig)
	if err != nil {
		return engine.Stream{}, err
	}
	defer contr.CloseAfterStream(&droplet)

	if err := contr.ExtractTo(config.AppTar, "/tmp/app"); err != nil {
		return engine.Stream{}, err
	}
	if !config.CacheEmpty {
		if err := contr.ExtractTo(config.Cache, "/tmp/cache"); err != nil {
			return engine.Stream{}, err
		}
	}

	status, err := contr.Start(config.Color("[%s] ", config.AppConfig.Name), s.Logs)
	if err != nil {
		return engine.Stream{}, err
	}
	if status != 0 {
		return engine.Stream{}, fmt.Errorf("container exited with status %d", status)
	}

	if err := config.Cache.Reset(); err != nil {
		return engine.Stream{}, err
	}
	if err := streamOut(contr, config.Cache, "/tmp/output-cache"); err != nil {
		return engine.Stream{}, err
	}

	return contr.CopyFrom("/tmp/droplet")
}

func (s *Stager) buildContainerConfig(config *AppConfig, buildpacks []string) (*container.Config, error) {
	// TODO: fill with real information -- get/set container limits
	vcapApp, err := json.Marshal(&vcapApplication{
		ApplicationID:      "01d31c12-d066-495e-aca2-8d3403165360",
		ApplicationName:    config.Name,
		ApplicationURIs:    []string{"localhost"},
		ApplicationVersion: "2b860df9-a0a1-474c-b02f-5985f53ea0bb",
		Limits:             map[string]uint{"fds": 16384, "mem": 512, "disk": 1024},
		Name:               config.Name,
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
	return &container.Config{
		Hostname:   "cflocal",
		User:       "root",
		Env:        mapToEnv(mergeMaps(env, config.StagingEnv, config.Env)),
		Image:      "cflocal",
		WorkingDir: "/home/vcap",
		Entrypoint: strslice.StrSlice{
			"/bin/bash", "-c", StagerScript,
			strings.Join(buildpacks, ","),
			strconv.FormatBool(len(buildpacks) == 1),
		},
	}, nil
}

func (*Stager) buildHostConfig(appDir string) *container.HostConfig {
	config := &container.HostConfig{}
	if appDir != "" {
		config.Binds = []string{appDir + ":/tmp/app"}
	}
	return config
}

func streamOut(contr Container, out io.Writer, path string) error {
	stream, err := contr.CopyFrom(path)
	if err != nil {
		return err
	}
	return stream.Out(out)
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
	buildpacks, err := s.buildpacks()
	if err == version.ErrNetwork || err == version.ErrUnavailable {
		s.UI.Output("Warning: cannot build image: %s", err)
		return nil
	}
	if err != nil {
		return err
	}
	dockerfileBuf := &bytes.Buffer{}
	dockerfileData := struct {
		DiegoVersion string
		GoVersion    string
		StackVersion string
		Buildpacks   []buildpackInfo
	}{
		s.DiegoVersion,
		s.GoVersion,
		s.StackVersion,
		buildpacks,
	}
	dockerfileTmpl := template.Must(template.New("Dockerfile").Parse(dockerfile))
	if err := dockerfileTmpl.Execute(dockerfileBuf, dockerfileData); err != nil {
		return err
	}
	dockerfileStream := engine.NewStream(ioutil.NopCloser(dockerfileBuf), int64(dockerfileBuf.Len()))
	return s.UI.Loading("Image", s.Image.Build("cflocal", dockerfileStream))
}

func (s *Stager) buildpacks() ([]buildpackInfo, error) {
	var buildpacks []buildpackInfo
	for _, buildpack := range Buildpacks {
		url, err := s.Versioner.Build(buildpack.URL, buildpack.VersionURL)
		if err != nil {
			return nil, err
		}
		checksum := fmt.Sprintf("%x", md5.Sum([]byte(buildpack.Name)))
		info := buildpackInfo{buildpack.Name, url, checksum}
		buildpacks = append(buildpacks, info)
	}
	return buildpacks, nil
}

type buildpackInfo struct {
	Name, URL, MD5 string
}

type BuildpackList []Buildpack

func (b BuildpackList) names() []string {
	var names []string
	for _, bp := range b {
		names = append(names, bp.Name)
	}
	return names
}
