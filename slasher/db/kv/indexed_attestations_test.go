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
			idxAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 0},
					Target: &ethpb.Checkpoint{Epoch: 1},
				},
				Signature: []byte{1, 2},
			},
		},
		{
			idxAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 2},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 0},
					Target: &ethpb.Checkpoint{Epoch: 2},
				},
				Signature: []byte{3, 4},
			},
		},
		{
			idxAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 1},
					Target: &ethpb.Checkpoint{Epoch: 2},
				},
				Signature: []byte{5, 6},
			},
		},
	}
}

func TestHasIndexedAttestation_NilDB(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(app, set, nil))
	defer teardownDB(t, db)
	ctx := context.Background()

	hasIdxAtt, err := db.HasIndexedAttestation(ctx, tests[0].idxAtt)
	if err != nil {
		t.Fatal(err)
	}
	if hasIdxAtt {
		t.Fatal("HasIndexedAttestation should return false")
	}
}

func TestSaveIndexedAttestation(t *testing.T) {
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

		exists, err := db.HasIndexedAttestation(ctx, tt.idxAtt)
		if err != nil {
			t.Fatalf("failed to get indexed attestation: %v", err)
		}

		if !exists {
			t.Fatal("Expected to find saved attestation in DB")
		}
	}
}

func TestIndexedAttestationWithPrefix(t *testing.T) {
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

		idxAtts, err := db.IndexedAttestationsWithPrefix(ctx, tt.idxAtt.Data.Target.Epoch, tt.idxAtt.Signature[:2])
		if err != nil {
			t.Fatalf("failed to get indexed attestation: %v", err)
		}

		if idxAtts == nil || !reflect.DeepEqual(idxAtts[0], tt.idxAtt) {
			t.Fatalf("get should return indexed attestation: %v", idxAtts)
		}
	}
}

func TestDeleteIndexedAttestation(t *testing.T) {
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
		found, err := db.HasIndexedAttestation(ctx, tt.idxAtt)
		if err != nil {
			t.Fatalf("Failed to check for index attestation: %v", err)
		}
		if !found {
			t.Fatalf("Expected indexed attestation: %v", tt.idxAtt)
		}

		if err = db.DeleteIndexedAttestation(ctx, tt.idxAtt); err != nil {
			t.Fatalf("Could not delete index attestation: %v", err)
		}

		found, err = db.HasIndexedAttestation(ctx, tt.idxAtt)
		if err != nil {
			t.Fatal(err)
		}
		if found {
			t.Error("Expected indexed attestation to be deleted")
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
		exists, err := db.HasIndexedAttestation(ctx, tt.idxAtt)
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
		exists, err := db.HasIndexedAttestation(ctx, tt.idxAtt)
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatal("has indexed attestation should return true")
		}
	}
}

func TestPruneHistoryIndexedAttestation(t *testing.T) {
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

		found, err := db.HasIndexedAttestation(ctx, tt.idxAtt)
		if err != nil {
			t.Fatalf("failed to get indexed attestation: %v", err)
		}

		if !found {
			t.Fatal("Expected to find attestation in DB")
		}
	}
	currentEpoch := uint64(3)
	historyToKeep := uint64(1)
	if err := db.PruneAttHistory(ctx, currentEpoch, historyToKeep); err != nil {
		t.Fatalf("failed to prune: %v", err)
	}

	for _, tt := range tests {
		exists, err := db.HasIndexedAttestation(ctx, tt.idxAtt)
		if err != nil {
			t.Fatal(err)
		}

		if tt.idxAtt.Data.Source.Epoch > currentEpoch-historyToKeep {
			if !exists {
				t.Fatal("Expected to find attestation newer than prune age in DB")
			}
		} else {
			if exists {
				t.Fatal("Expected to not find attestation older than prune age in DB")
			}
		}
	}
}
