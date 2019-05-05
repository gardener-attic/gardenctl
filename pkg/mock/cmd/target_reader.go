// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/gardenctl/cmd (interfaces: TargetReader)

// Package cmd is a generated GoMock package.
package cmd

import (
	cmd "github.com/gardener/gardenctl/cmd"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockTargetReader is a mock of TargetReader interface
type MockTargetReader struct {
	ctrl     *gomock.Controller
	recorder *MockTargetReaderMockRecorder
}

// MockTargetReaderMockRecorder is the mock recorder for MockTargetReader
type MockTargetReaderMockRecorder struct {
	mock *MockTargetReader
}

// NewMockTargetReader creates a new mock instance
func NewMockTargetReader(ctrl *gomock.Controller) *MockTargetReader {
	mock := &MockTargetReader{ctrl: ctrl}
	mock.recorder = &MockTargetReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockTargetReader) EXPECT() *MockTargetReaderMockRecorder {
	return m.recorder
}

// ReadTarget mocks base method
func (m *MockTargetReader) ReadTarget(arg0 string) cmd.TargetInterface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadTarget", arg0)
	ret0, _ := ret[0].(cmd.TargetInterface)
	return ret0
}

// ReadTarget indicates an expected call of ReadTarget
func (mr *MockTargetReaderMockRecorder) ReadTarget(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadTarget", reflect.TypeOf((*MockTargetReader)(nil).ReadTarget), arg0)
}
