package rpc

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
	"github.com/prysmaticlabs/prysm/slasher/detection"
)

func Test_DetectionFlow(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)

	savedAttestation := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{3},
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 3},
			Target: &ethpb.Checkpoint{Epoch: 4},
		},
		Signature: bytesutil.PadTo([]byte{1, 2}, 96),
	}
	incomingAtt := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{3},
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 2},
			Target: &ethpb.Checkpoint{Epoch: 4},
		},
		Signature: bytesutil.PadTo([]byte{1, 2}, 96),
	}
	cfg := &detection.Config{
		SlasherDB: db,
	}
	ctx := context.Background()
	ds := detection.NewDetectionService(ctx, cfg)
	server := Server{ctx: ctx, detector: ds, slasherDB: db}
	slashings, err := server.IsSlashableAttestation(ctx, savedAttestation)
	if err != nil {
		t.Fatalf("got error while trying to detect slashing: %v", err)
	}
	if len(slashings.AttesterSlashing) != 0 {
		t.Fatalf("Found slashings while no slashing should have been found on first attestation: %v slashing found: %v", savedAttestation, slashings)
	}

	slashing, err := server.IsSlashableAttestation(ctx, incomingAtt)
	if err != nil {
		t.Fatalf("got error while trying to detect slashing: %v", err)
	}
	if len(slashing.AttesterSlashing) != 1 {
		t.Fatalf("only one slashing should have been found. got: %v", len(slashing.AttesterSlashing))
	}
}
