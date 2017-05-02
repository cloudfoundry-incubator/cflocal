package mocks

import (
	"bytes"
	"io"
	"io/ioutil"
)

type MockBuffer struct {
	Buffer
	CloseErr error
	ResetErr error
	result   string
}

type Buffer interface {
	io.ReadWriter
	Len() int
}

func (m *MockBuffer) Close() error {
	result, err := ioutil.ReadAll(m.Buffer)
	if err != nil {
		panic("not readable: " + err.Error())
	}

	m.result = string(result)
	m.Buffer = nil
	return m.CloseErr
}

func (m *MockBuffer) Reset() error {
	if _, err := io.Copy(ioutil.Discard, m.Buffer); err != nil {
		panic("not discardable: " + err.Error())
	}
	return m.ResetErr
}

func (m *MockBuffer) Result() string {
	if m.Buffer != nil {
		panic("not closed")
	}
	return m.result
}

func NewMockBuffer(contents string) *MockBuffer {
	return &MockBuffer{Buffer: bytes.NewBufferString(contents)}
}
