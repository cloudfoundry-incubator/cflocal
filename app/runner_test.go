package app_test

import (
	"bytes"
	"context"
	"io"
	"time"

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

	XDescribe("#Run", func() {
		It("should run the provided droplet with the provided launcher", func(done Done) {
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
				Expect(runner.Run("some-app", colorize, launcher, droplet, launcherSize, dropletSize)).To(Succeed())
				close(done)
			}()
			time.Sleep(5 * time.Second)
			close(exitChan)

			//Eventually(logs, "5s").Should(gbytes.Say(`\[some-app\] % \S+ something`))

			// test that no "some-app-staging-GUID" containers exist
		}, 10)

		Context("on failure", func() {
			// test failure cases using reverse proxy
		})
	})
})

func containerInfo(client *docker.Client, id string) types.ContainerJSON {
	response, err := client.ContainerInspect(context.Background(), id)
	Expect(err).NotTo(HaveOccurred())
	return response
}
