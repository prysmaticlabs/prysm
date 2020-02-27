package detection

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations"
)

func TestDetect_detectSurroundVotes(t *testing.T) {
	type testStruct struct {
		name           string
		savedAtt       *ethpb.IndexedAttestation
		incomingAtt    *ethpb.IndexedAttestation
		slashingsFound int
	}
	tests := []testStruct{
		{
			name: "surrounding vote detected should report a slashing",
			savedAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{3},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 9},
					Target: &ethpb.Checkpoint{Epoch: 13},
				},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 3, 7},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 7},
					Target: &ethpb.Checkpoint{Epoch: 14},
				},
			},
			slashingsFound: 1,
		},
		{
			name: "surrounded vote detected should report a slashing",
			savedAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 2, 4, 8},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 6},
					Target: &ethpb.Checkpoint{Epoch: 10},
				},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 4},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 7},
					Target: &ethpb.Checkpoint{Epoch: 9},
				},
			},
			slashingsFound: 1,
		},
		{
			name: "no slashable detected should not report a slashing",
			savedAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 1},
					Target: &ethpb.Checkpoint{Epoch: 2},
				},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 0},
					Target: &ethpb.Checkpoint{Epoch: 1},
				},
			},
			slashingsFound: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testDB.SetupSlasherDB(t, false)
			ctx := context.Background()
			ds := Service{
				ctx:                ctx,
				slasherDB:          db,
				minMaxSpanDetector: &attestations.MockSpanDetector{},
			}
			if err := db.SaveIncomingIndexedAttestationByEpoch(ctx, tt.savedAtt); err != nil {
				t.Fatal(err)
			}

			slashableIndices := sliceutil.IntersectionUint64(tt.savedAtt.AttestingIndices, tt.incomingAtt.AttestingIndices)
			for _, valIdx := range slashableIndices {
				slashings, err := ds.detectSurroundVotes(ctx, valIdx, tt.incomingAtt)
				if err != nil {
					t.Fatal(err)
				}
				if len(slashings) != tt.slashingsFound {
					t.Fatalf("Unexpected amount of slashings found, received %d, expected %d", len(slashings), tt.slashingsFound)
				}

				for _, ss := range slashings {
					slashingAtt1 := ss.Attestation_1
					slashingAtt2 := ss.Attestation_2
					if !isSurrounding(slashingAtt1, slashingAtt2) && !isSurrounded(slashingAtt1, slashingAtt2) {
						t.Fatalf(
							"Expected slashing to be valid, received atts %d->%d and %d->%d",
							slashingAtt2.Data.Source.Epoch,
							slashingAtt2.Data.Target.Epoch,
							slashingAtt1.Data.Source.Epoch,
							slashingAtt1.Data.Target.Epoch,
						)
					}
				}
			}
			testDB.TeardownSlasherDB(t, db)
		})
	}
}
