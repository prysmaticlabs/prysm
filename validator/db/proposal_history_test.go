package db

import (
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

func TestNilDBHistoryBlkHdr(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)

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
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)
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

}

func TestHasHistoryBlkHdr(t *testing.T) {
}
