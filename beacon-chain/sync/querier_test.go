package sync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
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

func TestQuerier_StartStop(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	cfg := &QuerierConfig{
		P2P:                &mockP2P{},
		ResponseBufferSize: 100,
		PowChain:           &afterGenesisPowChain{},
		BeaconDB:           db,
		ChainService:       &mockChainService{},
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

func TestListenForStateInitialization_ContextCancelled(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	cfg := &QuerierConfig{
		P2P:                &mockP2P{},
		ResponseBufferSize: 100,
		ChainService:       &mockChainService{},
		BeaconDB:           db,
	}
	sq := NewQuerierService(context.Background(), cfg)
	exitRoutine := make(chan bool)

	defer func() {
		close(exitRoutine)
	}()

	go func() {
		sq.listenForStateInitialization()
		exitRoutine <- true
	}()

	sq.cancel()
	<-exitRoutine

	if sq.ctx.Done() == nil {
		t.Error("Despite context being cancelled, the done channel is nil")
	}
}

func TestListenForStateInitialization(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	cfg := &QuerierConfig{
		P2P:                &mockP2P{},
		ResponseBufferSize: 100,
		ChainService:       &mockChainService{},
		BeaconDB:           db,
	}
	sq := NewQuerierService(context.Background(), cfg)

	sq.chainStartBuf <- time.Now()
	sq.listenForStateInitialization()

	if !sq.chainStarted {
		t.Fatal("ChainStart in the querier service is not true despite the log being fired")
	}
	sq.cancel()
}

func TestQuerier_ChainReqResponse(t *testing.T) {
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
		Slot:                      0,
		Hash:                      []byte{'a', 'b'},
		FinalizedStateRootHash32S: []byte{'c', 'd'},
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
