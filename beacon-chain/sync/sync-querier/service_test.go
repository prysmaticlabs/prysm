package syncquerier

import (
	"context"
	"fmt"
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
	cfg := &Config{
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
		exitRoutine <- true
	}()

	sq.Stop()
	<-exitRoutine

	testutil.AssertLogsContain(t, hook, "Stopping service")

	hook.Reset()
}

func TestChainReqResponse(t *testing.T) {
	hook := logTest.NewGlobal()
	cfg := &Config{
		P2P:                &mockP2P{},
		ResponseBufferSize: 100,
	}
	sq := NewSyncQuerierService(context.Background(), cfg)

	exitRoutine := make(chan bool)

	defer func() {
		close(exitRoutine)
	}()

	go func() {
		sq.run()
		exitRoutine <- true
	}()

	response := &pb.ChainHeadResponse{
		Slot: 0,
		Hash: []byte{'a', 'b'},
	}

	msg := p2p.Message{
		Data: response,
	}

	sq.responseBuf <- msg

	expMsg := fmt.Sprintf("Latest Chain head is at slot: %d and hash %#x", response.Slot, response.Hash)

	testutil.WaitForLog(t, hook, expMsg)

	sq.cancel()
	<-exitRoutine

	hook.Reset()
}
