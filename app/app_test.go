package app_test

import (
	"compress/gzip"
	"io"
	"io/ioutil"

	docker "github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/sclevine/cflocal/app"
	"github.com/sclevine/cflocal/utils"
)

var _ = Describe("App", func() {
	var (
		app  *App
		logs *gbytes.Buffer
	)

	BeforeEach(func() {
		client, err := docker.NewEnvClient()
		Expect(err).NotTo(HaveOccurred())
		logs = gbytes.NewBuffer()
		app = &App{
			DiegoVersion: "0.1482.0",
			GoVersion:    "1.7",
			Docker:       client,
			Logs:         io.MultiWriter(logs, GinkgoWriter),
		}
	})

	Describe("#Stage", func() {
		It("should return a droplet of a staged app", func() {
			appTar, err := utils.TarFile("some-file", []byte("some-contents"))
			Expect(err).NotTo(HaveOccurred())
			colorize := func(text string) string { return text + " %" }
			droplet, err := app.Stage("some-app", colorize, appTar, []string{
				"https://github.com/cloudfoundry/staticfile-buildpack#v1.3.11",
			})
			Expect(err).NotTo(HaveOccurred())
			defer droplet.Close()

			Expect(logs).To(gbytes.Say(`\[some-app\] % \S+ -------> Buildpack version`))
			Expect(logs).To(gbytes.Say(`\[some-app\] % \S+ Downloaded`))
			Expect(logs).To(gbytes.Say("Using"))
			Expect(logs).To(gbytes.Say("Copying"))
			Expect(logs).To(gbytes.Say("Setting up"))

			dropletTar, err := gzip.NewReader(droplet)
			Expect(err).NotTo(HaveOccurred())
			testFile, err := utils.FileFromTar("./app/public/some-file", dropletTar)
			Expect(err).NotTo(HaveOccurred())
			Expect(ioutil.ReadAll(testFile)).To(Equal([]byte("some-contents")))
		})

		It("should make the appropriate requests to the Docker daemon", func() {

		})
	})
})
