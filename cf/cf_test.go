package cf_test

import (
	"errors"

	. "github.com/sclevine/cflocal/cf"
	"github.com/sclevine/cflocal/cf/mocks"
	sharedmocks "github.com/sclevine/cflocal/mocks"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("CF", func() {
	var (
		mockCtrl   *gomock.Controller
		mockUI     *sharedmocks.MockUI
		mockHelp   *mocks.MockHelp
		cmd1, cmd2 *mocks.MockCmd
		cf         *CF
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = sharedmocks.NewMockUI()
		mockHelp = mocks.NewMockHelp(mockCtrl)
		cmd1 = mocks.NewMockCmd(mockCtrl)
		cmd2 = mocks.NewMockCmd(mockCtrl)
		cf = &CF{
			UI:      mockUI,
			Help:    mockHelp,
			Cmds:    []Cmd{cmd1, cmd2},
			Version: "some-version",
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Run", func() {
		Context("when the subcommand is 'help'", func() {
			It("should show the long usage text", func() {
				mockHelp.EXPECT().Long()
				Expect(cf.Run([]string{"help"})).To(Succeed())
			})

			Context("when printing the usage text fails", func() {
				It("should print an error", func() {
					mockHelp.EXPECT().Long().Return(errors.New("some error"))
					err := cf.Run([]string{"help"})
					Expect(err).To(MatchError("some error"))
				})
			})
		})

		Context("when the subcommand is '[--]version'", func() {
			It("should output the version", func() {
				Expect(cf.Run([]string{"version"})).To(Succeed())
				Expect(mockUI.Out).To(gbytes.Say("CF Local version some-version"))
				Expect(cf.Run([]string{"--version"})).To(Succeed())
				Expect(mockUI.Out).To(gbytes.Say("CF Local version some-version"))
			})
		})

		Context("when the subcommand matches a command", func() {
			It("should run only that command", func() {
				cmd1.EXPECT().Match([]string{"some-cmd"}).Return(false)
				cmd2.EXPECT().Match([]string{"some-cmd"}).Return(true)
				cmd2.EXPECT().Run([]string{"some-cmd"}).Return(nil)

				Expect(cf.Run([]string{"some-cmd"})).To(Succeed())
			})

			Context("when the command returns an error", func() {
				It("should return the command error", func() {
					cmd1.EXPECT().Match([]string{"some-cmd"}).Return(false)
					cmd2.EXPECT().Match([]string{"some-cmd"}).Return(true)
					cmd2.EXPECT().Run([]string{"some-cmd"}).Return(errors.New("some error"))

					err := cf.Run([]string{"some-cmd"})
					Expect(err).To(MatchError("some error"))
				})
			})
		})

		Context("when the subcommand doesn't match a command", func() {
			It("should return an error", func() {
				cmd1.EXPECT().Match([]string{"some-cmd"}).Return(false)
				cmd2.EXPECT().Match([]string{"some-cmd"}).Return(false)

				err := cf.Run([]string{"some-cmd"})
				Expect(err).To(MatchError("invalid command"))
			})
		})
	})
})
