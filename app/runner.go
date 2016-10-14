package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	docker "github.com/docker/docker/client"

	"github.com/docker/go-connections/nat"
	"github.com/sclevine/cflocal/utils"
)

type Runner struct {
	Docker   *docker.Client
	Logs     io.Writer
	ExitChan <-chan struct{}
}

func (r *Runner) Run(name string, logColorizer Colorizer, launcher, droplet io.Reader, launcherSize, dropletSize int64) error {
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
		return err
	}
	untarDroplet := "tar -C /home/vcap -xzf /tmp/droplet"
	chownVCAP := "chown -R vcap:vcap /home/vcap"
	startCommand := "$(jq -r .start_command /home/vcap/staging_info.yml)"
	cont := utils.Container{Docker: r.Docker, Err: &err}
	id := cont.Create(name, &container.Config{
		Hostname:     "cflocal",
		User:         "vcap",
		ExposedPorts: map[nat.Port]struct{}{"8080/tcp": struct{}{}},
		Env: []string{
			"CF_INSTANCE_ADDR=0.0.0.0:8080",
			"CF_INSTANCE_GUID=999db41a-508b-46eb-74d8-6f9c06c006da",
			"CF_INSTANCE_INDEX=0",
			"CF_INSTANCE_IP=0.0.0.0",
			"CF_INSTANCE_PORT=8080",
			`CF_INSTANCE_PORTS=[{"external":8080,"internal":8080}]`,
			"HOME=/home/vcap",
			"INSTANCE_GUID=999db41a-508b-46eb-74d8-6f9c06c006da",
			"INSTANCE_INDEX=0",
			"LANG=en_US.UTF-8",
			"MEMORY_LIMIT=512m",
			"PATH=/usr/local/bin:/usr/bin:/bin",
			"PORT=8080",
			"TMPDIR=/home/vcap/tmp",
			"USER=vcap",
			fmt.Sprintf("VCAP_APPLICATION=%s", vcapApp),
			"VCAP_SERVICES={}",
		},
		Image:      "cloudfoundry/cflinuxfs2",
		WorkingDir: "/home/vcap/app",
		Entrypoint: strslice.StrSlice{
			"/bin/bash", "-c",
			fmt.Sprintf(
				`%s && %s && /tmp/lifecycle/launcher /home/vcap/app "%s" ''`,
				untarDroplet, chownVCAP, startCommand,
			),
		},
	})
	if id == "" {
		return err
	}
	defer cont.Remove(id)

	launcherTar, err := utils.TarFile("./lifecycle/launcher", launcher, launcherSize, 0755)
	if err != nil {
		return err
	}
	if err := r.Docker.CopyToContainer(context.Background(), id, "/tmp", launcherTar, types.CopyToContainerOptions{}); err != nil {
		return err
	}

	dropletTar, err := utils.TarFile("./droplet", droplet, dropletSize, 0755)
	if err != nil {
		return err
	}
	if err := r.Docker.CopyToContainer(context.Background(), id, "/tmp", dropletTar, types.CopyToContainerOptions{}); err != nil {
		return err
	}

	if err := r.Docker.ContainerStart(context.Background(), id, types.ContainerStartOptions{}); err != nil {
		return err
	}
	logs, err := r.Docker.ContainerLogs(context.Background(), id, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
	})
	if err != nil {
		return err
	}
	defer logs.Close()
	go utils.CopyStream(r.Logs, logs, logColorizer(fmt.Sprintf("[%s]", name))+" ")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-r.ExitChan
		cancel()
	}()
	status, err := r.Docker.ContainerWait(ctx, id)
	if err != nil {
		return err
	}
	if status != 0 {
		return fmt.Errorf("container exited with status %d", status)
	}

	return nil
}

func uintPtr(i uint) *uint {
	return &i
}
