package sync

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
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

func setupDB() {

}

func TestSetupTestingEnvironment(t *testing.T) {
	bd, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not set up simulated backend %v", err)
	}

	if err := bd.SetupBackend(100); err != nil {
		t.Fatalf("Could not set up backend %v", err)
	}

	t.Log(bd.State().Slot)

	mockPow := &mockPowChain{
		feed: new(event.Feed),
	}

	mockServer, err := p2p.MockServer(t)
	if err != nil {
		t.Fatalf("Could not create p2p server %v", err)
	}

	cfg := &Config{
		ChainService:     bd.ChainService,
		BeaconDB:         bd.BeaconDB,
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
	blocks := bd.InMemoryBlocks()

	ss.Stop()
	bd.Shutdown()
}
