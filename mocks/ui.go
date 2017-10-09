package mocks

import (
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega/gbytes"

	"github.com/sclevine/cflocal/engine"
)

type MockUI struct {
	Err      error
	Out      *gbytes.Buffer
	Reply    map[string]string
	Progress chan engine.Progress
}

func NewMockUI() *MockUI {
	return &MockUI{
		Out:      gbytes.NewBuffer(),
		Reply:    map[string]string{},
		Progress: make(chan engine.Progress, 1),
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
	fmt.Fprintln(m.Out, "Loading: "+message)
	for p := range progress {
		m.Progress <- p
	}
	return nil
}
