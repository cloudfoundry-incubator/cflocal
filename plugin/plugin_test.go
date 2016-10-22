package plugin_test

import (
	"os"

	. "github.com/sclevine/cflocal/plugin"
	"github.com/sclevine/cflocal/plugin/mocks"

	cfplugin "code.cloudfoundry.org/cli/plugin"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Plugin", func() {
	var (
		mockCtrl *gomock.Controller
		mockUI   *mocks.MockUI
		mockCLI  *mocks.MockCliConnection
		plugin   *Plugin
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI()
		mockCLI = mocks.NewMockCliConnection(mockCtrl)
		plugin = &Plugin{
			UI:      mockUI,
			Version: "100.200.300",
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Run", func() {
		Context("when uninstalling", func() {
			It("should return immediately", func() {
				plugin.Run(mockCLI, []string{"CLI-MESSAGE-UNINSTALL"})
				Expect(len(mockUI.Out.Contents())).To(BeZero())
				Expect(mockUI.Err).NotTo(HaveOccurred())
				Expect(plugin.RunErr).NotTo(HaveOccurred())
			})
		})

		Context("when the Docker daemon cannot be reached", func() {
			It("should set an error and return", func() {
				defer os.Setenv("DOCKER_HOST", os.Getenv("DOCKER_HOST"))
				os.Setenv("DOCKER_HOST", "#$%")
				plugin.Run(mockCLI, []string{"some-command"})
				Expect(plugin.RunErr).To(MatchError("unable to parse docker host `#$%`"))
			})
		})
	})

	Describe("#GetMetadata", func() {
		It("should return metadata about the plugin", func() {
			metadata := plugin.GetMetadata()
			Expect(metadata.Name).To(Equal("cflocal"))
			Expect(metadata.Version).To(Equal(cfplugin.VersionType{100, 200, 300}))
			Expect(metadata.Commands).To(HaveLen(1))
			Expect(metadata.Commands[0].Name).To(Equal("local"))
			Expect(metadata.Commands[0].HelpText).NotTo(BeEmpty())
			Expect(metadata.Commands[0].UsageDetails.Usage).NotTo(BeEmpty())
		})
	})

	Describe("#Help", func() {
		It("should describe how to install the plugin", func() {
			plugin.Help("some-command")
			Expect(mockUI.Out).To(gbytes.Say("Usage: some-command"))
			Expect(mockUI.Out).To(gbytes.Say("cf local help"))
			Expect(mockUI.Err).NotTo(HaveOccurred())
		})
	})
})
