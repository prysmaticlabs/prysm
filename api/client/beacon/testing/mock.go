package testing

import (
	"context"
	"reflect"

	"github.com/prysmaticlabs/prysm/v5/api/client/beacon"
	"go.uber.org/mock/gomock"
)

var (
	_ = beacon.HealthNode(&MockHealthClient{})
)

// MockHealthClient is a mock of HealthClient interface.
type MockHealthClient struct {
	ctrl          *gomock.Controller
	recorder      *MockHealthClientMockRecorder
	healthTracker *beacon.NodeHealthTracker
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
	mock.healthTracker = beacon.NewNodeHealthTracker(mock)
	return mock
}

// NewMockNodeHealthTracker returns a mock tracker with mock health client
func NewMockNodeHealthTracker(ctrl *gomock.Controller) (*beacon.NodeHealthTracker, *MockHealthClient) {
	client := NewMockHealthClient(ctrl)
	return beacon.NewNodeHealthTracker(client), client
}
