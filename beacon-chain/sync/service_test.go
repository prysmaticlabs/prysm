package sync

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
)

func setupTestSyncService(t *testing.T) *Service {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	if err := db.InitializeState(); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	cfg := &Config{
		ChainService:  &mockChainService{},
		P2P:           &mockP2P{},
		BeaconDB:      db,
		AttestService: &mockAttestService{},
	}

	return NewSyncService(context.Background(), cfg)
}

func TestStatusWithNetwork(t *testing.T) {
	ss := setupTestSyncService(t)
	synced, querierErr := ss.Querier.IsSynced()

	if synced {
		if querierErr != nil {
			t.Errorf("querierErr expected to be nil but got %v", querierErr)
		}
		if statusErr := ss.Status(); statusErr != nil {
			t.Errorf("Expected nil but got %v", statusErr)
		}
	}

	if !synced {
		if querierErr == nil {
			t.Errorf("querierErr expected to be not nil but got %v", querierErr)
		}
		if statusErr := ss.Status(); statusErr != querierErr {
			t.Errorf("Expected %v, but got %v", querierErr, statusErr)
		}
	}
}
