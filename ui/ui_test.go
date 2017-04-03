package ui_test

import (
	"errors"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/sclevine/cflocal/ui"
)

var _ = Describe("UI", func() {
	var (
		out, err, in *gbytes.Buffer
		ui           *UI
	)

	BeforeEach(func() {
		out = gbytes.NewBuffer()
		err = gbytes.NewBuffer()
		in = gbytes.NewBuffer()
		ui = &UI{Out: out, Err: err, In: in}
	})

	Describe("#Prompt", func() {
		It("should output the prompt and return the user's entry", func() {
			io.WriteString(in, "some answer\n")
			response := ui.Prompt("some question")
			Expect(out).To(gbytes.Say("some question"))
			Expect(response).To(Equal("some answer"))
		})

		Context("when the input cannot be read", func() {
			It("should output the prompt and return an empty string", func() {
				response := ui.Prompt("some question")
				Expect(out).To(gbytes.Say("some question"))
				Expect(response).To(BeEmpty())
			})
		})
	})

	Describe("#Output", func() {
		It("should output the provided format string", func() {
			ui.Output("%s format", "some")
			Expect(out).To(gbytes.Say("some format"))
		})
	})

	Describe("#Warning", func() {
		Context("when stderr is connected", func() {
			It("should output the provided warning as to stderr", func() {
				ui.ErrIsTerm = true
				ui.Warn("%s warning", "some")
				Expect(err).To(gbytes.Say("Warning: some warning"))
			})
		})

		Context("when stderr is not connected", func() {
			It("should output the provided warning to stdout", func() {
				ui.Warn("%s warning", "some")
				Expect(out).To(gbytes.Say("Warning: some warning"))
			})
		})
	})

	Describe("#Error", func() {
		Context("when stderr is connected", func() {
			It("should output the provided error as to stderr followed by FAILED", func() {
				ui.ErrIsTerm = true
				ui.Error(errors.New("some error"))
				Expect(err).To(gbytes.Say("Error: some error"))
				Expect(out).To(gbytes.Say("FAILED"))
			})
		})

		Context("when stderr is not connected", func() {
			It("should output the provided error to stdout followed by FAILED", func() {
				ui.Error(errors.New("some error"))
				Expect(out).To(gbytes.Say("Error: some error"))
				Expect(out).To(gbytes.Say("FAILED"))
			})
		})
	})

	PDescribe("#Loading", func() {
		// TODO: test loading bar
	})
})
