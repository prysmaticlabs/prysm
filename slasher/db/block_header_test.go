package db

import (
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func TestNilDBHistoryBlkHdr(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	epoch := uint64(1)
	validatorID := uint64(1)

	hasBlockHeader := db.HasBlockHeader(epoch, validatorID)
	if hasBlockHeader {
		t.Fatal("HasBlockHeader should return false")
	}

	bPrime, err := db.BlockHeader(epoch, validatorID)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	if bPrime != nil {
		t.Fatalf("get should return nil for a non existent key")
	}
}

func TestSaveHistoryBlkHdr(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	tests := []struct {
		epoch uint64
		vID   uint64
		bh    *ethpb.BeaconBlockHeader
	}{
		{
			epoch: uint64(0),
			vID:   uint64(0),
			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in")},
		},
		{
			epoch: uint64(0),
			vID:   uint64(1),
			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in 2nd")},
		},
		{
			epoch: uint64(1),
			vID:   uint64(0),
			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in 3rd")},
		},
	}

	for _, tt := range tests {
		err := db.SaveBlockHeader(tt.epoch, tt.vID, tt.bh)
		if err != nil {
			t.Fatalf("save block failed: %v", err)
		}

		bha, err := db.BlockHeader(tt.epoch, tt.vID)
		if err != nil {
			t.Fatalf("failed to get block: %v", err)
		}

		if bha == nil || !reflect.DeepEqual(bha[0], tt.bh) {
			t.Fatalf("get should return bh: %v", bha)
		}
	}

}

func TestDeleteHistoryBlkHdr(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	tests := []struct {
		epoch uint64
		vID   uint64
		bh    *ethpb.BeaconBlockHeader
	}{
		{
			epoch: uint64(0),
			vID:   uint64(0),
			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in")},
		},
		{
			epoch: uint64(0),
			vID:   uint64(1),
			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in 2nd")},
		},
		{
			epoch: uint64(1),
			vID:   uint64(0),
			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in 3rd")},
		},
	}
	for _, tt := range tests {

		err := db.SaveBlockHeader(tt.epoch, tt.vID, tt.bh)
		if err != nil {
			t.Fatalf("save block failed: %v", err)
		}
	}

	for _, tt := range tests {
		bha, err := db.BlockHeader(tt.epoch, tt.vID)
		if err != nil {
			t.Fatalf("failed to get block: %v", err)
		}

		if bha == nil || !reflect.DeepEqual(bha[0], tt.bh) {
			t.Fatalf("get should return bh: %v", bha)
		}
		err = db.DeleteBlockHeader(tt.epoch, tt.vID, tt.bh)
		if err != nil {
			t.Fatalf("save block failed: %v", err)
		}
		bh, err := db.BlockHeader(tt.epoch, tt.vID)

		if err != nil {
			t.Fatal(err)
		}
		if bh != nil {
			t.Errorf("Expected block to have been deleted, received: %v", bh)
		}

	}

}

func TestHasHistoryBlkHdr(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	tests := []struct {
		epoch uint64
		vID   uint64
		bh    *ethpb.BeaconBlockHeader
	}{
		{
			epoch: uint64(0),
			vID:   uint64(0),
			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in")},
		},
		{
			epoch: uint64(0),
			vID:   uint64(1),
			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in 2nd")},
		},
		{
			epoch: uint64(1),
			vID:   uint64(0),
			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in 3rd")},
		},
	}
	for _, tt := range tests {

		found := db.HasBlockHeader(tt.epoch, tt.vID)
		if found {
			t.Fatal("has block header should return false for block headers that are not in db")
		}
		err := db.SaveBlockHeader(tt.epoch, tt.vID, tt.bh)
		if err != nil {
			t.Fatalf("save block failed: %v", err)
		}
	}
	for _, tt := range tests {
		err := db.SaveBlockHeader(tt.epoch, tt.vID, tt.bh)
		if err != nil {
			t.Fatalf("save block failed: %v", err)
		}

		found := db.HasBlockHeader(tt.epoch, tt.vID)

		if !found {
			t.Fatal("has block header should return true")
		}
	}
}

func TestPruneHistoryBlkHdr(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	tests := []struct {
		epoch uint64
		vID   uint64
		bh    *ethpb.BeaconBlockHeader
	}{
		{
			epoch: uint64(0),
			vID:   uint64(0),
			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in")},
		},
		{
			epoch: uint64(0),
			vID:   uint64(1),
			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in 2nd")},
		},
		{
			epoch: uint64(1),
			vID:   uint64(0),
			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in 3rd")},
		},
		{
			epoch: uint64(2),
			vID:   uint64(0),
			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in 4th")},
		},
		{
			epoch: uint64(3),
			vID:   uint64(0),
			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in 5th")},
		},
	}

	for _, tt := range tests {
		err := db.SaveBlockHeader(tt.epoch, tt.vID, tt.bh)
		if err != nil {
			t.Fatalf("save block header failed: %v", err)
		}

		bha, err := db.BlockHeader(tt.epoch, tt.vID)
		if err != nil {
			t.Fatalf("failed to get block header: %v", err)
		}

		if bha == nil || !reflect.DeepEqual(bha[0], tt.bh) {
			t.Fatalf("get should return bh: %v", bha)
		}
	}
	currentEpoch := uint64(3)
	historyToKeep := uint64(2)
	err := db.pruneHistory(currentEpoch, historyToKeep)
	if err != nil {
		t.Fatalf("failed to prune: %v", err)
	}

	for _, tt := range tests {
		bha, err := db.BlockHeader(tt.epoch, tt.vID)
		if err != nil {
			t.Fatalf("failed to get block header: %v", err)
		}
		if tt.epoch > currentEpoch-historyToKeep {
			if bha == nil || !reflect.DeepEqual(bha[0], tt.bh) {
				t.Fatalf("get should return bh: %v", bha)
			}
		} else {
			if bha != nil {
				t.Fatalf("block header should have been pruned: %v", bha)
			}
		}

	}
}
