package sync

import (
	"context"
	"errors"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
)

func TestStatus(t *testing.T) {

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

    ss := NewSyncService(context.Background(), cfg)
	synced, errSync := ss.Querier.IsSynced()

	if errSync != nil {
		if err := ss.Status(); err != errSync {
			t.Errorf("Expected errSync %v but got %v", errSync, err)
		}
	}

	errNotSync := errors.New("not in sync")
	if !synced {
		if err := ss.Status(); err != errNotSync {
			t.Errorf("Expected errNotSync %v but got %v", errSync, err)
		}
	} 
	
	if synced {
		if err := ss.Status(); err != nil {
			t.Errorf("Expected nil but got %v", err)
		}
	}
}