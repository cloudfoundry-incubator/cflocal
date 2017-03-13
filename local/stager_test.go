package local_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"

	docker "github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/sclevine/cflocal/local"
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
			StackVersion: "1.86.0",
			UpdateRootFS: true,
			Docker:       client,
			Logs:         io.MultiWriter(logs, GinkgoWriter),
		}
	})

	Describe("#Stage", func() {
		It("should return a droplet of a staged app", func() {
			appFileContents := bytes.NewBufferString("some-contents")
			appTar, err := utils.TarFile("some-file", appFileContents, int64(appFileContents.Len()), 0644)
			Expect(err).NotTo(HaveOccurred())
			droplet, err := stager.Stage(&StageConfig{
				AppTar:     appTar,
				Buildpacks: []string{"https://github.com/sclevine/cflocal-buildpack#v0.0.1"},
				AppConfig: &AppConfig{
					Name: "some-app",
					StagingEnv: map[string]string{
						"TEST_STAGING_ENV_KEY": "test-staging-env-value",
						"MEMORY_LIMIT":         "256m",
					},
					RunningEnv: map[string]string{
						"SOME_NA_KEY": "some-na-value",
					},
					Env: map[string]string{
						"TEST_ENV_KEY": "test-env-value",
						"MEMORY_LIMIT": "1024m",
					},
				},
			}, percentColor)
			Expect(err).NotTo(HaveOccurred())
			defer droplet.Close()

			Expect(logs.Contents()).To(MatchRegexp(`\[some-app\] % \S+ Compile message from stderr\.`))
			Expect(logs).To(gbytes.Say(`\[some-app\] % \S+ Compile arguments: /tmp/app /tmp/cache`))
			Expect(logs).To(gbytes.Say(`\[some-app\] % \S+ Compile message from stdout\.`))

			Expect(droplet.Size).To(BeNumerically(">", 500))
			Expect(droplet.Size).To(BeNumerically("<", 1000))

			dropletTar, err := gzip.NewReader(droplet)
			Expect(err).NotTo(HaveOccurred())
			dropletBuffer, err := ioutil.ReadAll(dropletTar)
			Expect(err).NotTo(HaveOccurred())

			Expect(fileFromTar("./app/some-file", dropletBuffer)).To(Equal("some-contents"))
			Expect(fileFromTar("./staging_info.yml", dropletBuffer)).To(ContainSubstring("start_command"))
			Expect(fileFromTar("./app/env", dropletBuffer)).To(Equal(stagingEnvFixture))

			// test that no "some-app-staging-GUID" containers exist

			// test that termination via ExitChan works

			// test skipping detection when only one buildpack
		})

		Context("on failure", func() {
			// test failure cases using reverse proxy
		})
	})

	Describe("#Download", func() {
		It("should return the specified file", func() {
			launcher, err := stager.Download("/tmp/lifecycle/launcher")
			Expect(err).NotTo(HaveOccurred())
			defer launcher.Close()

			Expect(launcher.Size).To(Equal(int64(3053594)))

			launcherBytes, err := ioutil.ReadAll(launcher)
			Expect(err).NotTo(HaveOccurred())
			Expect(launcherBytes).To(HaveLen(3053594))

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
