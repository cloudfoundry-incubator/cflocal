package engine_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/sclevine/cflocal/engine"
	"github.com/sclevine/cflocal/utils"
)

var _ = Describe("Container", func() {
	var (
		contr      *Container
		client     *docker.Client
		config     *container.Config
		entrypoint strslice.StrSlice
	)

	BeforeEach(func() {
		entrypoint = strslice.StrSlice{"bash"}
	})

	JustBeforeEach(func() {
		var err error
		client, err = docker.NewEnvClient()
		Expect(err).NotTo(HaveOccurred())
		client.UpdateClientVersion("")

		progress, done := (&Image{Docker: client}).Pull("cloudfoundry/cflinuxfs2")
		go drain(progress)
		Eventually(done, "5m").Should(BeClosed())

		config = &container.Config{
			Hostname:   "test-container",
			Image:      "cloudfoundry/cflinuxfs2",
			Env:        []string{"SOME-KEY=some-value"},
			Labels:     map[string]string{"some-label-key": "some-label-value"},
			Entrypoint: entrypoint,
		}
		hostConfig := &container.HostConfig{
			PortBindings: nat.PortMap{
				"8080/tcp": {{HostIP: "127.0.0.1", HostPort: freePort()}},
			},
		}
		contr, err = NewContainer(client, config, hostConfig)
		Expect(err).NotTo(HaveOccurred())
	})

	containerFound := func(id string) bool {
		_, err := client.ContainerInspect(context.Background(), contr.ID)
		if err != nil {
			ExpectWithOffset(1, docker.IsErrContainerNotFound(err)).To(BeTrue())
			return false
		}
		return true
	}

	containerRunning := func(id string) bool {
		info, err := client.ContainerInspect(context.Background(), contr.ID)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		return info.State.Running
	}

	AfterEach(func() {
		if containerFound(contr.ID) {
			Expect(contr.Close()).To(Succeed())
			Expect(containerFound(contr.ID)).To(BeFalse())
		}
		Expect(client.Close()).To(Succeed())
	})

	Describe("#Close", func() {
		It("should remove the container", func() {
			Expect(containerFound(contr.ID)).To(BeTrue())
			Expect(contr.Close()).To(Succeed())
			Expect(containerFound(contr.ID)).To(BeFalse())
		})

		It("should return an error if already closed", func() {
			Expect(contr.Close()).To(Succeed())
			Expect(contr.Close()).To(MatchError(ContainSubstring("No such container")))
		})
	})

	Describe("#CloseAfterStream", func() {
		It("should configure the provided stream to remove the container when it's closed", func() {
			closer := &closeTester{}
			stream := NewStream(closer, 100)
			Expect(contr.CloseAfterStream(&stream)).To(Succeed())

			Expect(closer.closed).To(BeFalse())
			Expect(containerFound(contr.ID)).To(BeTrue())

			Expect(stream.Close()).To(Succeed())

			Expect(closer.closed).To(BeTrue())
			Expect(containerFound(contr.ID)).To(BeFalse())
		})

		It("should return a container removal error if no other close error occurs", func() {
			Expect(contr.Close()).To(Succeed())

			closer := &closeTester{}
			stream := NewStream(closer, 100)
			Expect(contr.CloseAfterStream(&stream)).To(Succeed())

			Expect(contr.Close()).To(MatchError(ContainSubstring("No such container")))
			closer.err = errors.New("some error")
			Expect(stream.Close()).To(MatchError("some error"))
		})

		It("should close the container immediately if the stream is empty", func() {
			stream := NewStream(nil, 100)
			Expect(contr.CloseAfterStream(&stream)).To(Succeed())
			Expect(containerFound(contr.ID)).To(BeFalse())
		})
	})

	Describe("#Start", func() {
		Context("when signaled to exit", func() {
			BeforeEach(func() {
				entrypoint = strslice.StrSlice{
					"bash", "-c",
					`echo some-logs-stdout && \
					 >&2 echo some-logs-stderr && \
					 sleep 60`,
				}
			})

			It("should start the container, stream logs, and return status 128", func(done Done) {
				exit := make(chan struct{})
				contr.Exit = exit

				logs := gbytes.NewBuffer()
				go func() {
					defer GinkgoRecover()
					Expect(contr.Start("some-prefix", logs)).To(Equal(int64(128)))
					close(done)
				}()
				Eventually(try(containerRunning, contr.ID)).Should(BeTrue())
				Eventually(logs.Contents).Should(ContainSubstring("Z some-logs-stdout"))
				Eventually(logs.Contents).Should(ContainSubstring("Z some-logs-stderr"))

				close(exit)

				Eventually(try(containerRunning, contr.ID)).Should(BeFalse())
			}, 5)
		})

		Context("when the command finishes successfully", func() {
			BeforeEach(func() {
				entrypoint = strslice.StrSlice{
					"bash", "-c",
					`echo some-logs-stdout && \
					 >&2 echo some-logs-stderr && \
					 sleep 0`,
				}
			})

			It("should start the container, stream logs, and return status 0", func() {
				logs := gbytes.NewBuffer()
				Expect(contr.Start("some-prefix", logs)).To(Equal(int64(0)))
				Expect(containerRunning(contr.ID)).To(BeFalse())
				Expect(logs.Contents()).To(ContainSubstring("Z some-logs-stdout"))
				Expect(logs.Contents()).To(ContainSubstring("Z some-logs-stderr"))
			})
		})

		It("should return an error when the container cannot be started", func() {
			Expect(contr.Close()).To(Succeed())
			_, err := contr.Start("some-prefix", gbytes.NewBuffer())
			Expect(err).To(MatchError(ContainSubstring("No such container")))
		})
	})

	Describe("#Commit", func() {
		It("should create an image using the state of the container", func() {
			ctx := context.Background()

			inBuffer := bytes.NewBufferString("some-data")
			inStream := NewStream(ioutil.NopCloser(inBuffer), int64(inBuffer.Len()))
			Expect(contr.CopyTo(inStream, "/some-path")).To(Succeed())

			id, err := contr.Commit("some-ref")
			Expect(err).NotTo(HaveOccurred())
			defer client.ImageRemove(ctx, id, types.ImageRemoveOptions{
				Force:         true,
				PruneChildren: true,
			})

			info, _, err := client.ImageInspectWithRaw(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			info.Config.Env = scrubProxyEnv(info.Config.Env)
			Expect(info.Config).To(Equal(config))
			Expect(info.Author).To(Equal("CF Local"))
			Expect(info.RepoTags[0]).To(Equal("some-ref:latest"))

			config.Image = "some-ref:latest"
			contr2, err := NewContainer(client, config, nil)
			Expect(err).NotTo(HaveOccurred())
			defer contr2.Close()

			outStream, err := contr2.CopyFrom("/some-path")
			Expect(err).NotTo(HaveOccurred())
			Expect(ioutil.ReadAll(outStream)).To(Equal([]byte("some-data")))
			Expect(outStream.Size).To(Equal(inStream.Size))
		})

		It("should return an error if committing fails", func() {
			_, err := contr.Commit("$%^some-ref")
			Expect(err).To(MatchError("invalid reference format"))
		})
	})

	Describe("#ExtractTo", func() {
		It("should extract the provided tarball into the container", func() {
			inBuffer := bytes.NewBufferString("some-data")
			inSize := int64(inBuffer.Len())
			inTar, err := utils.TarFile("some-file", inBuffer, inSize, 0755)
			Expect(err).NotTo(HaveOccurred())

			Expect(contr.ExtractTo(inTar, "/root")).To(Succeed())
			outStream, err := contr.CopyFrom("/root/some-file")
			Expect(err).NotTo(HaveOccurred())
			Expect(ioutil.ReadAll(outStream)).To(Equal([]byte("some-data")))
			Expect(outStream.Size).To(Equal(inSize))
		})

		It("should return an error if extracting fails", func() {
			err := contr.ExtractTo(nil, "/some-bad-path")
			Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
		})
	})

	Describe("#CopyTo", func() {
		It("should copy the stream into the container and close it", func() {
			inBuffer := bytes.NewBufferString("some-data")
			inCloseTester := &closeTester{Reader: inBuffer}
			inStream := NewStream(inCloseTester, int64(inBuffer.Len()))
			Expect(contr.CopyTo(inStream, "/some-path/some-file")).To(Succeed())

			Expect(inCloseTester.closed).To(BeTrue())

			outStream, err := contr.CopyFrom("/some-path/some-file")
			Expect(err).NotTo(HaveOccurred())
			Expect(ioutil.ReadAll(outStream)).To(Equal([]byte("some-data")))
			Expect(outStream.Size).To(Equal(inStream.Size))
		})

		It("should return an error if tarring fails", func() {
			inBuffer := bytes.NewBufferString("some-data")
			inStream := NewStream(&closeTester{Reader: inBuffer}, 100)
			err := contr.CopyTo(inStream, "/some-path/some-file")
			Expect(err).To(MatchError("EOF"))
		})

		It("should return an error if extracting fails", func() {
			inBuffer := bytes.NewBufferString("some-data")
			inStream := NewStream(&closeTester{Reader: inBuffer}, int64(inBuffer.Len()))
			err := contr.CopyTo(inStream, "/dev/stdout/bad-path")
			Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
		})

		It("should return an error if closing fails", func() {
			inBuffer := bytes.NewBufferString("some-data")
			inCloseTester := &closeTester{Reader: inBuffer, err: errors.New("some error")}
			inStream := NewStream(inCloseTester, int64(inBuffer.Len()))
			err := contr.CopyTo(inStream, "/some-path/some-file")
			Expect(err).To(MatchError("some error"))
		})
	})

	Describe("#CopyFrom", func() {
		It("should copy the contents of a file out of the container", func() {
			stream, err := contr.CopyFrom("/etc/timezone")
			Expect(err).NotTo(HaveOccurred())
			defer stream.Close()
			Expect(ioutil.ReadAll(stream)).To(Equal([]byte("Etc/UTC\n")))
			Expect(stream.Size).To(Equal(int64(8)))
			Expect(stream.Close()).To(Succeed())
			// TODO: test closing of tar
		})

		It("should return an error if copying fails", func() {
			_, err := contr.CopyFrom("/some-bad-path")
			Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
		})

		It("should return an error if untarring fails", func() {
			_, err := contr.CopyFrom("/root")
			Expect(err).To(MatchError("EOF"))
			// TODO: test closing of tar
		})
	})
})

func try(f func(string) bool, id string) func() bool {
	return func() bool {
		return f(id)
	}
}

func drain(c <-chan string) {
	for range c {
	}
}

func freePort() string {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	defer listener.Close()
	address := listener.Addr().String()
	return strings.SplitN(address, ":", 2)[1]
}

func scrubProxyEnv(old []string) (new []string) {
	for _, v := range old {
		if !strings.Contains(v, "proxy") {
			new = append(new, v)
		}
	}
	return new
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
