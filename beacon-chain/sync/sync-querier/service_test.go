package syncquerier

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mockP2P struct {
}

func (mp *mockP2P) Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription {
	return new(event.Feed).Subscribe(channel)
}

func (mp *mockP2P) Broadcast(msg proto.Message) {}

func (mp *mockP2P) Send(msg proto.Message, peer p2p.Peer) {
}

func TestStartStop(t *testing.T) {
	hook := logTest.NewGlobal()
	cfg := Config{
		P2P:                &mockP2P{},
		ResponseBufferSize: 100,
	}
	sq := NewSyncQuerierService(context.Background(), cfg)

	exitRoutine := make(chan bool)

	defer func() {
		close(exitRoutine)
	}()

	go func() {
		sq.Start()
		<-exitRoutine
		sq.Stop()
	}()

	msg := p2p.Message{
		Data: &pb.ChainHeadResponse{
			Hash: []byte{},
			Slot: 0,
		},
	}

	sq.responseBuf <- msg
	exitRoutine <- true

	testutil.WaitForLog(t, hook, "Exiting goroutine")
	testutil.AssertLogsContain(t, hook, "Stopping service")

	hook.Reset()
}
