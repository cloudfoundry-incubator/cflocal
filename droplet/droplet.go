package droplet

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	gouuid "github.com/nu7hatch/gouuid"
)

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

type Droplet struct {
	DiegoVersion string
	GoVersion    string
	UpdateRootFS bool
	Docker       *docker.Client
	Logs         io.Writer
}

func (d *Droplet) Build(appName string, appTar io.Reader, buildpacks []string) (droplet io.ReadCloser, err error) {
	if err := d.buildDockerfile(); err != nil {
		return nil, err
	}
	vcapApp, err := json.Marshal(&vcapApplication{
		ApplicationID:      "01d31c12-d066-495e-aca2-8d3403165360",
		ApplicationName:    appName,
		ApplicationURIs:    []string{"localhost"},
		ApplicationVersion: "2b860df9-a0a1-474c-b02f-5985f53ea0bb",
		Limits:             map[string]uint{"fds": 16384, "mem": 512, "disk": 1024},
		Name:               appName,
		SpaceID:            "18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1",
		SpaceName:          "cflocal-space",
		URIs:               []string{"localhost"},
		Version:            "18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1",
	})
	if err != nil {
		return nil, err
	}
	uuid, err := gouuid.NewV4()
	if err != nil {
		return nil, err
	}
	containerName := fmt.Sprintf("%s-build-%s", appName, uuid)
	if err := d.createContainer(containerName, &container.Config{
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
	}); err != nil {
		return nil, err
	}
	defer d.removeContainerAfterCopy(containerName, &droplet, &err)

	if err := d.Docker.CopyToContainer(context.Background(), containerName, "/tmp/app", appTar, types.CopyToContainerOptions{}); err != nil {
		return nil, err
	}
	if err := d.Docker.ContainerStart(context.Background(), containerName, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}
	logs, err := d.Docker.ContainerLogs(context.Background(), containerName, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
	})
	if err != nil {
		return nil, err
	}
	defer logs.Close()
	go copyStream(d.Logs, logs)

	status, err := d.Docker.ContainerWait(context.Background(), containerName)
	if err != nil {
		return nil, err
	}
	if status != 0 {
		return nil, fmt.Errorf("container exited with status %d", status)
	}

	droplet, _, err = d.Docker.CopyFromContainer(context.Background(), containerName, "/tmp/droplet")
	if err != nil {
		return nil, err
	}
	return droplet, nil
}

func UNUSEDappTar() error {
	appTar, err := archive.Tar(".", archive.Gzip)
	if err != nil {
		return err
	}
	defer appTar.Close()
	return nil
}

func (d *Droplet) Launcher() (launcher io.ReadCloser, err error) {
	if err := d.buildDockerfile(); err != nil {
		return nil, err
	}
	uuid, err := gouuid.NewV4()
	if err != nil {
		return nil, err
	}
	containerName := fmt.Sprintf("launcher-%s", uuid)
	if err := d.createContainer(containerName, &container.Config{
		Image: "cflocal",
	}); err != nil {
		return nil, err
	}
	defer d.removeContainerAfterCopy(containerName, &launcher, &err)
	launcher, _, err = d.Docker.CopyFromContainer(context.Background(), containerName, "/tmp/droplet")
	return launcher, err
}

func (d *Droplet) buildDockerfile() error {
	dockerfileBuf := &bytes.Buffer{}
	dockerfileTmpl := template.Must(template.New("Dockerfile").Parse(dockerfile))
	if err := dockerfileTmpl.Execute(dockerfileBuf, d); err != nil {
		return err
	}
	dockerfileTar, err := tarFile("Dockerfile", dockerfileBuf.Bytes())
	if err != nil {
		return err
	}
	response, err := d.Docker.ImageBuild(context.Background(), bytes.NewReader(dockerfileTar), types.ImageBuildOptions{
		Tags:           []string{"cflocal"},
		SuppressOutput: true,
		PullParent:     d.UpdateRootFS,
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

func (d *Droplet) createContainer(name string, config *container.Config) error {
	_, err := d.Docker.ContainerCreate(context.Background(), config, nil, nil, name)
	return err
}

func (d *Droplet) removeContainer(name string, err *error) {
	rmErr := d.Docker.ContainerRemove(context.Background(), name, types.ContainerRemoveOptions{
		Force: true,
	})
	if *err == nil {
		*err = rmErr
	}
}

func (d *Droplet) removeContainerAfterCopy(name string, file *io.ReadCloser, err *error) {
	if *file == nil {
		d.removeContainer(name, err)
		return
	}

	*file = &closeWrapper{
		ReadCloser: *file,
		After: func() {
			d.removeContainer(name, err)
		},
	}
}

// TODO: new package for below + unit tests
func tarFile(name string, contents []byte) ([]byte, error) {
	tarBuffer := &bytes.Buffer{}
	tarball := tar.NewWriter(tarBuffer)
	defer tarball.Close()
	header := &tar.Header{
		Name: name,
		Size: int64(len(contents)),
		Mode: 0644,
	}
	if err := tarball.WriteHeader(header); err != nil {
		return nil, err
	}
	if _, err := tarball.Write(contents); err != nil {
		return nil, err
	}
	return tarBuffer.Bytes(), nil
}

type closeWrapper struct {
	io.ReadCloser
	After func()
}

func (c *closeWrapper) Close() error {
	defer c.After()
	return c.ReadCloser.Close()
}

func copyStream(dst io.Writer, src io.Reader) {
	header := make([]byte, 8)
	for {
		if _, err := io.ReadFull(src, header); err != nil {
			break
		}
		if _, err := io.CopyN(dst, src, int64(binary.BigEndian.Uint32(header[4:]))); err != nil {
			break
		}
	}
}
