package plugin_test

import (
	"errors"

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
			UI: mockUI,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Run", func() {
		Context("when the subcommand is 'help'", func() {
			It("should run `cf local help`", func() {
				mockCLI.EXPECT().CliCommand("help", "local")
				plugin.Run(mockCLI, []string{"dev", "help"})
			})

			Context("when printing the help text fails", func() {
				It("should print an error", func() {
					mockCLI.EXPECT().CliCommand("help", "local").Return(nil, errors.New("some error"))
					plugin.Run(mockCLI, []string{"local", "help"})
					Expect(mockUI.Err).To(MatchError("some error"))
				})
			})
		})

		Context("when the subcommand is '[--]version'", func() {
			It("should output the version", func() {
				plugin.Version = cfplugin.VersionType{100, 200, 300}
				plugin.Run(mockCLI, []string{"local", "version"})
				Expect(mockUI.Out).To(gbytes.Say("CF Local version 100.200.300"))
				plugin.Run(mockCLI, []string{"local", "--version"})
				Expect(mockUI.Out).To(gbytes.Say("CF Local version 100.200.300"))
			})
		})

		Context("when the subcommand is 'build'", func() {
			It("should build a droplet", func() {
			})
		})

		Context("when uninstalling", func() {
			It("should return immediately without panicking", func() {
				Expect(func() {
					plugin.Run(mockCLI, []string{"CLI-MESSAGE-UNINSTALL"})
				}).NotTo(Panic())
			})
		})
	})

	Describe("#GetMetadata", func() {
		It("should populate the version field", func() {
			plugin.Version = cfplugin.VersionType{100, 200, 300}
			Expect(plugin.GetMetadata().Version).To(Equal(cfplugin.VersionType{100, 200, 300}))
		})
	})
})
