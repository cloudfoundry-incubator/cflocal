package utils

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"
	gouuid "github.com/nu7hatch/gouuid"
)

type Container struct {
	Name       string
	Config     *container.Config
	HostConfig *container.HostConfig
	Docker     *docker.Client
	Err        *error
	mutex      *sync.Mutex
	id         string
}

func (c *Container) ID() string {
	return c.id
}

func (c *Container) Create() {
	uuid, err := gouuid.NewV4()
	if err != nil {
		*c.Err = err
		return
	}

	response, err := c.Docker.ContainerCreate(context.Background(), c.Config, c.HostConfig, nil, fmt.Sprintf("%s-%s", c.Name, uuid))
	if err != nil {
		*c.Err = err
		return
	}
	c.mutex = &sync.Mutex{}
	c.id = response.ID
}

func (c *Container) Remove() {
	c.mutex.Lock()
	if c.id == "" {
		return
	}
	rmErr := c.Docker.ContainerRemove(context.Background(), c.id, types.ContainerRemoveOptions{
		Force: true,
	})
	if *c.Err == nil {
		*c.Err = rmErr
	}
	if rmErr == nil {
		c.id = ""
	}
	c.mutex.Unlock()
}

func (c *Container) RemoveAfterCopy(file *io.ReadCloser) {
	if *file == nil {
		c.Remove()
		return
	}

	*file = &closeWrapper{
		ReadCloser: *file,
		After: func() {
			c.Remove()
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
