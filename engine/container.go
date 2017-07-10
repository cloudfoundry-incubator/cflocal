package engine

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	gopath "path"
	"strings"
	"time"

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

func (c *Container) Start(logPrefix string, logs io.Writer, config <-chan interface{}, restart <-chan time.Time) (status int64, err error) {
	defer func() {
		if isErrCanceled(err) {
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
			cancel()
		case <-c.Exit:
			restart = nil
			cancel()
		}
	}()
	logQueue := copyStreams(logs, logPrefix)
	defer close(logQueue)

	if err := c.Docker.ContainerStart(ctx, c.ID, types.ContainerStartOptions{}); err != nil {
		return 0, err
	}
	contLogs, err := c.Docker.ContainerLogs(ctx, c.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
	})
	if err != nil {
		return 0, err
	}
	logQueue <- contLogs

	if restart != nil {
		return c.restart(ctx, contLogs, logQueue, config, restart)
	}
	defer contLogs.Close()
	return c.Docker.ContainerWait(ctx, c.ID)
}

func (c *Container) restart(ctx context.Context, contLogs io.ReadCloser, logQueue chan io.Reader, config <-chan interface{}, restart <-chan time.Time) (status int64, err error) {
	// TODO: log on each continue

	// FIXME: new strategy: restart channel backed by container-wait OR by watcher

	for {
		select {
		case config := <-config:
			if config != nil {
				configJSON := &bytes.Buffer{}
				if err := json.NewEncoder(configJSON).Encode(config); err != nil {
					continue
				}
				if err := c.Docker.CopyToContainer(ctx, c.ID, "/tmp/config.json", configJSON, types.CopyToContainerOptions{}); err != nil {
					continue
				}
			}
			wait := time.Second
			if err := c.Docker.ContainerRestart(ctx, c.ID, &wait); err != nil {
				continue
			}
			contJSON, err := c.Docker.ContainerInspect(ctx, c.ID)
			if err != nil {
				continue
			}
			startedAt, err := time.Parse(time.RFC3339Nano, contJSON.State.StartedAt)
			if err != nil {
				startedAt = time.Unix(0, 0)
			}
			contLogs.Close()
			contLogs, err = c.Docker.ContainerLogs(ctx, c.ID, types.ContainerLogsOptions{
				Timestamps: true,
				ShowStdout: true,
				ShowStderr: true,
				Follow:     true,
				Since:      startedAt.Add(-10 * time.Millisecond).Format(time.RFC3339Nano),
			})
			if err != nil {
				continue
			}
			logQueue <- contLogs
		case <-c.Exit:
			defer contLogs.Close()
			return 128, nil
		}
	}
}

func isErrCanceled(err error) bool {
	return err == context.Canceled || (err != nil && strings.HasSuffix(err.Error(), "context canceled"))
}

func copyStreams(dst io.Writer, prefix string) chan<- io.Reader {
	srcs := make(chan io.Reader)
	go func() {
		header := make([]byte, 8)
		for src := range srcs {
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
	}()
	return srcs
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
