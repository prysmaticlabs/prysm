package testing

import (
	"context"
	"reflect"

	"github.com/prysmaticlabs/prysm/v5/api/client/beacon/iface"
	"go.uber.org/mock/gomock"
)

var (
	_ = iface.HealthNode(&MockHealthClient{})
)

// MockHealthClient is a mock of HealthClient interface.
type MockHealthClient struct {
	ctrl     *gomock.Controller
	recorder *MockHealthClientMockRecorder
}

// MockHealthClientMockRecorder is the mock recorder for MockHealthClient.
type MockHealthClientMockRecorder struct {
	mock *MockHealthClient
}

// IsHealthy mocks base method.
func (m *MockHealthClient) IsHealthy(arg0 context.Context) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsHealthy", arg0)
	ret0, ok := ret[0].(bool)
	if !ok {
		return false
	}
	return ret0
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockHealthClient) EXPECT() *MockHealthClientMockRecorder {
	return m.recorder
}

// IsHealthy indicates an expected call of IsHealthy.
func (mr *MockHealthClientMockRecorder) IsHealthy(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsHealthy", reflect.TypeOf((*MockHealthClient)(nil).IsHealthy), arg0)
}

// NewMockHealthClient creates a new mock instance.
func NewMockHealthClient(ctrl *gomock.Controller) *MockHealthClient {
	mock := &MockHealthClient{ctrl: ctrl}
	mock.recorder = &MockHealthClientMockRecorder{mock}
	return mock
}
