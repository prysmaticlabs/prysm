package sync

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
)

type mockPowChain struct {
	feed *event.Feed
}

func (mp *mockPowChain) HasChainStartLogOccurred() (bool, uint64, error) {
	return false, 0, nil
}

func (mp *mockPowChain) ChainStartFeed() *event.Feed {
	return mp.feed
}

func TestSetupTestingEnvironment(t *testing.T) {
	bd, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not set up simulated backend %v", err)
	}

	if err := bd.SetupBackend(100); err != nil {
		t.Fatalf("Could not set up backend %v", err)
	}

	beacondb, err := db.SetupDB()
	if err != nil {
		t.Fatalf("Could not setup beacon db %v", err)
	}
	defer db.TeardownDB(beacondb)

	if err := beacondb.SaveState(bd.State()); err != nil {
		t.Fatalf("Could not save state %v", err)
	}

	mockPow := &mockPowChain{
		feed: new(event.Feed),
	}

	mockChain := &mockChainService{
		feed: new(event.Feed),
	}

	mockServer, err := p2p.MockServer(t)
	if err != nil {
		t.Fatalf("Could not create p2p server %v", err)
	}

	cfg := &Config{
		ChainService:     mockChain,
		BeaconDB:         beacondb,
		OperationService: &mockOperationService{},
		P2P:              mockServer,
		Powchain:         mockPow,
	}

	ss := NewSyncService(context.Background(), cfg)

	go ss.run()
	for !ss.Querier.isChainStart {
		mockPow.feed.Send(time.Now())
	}

	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
	ctx := context.Background()
	blocks := bd.InMemoryBlocks()
	newChan := make(chan p2p.Message, 5)
	mockServer.Feed(&pb.BeaconBlockResponse{}).Send(p2p.Message{
		Ctx: ctx,
		Data: &pb.BeaconBlockResponse{
			Block: blocks[0],
		}})
	mockServer.Feed(&pb.BeaconBlockResponse{}).Send(p2p.Message{
		Ctx: ctx,
		Data: &pb.BeaconBlockResponse{
			Block: blocks[1],
		}})
	mockServer.Feed(&pb.BeaconBlockResponse{}).Send(p2p.Message{
		Ctx: ctx,
		Data: &pb.BeaconBlockResponse{
			Block: blocks[2],
		}})
	mockServer.Feed(&pb.BeaconBlockResponse{}).Send(p2p.Message{
		Ctx: ctx,
		Data: &pb.BeaconBlockResponse{
			Block: blocks[3],
		}})

	sub := mockChain.feed.Subscribe(newChan)
	defer sub.Unsubscribe()

	msg := <-newChan
	blk := msg.Data.(*pb.BeaconBlockResponse)
	t.Log(blk)

	ss.Stop()
	bd.Shutdown()
}
