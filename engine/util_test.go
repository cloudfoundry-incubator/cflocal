package engine_test

import (
	"context"
	"io"
	"net"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	. "github.com/onsi/gomega"
)

func containerFound(id string) bool {
	_, err := client.ContainerInspect(context.Background(), id)
	if err != nil {
		ExpectWithOffset(1, docker.IsErrContainerNotFound(err)).To(BeTrue())
		return false
	}
	return true
}

func containerRunning(id string) bool {
	info, err := client.ContainerInspect(context.Background(), id)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return info.State.Running
}

func clearImage(image string) {
	ctx := context.Background()
	client.ImageRemove(ctx, image, types.ImageRemoveOptions{
		Force:         true,
		PruneChildren: true,
	})
}

func try(f func(string) bool, id string) func() bool {
	return func() bool {
		return f(id)
	}
}

func freePort() string {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	defer listener.Close()
	address := listener.Addr().String()
	return strings.SplitN(address, ":", 2)[1]
}

func scrubEnv(old []string) (new []string) {
	for _, v := range old {
		switch {
		case strings.Contains(v, "proxy="):
		case strings.Contains(v, "PATH="):
		default:
			new = append(new, v)
		}
	}
	return new
}

func changesStatus(interval chan<- time.Time, check <-chan string, status string) bool {
	for total, last := 0, false; total < 5; total++ {
		interval <- time.Time{}
		match := <-check == status
		if last && match {
			return true
		}
		last = match
		time.Sleep(100*time.Millisecond)
	}
	return false
}

type closeTester struct {
	io.Reader
	closed bool
	err    error
}

func (c *closeTester) Close() (err error) {
	c.closed = true
	return c.err
}
