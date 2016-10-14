package app_test

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"

	docker "github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/sclevine/cflocal/app"
	"github.com/sclevine/cflocal/utils"
)

var _ = Describe("Stager", func() {
	var (
		stager *Stager
		logs   *gbytes.Buffer
	)

	BeforeEach(func() {
		client, err := docker.NewEnvClient()
		Expect(err).NotTo(HaveOccurred())
		logs = gbytes.NewBuffer()
		stager = &Stager{
			DiegoVersion: "0.1482.0",
			GoVersion:    "1.7",
			Docker:       client,
			Logs:         io.MultiWriter(logs, GinkgoWriter),
		}
	})

	Describe("#Stage", func() {
		It("should return a droplet of a staged app", func() {
			appFileContents := bytes.NewBufferString("some-contents")
			appTar, err := utils.TarFile("some-file", appFileContents, int64(appFileContents.Len()), 0644)
			Expect(err).NotTo(HaveOccurred())
			colorize := func(text string) string { return text + " %" }
			droplet, size, err := stager.Stage("some-app", colorize, appTar, []string{
				"https://github.com/sclevine/cflocal-buildpack#v0.0.1",
			})
			Expect(err).NotTo(HaveOccurred())
			defer droplet.Close()

			Expect(logs.Contents()).To(MatchRegexp(`\[some-app\] % \S+ Compile message from stderr\.`))
			Expect(logs).To(gbytes.Say(`\[some-app\] % \S+ Compile arguments: /tmp/app /tmp/cache`))
			Expect(logs).To(gbytes.Say(`\[some-app\] % \S+ Compile message from stdout\.`))

			Expect(size).To(BeNumerically(">", 750))
			Expect(size).To(BeNumerically("<", 850))

			dropletTar, err := gzip.NewReader(droplet)
			Expect(err).NotTo(HaveOccurred())
			dropletBuffer, err := ioutil.ReadAll(dropletTar)
			Expect(err).NotTo(HaveOccurred())

			Expect(fileFromTar("./app/some-file", dropletBuffer)).To(Equal("some-contents"))
			Expect(fileFromTar("./staging_info.yml", dropletBuffer)).To(ContainSubstring("start_command"))
			Expect(fileFromTar("./app/env", dropletBuffer)).To(Equal(stagingEnvFixture))

			// test that no "some-app-staging-GUID" containers exist
		})

		Context("on failure", func() {
			// test failure cases using reverse proxy
		})
	})

	Describe("#Launcher", func() {
		It("should return the Diego launcher", func() {
			launcher, size, err := stager.Launcher()
			Expect(err).NotTo(HaveOccurred())
			defer launcher.Close()

			Expect(size).To(Equal(int64(3053594)))

			launcherBytes, err := ioutil.ReadAll(launcher)
			Expect(err).NotTo(HaveOccurred())
			launcherSum := fmt.Sprintf("%x", md5.Sum(launcherBytes))
			Expect(launcherSum).To(Equal("05cd65b3ee0e98acb06bf39f7accb94d"))

			// test that no "some-app-launcher-GUID" containers exist
		})

		Context("on failure", func() {
			// test failure cases using reverse proxy
		})
	})

	Describe("Dockerfile", func() {
		// test docker image via docker info?
	})
})

func fileFromTar(path string, tarball []byte) string {
	file, err := utils.FileFromTar(path, bytes.NewReader(tarball))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	contents, err := ioutil.ReadAll(file)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return string(contents)
}
