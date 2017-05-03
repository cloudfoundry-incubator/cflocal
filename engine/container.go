package engine

import (
	"archive/tar"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	gopath "path"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"
	gouuid "github.com/nu7hatch/gouuid"
)

type Container struct {
	Docker *docker.Client
	Exit   <-chan struct{}
	ID     string
	config *container.Config
}

func NewContainer(docker *docker.Client, config *container.Config, hostConfig *container.HostConfig) (*Container, error) {
	uuid, err := gouuid.NewV4()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	response, err := docker.ContainerCreate(ctx, config, hostConfig, nil, fmt.Sprintf("%s-%s", config.Hostname, uuid))
	if err != nil {
		return nil, err
	}
	return &Container{docker, nil, response.ID, config}, nil
}

func (c *Container) Close() error {
	ctx := context.Background()
	return c.Docker.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{
		Force: true,
	})
}

func (c *Container) CloseAfterStream(stream *Stream) error {
	if stream == nil || stream.ReadCloser == nil {
		return c.Close()
	}
	stream.ReadCloser = &closeWrapper{
		ReadCloser: stream.ReadCloser,
		After:      c.Close,
	}
	return nil
}

type causer interface {
	Cause() error
}

func (c *Container) Start(logPrefix string, logs io.Writer) (status int64, err error) {
	defer func() {
		if err == context.Canceled || (err != nil && strings.HasSuffix(err.Error(), "context canceled")) {
			status, err = 128, nil
			return
		}
	}()
	done := make(chan struct{})
	defer close(done)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-done:
		case <-c.Exit:
			cancel()
		}
	}()

	if err := c.Docker.ContainerStart(ctx, c.ID, types.ContainerStartOptions{}); err != nil {
		return 0, err
	}
	out, err := c.Docker.ContainerLogs(ctx, c.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
	})
	if err != nil {
		return 0, err
	}
	defer out.Close()
	go copyStream(logs, out, logPrefix)
	return c.Docker.ContainerWait(ctx, c.ID)
}

func copyStream(dst io.Writer, src io.Reader, prefix string) {
	header := make([]byte, 8)
	for {
		if _, err := io.ReadFull(src, header); err != nil {
			break
		}
		if n, err := io.WriteString(dst, prefix); err != nil || n != len(prefix) {
			break
		}
		// TODO: bold STDERR
		if _, err := io.CopyN(dst, src, int64(binary.BigEndian.Uint32(header[4:]))); err != nil {
			break
		}
	}
}

func (c *Container) Commit(ref string) (imageID string, err error) {
	ctx := context.Background()
	response, err := c.Docker.ContainerCommit(ctx, c.ID, types.ContainerCommitOptions{
		Reference: ref,
		Author:    "CF Local",
		Pause:     true,
		Config:    c.config,
	})
	return response.ID, err
}

func (c *Container) ExtractTo(tar io.Reader, path string) error {
	ctx := context.Background()
	return c.Docker.CopyToContainer(ctx, c.ID, path, onlyReader(tar), types.CopyToContainerOptions{})
}

func onlyReader(r io.Reader) io.Reader {
	if r == nil {
		return nil
	}
	return struct{ io.Reader }{r}
}

func (c *Container) CopyTo(stream Stream, path string) error {
	tar, err := tarFile(path, stream, stream.Size, 0755)
	if err != nil {
		return err
	}
	if err := c.ExtractTo(tar, "/"); err != nil {
		return err
	}
	return stream.Close()
}

func (c *Container) CopyFrom(path string) (Stream, error) {
	ctx := context.Background()
	tar, stat, err := c.Docker.CopyFromContainer(ctx, c.ID, path)
	if err != nil {
		return Stream{}, err
	}
	reader, _, err := fileFromTar(gopath.Base(path), tar)
	if err != nil {
		tar.Close()
		return Stream{}, err
	}
	return NewStream(splitReadCloser{reader, tar}, stat.Size), nil
}

func fileFromTar(name string, archive io.Reader) (file io.Reader, header *tar.Header, err error) {
	tarball := tar.NewReader(archive)
	for {
		header, err = tarball.Next()
		if err != nil {
			return nil, nil, err
		}
		if header.Name == name {
			break
		}
	}
	return tarball, header, nil
}

type splitReadCloser struct {
	io.Reader
	io.Closer
}

type closeWrapper struct {
	io.ReadCloser
	After func() error
}

func (c *closeWrapper) Close() (err error) {
	defer func() {
		if afterErr := c.After(); err == nil {
			err = afterErr
		}
	}()
	return c.ReadCloser.Close()
}
