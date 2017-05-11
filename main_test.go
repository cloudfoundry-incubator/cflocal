package main_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/docker/docker/pkg/archive"
	gouuid "github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/sclevine/cflocal/fixtures"
)

var (
	pluginPath string
	tempHome   string
	spaceName  string

	resetEnv func()
)

var _ = BeforeSuite(func() {
	var err error

	tempHome, err = ioutil.TempDir("", "cflocal")
	Expect(err).NotTo(HaveOccurred())

	var set func(k, v string)
	set, resetEnv = setEnv()

	set("CF_HOME", tempHome)
	set("CF_PLUGIN_HOME", filepath.Join(tempHome, "plugins"))

	pluginPath, err = gexec.Build("github.com/sclevine/cflocal", "-ldflags", "-X main.Version=100.200.300")
	Expect(err).NotTo(HaveOccurred())

	session, err := gexec.Start(exec.Command(pluginPath), GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, "1m").Should(gexec.Exit(0))
	Expect(session).To(gbytes.Say("Plugin successfully installed. Current version: 100.200.300"))

	spaceName = uniqueName("cflocal-space")

	cf("api", getEnv("CF_API"), ifSet("CF_SKIP_SSL_VALIDATION", "--skip-ssl-validation"))
	cf("auth", getEnv("CF_USER"), getEnv("CF_PASSWORD"))
	cf("create-space", spaceName, "-o", getEnv("CF_ORG"))
	cf("target", "-s", spaceName, "-o", getEnv("CF_ORG"))
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	cf("delete-space", spaceName, "-o", getEnv("CF_ORG"), "-f")
	Expect(os.RemoveAll(tempHome)).To(Succeed())
	resetEnv()
})

var _ = Describe("CF Local", func() {
	Context("when executed directly", func() {
		It("should output a helpful usage message when run with help flags", func() {
			pluginCmd := exec.Command(pluginPath, "--help")
			session, err := gexec.Start(pluginCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "5s").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("After installing, run: cf local help"))
		})

		It("should upgrade the plugin if it is already installed", func() {
			pluginCmd := exec.Command(pluginPath)
			session, err := gexec.Start(pluginCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "1m").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("Plugin successfully upgraded. Current version: 100.200.300"))
		})

		It("should output an error message when the cf CLI is unavailable", func() {
			pluginCmd := exec.Command(pluginPath)
			pluginCmd.Env = []string{}
			session, err := gexec.Start(pluginCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "1m").Should(gexec.Exit(1))
			Expect(session.Out).To(gbytes.Say("Error: failed to determine cf CLI version"))
			Expect(session.Out).To(gbytes.Say("FAILED"))
		})
	})

	Context("when used as a cf CLI plugin", func() {
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = ioutil.TempDir("", "cflocal")
			Expect(err).NotTo(HaveOccurred())
			Expect(archive.CopyResource(filepath.Join("fixtures", "go-app"), tempDir, false)).To(Succeed())
			Expect(archive.CopyResource(filepath.Join("fixtures", "test-app"), tempDir, false)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tempDir)).To(Succeed())
		})

		It("should setup the staging and running environments to mimic CF", func() {
			By("staging", func() {
				stageCmd := exec.Command("cf", "local", "stage", "some-app", "-b", "https://github.com/sclevine/cflocal-buildpack#v0.0.6")
				stageCmd.Dir = filepath.Join(tempDir, "test-app")
				session, err := gexec.Start(stageCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session, "10m").Should(gexec.Exit(0))
				Expect(session).To(gbytes.Say("Compile message from stdout."))
				Expect(session).To(gbytes.Say("Cache not detected."))
				Expect(session.Out.Contents()).To(ContainSubstring("Compile message from stderr."))

				Expect(os.Stat(filepath.Join(tempDir, "test-app", "some-app.droplet"))).To(WithTransform(os.FileInfo.Size, BeNumerically(">", 0)))
				Expect(os.Stat(filepath.Join(tempDir, "test-app", ".some-app.cache"))).To(WithTransform(os.FileInfo.Size, BeNumerically(">", 0)))
			})

			By("restaging", func() {
				stageCmd := exec.Command("cf", "local", "stage", "some-app", "-b", "https://github.com/sclevine/cflocal-buildpack#v0.0.6")
				stageCmd.Dir = filepath.Join(tempDir, "test-app")
				session, err := gexec.Start(stageCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session, "10m").Should(gexec.Exit(0))
				Expect(session).To(gbytes.Say("Compile message from stdout."))
				Expect(session).To(gbytes.Say("Cache detected."))
				Expect(session.Out.Contents()).To(ContainSubstring("Compile message from stderr."))
			})

			By("running", func() {
				runCmd := exec.Command("cf", "local", "run", "some-app")
				runCmd.Dir = filepath.Join(tempDir, "test-app")
				runCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				session, err := gexec.Start(runCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				message := `Running some-app on port ([\d]+)\.\.\.`
				Eventually(session, "10s").Should(gbytes.Say(message))
				port := regexp.MustCompile(message).FindSubmatch(session.Out.Contents())[1]
				url := fmt.Sprintf("http://localhost:%s", port)

				response := fmt.Sprintf(
					"Staging:\n%s\nRunning:\n%s\n",
					strings.Join(fixtures.StagingEnv("LC_COLLATE=C"), "\n"),
					strings.Join(fixtures.RunningEnv("LC_COLLATE=C"), "\n"),
				)
				Expect(get(url, "10s")).To(Equal(response))

				Eventually(session).Should(gbytes.Say("Log message from stdout."))
				Eventually(session.Out.Contents).Should(ContainSubstring("Log message from stderr."))

				Expect(syscall.Kill(-runCmd.Process.Pid, syscall.SIGINT)).To(Succeed())
				Eventually(session, "5s").Should(gexec.Exit(130))
			})
		})

		It("should successfully stage, run, and push a local app", func() {
			By("staging", func() {
				stageCmd := exec.Command("cf", "local", "stage", "some-app")
				stageCmd.Dir = filepath.Join(tempDir, "go-app")
				session, err := gexec.Start(stageCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session, "10m").Should(gexec.Exit(0))
				Expect(session).To(gbytes.Say("Successfully staged: some-app"))
			})

			By("running", func() {
				runCmd := exec.Command("cf", "local", "run", "some-app")
				runCmd.Dir = filepath.Join(tempDir, "go-app")
				runCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				session, err := gexec.Start(runCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				message := `Running some-app on port ([\d]+)\.\.\.`
				Eventually(session, "10s").Should(gbytes.Say(message))
				port := regexp.MustCompile(message).FindSubmatch(session.Out.Contents())[1]
				url := fmt.Sprintf("http://localhost:%s/some-path", port)

				Expect(get(url, "10s")).To(Equal("Path: /some-path"))
				Expect(syscall.Kill(-runCmd.Process.Pid, syscall.SIGINT)).To(Succeed())

				Eventually(session, "5s").Should(gexec.Exit(130))
			})

			By("pushing", func() {
				cfPushCmd := exec.Command("cf", "push", "some-app", "--no-start", "--random-route")
				cfPushCmd.Dir = filepath.Join(tempDir, "test-app")
				cfSession, err := gexec.Start(cfPushCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(cfSession, "1m").Should(gexec.Exit(0))

				pushCmd := exec.Command("cf", "local", "push", "some-app")
				pushCmd.Dir = filepath.Join(tempDir, "go-app")
				session, err := gexec.Start(pushCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session, "10m").Should(gexec.Exit(0))

				message := `\nurls: (\S+)\n`
				Expect(session).To(gbytes.Say(message))
				route := regexp.MustCompile(message).FindSubmatch(session.Out.Contents())[1]
				url := fmt.Sprintf("http://%s/some-path", route)
				Expect(get(url, "10s")).To(Equal("Path: /some-path"))
			})
		})

		It("should successfully pull, run, and push an app from CF", func() {
			Expect(os.Mkdir(filepath.Join(tempDir, "local-app"), 0777)).To(Succeed())

			By("pulling", func() {
				cfPushCmd := exec.Command("cf", "push", "some-app", "--random-route")
				cfPushCmd.Dir = filepath.Join(tempDir, "go-app")
				cfSession, err := gexec.Start(cfPushCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(cfSession, "5m").Should(gexec.Exit(0))

				pullCmd := exec.Command("cf", "local", "pull", "some-app")
				pullCmd.Dir = filepath.Join(tempDir, "local-app")
				session, err := gexec.Start(pullCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session, "10m").Should(gexec.Exit(0))
				Expect(session).To(gbytes.Say("Successfully downloaded: some-app"))

				cf("delete", "some-app")
			})

			By("running", func() {
				runCmd := exec.Command("cf", "local", "run", "some-app")
				runCmd.Dir = filepath.Join(tempDir, "local-app")
				runCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				session, err := gexec.Start(runCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				message := `Running some-app on port ([\d]+)\.\.\.`
				Eventually(session, "10s").Should(gbytes.Say(message))
				port := regexp.MustCompile(message).FindSubmatch(session.Out.Contents())[1]
				url := fmt.Sprintf("http://localhost:%s/some-path", port)

				Expect(get(url, "10s")).To(Equal("Path: /some-path"))
				Expect(syscall.Kill(-runCmd.Process.Pid, syscall.SIGINT)).To(Succeed())

				Eventually(session, "5s").Should(gexec.Exit(130))
			})

			By("pushing", func() {
				cfPushCmd := exec.Command("cf", "push", "some-app", "--no-start", "--random-route")
				cfPushCmd.Dir = filepath.Join(tempDir, "test-app")
				cfSession, err := gexec.Start(cfPushCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(cfSession, "1m").Should(gexec.Exit(0))

				pushCmd := exec.Command("cf", "local", "push", "some-app")
				pushCmd.Dir = filepath.Join(tempDir, "local-app")
				session, err := gexec.Start(pushCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session, "10m").Should(gexec.Exit(0))

				message := `\nurls: (\S+)\n`
				Expect(session).To(gbytes.Say(message))
				route := regexp.MustCompile(message).FindSubmatch(session.Out.Contents())[1]
				url := fmt.Sprintf("http://%s/some-path", route)
				Expect(get(url, "10s")).To(Equal("Path: /some-path"))
			})
		})

		// TODO: service forwarding / app dir mounts
		// TODO: confirm coverage matches previous local package coverage
	})
})

func setEnv() (set func(k, v string), reset func()) {
	var new []string
	saved := map[string]string{}
	return func(k, v string) {
		if old, ok := os.LookupEnv(k); ok {
			saved[k] = old
		} else {
			new = append(new, k)
		}
		if err := os.Setenv(k, v); err != nil {
			Fail(err.Error(), 1)
		}
	}, func() {
		for k, v := range saved {
			if err := os.Setenv(k, v); err != nil {
				Fail(err.Error(), 1)
			}
			delete(saved, k)
		}
		for _, k := range new {
			if err := os.Unsetenv(k); err != nil {
				Fail(err.Error(), 1)
			}
		}
		new = nil
	}
}

func getEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		Fail("Not set: " + k)
	}
	return v
}

func ifSet(k string, ret string) string {
	if v := os.Getenv(k); v != "true" && v != "1" {
		return ""
	}
	return ret
}

func uniqueName(s string) string {
	guid, err := gouuid.NewV4()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return fmt.Sprintf("%s-%s", s, guid)
}

func cf(args ...string) {
	cmd := exec.Command("cf", args...)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	EventuallyWithOffset(1, session, "10s").Should(gexec.Exit(0))
}

func get(url, timeout string) string {
	var body io.ReadCloser
	EventuallyWithOffset(1, func() error {
		response, err := http.Get(url)
		if err != nil {
			return err
		}
		body = response.Body
		return nil
	}, timeout).Should(Succeed())
	defer body.Close()
	bodyBytes, err := ioutil.ReadAll(body)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return string(bodyBytes)
}
