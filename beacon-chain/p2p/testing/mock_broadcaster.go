package testing

import (
	"context"
)

// MockBroadcaster implements p2p.Broadcaster for testing.
type MockBroadcaster struct {
	BroadcastCalled bool
}

// Broadcast records a broadcast occurred.
func (m *MockBroadcaster) Broadcast(context.Context, interface{}) error {
	m.BroadcastCalled = true
	return nil
}
