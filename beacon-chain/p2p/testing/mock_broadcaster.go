package testing

import (
	"context"

	"github.com/gogo/protobuf/proto"
)

type MockBroadcaster struct {
	BroadcastCalled bool
}

func (m *MockBroadcaster) Broadcast(context.Context, proto.Message) error {
	m.BroadcastCalled = true
	return nil
}
