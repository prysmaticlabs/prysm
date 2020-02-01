package db

import (
	"flag"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/urfave/cli"
)

type testStruct struct {
	idxAtt *ethpb.IndexedAttestation
}

var tests []testStruct

func init() {
	tests = []testStruct{
		{
			idxAtt: &ethpb.IndexedAttestation{Signature: []byte("let me in"), AttestingIndices: []uint64{0}, Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 0},
				Target: &ethpb.Checkpoint{Epoch: 1},
			}},
		},
		{
			idxAtt: &ethpb.IndexedAttestation{Signature: []byte("let me in 2nd"), AttestingIndices: []uint64{1, 2}, Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 0},
				Target: &ethpb.Checkpoint{Epoch: 2},
			}},
		},
		{
			idxAtt: &ethpb.IndexedAttestation{Signature: []byte("let me in 3rd"), AttestingIndices: []uint64{0}, Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 1},
				Target: &ethpb.Checkpoint{Epoch: 2},
			}},
		},
	}
}

func TestNilDBHistoryIdxAtt(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(app, set, nil)
	db := SetupSlasherDB(t, ctx)
	defer TeardownSlasherDB(t, db)

	epoch := uint64(1)
	validatorID := uint64(1)

	hasIdxAtt, err := db.HasIndexedAttestation(epoch, validatorID)
	if err != nil {
		t.Fatal(err)
	}
	if hasIdxAtt {
		t.Fatal("HasIndexedAttestation should return false")
	}

	idxAtt, err := db.IndexedAttestation(epoch, validatorID)
	if err != nil {
		t.Fatalf("failed to get indexed attestation: %v", err)
	}
	if idxAtt != nil {
		t.Fatalf("get should return nil for a non existent key")
	}
}

func TestSaveIdxAtt(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(app, set, nil)
	db := SetupSlasherDB(t, ctx)
	defer TeardownSlasherDB(t, db)

	for _, tt := range tests {
		err := db.SaveIndexedAttestation(tt.idxAtt)
		if err != nil {
			t.Fatalf("save indexed attestation failed: %v", err)
		}

		idxAttarray, err := db.IndexedAttestation(tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatalf("failed to get indexed attestation: %v", err)
		}

		if idxAttarray == nil || !reflect.DeepEqual(idxAttarray[0], tt.idxAtt) {
			t.Fatalf("get should return indexed attestation: %v", idxAttarray)
		}
	}

}

func TestDeleteHistoryIdxAtt(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(app, set, nil)
	db := SetupSlasherDB(t, ctx)
	defer TeardownSlasherDB(t, db)

	for _, tt := range tests {

		err := db.SaveIndexedAttestation(tt.idxAtt)
		if err != nil {
			t.Fatalf("save indexed attestation failed: %v", err)
		}
	}

	for _, tt := range tests {
		idxAttarray, err := db.IndexedAttestation(tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatalf("failed to get index attestation: %v", err)
		}

		if idxAttarray == nil || !reflect.DeepEqual(idxAttarray[0], tt.idxAtt) {
			t.Fatalf("get should return indexed attestation: %v", idxAttarray)
		}
		err = db.DeleteIndexedAttestation(tt.idxAtt)
		if err != nil {
			t.Fatalf("delete index attestation failed: %v", err)
		}

		idxAttarray, err = db.IndexedAttestation(tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatal(err)
		}
		hasA, err := db.HasIndexedAttestation(tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatal(err)
		}
		if len(idxAttarray) != 0 {
			t.Errorf("Expected index attestation to have been deleted, received: %v", idxAttarray)
		}
		if hasA {
			t.Errorf("Expected indexed attestation indexes list to be deleted, received: %v", hasA)
		}

	}

}

func TestHasIdxAtt(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(app, set, nil)
	db := SetupSlasherDB(t, ctx)
	defer TeardownSlasherDB(t, db)

	for _, tt := range tests {
		exists, err := db.HasIndexedAttestation(tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Fatal("has indexed attestation should return false for indexed attestations that are not in db")
		}

		if err := db.SaveIndexedAttestation(tt.idxAtt); err != nil {
			t.Fatalf("save indexed attestation failed: %v", err)
		}
	}
	for _, tt := range tests {
		exists, err := db.HasIndexedAttestation(tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatal("has indexed attestation should return true")
		}
	}
}

func TestPruneHistoryIdxAtt(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(app, set, nil)
	db := SetupSlasherDB(t, ctx)
	defer TeardownSlasherDB(t, db)

	for _, tt := range tests {
		err := db.SaveIndexedAttestation(tt.idxAtt)
		if err != nil {
			t.Fatalf("save indexed attestation failed: %v", err)
		}

		idxAttarray, err := db.IndexedAttestation(tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatalf("failed to get indexed attestation: %v", err)
		}

		if idxAttarray == nil || !reflect.DeepEqual(idxAttarray[0], tt.idxAtt) {
			t.Fatalf("get should return bh: %v", idxAttarray)
		}
	}
	currentEpoch := uint64(3)
	historyToKeep := uint64(1)
	err := db.pruneAttHistory(currentEpoch, historyToKeep)
	if err != nil {
		t.Fatalf("failed to prune: %v", err)
	}

	for _, tt := range tests {
		idxAttarray, err := db.IndexedAttestation(tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatalf("failed to get indexed attestation: %v", err)
		}
		exists, err := db.HasIndexedAttestation(tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatal(err)
		}

		if tt.idxAtt.Data.Source.Epoch > currentEpoch-historyToKeep {
			if idxAttarray == nil || !reflect.DeepEqual(idxAttarray[0], tt.idxAtt) {
				t.Fatalf("get should return indexed attestation: %v", idxAttarray)
			}
			if !exists {
				t.Fatalf("get should have indexed attestation for epoch: %v", idxAttarray)
			}
		} else {
			if idxAttarray != nil || exists {
				t.Fatalf("indexed attestation should have been pruned: %v has attestation: %t", idxAttarray, exists)
			}
		}
	}
}
