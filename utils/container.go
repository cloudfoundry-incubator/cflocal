package utils

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"
	gouuid "github.com/nu7hatch/gouuid"
)

type Container struct {
	Docker *docker.Client
	Err    *error
}

func (c *Container) Create(name string, config *container.Config) (id string) {
	uuid, err := gouuid.NewV4()
	if err != nil {
		*c.Err = err
		return ""
	}

	response, err := c.Docker.ContainerCreate(context.Background(), config, nil, nil, fmt.Sprintf("%s-%s", name, uuid))
	if err != nil {
		*c.Err = err
		return ""
	}
	return response.ID
}

func (c *Container) Remove(id string) {
	if id == "" {
		return
	}
	rmErr := c.Docker.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{
		Force: true,
	})
	if *c.Err == nil {
		*c.Err = rmErr
	}
}

func (c *Container) RemoveAfterCopy(id string, file *io.ReadCloser) {
	if *file == nil {
		c.Remove(id)
		return
	}

	*file = &closeWrapper{
		ReadCloser: *file,
		After: func() {
			c.Remove(id)
		},
	}
}

type closeWrapper struct {
	io.ReadCloser
	After func()
}

func (c *closeWrapper) Close() error {
	defer c.After()
	return c.ReadCloser.Close()
}
