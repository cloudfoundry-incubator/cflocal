package mocks

import (
	"fmt"

	"github.com/onsi/gomega/gbytes"
)

// Mock of UI interface
type MockUI struct {
	Stdout *gbytes.Buffer
	Stderr *gbytes.Buffer
	Reply  map[string]string
}

func NewMockUI() *MockUI {
	return &MockUI{
		Stdout: gbytes.NewBuffer(),
		Stderr: gbytes.NewBuffer(),
		Reply:  map[string]string{},
	}
}

func (m *MockUI) Ask(prompt string) string {
	fmt.Fprint(m.Stdout, prompt, "\n")
	return m.Reply[prompt]
}

func (m *MockUI) Failed(message string, args ...interface{}) {
	fmt.Fprintf(m.Stderr, message+"\n", args...)
	panic("UI PANIC")
}

func (m *MockUI) Say(message string, args ...interface{}) {
	fmt.Fprintf(m.Stdout, message+"\n", args...)
}
