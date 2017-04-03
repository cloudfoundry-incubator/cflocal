package engine

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"

	"github.com/sclevine/cflocal/utils"
)

type Image struct {
	Docker *docker.Client
	Exit   <-chan struct{}
}

func (i *Image) Build(tag string, dockerfile Stream) (<-chan string, <-chan error) {
	progress, done := make(chan string), make(chan error, 1)

	dockerfileTar, err := utils.TarFile("Dockerfile", dockerfile, dockerfile.Size, 0644)
	if err != nil {
		done <- err
		return nil, done
	}
	response, err := i.Docker.ImageBuild(context.Background(), dockerfileTar, types.ImageBuildOptions{
		Tags:        []string{"cflocal"},
		PullParent:  true,
		Remove:      true,
		ForceRemove: true,
	})
	if err != nil {
		done <- err
		return nil, done
	}
	go i.checkBody(response.Body, progress, done)
	return progress, done
}

func (i *Image) Pull(image string) (<-chan string, <-chan error) {
	progress, done := make(chan string), make(chan error, 1)

	body, err := i.Docker.ImagePull(context.Background(), image, types.ImagePullOptions{})
	if err != nil {
		done <- err
		return nil, done
	}
	go i.checkBody(body, progress, done)
	return progress, done
}

func (i *Image) checkBody(body io.ReadCloser, progress chan<- string, done chan<- error) {
	defer body.Close()
	defer close(progress)
	defer close(done)

	decoder := json.NewDecoder(body)
	for {
		select {
		case <-i.Exit:
			done <- errors.New("interrupted")
			return
		default:
			var stream struct {
				Error    string
				Progress string
			}
			if err := decoder.Decode(&stream); err != nil {
				if err != io.EOF {
					done <- err
				}
				return
			}
			if stream.Error != "" {
				done <- errors.New(stream.Error)
				return
			}
			progress <- stream.Progress
		}
	}
}
