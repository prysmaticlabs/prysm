package p2p

import (
	"context"

	"github.com/gogo/protobuf/proto"
)

// MockBroadcaster implements p2p.Broadcaster for p2p.
type MockBroadcaster struct {
	BroadcastCalled bool
}

// Broadcast records a broadcast occurred.
func (m *MockBroadcaster) Broadcast(context.Context, proto.Message) error {
	m.BroadcastCalled = true
	return nil
}
