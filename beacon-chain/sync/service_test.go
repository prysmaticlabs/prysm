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
	synced, querierErr := ss.Querier.IsSynced()

	if querierErr != nil {
		if statusErr := ss.Status(); statusErr != querierErr {
			t.Errorf("Expected err %v but got %v", querierErr, statusErr)
		}
	}

	errNotSync := errors.New("node is not synced with the rest of the network")
	if !synced {
		if statusErr := ss.Status(); statusErr != errNotSync {
			t.Errorf("Expected errNotSync %v but got %v", errNotSync, statusErr)
		}
	}

	if synced {
		if statusErr := ss.Status(); statusErr != nil {
			t.Errorf("Expected nil but got %v", statusErr)
		}
	}
}
