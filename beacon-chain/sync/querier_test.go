package sync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/event"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type genesisPowChain struct {
	feed *event.Feed
}

func (mp *genesisPowChain) HasChainStartLogOccurred() (bool, uint64, error) {
	return false, 0, nil
}

func (mp *genesisPowChain) ChainStartFeed() *event.Feed {
	return mp.feed
}

type afterGenesisPowChain struct {
	feed *event.Feed
}

func (mp *afterGenesisPowChain) HasChainStartLogOccurred() (bool, uint64, error) {
	return true, 0, nil
}

func (mp *afterGenesisPowChain) ChainStartFeed() *event.Feed {
	return mp.feed
}

func TestStartStop(t *testing.T) {
	hook := logTest.NewGlobal()
	cfg := &QuerierConfig{
		P2P:                &mockP2P{},
		ResponseBufferSize: 100,
		PowChain:           &afterGenesisPowChain{},
	}
	sq := NewQuerierService(context.Background(), cfg)

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

func TestListenForChainStart_ContextCancelled(t *testing.T) {
	cfg := &QuerierConfig{
		P2P:                &mockP2P{},
		ResponseBufferSize: 100,
		PowChain: &afterGenesisPowChain{
			feed: new(event.Feed),
		},
	}
	sq := NewQuerierService(context.Background(), cfg)
	exitRoutine := make(chan bool)

	defer func() {
		close(exitRoutine)
	}()

	go func() {
		sq.listenForChainStart()
		exitRoutine <- true
	}()

	sq.cancel()
	<-exitRoutine

	if sq.ctx.Done() == nil {
		t.Error("Despite context being cancelled, the done channel is nil")
	}
}

func TestListenForChainStart(t *testing.T) {
	cfg := &QuerierConfig{
		P2P:                &mockP2P{},
		ResponseBufferSize: 100,
		PowChain: &afterGenesisPowChain{
			feed: new(event.Feed),
		},
	}
	sq := NewQuerierService(context.Background(), cfg)

	sq.chainStartBuf <- time.Now()
	sq.listenForChainStart()

	if !sq.chainStarted {
		t.Fatal("ChainStart in the querier service is not true despite the log being fired")
	}
	sq.cancel()
}

func TestChainReqResponse(t *testing.T) {
	hook := logTest.NewGlobal()
	cfg := &QuerierConfig{
		P2P:                &mockP2P{},
		ResponseBufferSize: 100,
		PowChain:           &afterGenesisPowChain{},
	}
	sq := NewQuerierService(context.Background(), cfg)

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

	expMsg := fmt.Sprintf("Latest chain head is at slot: %d and hash %#x", response.Slot, response.Hash)

	testutil.WaitForLog(t, hook, expMsg)

	sq.cancel()
	<-exitRoutine

	hook.Reset()
}
