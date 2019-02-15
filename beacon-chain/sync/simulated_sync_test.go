package sync

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
)

func TestSetupTestingEnvironment(t *testing.T) {
	bd, err := backend.NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not set up simulated backend %v", err)
	}

	cfg := &Config{
		ChainService:     bd.ChainService,
		BeaconDB:         bd.BeaconDB,
		OperationService: &mockOperationService{},
		P2P:              &mockP2P{},
	}

	ss := NewSyncService(context.Background(), cfg)
	ss.run()
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
	bd.GenerateBlockAndAdvanceChain(&backend.SimulatedObjects{})
	ss.Stop()

}
