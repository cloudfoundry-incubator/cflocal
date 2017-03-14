package mocks

import (
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega/gbytes"
)

type MockUI struct {
	Out   *gbytes.Buffer
	Err   error
	Reply map[string]string
}

func NewMockUI() *MockUI {
	return &MockUI{
		Out:   gbytes.NewBuffer(),
		Reply: map[string]string{},
	}
}

func (m *MockUI) Prompt(prompt string) string {
	fmt.Fprint(m.Out, prompt, "\n")
	return m.Reply[prompt]
}


func (m *MockUI) Output(message string, args ...interface{}) {
	fmt.Fprintf(m.Out, message+"\n", args...)
}

func (m *MockUI) Warn(message string, args ...interface{}) {
	fmt.Fprintf(m.Out, "Warning: "+message+"\n", args...)
}

func (m *MockUI) Error(err error) {
	if m.Err != nil {
		ginkgo.Fail("Error should not be called twice.")
	}
	m.Err = err
}
