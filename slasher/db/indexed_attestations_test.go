package db

import (
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

type testStruct struct {
	iA *ethpb.IndexedAttestation
}

var tests []testStruct

func init() {
	tests = []testStruct{
		{
			iA: &ethpb.IndexedAttestation{Signature: []byte("let me in"), CustodyBit_0Indices: []uint64{0}, Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 0},
				Target: &ethpb.Checkpoint{Epoch: 1},
			}},
		},
		{
			iA: &ethpb.IndexedAttestation{Signature: []byte("let me in 2nd"), CustodyBit_0Indices: []uint64{1, 2}, Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 0},
				Target: &ethpb.Checkpoint{Epoch: 2},
			}},
		},
		{
			iA: &ethpb.IndexedAttestation{Signature: []byte("let me in 3rd"), CustodyBit_0Indices: []uint64{0}, Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 1},
				Target: &ethpb.Checkpoint{Epoch: 2},
			}},
		},
	}
}

func TestNilDBHistoryIdxAtt(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)

	epoch := uint64(1)
	validatorID := uint64(1)

	hasIdxAtt := db.HasIndexedAttestation(epoch, epoch, validatorID)
	if hasIdxAtt {
		t.Fatal("HasIndexedAttestation should return false")
	}

	idxAtt, err := db.IndexedAttestation(epoch, epoch, validatorID)
	if err != nil {
		t.Fatalf("failed to get indexed attestation: %v", err)
	}
	if idxAtt != nil {
		t.Fatalf("get should return nil for a non existent key")
	}
}

func TestSaveIdxAtt(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)

	for _, tt := range tests {
		err := db.SaveIndexedAttestation(tt.iA)
		if err != nil {
			t.Fatalf("save indexed attestation failed: %v", err)
		}

		iAarray, err := db.IndexedAttestation(tt.iA.Data.Source.Epoch, tt.iA.Data.Target.Epoch, tt.iA.CustodyBit_0Indices[0])
		if err != nil {
			t.Fatalf("failed to get indexed attestation: %v", err)
		}

		if iAarray == nil || !reflect.DeepEqual(iAarray[0], tt.iA) {
			t.Fatalf("get should return indexed attestation: %v", iAarray)
		}
	}

}

func TestDeleteHistoryIdxAtt(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)

	for _, tt := range tests {

		err := db.SaveIndexedAttestation(tt.iA)
		if err != nil {
			t.Fatalf("save indexed attestation failed: %v", err)
		}
	}

	for _, tt := range tests {
		iAarray, err := db.IndexedAttestation(tt.iA.Data.Source.Epoch, tt.iA.Data.Target.Epoch, tt.iA.CustodyBit_0Indices[0])
		if err != nil {
			t.Fatalf("failed to get index attestation: %v", err)
		}

		if iAarray == nil || !reflect.DeepEqual(iAarray[0], tt.iA) {
			t.Fatalf("get should return indexed attestation: %v", iAarray)
		}
		err = db.DeleteIndexedAttestation(tt.iA)
		if err != nil {
			t.Fatalf("delete index attestation failed: %v", err)
		}
		iAarray, err = db.IndexedAttestation(tt.iA.Data.Source.Epoch, tt.iA.Data.Target.Epoch, tt.iA.CustodyBit_0Indices[0])
		hasA := db.HasIndexedAttestation(tt.iA.Data.Source.Epoch, tt.iA.Data.Target.Epoch, tt.iA.CustodyBit_0Indices[0])
		if err != nil {
			t.Fatal(err)
		}
		if len(iAarray) != 0 {
			t.Errorf("Expected index attestation to have been deleted, received: %v", iAarray)
		}
		if hasA {
			t.Errorf("Expected indexed attestation indexes list to be deleted, received: %v", hasA)
		}

	}

}

func TestHasIdxAtt(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)

	for _, tt := range tests {

		found := db.HasIndexedAttestation(tt.iA.Data.Source.Epoch, tt.iA.Data.Target.Epoch, tt.iA.CustodyBit_0Indices[0])
		if found {
			t.Fatal("has indexed attestation should return false for indexed attestations that are not in db")
		}
		err := db.SaveIndexedAttestation(tt.iA)
		if err != nil {
			t.Fatalf("save indexed attestation failed: %v", err)
		}
	}
	for _, tt := range tests {

		found := db.HasIndexedAttestation(tt.iA.Data.Source.Epoch, tt.iA.Data.Target.Epoch, tt.iA.CustodyBit_0Indices[0])

		if !found {
			t.Fatal("has indexed attestation should return true")
		}
	}
}

func TestPruneHistoryIdxAtt(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)

	for _, tt := range tests {
		err := db.SaveIndexedAttestation(tt.iA)
		if err != nil {
			t.Fatalf("save indexed attestation failed: %v", err)
		}

		iAarray, err := db.IndexedAttestation(tt.iA.Data.Source.Epoch, tt.iA.Data.Target.Epoch, tt.iA.CustodyBit_0Indices[0])
		if err != nil {
			t.Fatalf("failed to get indexed attestation: %v", err)
		}

		if iAarray == nil || !reflect.DeepEqual(iAarray[0], tt.iA) {
			t.Fatalf("get should return bh: %v", iAarray)
		}
	}
	currentEpoch := uint64(3)
	historyToKeep := uint64(2)
	err := db.pruneAttHistory(currentEpoch, historyToKeep)
	if err != nil {
		t.Fatalf("failed to prune: %v", err)
	}

	for _, tt := range tests {
		iAarray, err := db.IndexedAttestation(tt.iA.Data.Source.Epoch, tt.iA.Data.Target.Epoch, tt.iA.CustodyBit_0Indices[0])
		if err != nil {
			t.Fatalf("failed to get indexed attestation: %v", err)
		}
		hasIa := db.HasIndexedAttestation(tt.iA.Data.Source.Epoch, tt.iA.Data.Target.Epoch, tt.iA.CustodyBit_0Indices[0])

		if tt.iA.Data.Source.Epoch > currentEpoch-historyToKeep {
			if iAarray == nil || !reflect.DeepEqual(iAarray[0], tt.iA) {
				t.Fatalf("get should return indexed attestation: %v", iAarray)
			}
			if !hasIa {
				t.Fatalf("get should have indexed attestation for epoch: %v", iAarray)
			}
		} else {
			if iAarray != nil || hasIa {
				t.Fatalf("indexed attestation should have been pruned: %v has attestation: %v", iAarray, hasIa)
			}
		}

	}
}
