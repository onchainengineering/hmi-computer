// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/coder/coder/v2/tailnet (interfaces: Subscription)
//
// Generated by this command:
//
//	mockgen -destination ./subscriptionmock.go -package tailnettest github.com/coder/coder/v2/tailnet Subscription
//

// Package tailnettest is a generated GoMock package.
package tailnettest

import (
	reflect "reflect"

	proto "github.com/coder/coder/v2/tailnet/proto"
	gomock "go.uber.org/mock/gomock"
)

// MockSubscription is a mock of Subscription interface.
type MockSubscription struct {
	ctrl     *gomock.Controller
	recorder *MockSubscriptionMockRecorder
}

// MockSubscriptionMockRecorder is the mock recorder for MockSubscription.
type MockSubscriptionMockRecorder struct {
	mock *MockSubscription
}

// NewMockSubscription creates a new mock instance.
func NewMockSubscription(ctrl *gomock.Controller) *MockSubscription {
	mock := &MockSubscription{ctrl: ctrl}
	mock.recorder = &MockSubscriptionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSubscription) EXPECT() *MockSubscriptionMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockSubscription) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockSubscriptionMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockSubscription)(nil).Close))
}

// Updates mocks base method.
func (m *MockSubscription) Updates() <-chan *proto.WorkspaceUpdate {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Updates")
	ret0, _ := ret[0].(<-chan *proto.WorkspaceUpdate)
	return ret0
}

// Updates indicates an expected call of Updates.
func (mr *MockSubscriptionMockRecorder) Updates() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Updates", reflect.TypeOf((*MockSubscription)(nil).Updates))
}
