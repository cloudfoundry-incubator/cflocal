package plugin_test

import (
	"errors"

	cfplugin "code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/pluginfakes"
	"github.com/sclevine/cflocal/plugin"
	"github.com/sclevine/cflocal/plugin/mocks"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin", func() {
	var (
		mockCtrl          *gomock.Controller
		mockUI            *mocks.MockUI
		fakeCliConnection *pluginfakes.FakeCliConnection
		cflocal           *plugin.Plugin
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockCtrl)
		fakeCliConnection = &pluginfakes.FakeCliConnection{}
		cflocal = &plugin.Plugin{
			UI: mockUI,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Run", func() {
		Context("when printing the help text fails", func() {
			It("should print an error", func() {
				mockUI.EXPECT().Failed("Error: %s.", errors.New("some-error"))
				fakeCliConnection.CliCommandReturns(nil, errors.New("some-error"))

				cflocal.Run(fakeCliConnection, []string{"dev", "help"})
			})
		})
	})

	Describe("#GetMetadata", func() {
		It("should populate the version field", func() {
			cflocal.Version = cfplugin.VersionType{100, 200, 300}
			Expect(cflocal.GetMetadata().Version).To(Equal(cfplugin.VersionType{100, 200, 300}))
		})
	})

	Context("Uninstalling", func() {
		It("should return immediately", func() {
			cflocal.Run(&pluginfakes.FakeCliConnection{}, []string{"CLI-MESSAGE-UNINSTALL"})
			// finish
		})
	})
})
