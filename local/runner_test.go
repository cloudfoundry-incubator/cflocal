package local_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	docker "github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	. "github.com/sclevine/cflocal/local"
	"github.com/sclevine/cflocal/service"
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
				StackVersion: "1.86.0",
				Docker:       client,
				Logs:         GinkgoWriter,
			}

			appFileContents := bytes.NewBufferString("some-contents")
			appTar, err := utils.TarFile("some-file", appFileContents, int64(appFileContents.Len()), 0644)
			Expect(err).NotTo(HaveOccurred())

			droplet, err := stager.Stage(&StageConfig{
				AppTar:     appTar,
				Buildpacks: []string{"https://github.com/sclevine/cflocal-buildpack#v0.0.1"},
				AppConfig:  &AppConfig{Name: "some-app"},
			}, percentColor)
			Expect(err).NotTo(HaveOccurred())
			defer droplet.Close()

			launcher, err := stager.Download("/tmp/lifecycle/launcher")
			Expect(err).NotTo(HaveOccurred())
			defer launcher.Close()

			sshpassBuf := bytes.NewBufferString("echo sshpass $@")
			sshpass := Stream{ReadCloser: ioutil.NopCloser(sshpassBuf), Size: int64(sshpassBuf.Len())}

			port := freePort()

			go func() {
				defer GinkgoRecover()
				defer close(exitChan)
				Expect(get(fmt.Sprintf("http://localhost:%d/", port))).To(Equal(runningEnvFixture))
			}()

			config := &RunConfig{
				Droplet:  droplet,
				Launcher: launcher,
				Forwarder: Forwarder{
					SSHPass: sshpass,
					Config: &service.ForwardConfig{
						Host: "some-ssh-host",
						Port: "some-port",
						User: "some-user",
						Code: "some-code",
						Forwards: []service.Forward{
							{
								Name: "some-name",
								From: "some-from",
								To:   "some-to",
							},
							{
								Name: "some-other-name",
								From: "some-other-from",
								To:   "some-other-to",
							},
						},
					},
				},
				Port: port,
				AppConfig: &AppConfig{
					Name: "some-app",
					StagingEnv: map[string]string{
						"SOME_NA_KEY": "some-na-value",
					},
					RunningEnv: map[string]string{
						"TEST_RUNNING_ENV_KEY": "test-running-env-value",
						"MEMORY_LIMIT":         "256m",
					},
					Env: map[string]string{
						"TEST_ENV_KEY": "test-env-value",
						"MEMORY_LIMIT": "1024m",
					},
					Services: service.Services{
						"some-type": {{
							Name: "some-name",
						}},
					},
				},
			}
			status, err := runner.Run(config, percentColor)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(137))

			Expect(logs.Contents()).To(MatchRegexp(`\[some-app\] % \S+ Forwarding: some-name some-other-name`))
			Expect(logs.Contents()).To(MatchRegexp(
				`\[some-app\] % \S+ sshpass -p some-code ssh -f -N -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no` +
					` -o LogLevel=ERROR -o ExitOnForwardFailure=yes -o ServerAliveInterval=10 -o ServerAliveCountMax=60` +
					` -p some-port some-user@some-ssh-host -L some-from:some-to -L some-other-from:some-other-to`,
			))
			Expect(logs.Contents()).To(MatchRegexp(`\[some-app\] % \S+ Log message from stdout.`))
			Expect(logs.Contents()).To(MatchRegexp(`\[some-app\] % \S+ Log message from stderr.`))

			Eventually(exitChan, "5s").Should(BeClosed())

			// test that droplet and launcher are closed
			// test that no containers exist
		})

		// test with custom start command
		// test with mounted app dir

		Context("on failure", func() {
			// test failure cases using reverse proxy
		})
	})

	Describe("#Export", func() {
		It("should load the provided droplet into a Docker image with the launcher", func() {
			stager := &Stager{
				DiegoVersion: "0.1482.0",
				GoVersion:    "1.7",
				StackVersion: "1.86.0",
				Docker:       client,
				Logs:         GinkgoWriter,
			}

			appFileContents := bytes.NewBufferString("some-contents")
			appTar, err := utils.TarFile("some-file", appFileContents, int64(appFileContents.Len()), 0644)
			Expect(err).NotTo(HaveOccurred())

			droplet, err := stager.Stage(&StageConfig{
				AppTar:     appTar,
				Buildpacks: []string{"https://github.com/sclevine/cflocal-buildpack#v0.0.1"},
				AppConfig:  &AppConfig{Name: "some-app"},
			}, percentColor)
			Expect(err).NotTo(HaveOccurred())
			defer droplet.Close()

			launcher, err := stager.Download("/tmp/lifecycle/launcher")
			Expect(err).NotTo(HaveOccurred())
			defer launcher.Close()

			config := &ExportConfig{
				Droplet:  droplet,
				Launcher: launcher,
				AppConfig: &AppConfig{
					Name: "some-app",
					StagingEnv: map[string]string{
						"SOME_NA_KEY": "some-na-value",
					},
					RunningEnv: map[string]string{
						"TEST_RUNNING_ENV_KEY": "test-running-env-value",
						"MEMORY_LIMIT":         "256m",
					},
					Env: map[string]string{
						"TEST_ENV_KEY": "test-env-value",
						"MEMORY_LIMIT": "1024m",
					},

					Services: service.Services{
						"some-type": {{
							Name: "some-name",
						}},
					},
				},
			}
			id, err := runner.Export(config, "")
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				rmiCmd := exec.Command("docker", "rmi", "-f", id)
				session, err := gexec.Start(rmiCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, "5s").Should(gexec.Exit(0))
			}()

			port := freePort()

			runCmd := exec.Command("docker", "run", "-d", "-h", "cflocal", "-p", fmt.Sprintf("%d:8080", port), id)
			session, err := gexec.Start(runCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "5s").Should(gexec.Exit(0))
			containerID := strings.TrimSpace(string(session.Out.Contents()))
			defer func() {
				rmCmd := exec.Command("docker", "rm", "-f", containerID)
				session, err := gexec.Start(rmCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, "5s").Should(gexec.Exit(0))
			}()

			Expect(get(fmt.Sprintf("http://localhost:%d/", port))).To(Equal(runningEnvFixture))

			// test that droplet and launcher are closed
			// test that no containers exist
		})

		// test with custom start command

		Context("on failure", func() {
			// test failure cases using reverse proxy
		})
	})
})

func get(url string) string {
	var body io.ReadCloser
	EventuallyWithOffset(1, func() error {
		response, err := http.Get(url)
		if err != nil {
			return err
		}
		body = response.Body
		return nil
	}, "5s").Should(Succeed())
	defer body.Close()
	bodyBytes, err := ioutil.ReadAll(body)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return string(bodyBytes)
}

func freePort() uint {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	defer listener.Close()

	address := listener.Addr().String()
	portStr := strings.SplitN(address, ":", 2)[1]
	port, err := strconv.ParseUint(portStr, 10, 32)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return uint(port)
}
