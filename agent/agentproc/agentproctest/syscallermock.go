// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/coder/coder/v2/agent/agentproc (interfaces: Syscaller)

// Package agentproctest is a generated GoMock package.
package agentproctest

import (
	reflect "reflect"
	syscall "syscall"

	gomock "go.uber.org/mock/gomock"
)

// MockSyscaller is a mock of Syscaller interface.
type MockSyscaller struct {
	ctrl     *gomock.Controller
	recorder *MockSyscallerMockRecorder
}

// MockSyscallerMockRecorder is the mock recorder for MockSyscaller.
type MockSyscallerMockRecorder struct {
	mock *MockSyscaller
}

// NewMockSyscaller creates a new mock instance.
func NewMockSyscaller(ctrl *gomock.Controller) *MockSyscaller {
	mock := &MockSyscaller{ctrl: ctrl}
	mock.recorder = &MockSyscallerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSyscaller) EXPECT() *MockSyscallerMockRecorder {
	return m.recorder
}

// GetPriority mocks base method.
func (m *MockSyscaller) GetPriority(arg0 int32) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPriority", arg0)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPriority indicates an expected call of GetPriority.
func (mr *MockSyscallerMockRecorder) GetPriority(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPriority", reflect.TypeOf((*MockSyscaller)(nil).GetPriority), arg0)
}

// Kill mocks base method.
func (m *MockSyscaller) Kill(arg0 int32, arg1 syscall.Signal) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Kill", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Kill indicates an expected call of Kill.
func (mr *MockSyscallerMockRecorder) Kill(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Kill", reflect.TypeOf((*MockSyscaller)(nil).Kill), arg0, arg1)
}

// SetPriority mocks base method.
func (m *MockSyscaller) SetPriority(arg0 int32, arg1 int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetPriority", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetPriority indicates an expected call of SetPriority.
func (mr *MockSyscallerMockRecorder) SetPriority(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetPriority", reflect.TypeOf((*MockSyscaller)(nil).SetPriority), arg0, arg1)
}
