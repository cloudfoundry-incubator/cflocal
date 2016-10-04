package plugin_test

import (
	"github.com/golang/mock/gomock"
	"github.com/sclevine/cflocal/plugin"
	"github.com/sclevine/cflocal/plugin/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NonTranslatingUI", func() {

	var (
		mockCtrl *gomock.Controller
		ui       *plugin.NonTranslatingUI
		mockCFUI *mocks.MockUI
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCFUI = mocks.NewMockUI(mockCtrl)
		ui = &plugin.NonTranslatingUI{
			mockCFUI,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Confirm", func() {
		Context("when the user enters yes", func() {
			It("should return true", func() {
				mockCFUI.EXPECT().Ask("some-question").Return("yes")
				Expect(ui.Confirm("some-question")).To(BeTrue())
			})
		})

		Context("when the user enters y", func() {
			It("should return true", func() {
				mockCFUI.EXPECT().Ask("some-question").Return("y")
				Expect(ui.Confirm("some-question")).To(BeTrue())
			})
		})

		Context("when the user enters Y", func() {
			It("should return true", func() {
				mockCFUI.EXPECT().Ask("some-question").Return("Y")
				Expect(ui.Confirm("some-question")).To(BeTrue())
			})
		})

		Context("when the user enters anything else", func() {
			It("should return false", func() {
				mockCFUI.EXPECT().Ask("some-question").Return("some-answer")
				Expect(ui.Confirm("some-question")).To(BeFalse())
			})
		})
	})
})
