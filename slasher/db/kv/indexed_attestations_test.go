package kv

import (
	"context"
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
	db := setupDB(t, cli.NewContext(app, set, nil))
	defer teardownDB(t, db)
	ctx := context.Background()

	epoch := uint64(1)
	validatorID := uint64(1)

	hasIdxAtt, err := db.HasIndexedAttestation(ctx, epoch, validatorID)
	if err != nil {
		t.Fatal(err)
	}
	if hasIdxAtt {
		t.Fatal("HasIndexedAttestation should return false")
	}

	idxAtts, err := db.IdxAttsForTargetFromID(ctx, epoch, validatorID)
	if err != nil {
		t.Fatalf("failed to get indexed attestation: %v", err)
	}
	if idxAtts != nil {
		t.Fatalf("get should return nil for a non existent key")
	}
}

func TestSaveIdxAtt(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(app, set, nil))
	defer teardownDB(t, db)
	ctx := context.Background()

	for _, tt := range tests {
		err := db.SaveIndexedAttestation(ctx, tt.idxAtt)
		if err != nil {
			t.Fatalf("save indexed attestation failed: %v", err)
		}

		idxAtts, err := db.IdxAttsForTargetFromID(ctx, tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatalf("failed to get indexed attestation: %v", err)
		}

		if idxAtts == nil || !reflect.DeepEqual(idxAtts[0], tt.idxAtt) {
			t.Fatalf("get should return indexed attestation: %v", idxAtts)
		}
	}

}

func TestDeleteHistoryIdxAtt(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(app, set, nil))
	defer teardownDB(t, db)
	ctx := context.Background()

	for _, tt := range tests {

		err := db.SaveIndexedAttestation(ctx, tt.idxAtt)
		if err != nil {
			t.Fatalf("save indexed attestation failed: %v", err)
		}
	}

	for _, tt := range tests {
		idxAtts, err := db.IdxAttsForTargetFromID(ctx, tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatalf("failed to get index attestation: %v", err)
		}

		if idxAtts == nil || !reflect.DeepEqual(idxAtts[0], tt.idxAtt) {
			t.Fatalf("get should return indexed attestation: %v", idxAtts)
		}
		err = db.DeleteIndexedAttestation(ctx, tt.idxAtt)
		if err != nil {
			t.Fatalf("delete index attestation failed: %v", err)
		}

		idxAtts, err = db.IdxAttsForTargetFromID(ctx, tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatal(err)
		}
		hasA, err := db.HasIndexedAttestation(ctx, tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatal(err)
		}
		if len(idxAtts) != 0 {
			t.Errorf("Expected index attestation to have been deleted, received: %v", idxAtts)
		}
		if hasA {
			t.Errorf("Expected indexed attestation indexes list to be deleted, received: %v", hasA)
		}

	}

}

func TestHasIndexedAttestation(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(app, set, nil))
	defer teardownDB(t, db)
	ctx := context.Background()

	for _, tt := range tests {
		exists, err := db.HasIndexedAttestation(ctx, tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Fatal("has indexed attestation should return false for indexed attestations that are not in db")
		}

		if err := db.SaveIndexedAttestation(ctx, tt.idxAtt); err != nil {
			t.Fatalf("save indexed attestation failed: %v", err)
		}
	}

	for _, tt := range tests {
		exists, err := db.HasIndexedAttestation(ctx, tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
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
	db := setupDB(t, cli.NewContext(app, set, nil))
	defer teardownDB(t, db)
	ctx := context.Background()

	for _, tt := range tests {
		err := db.SaveIndexedAttestation(ctx, tt.idxAtt)
		if err != nil {
			t.Fatalf("save indexed attestation failed: %v", err)
		}

		idxAtts, err := db.IdxAttsForTargetFromID(ctx, tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatalf("failed to get indexed attestation: %v", err)
		}

		if idxAtts == nil || !reflect.DeepEqual(idxAtts[0], tt.idxAtt) {
			t.Fatalf("get should return bh: %v", idxAtts)
		}
	}
	currentEpoch := uint64(3)
	historyToKeep := uint64(1)
	err := db.PruneAttHistory(ctx, currentEpoch, historyToKeep)
	if err != nil {
		t.Fatalf("failed to prune: %v", err)
	}

	for _, tt := range tests {
		idxAtts, err := db.IdxAttsForTargetFromID(ctx, tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatalf("failed to get indexed attestation: %v", err)
		}
		exists, err := db.HasIndexedAttestation(ctx, tt.idxAtt.Data.Target.Epoch, tt.idxAtt.AttestingIndices[0])
		if err != nil {
			t.Fatal(err)
		}

		if tt.idxAtt.Data.Source.Epoch > currentEpoch-historyToKeep {
			if idxAtts == nil || !reflect.DeepEqual(idxAtts[0], tt.idxAtt) {
				t.Fatalf("get should return indexed attestation: %v", idxAtts)
			}
			if !exists {
				t.Fatalf("get should have indexed attestation for epoch: %v", idxAtts)
			}
		} else {
			if idxAtts != nil || exists {
				t.Fatalf("indexed attestation should have been pruned: %v has attestation: %t", idxAtts, exists)
			}
		}
	}
}
