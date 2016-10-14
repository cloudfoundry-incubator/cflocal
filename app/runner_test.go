package app_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	docker "github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"github.com/docker/docker/api/types"
	. "github.com/sclevine/cflocal/app"
	"github.com/sclevine/cflocal/utils"
)

var _ = Describe("Runner", func() {
	var (
		runner   *Runner
		client   *docker.Client
		logs     *gbytes.Buffer
		exitChan chan struct{}
	)

	BeforeEach(func() {
		var err error
		client, err = docker.NewEnvClient()
		Expect(err).NotTo(HaveOccurred())
		logs = gbytes.NewBuffer()
		exitChan = make(chan struct{})
		runner = &Runner{
			Docker:   client,
			Logs:     io.MultiWriter(logs, GinkgoWriter),
			ExitChan: exitChan,
		}
	})

	Describe("#Run", func() {
		It("should run the provided droplet with the provided launcher", func() {
			stager := &Stager{
				DiegoVersion: "0.1482.0",
				GoVersion:    "1.7",
				Docker:       client,
				Logs:         GinkgoWriter,
			}
			appFileContents := bytes.NewBufferString("some-contents")
			appTar, err := utils.TarFile("some-file", appFileContents, int64(appFileContents.Len()), 0644)
			Expect(err).NotTo(HaveOccurred())
			colorize := func(text string) string { return text + " %" }
			droplet, dropletSize, err := stager.Stage("some-app", colorize, appTar, []string{
				"https://github.com/sclevine/cflocal-buildpack#v0.0.1",
			})
			Expect(err).NotTo(HaveOccurred())
			defer droplet.Close()

			launcher, launcherSize, err := stager.Launcher()
			Expect(err).NotTo(HaveOccurred())
			defer launcher.Close()

			go func() {
				defer GinkgoRecover()
				response, err := http.Get(fmt.Sprintf("http://localhost:%s/", port))
				defer response.Body.Close()
				Expect(err).NotTo(HaveOccurred())
				Expect(ioutil.ReadAll(response.Body)).To(Equal(envFixture))
				close(exitChan)
			}()

			config := &RunConfig{
				Droplet:      droplet,
				DropletSize:  dropletSize,
				Launcher:     launcher,
				LauncherSize: launcherSize,
				IDChan:       idChan,
			}
			status, err := runner.Run("some-app", colorize, config)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(137))

			Expect(logs.Contents()).To(MatchRegexp(`\[some-app\] % \S+ Log message from stdout.`))
			Expect(logs.Contents()).To(MatchRegexp(`\[some-app\] % \S+ Log message from stderr.`))

			Eventually(exitChan, "5s").Should(BeClosed())

			// test that no "some-app-staging-GUID" containers exist
		})

		Context("on failure", func() {
			// test failure cases using reverse proxy
		})
	})
})

func containerInfo(client *docker.Client, id string) (info types.ContainerJSON) {
	EventuallyWithOffset(1, func() (err error) {
		info, err = client.ContainerInspect(context.Background(), id)
		return err
	}, "5s").Should(Succeed())

	return info
}
