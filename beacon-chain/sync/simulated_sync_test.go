package sync

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
	"github.com/prysmaticlabs/prysm/shared/event"
)

type mockPowChain struct{}

func (mp *mockPowChain) HasChainStartLogOccurred() (bool, uint64, error) {
	return false, 0, nil
}

func (mp *mockPowChain) ChainStartFeed() *event.Feed {
	return new(event.Feed)
}

func TestSetupTestingEnvironment(t *testing.T) {
	bd, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not set up simulated backend %v", err)
	}

	if err := bd.SetupBackend(1000); err != nil {
		t.Fatalf("Could not set up backend %v", err)
	}

	cfg := &Config{
		ChainService:     bd.ChainService,
		BeaconDB:         bd.BeaconDB,
		OperationService: &mockOperationService{},
		P2P:              &mockP2P{},
		Powchain:         &mockPowChain{},
	}

	ss := NewSyncService(context.Background(), cfg)

	go ss.run()
	ss.Querier.powchain.ChainStartFeed().Send(time.Now())

	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
	ss.Stop()

}
