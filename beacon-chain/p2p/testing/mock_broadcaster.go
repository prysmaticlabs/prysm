package testing

import (
	"github.com/gogo/protobuf/proto"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
)

var _ = p2p.Broadcaster(&MockBroadcaster{})

type MockBroadcaster struct {
	BroadcastCalled bool
}

func (m *MockBroadcaster) Broadcast(proto.Message) {
	m.BroadcastCalled = true
}
