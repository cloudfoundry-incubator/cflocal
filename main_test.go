package main_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

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
	Expect(os.RemoveAll(tempHome)).To(Succeed())
	os.Setenv("CF_HOME", oldCFHome)
	os.Setenv("CF_PLUGIN_HOME", oldCFPluginHome)
})

var _ = Describe("CF Local", func() {
	Context("when executed directly", func() {
		It("should output a helpful usage message when run with help flags", func() {
			pluginCommand := exec.Command(pluginPath, "--help")
			session, err := gexec.Start(pluginCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "5s").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("After installing, run: cf local help"))
		})

		It("should upgrade the plugin if it is already installed", func() {
			pluginCommand := exec.Command(pluginPath)
			session, err := gexec.Start(pluginCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "1m").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("Plugin successfully upgraded. Current version: 100.200.300"))
		})

		It("should output an error message when the cf CLI in unavailable", func() {
			pluginCommand := exec.Command(pluginPath)
			pluginCommand.Env = []string{}
			session, err := gexec.Start(pluginCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "1m").Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("Error: failed to determine cf CLI version"))
			Expect(session.Out).To(gbytes.Say("FAILED"))
		})
	})

	Describe("cf local stage", func() {
		It("should build a Docker image that launches the app in the current directory", func() {
			pluginCommand := exec.Command("cf", "local", "stage", "some-app")
			pluginCommand.Dir = "./fixtures/go-app"
			session, err := gexec.Start(pluginCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "10m").Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("Staging of some-app successful."))
		})
	})
})

func loadEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		Fail("missing "+name, 1)
	}
	return value
}

func cf(args ...string) *gexec.Session {
	command := exec.Command("cf", args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return session
}
