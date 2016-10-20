package plugin_test

import (
	. "github.com/sclevine/cflocal/plugin"
	"github.com/sclevine/cflocal/plugin/mocks"

	cfplugin "code.cloudfoundry.org/cli/plugin"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
			It("should return immediately", func() {
			})
		})

		// test invalid Docker client
	})

	Describe("#GetMetadata", func() {
		It("should populate the version field", func() {
			Expect(plugin.GetMetadata().Version).To(Equal(cfplugin.VersionType{100, 200, 300}))
		})
	})
})
