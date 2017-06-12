package engine_test

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
	gouuid "github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"time"

	. "github.com/sclevine/cflocal/engine"
)

var _ = Describe("Container", func() {
	var (
		contr      *Container
		config     *container.Config
		entrypoint strslice.StrSlice
	)

	BeforeEach(func() {
		entrypoint = strslice.StrSlice{"bash"}
	})

	JustBeforeEach(func() {
		config = &container.Config{
			Hostname:   "test-container",
			Image:      "sclevine/test",
			Env:        []string{"SOME-KEY=some-value"},
			Labels:     map[string]string{"some-label-key": "some-label-value"},
			Entrypoint: entrypoint,
		}
		hostConfig := &container.HostConfig{
			PortBindings: nat.PortMap{
				"8080/tcp": {{HostIP: "127.0.0.1", HostPort: freePort()}},
			},
		}
		var err error
		contr, err = NewContainer(client, config, hostConfig)
		Expect(err).NotTo(HaveOccurred())
	})

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
					"sh", "-c",
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
					defer close(done)
					defer GinkgoRecover()
					Expect(contr.Start("some-prefix", logs, nil)).To(Equal(int64(128)))
				}()
				Eventually(try(containerRunning, contr.ID)).Should(BeTrue())
				Eventually(logs.Contents).Should(ContainSubstring("Z some-logs-stdout"))
				Eventually(logs.Contents).Should(ContainSubstring("Z some-logs-stderr"))

				close(exit)

				Eventually(try(containerRunning, contr.ID)).Should(BeFalse())
			}, 5)
		})

		Context("when signaled to restart", func() {
			BeforeEach(func() {
				entrypoint = strslice.StrSlice{
					"sh", "-c",
					`echo some-logs-stdout && \
					 >&2 echo some-logs-stderr && \
					 sleep 60`,
				}
			})

			It("should restart until signaled to exit then return status 128", func(done Done) {
				exit := make(chan struct{})
				restart := make(chan time.Time)
				contr.Exit = exit

				logs := gbytes.NewBuffer()
				go func() {
					defer close(done)
					defer GinkgoRecover()
					Expect(contr.Start("some-prefix", logs, restart)).To(Equal(int64(128)))
				}()
				Eventually(try(containerRunning, contr.ID)).Should(BeTrue())
				Eventually(logs).Should(gbytes.Say("Z some-logs-stdout"))
				restart <- time.Time{}
				Eventually(logs, "5s").Should(gbytes.Say("Z some-logs-stdout"))
				restart <- time.Time{}
				Eventually(logs, "5s").Should(gbytes.Say("Z some-logs-stdout"))

				Consistently(logs, "2s").ShouldNot(gbytes.Say("Z some-logs-stdout"))
				close(exit)

				Eventually(try(containerRunning, contr.ID)).Should(BeFalse())
			}, 15)
		})

		Context("when the command finishes successfully", func() {
			BeforeEach(func() {
				entrypoint = strslice.StrSlice{
					"sh", "-c",
					`echo some-logs-stdout && \
					 >&2 echo some-logs-stderr && \
					 sleep 0`,
				}
			})

			It("should start the container, stream logs, and return status 0", func() {
				logs := gbytes.NewBuffer()
				Expect(contr.Start("some-prefix", logs, nil)).To(Equal(int64(0)))
				Expect(containerRunning(contr.ID)).To(BeFalse())
				Expect(logs.Contents()).To(ContainSubstring("Z some-logs-stdout"))
				Expect(logs.Contents()).To(ContainSubstring("Z some-logs-stderr"))
			})
		})

		It("should return an error when the container cannot be started", func() {
			Expect(contr.Close()).To(Succeed())
			_, err := contr.Start("some-prefix", gbytes.NewBuffer(), nil)
			Expect(err).To(MatchError(ContainSubstring("No such container")))
		})
	})

	Describe("#Commit", func() {
		It("should create an image using the state of the container", func() {
			ctx := context.Background()

			inBuffer := bytes.NewBufferString("some-data")
			inStream := NewStream(ioutil.NopCloser(inBuffer), int64(inBuffer.Len()))
			Expect(contr.CopyTo(inStream, "/some-path")).To(Succeed())

			uuid, err := gouuid.NewV4()
			Expect(err).NotTo(HaveOccurred())
			ref := fmt.Sprintf("some-ref-%s", uuid)
			id, err := contr.Commit(ref)
			Expect(err).NotTo(HaveOccurred())
			defer client.ImageRemove(ctx, id, types.ImageRemoveOptions{
				Force:         true,
				PruneChildren: true,
			})

			info, _, err := client.ImageInspectWithRaw(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			info.Config.Env = scrubEnv(info.Config.Env)
			Expect(info.Config).To(Equal(config))
			Expect(info.Author).To(Equal("CF Local"))
			Expect(info.RepoTags[0]).To(Equal(ref + ":latest"))

			config.Image = ref + ":latest"
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
		It("should extract the provided tarball into the container without closing it", func() {
			tarBuffer := &bytes.Buffer{}
			tarball := tar.NewWriter(tarBuffer)
			Expect(tarball.WriteHeader(&tar.Header{Name: "some-file", Size: 9, Mode: 0755})).To(Succeed())
			Expect(tarball.Write([]byte("some-data"))).To(Equal(9))
			Expect(tarball.Close()).To(Succeed())

			tar := &closeTester{Reader: tarBuffer}
			Expect(contr.ExtractTo(tar, "/root")).To(Succeed())
			outStream, err := contr.CopyFrom("/root/some-file")
			Expect(err).NotTo(HaveOccurred())
			Expect(ioutil.ReadAll(outStream)).To(Equal([]byte("some-data")))
			Expect(outStream.Size).To(Equal(int64(9)))
			Expect(tar.closed).To(BeFalse())
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
			err := contr.CopyTo(inStream, "/")
			Expect(err).To(MatchError(ContainSubstring("cannot overwrite")))
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
			stream, err := contr.CopyFrom("/testfile")
			Expect(err).NotTo(HaveOccurred())
			defer stream.Close()
			Expect(ioutil.ReadAll(stream)).To(Equal([]byte("test-data\n")))
			Expect(stream.Size).To(Equal(int64(10)))
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
