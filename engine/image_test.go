package engine_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	"github.com/docker/docker/api/types/strslice"
	gouuid "github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/sclevine/cflocal/engine"
	"github.com/sclevine/cflocal/ui"
)

var _ = Describe("Image", func() {
	var image *Image

	BeforeEach(func() {
		image = &Image{Docker: client}
	})

	Describe("#Build", func() {
		var tag string

		BeforeEach(func() {
			uuid, err := gouuid.NewV4()
			Expect(err).NotTo(HaveOccurred())
			tag = fmt.Sprintf("some-image-%s", uuid)
		})

		AfterEach(func() {
			clearImage(tag)
		})

		It("should build a Dockerfile and tag the resulting image", func() {
			dockerfile := bytes.NewBufferString(`
				FROM sclevine/test
				RUN echo some-data > /some-path
			`)
			dockerfileStream := NewStream(ioutil.NopCloser(dockerfile), int64(dockerfile.Len()))

			progress := image.Build(tag, dockerfileStream)
			naCount := 0
			for p := range progress {
				Expect(p.Err()).NotTo(HaveOccurred())
				if p.Status() == "N/A" {
					naCount++
				} else {
					Expect(p.Status()).To(HaveSuffix("MB"))
				}
			}
			Expect(naCount).To(BeNumerically(">", 0))
			Expect(naCount).To(BeNumerically("<", 20))

			ctx := context.Background()
			info, _, err := client.ImageInspectWithRaw(ctx, tag)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.RepoTags[0]).To(Equal(tag + ":latest"))

			info.Config.Image = tag + ":latest"
			info.Config.Entrypoint = strslice.StrSlice{"bash"}
			contr, err := NewContainer(client, info.Config, nil)
			Expect(err).NotTo(HaveOccurred())
			defer contr.Close()

			outStream, err := contr.CopyFrom("/some-path")
			Expect(err).NotTo(HaveOccurred())
			Expect(ioutil.ReadAll(outStream)).To(Equal([]byte("some-data\n")))
		})

		It("should send an error when the Dockerfile cannot be tarred", func() {
			dockerfile := bytes.NewBufferString(`
				FROM sclevine/test
				RUN echo some-data > /some-path
			`)
			dockerfileStream := NewStream(ioutil.NopCloser(dockerfile), int64(dockerfile.Len())+100)

			progress := image.Build(tag, dockerfileStream)
			var err error
			for p := range progress {
				if pErr := p.Err(); pErr != nil {
					err = pErr
				}
			}
			Expect(err).To(MatchError("EOF"))

			ctx := context.Background()
			_, _, err = client.ImageInspectWithRaw(ctx, tag)
			Expect(err).To(MatchError("Error: No such image: " + tag))
		})

		It("should send an error when the image build request is invalid", func() {
			dockerfile := bytes.NewBufferString(`
				SOME BAD DOCKERFILE
			`)
			dockerfileStream := NewStream(ioutil.NopCloser(dockerfile), int64(dockerfile.Len()))

			progress := image.Build(tag, dockerfileStream)
			var err error
			for p := range progress {
				if pErr := p.Err(); pErr != nil {
					err = pErr
				}
			}
			Expect(err).To(MatchError(HaveSuffix("SOME")))

			ctx := context.Background()
			_, _, err = client.ImageInspectWithRaw(ctx, tag)
			Expect(err).To(MatchError("Error: No such image: " + tag))
		})

		It("should send an error when an error occurs during the image build", func() {
			dockerfile := bytes.NewBufferString(`
				FROM sclevine/test
				RUN false
			`)
			dockerfileStream := NewStream(ioutil.NopCloser(dockerfile), int64(dockerfile.Len()))

			progress := image.Build(tag, dockerfileStream)
			var progressErr ui.Progress
			for progressErr = range progress {
				if progressErr.Err() != nil {
					break
				}
			}
			Expect(progressErr.Err()).To(MatchError(ContainSubstring("non-zero code")))
			Expect(progress).To(BeClosed())

			ctx := context.Background()
			_, _, err := client.ImageInspectWithRaw(ctx, tag)
			Expect(err).To(MatchError("Error: No such image: " + tag))
		})
	})

	Describe("#Pull", func() {
		// TODO: consider using a new image for this test
		It("should pull a Docker image", func() {
			progress := image.Pull("sclevine/test")
			naCount := 0
			for p := range progress {
				Expect(p.Err()).NotTo(HaveOccurred())
				if p.Status() == "N/A" {
					naCount++
				} else {
					Expect(p.Status()).To(HaveSuffix("MB"))
				}
			}
			Expect(naCount).To(BeNumerically(">", 0))
			Expect(naCount).To(BeNumerically("<", 20))

			ctx := context.Background()
			info, _, err := client.ImageInspectWithRaw(ctx, "sclevine/test")
			Expect(err).NotTo(HaveOccurred())
			Expect(info.RepoTags[0]).To(Equal("sclevine/test:latest"))

			info.Config.Image = "sclevine/test:latest"
			info.Config.Entrypoint = strslice.StrSlice{"sh"}
			contr, err := NewContainer(client, info.Config, nil)
			Expect(err).NotTo(HaveOccurred())
			defer contr.Close()

			outStream, err := contr.CopyFrom("/testfile")
			Expect(err).NotTo(HaveOccurred())
			Expect(ioutil.ReadAll(outStream)).To(Equal([]byte("test-data\n")))
		})

		It("should send an error when the image pull request is invalid", func() {
			progress := image.Pull("-----")

			var progressErr ui.Progress
			Expect(progress).To(Receive(&progressErr))
			Expect(progressErr.Err()).To(MatchError(HaveSuffix("invalid reference format")))
			Expect(progress).To(BeClosed())
		})

		It("should send an error when an error occurs during the image build", func() {
			progress := image.Pull("sclevine/bad-test")
			var progressErr ui.Progress
			for progressErr = range progress {
				if progressErr.Err() != nil {
					break
				}
			}
			Expect(progressErr.Err()).To(MatchError(ContainSubstring("not found")))
			Expect(progress).To(BeClosed())
		})
	})
})
