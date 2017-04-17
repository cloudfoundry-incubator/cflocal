package mocks

import (
	"bytes"
	"io"
	"io/ioutil"
)

type MockBuffer struct {
	io.ReadWriter
	CloseErr error
	result   string
}

func (m *MockBuffer) Close() error {
	result, err := ioutil.ReadAll(m.ReadWriter)
	if err != nil {
		panic("not readable: " + err.Error())
	}

	m.result = string(result)
	m.ReadWriter = nil
	return m.CloseErr
}

func (m *MockBuffer) Result() string {
	if m.ReadWriter != nil {
		panic("not closed")
	}
	return m.result
}

func NewMockBuffer(contents string) *MockBuffer {
	return &MockBuffer{ReadWriter: bytes.NewBufferString(contents)}
}
