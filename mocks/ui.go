package mocks

import (
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega/gbytes"
	"github.com/buildpack/forge/engine"
)

type MockUI struct {
	Err   error
	Out   *gbytes.Buffer
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

func (m *MockUI) Output(format string, args ...interface{}) {
	fmt.Fprintf(m.Out, format+"\n", args...)
}

func (m *MockUI) Warn(format string, args ...interface{}) {
	fmt.Fprintf(m.Out, "Warning: "+format+"\n", args...)
}

func (m *MockUI) Error(err error) {
	if m.Err != nil {
		ginkgo.Fail("Error should not be called twice.")
	}
	m.Err = err
}

func (m *MockUI) Loading(message string, progress <-chan engine.Progress) error {
	ginkgo.Fail("UI mock does not support Loading method.")
	return nil
}
