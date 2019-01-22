package sync

import (
	"context"
	"testing"
	"errors"

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


func TestStatusQuerierError(t *testing.T) {
    ss := setupTestSyncService(t)
	_, querierErr := ss.Querier.IsSynced()

	if querierErr != nil {
		if statusErr := ss.Status(); statusErr != querierErr {
			t.Errorf("Expected err %v but got %v", querierErr, statusErr)
		}
	}

}

func TestStatusSyncedNetwork(t *testing.T) {
	ss := setupTestSyncService(t)
	synced, _ := ss.Querier.IsSynced()
	
	if synced {
		if statusErr := ss.Status(); statusErr != nil {
			t.Errorf("Expected nil but got %v", statusErr)
		}
	}
}

func TestStatusNotSyncedNetwork(t *testing.T) {
	ss := setupTestSyncService(t)
	synced, querierErr := ss.Querier.IsSynced()

	if !synced {
		if querierErr != nil {
	   		if statusErr := ss.Status(); statusErr != querierErr {
				t.Errorf("Expected %v, but got %v", querierErr, statusErr)	
			}
		 } else {
			errNotSync := errors.New("node is not in sync with the rest of the network")
			if statusErr := ss.Status(); statusErr != errNotSync {
				t.Errorf("Expected %v, but got %v", errNotSync, statusErr)	
			} 
		}
	}
}

