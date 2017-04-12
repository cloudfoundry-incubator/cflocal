package engine

import (
	"context"
	"encoding/json"
	"io"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"

	"github.com/sclevine/cflocal/ui"
)

type Image struct {
	Docker *docker.Client
	Exit   <-chan struct{}
}

func (i *Image) Build(tag string, dockerfile Stream) <-chan ui.Progress {
	ctx := context.Background()
	progress := make(chan ui.Progress, 1)

	dockerfileTar, err := tarFile("Dockerfile", dockerfile, dockerfile.Size, 0644)
	if err != nil {
		progress <- progressError{err}
		close(progress)
		return progress
	}
	response, err := i.Docker.ImageBuild(ctx, dockerfileTar, types.ImageBuildOptions{
		Tags:        []string{tag},
		PullParent:  true,
		Remove:      true,
		ForceRemove: true,
	})
	if err != nil {
		progress <- progressError{err}
		close(progress)
		return progress
	}
	go i.checkBody(response.Body, progress)
	return progress
}

func (i *Image) Pull(image string) <-chan ui.Progress {
	ctx := context.Background()
	progress := make(chan ui.Progress, 1)

	body, err := i.Docker.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		progress <- progressError{err}
		close(progress)
		return progress
	}
	go i.checkBody(body, progress)
	return progress
}

func (i *Image) checkBody(body io.ReadCloser, progress chan<- ui.Progress) {
	defer body.Close()
	defer close(progress)

	decoder := json.NewDecoder(body)
	for {
		select {
		case <-i.Exit:
			progress <- progressErrorString("interrupted")
			return
		default:
			var stream struct {
				Error    string
				Progress string
			}
			if err := decoder.Decode(&stream); err != nil {
				if err != io.EOF {
					progress <- progressError{err}
				}
				return
			}
			if stream.Error != "" {
				progress <- progressErrorString(stream.Error)
				return
			}
			if stream.Progress == "" {
				progress <- progressNA{}
			} else {
				progress <- progressMsg(stream.Progress)
			}
		}
	}
}
