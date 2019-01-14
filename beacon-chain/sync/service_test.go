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
	synced, err := ss.Querier.IsSynced()

	if err != nil {
		if err1 := ss.Status(); err1 != err {
			t.Errorf("Expected err %v but got %v", err, err1)
		}
	}

	errNotSync := errors.New("node is not synced with the rest of the network")
	if !synced {
		if err1 := ss.Status(); err1 != errNotSync {
			t.Errorf("Expected errNotSync %v but got %v", errNotSync, err1)
		}
	}

	if synced {
		if err1 := ss.Status(); err1 != nil {
			t.Errorf("Expected nil but got %v", err1)
		}
	}
}
