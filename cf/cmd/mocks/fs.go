// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/sclevine/cflocal/cf/cmd (interfaces: FS)

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	fs "github.com/sclevine/cflocal/fs"
	io "io"
	reflect "reflect"
	time "time"
)

// MockFS is a mock of FS interface
type MockFS struct {
	ctrl     *gomock.Controller
	recorder *MockFSMockRecorder
}

// MockFSMockRecorder is the mock recorder for MockFS
type MockFSMockRecorder struct {
	mock *MockFS
}

// NewMockFS creates a new mock instance
func NewMockFS(ctrl *gomock.Controller) *MockFS {
	mock := &MockFS{ctrl: ctrl}
	mock.recorder = &MockFSMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockFS) EXPECT() *MockFSMockRecorder {
	return m.recorder
}

// Abs mocks base method
func (m *MockFS) Abs(arg0 string) (string, error) {
	ret := m.ctrl.Call(m, "Abs", arg0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Abs indicates an expected call of Abs
func (mr *MockFSMockRecorder) Abs(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Abs", reflect.TypeOf((*MockFS)(nil).Abs), arg0)
}

// MakeDirAll mocks base method
func (m *MockFS) MakeDirAll(arg0 string) error {
	ret := m.ctrl.Call(m, "MakeDirAll", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// MakeDirAll indicates an expected call of MakeDirAll
func (mr *MockFSMockRecorder) MakeDirAll(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MakeDirAll", reflect.TypeOf((*MockFS)(nil).MakeDirAll), arg0)
}

// OpenFile mocks base method
func (m *MockFS) OpenFile(arg0 string) (fs.ReadResetWriteCloser, int64, error) {
	ret := m.ctrl.Call(m, "OpenFile", arg0)
	ret0, _ := ret[0].(fs.ReadResetWriteCloser)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// OpenFile indicates an expected call of OpenFile
func (mr *MockFSMockRecorder) OpenFile(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OpenFile", reflect.TypeOf((*MockFS)(nil).OpenFile), arg0)
}

// ReadFile mocks base method
func (m *MockFS) ReadFile(arg0 string) (io.ReadCloser, int64, error) {
	ret := m.ctrl.Call(m, "ReadFile", arg0)
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ReadFile indicates an expected call of ReadFile
func (mr *MockFSMockRecorder) ReadFile(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadFile", reflect.TypeOf((*MockFS)(nil).ReadFile), arg0)
}

// Watch mocks base method
func (m *MockFS) Watch(arg0 string, arg1 time.Duration) (<-chan time.Time, chan<- struct{}, error) {
	ret := m.ctrl.Call(m, "Watch", arg0, arg1)
	ret0, _ := ret[0].(<-chan time.Time)
	ret1, _ := ret[1].(chan<- struct{})
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Watch indicates an expected call of Watch
func (mr *MockFSMockRecorder) Watch(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockFS)(nil).Watch), arg0, arg1)
}

// WriteFile mocks base method
func (m *MockFS) WriteFile(arg0 string) (io.WriteCloser, error) {
	ret := m.ctrl.Call(m, "WriteFile", arg0)
	ret0, _ := ret[0].(io.WriteCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WriteFile indicates an expected call of WriteFile
func (mr *MockFSMockRecorder) WriteFile(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteFile", reflect.TypeOf((*MockFS)(nil).WriteFile), arg0)
}
