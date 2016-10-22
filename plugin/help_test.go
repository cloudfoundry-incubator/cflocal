package plugin_test

import (
	"errors"

	. "github.com/sclevine/cflocal/plugin"
	"github.com/sclevine/cflocal/plugin/mocks"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Help", func() {
	var (
		mockCtrl *gomock.Controller
		mockCLI  *mocks.MockCliConnection
		help     *Help
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCLI = mocks.NewMockCliConnection(mockCtrl)
		help = &Help{
			CLI: mockCLI,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Show", func() {
		It("should run `cf help local`", func() {
			mockCLI.EXPECT().CliCommand("help", "local")
			Expect(help.Show()).To(Succeed())
		})

		Context("when `cf help local` fails", func() {
			It("should return the error", func() {
				mockCLI.EXPECT().CliCommand("help", "local").Return(nil, errors.New("some error"))
				err := help.Show()
				Expect(err).To(MatchError("some error"))
			})
		})
	})
})
