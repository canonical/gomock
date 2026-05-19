// Pending MockGen update. Hand-maintained until mockgen emits the
// new typed NewCallN_M / DispatchN_M / Track API.
// Source: github.com/canonical/gomock/gomock_test (interfaces: Foo)

// Package gomock_test is a generated GoMock package.
package gomock_test

import "github.com/canonical/gomock/gomock"

// MockFoo is a mock of Foo interface.
type MockFoo struct {
	ctrl     *gomock.Controller
	recorder *MockFooMockRecorder
	isgomock struct{}
}

// MockFooMockRecorder is the mock recorder for MockFoo.
type MockFooMockRecorder struct {
	mock          *MockFoo
	barExpects    []*gomock.Call1_1[string, string]
	stringExpects []*gomock.Call0_1[string]
}

// NewMockFoo creates a new mock instance.
func NewMockFoo(ctrl *gomock.Controller) *MockFoo {
	mock := &MockFoo{ctrl: ctrl}
	mock.recorder = &MockFooMockRecorder{mock: mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFoo) EXPECT() *MockFooMockRecorder {
	return m.recorder
}

// Bar mocks base method.
func (m *MockFoo) Bar(arg0 string) string {
	m.ctrl.T.Helper()
	return gomock.Dispatch1_1(&m.recorder.barExpects, m.ctrl, m, "Bar", arg0)
}

// Bar indicates an expected call of Bar.
func (mr *MockFooMockRecorder) Bar(arg0 any) *MockFooBarCall {
	mr.mock.ctrl.T.Helper()
	call := gomock.NewCall1_1[string, string](mr.mock.ctrl.T, mr.mock, "Bar", gomock.EnsureMatcher(arg0))
	mr.barExpects = append(mr.barExpects, call)
	mr.mock.ctrl.Track(call.Call)
	return call
}

// MockFooBarCall is the typed call wrapper for Bar.
type MockFooBarCall = gomock.Call1_1[string, string]

// String mocks base method.
func (m *MockFoo) String() string {
	m.ctrl.T.Helper()
	return gomock.Dispatch0_1(&m.recorder.stringExpects, m.ctrl, m, "String")
}

// String indicates an expected call of String.
func (mr *MockFooMockRecorder) String() *MockFooStringCall {
	mr.mock.ctrl.T.Helper()
	call := gomock.NewCall0_1[string](mr.mock.ctrl.T, mr.mock, "String")
	mr.stringExpects = append(mr.stringExpects, call)
	mr.mock.ctrl.Track(call.Call)
	return call
}

// MockFooStringCall is the typed call wrapper for String.
type MockFooStringCall = gomock.Call0_1[string]
