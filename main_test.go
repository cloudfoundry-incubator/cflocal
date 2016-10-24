package main_test

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"

	"github.com/docker/docker/pkg/archive"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var (
	pluginPath      string
	tempHome        string
	oldCFHome       string
	oldCFPluginHome string
)

var _ = BeforeSuite(func() {
	oldCFHome = os.Getenv("CF_HOME")
	oldCFPluginHome = os.Getenv("CF_PLUGIN_HOME")

	var err error
	tempHome, err = ioutil.TempDir("", "cflocal")
	Expect(err).NotTo(HaveOccurred())

	os.Setenv("CF_HOME", tempHome)
	os.Setenv("CF_PLUGIN_HOME", filepath.Join(tempHome, "plugins"))

	pluginPath, err = gexec.Build("github.com/sclevine/cflocal", "-ldflags", "-X main.Version=100.200.300")
	Expect(err).NotTo(HaveOccurred())

	session, err := gexec.Start(exec.Command(pluginPath), GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, "1m").Should(gexec.Exit(0))
	Expect(session).To(gbytes.Say("Plugin successfully installed. Current version: 100.200.300"))
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	Expect(os.RemoveAll(tempHome)).To(Succeed())
	os.Setenv("CF_HOME", oldCFHome)
	os.Setenv("CF_PLUGIN_HOME", oldCFPluginHome)
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

	Describe("staging and running a local app", func() {
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = ioutil.TempDir("", "cflocal")
			Expect(err).NotTo(HaveOccurred())
			Expect(archive.CopyResource(filepath.Join("fixtures", "go-app"), tempDir, false)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tempDir)).To(Succeed())
		})

		It("should successfully stage and run a local app", func() {
			By("staging", func() {
				stageCmd := exec.Command("cf", "local", "stage", "some-app")
				stageCmd.Dir = path.Join(tempDir, "go-app")
				session, err := gexec.Start(stageCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session, "10m").Should(gexec.Exit(0))
				Expect(session).To(gbytes.Say("Successfully staged: some-app"))
			})

			By("running", func() {
				runCmd := exec.Command("cf", "local", "run", "some-app")
				runCmd.Dir = path.Join(tempDir, "go-app")
				runCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				session, err := gexec.Start(runCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Expect(get("http://localhost:3000/some-path", "8s")).To(Equal("Path: /some-path"))
				Expect(syscall.Kill(-runCmd.Process.Pid, syscall.SIGINT)).To(Succeed())

				Eventually(session, "5s").Should(gexec.Exit(130))
				Expect(session).To(gbytes.Say("Running some-app..."))
			})
		})

		PIt("should successfully download and run an app from CF", func() {
			By("staging", func() {
				pullCmd := exec.Command("cf", "local", "pull", "some-app")
				pullCmd.Dir = path.Join(tempDir)
				session, err := gexec.Start(pullCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session, "10m").Should(gexec.Exit(0))
				Expect(session).To(gbytes.Say("Successfully downloaded: some-app"))
			})

			By("running", func() {
				runCmd := exec.Command("cf", "local", "run", "some-app")
				runCmd.Dir = path.Join(tempDir, "go-app")
				runCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				session, err := gexec.Start(runCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Expect(get("http://localhost:3000/some-path", "8s")).To(Equal("Path: /some-path"))
				Expect(syscall.Kill(-runCmd.Process.Pid, syscall.SIGINT)).To(Succeed())

				Eventually(session, "5s").Should(gexec.Exit(130))
				Expect(session).To(gbytes.Say("Running some-app..."))
			})
		})
	})
})

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
