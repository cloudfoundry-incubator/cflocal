package plugin_test

import (
	"errors"

	cfplugin "code.cloudfoundry.org/cli/plugin"
	"github.com/sclevine/cflocal/plugin"
	"github.com/sclevine/cflocal/plugin/mocks"

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
		cflocal  *plugin.Plugin
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI()
		mockCLI = mocks.NewMockCliConnection(mockCtrl)
		cflocal = &plugin.Plugin{
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
				cflocal.Run(mockCLI, []string{"dev", "help"})
			})

			Context("when printing the help text fails", func() {
				It("should print an error", func() {
					mockCLI.EXPECT().CliCommand("help", "local").Return(nil, errors.New("some error"))
					Expect(func() {
						cflocal.Run(mockCLI, []string{"local", "help"})
					}).To(Panic())
					Expect(mockUI.Stderr).To(gbytes.Say("Error: some error"))
				})
			})
		})

		Context("when the subcommand is '[--]version'", func() {
			It("should output the version", func() {
				cflocal.Version = cfplugin.VersionType{100, 200, 300}
				cflocal.Run(mockCLI, []string{"local", "version"})
				Expect(mockUI.Stdout).To(gbytes.Say("CF Local version 100.200.300"))
				cflocal.Run(mockCLI, []string{"local", "--version"})
				Expect(mockUI.Stdout).To(gbytes.Say("CF Local version 100.200.300"))
			})
		})

		Context("when the subcommand is 'build'", func() {
			It("should build a droplet", func() {
			})
		})

		Context("when uninstalling", func() {
			It("should return immediately", func() {
				Expect(func() {
					cflocal.Run(mockCLI, []string{"CLI-MESSAGE-UNINSTALL"})
				}).NotTo(Panic())
			})
		})
	})

	Describe("#GetMetadata", func() {
		It("should populate the version field", func() {
			cflocal.Version = cfplugin.VersionType{100, 200, 300}
			Expect(cflocal.GetMetadata().Version).To(Equal(cfplugin.VersionType{100, 200, 300}))
		})
	})
})
