package plugin_test

import (
	"github.com/sclevine/cflocal/plugin"
	"github.com/sclevine/cflocal/plugin/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("NonTranslatingUI", func() {
	var (
		mockUI *mocks.MockUI
		ui     *plugin.NonTranslatingUI
	)

	BeforeEach(func() {
		mockUI = mocks.NewMockUI()
		ui = &plugin.NonTranslatingUI{UI: mockUI}
	})

	Describe("#Confirm", func() {
		Context("when the user enters yes", func() {
			It("should return true", func() {
				mockUI.Reply["some question"] = "yes"
				Expect(ui.Confirm("some question")).To(BeTrue())
				Expect(mockUI.Stdout).To(gbytes.Say("some question"))
			})
		})

		Context("when the user enters y", func() {
			It("should return true", func() {
				mockUI.Reply["some question"] = "y"
				Expect(ui.Confirm("some question")).To(BeTrue())
				Expect(mockUI.Stdout).To(gbytes.Say("some question"))
			})
		})

		Context("when the user enters Y", func() {
			It("should return true", func() {
				mockUI.Reply["some question"] = "Y"
				Expect(ui.Confirm("some question")).To(BeTrue())
				Expect(mockUI.Stdout).To(gbytes.Say("some question"))
			})
		})

		Context("when the user enters anything else", func() {
			It("should return false", func() {
				mockUI.Reply["some question"] = "some answer"
				Expect(ui.Confirm("some question")).To(BeFalse())
				Expect(mockUI.Stdout).To(gbytes.Say("some question"))
			})
		})
	})
})
