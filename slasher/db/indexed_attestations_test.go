package db

import (
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func TestNilDBHistoryIdxAtt(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	epoch := uint64(1)
	validatorID := uint64(1)

	hasIdxAtt := db.HasIndexedAttestation(epoch, validatorID)
	if hasIdxAtt {
		t.Fatal("HasBlockHeader should return false")
	}

	idxAtt, err := db.IndexedAttestation(epoch, validatorID)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	if idxAtt != nil {
		t.Fatalf("get should return nil for a non existent key")
	}
}

func TestSaveIdxAtt(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	tests := []struct {
		epoch uint64
		iA    *ethpb.IndexedAttestation
	}{
		{
			epoch: uint64(0),
			iA: &ethpb.IndexedAttestation{Signature: []byte("let me in"), CustodyBit_0Indices: []uint64{0}, Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 0},
				Target: &ethpb.Checkpoint{Epoch: 0},
				Crosslink: &ethpb.Crosslink{
					Shard: 4,
				},
			}},
		},
		{
			epoch: uint64(0),
			iA: &ethpb.IndexedAttestation{Signature: []byte("let me in 2nd"), CustodyBit_0Indices: []uint64{1, 2}, Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 0},
				Target: &ethpb.Checkpoint{Epoch: 0},
				Crosslink: &ethpb.Crosslink{
					Shard: 4,
				},
			}},
		},
		{
			epoch: uint64(1),
			iA: &ethpb.IndexedAttestation{Signature: []byte("let me in 3rd"), CustodyBit_0Indices: []uint64{0}, Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 0},
				Target: &ethpb.Checkpoint{Epoch: 0},
				Crosslink: &ethpb.Crosslink{
					Shard: 4,
				},
			}},
		},
	}

	for _, tt := range tests {
		err := db.SaveIndexedAttestation(tt.epoch, tt.iA)
		if err != nil {
			t.Fatalf("save block failed: %v", err)
		}

		iAarray, err := db.IndexedAttestation(tt.epoch, tt.iA.CustodyBit_0Indices[0])
		if err != nil {
			t.Fatalf("failed to get block: %v", err)
		}

		if iAarray == nil || !reflect.DeepEqual(iAarray[0], tt.iA) {
			t.Fatalf("get should return indexed attestation: %v", iAarray)
		}
	}

}

func TestDeleteHistoryIdxAtt(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	tests := []struct {
		epoch uint64
		iA    *ethpb.IndexedAttestation
	}{
		{
			epoch: uint64(0),
			iA: &ethpb.IndexedAttestation{Signature: []byte("let me in"), CustodyBit_0Indices: []uint64{0}, Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 0},
				Target: &ethpb.Checkpoint{Epoch: 0},
				Crosslink: &ethpb.Crosslink{
					Shard: 4,
				},
			}},
		},
		{
			epoch: uint64(0),
			iA: &ethpb.IndexedAttestation{Signature: []byte("let me in 2nd"), CustodyBit_0Indices: []uint64{1, 2}, Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 0},
				Target: &ethpb.Checkpoint{Epoch: 0},
				Crosslink: &ethpb.Crosslink{
					Shard: 4,
				},
			}},
		},
		{
			epoch: uint64(1),
			iA: &ethpb.IndexedAttestation{Signature: []byte("let me in 3rd"), CustodyBit_0Indices: []uint64{0}, Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 0},
				Target: &ethpb.Checkpoint{Epoch: 0},
				Crosslink: &ethpb.Crosslink{
					Shard: 4,
				},
			}},
		},
	}
	for _, tt := range tests {

		err := db.SaveIndexedAttestation(tt.epoch, tt.iA)
		if err != nil {
			t.Fatalf("save block failed: %v", err)
		}
	}

	for _, tt := range tests {
		iAarray, err := db.IndexedAttestation(tt.epoch, tt.iA.CustodyBit_0Indices[0])
		if err != nil {
			t.Fatalf("failed to get block: %v", err)
		}

		if iAarray == nil || !reflect.DeepEqual(iAarray[0], tt.iA) {
			t.Fatalf("get should return bh: %v", iAarray)
		}
		err = db.DeleteIndexedAttestation(tt.epoch, tt.iA)
		if err != nil {
			t.Fatalf("save block failed: %v", err)
		}
		iAarray, err = db.IndexedAttestation(tt.epoch, tt.iA.CustodyBit_0Indices[0])

		if err != nil {
			t.Fatal(err)
		}
		if len(iAarray) != 0 {
			t.Errorf("Expected block to have been deleted, received: %v", iAarray)
		}

	}

}

/*
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
*/
