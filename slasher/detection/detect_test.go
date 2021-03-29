package detection

import (
	"context"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/slashutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
	status "github.com/prysmaticlabs/prysm/slasher/db/types"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"github.com/prysmaticlabs/prysm/slasher/detection/proposals"
	testDetect "github.com/prysmaticlabs/prysm/slasher/detection/testing"
)

func TestDetect_detectAttesterSlashings_Surround(t *testing.T) {
	type testStruct struct {
		name           string
		savedAtts      []*ethpb.IndexedAttestation
		incomingAtt    *ethpb.IndexedAttestation
		slashingsFound int
	}
	tests := []testStruct{
		{
			name: "surrounding vote detected should report a slashing",
			savedAtts: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{3},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 9, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 13, Root: make([]byte, 32)},
						BeaconBlockRoot: make([]byte, 32),
					},
					Signature: bytesutil.PadTo([]byte{1, 2}, 96),
				},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 3, 7},
				Data: &ethpb.AttestationData{
					Source:          &ethpb.Checkpoint{Epoch: 7, Root: make([]byte, 32)},
					Target:          &ethpb.Checkpoint{Epoch: 14, Root: make([]byte, 32)},
					BeaconBlockRoot: make([]byte, 32),
				},
				Signature: make([]byte, 96),
			},
			slashingsFound: 1,
		},
		{
			name: "surrounded vote detected should report a slashing",
			savedAtts: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0, 2, 4, 8},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 6, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 10, Root: make([]byte, 32)},
						BeaconBlockRoot: make([]byte, 32),
					},
					Signature: bytesutil.PadTo([]byte{1, 2}, 96),
				},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 4},
				Data: &ethpb.AttestationData{
					Source:          &ethpb.Checkpoint{Epoch: 7, Root: make([]byte, 32)},
					Target:          &ethpb.Checkpoint{Epoch: 9, Root: make([]byte, 32)},
					BeaconBlockRoot: make([]byte, 32),
				},
				Signature: make([]byte, 96),
			},
			slashingsFound: 1,
		},
		{
			name: "2 different surrounded votes detected should report 2 slashings",
			savedAtts: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0, 2},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 5, Root: make([]byte, 32)},
						BeaconBlockRoot: make([]byte, 32),
					},
					Signature: bytesutil.PadTo([]byte{1, 2}, 96),
				},
				{
					AttestingIndices: []uint64{4, 8},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
						BeaconBlockRoot: make([]byte, 32),
					},
					Signature: bytesutil.PadTo([]byte{1, 3}, 96),
				},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 4},
				Data: &ethpb.AttestationData{
					Source:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
					Target:          &ethpb.Checkpoint{Epoch: 7, Root: make([]byte, 32)},
					BeaconBlockRoot: make([]byte, 32),
				},
				Signature: make([]byte, 96),
			},
			slashingsFound: 2,
		},
		{
			name: "2 different surrounding votes detected should report 2 slashings",
			savedAtts: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0, 2},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 10, Root: make([]byte, 32)},
						BeaconBlockRoot: make([]byte, 32),
					},
					Signature: bytesutil.PadTo([]byte{1, 2}, 96),
				},
				{
					AttestingIndices: []uint64{4, 8},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 5, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 9, Root: make([]byte, 32)},
						BeaconBlockRoot: make([]byte, 32),
					},
					Signature: bytesutil.PadTo([]byte{1, 3}, 96),
				},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 4},
				Data: &ethpb.AttestationData{
					Source:          &ethpb.Checkpoint{Epoch: 7, Root: make([]byte, 32)},
					Target:          &ethpb.Checkpoint{Epoch: 8, Root: make([]byte, 32)},
					BeaconBlockRoot: make([]byte, 32),
				},
				Signature: make([]byte, 96),
			},
			slashingsFound: 2,
		},
		{
			name: "no slashable detected should not report a slashing",
			savedAtts: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
						BeaconBlockRoot: make([]byte, 32),
					},
					Signature: bytesutil.PadTo([]byte{1, 2}, 96),
				},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0},
				Data: &ethpb.AttestationData{
					Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
					Target:          &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					BeaconBlockRoot: make([]byte, 32),
				},
				Signature: make([]byte, 96),
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
				cfg:                &Config{SlasherDB: db},
				minMaxSpanDetector: attestations.NewSpanDetector(db),
			}
			require.NoError(t, db.SaveIndexedAttestations(ctx, tt.savedAtts))
			for _, att := range tt.savedAtts {
				require.NoError(t, ds.minMaxSpanDetector.UpdateSpans(ctx, att))
			}

			slashings, err := ds.DetectAttesterSlashings(ctx, tt.incomingAtt)
			require.NoError(t, err)
			require.Equal(t, tt.slashingsFound, len(slashings), "Unexpected amount of slashings found")
			attsl, err := db.AttesterSlashings(ctx, status.Active)
			require.NoError(t, err)
			require.Equal(t, tt.slashingsFound, len(attsl), "Didnt save slashing to db")
			for _, ss := range slashings {
				slashingAtt1 := ss.Attestation_1
				slashingAtt2 := ss.Attestation_2
				if !slashutil.IsSurround(slashingAtt1, slashingAtt2) {
					t.Fatalf(
						"Expected slashing to be valid, received atts %d->%d and %d->%d",
						slashingAtt2.Data.Source.Epoch,
						slashingAtt2.Data.Target.Epoch,
						slashingAtt1.Data.Source.Epoch,
						slashingAtt1.Data.Target.Epoch,
					)
				}
			}
		})
	}
}

func TestDetect_detectAttesterSlashings_Double(t *testing.T) {
	type testStruct struct {
		name           string
		savedAtts      []*ethpb.IndexedAttestation
		incomingAtt    *ethpb.IndexedAttestation
		slashingsFound int
	}
	tests := []testStruct{
		{
			name: "different source, same target, should report a slashing",
			savedAtts: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{3},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
						BeaconBlockRoot: make([]byte, 32),
					},
					Signature: bytesutil.PadTo([]byte{1, 2}, 96),
				},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 3, 7},
				Data: &ethpb.AttestationData{
					Source:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
					Target:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
					BeaconBlockRoot: make([]byte, 32),
				},
				Signature: bytesutil.PadTo([]byte{1, 2}, 96),
			},
			slashingsFound: 1,
		},
		{
			name: "different histories, same target, should report 2 slashings",
			savedAtts: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{1},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
						BeaconBlockRoot: make([]byte, 32),
					},
					Signature: bytesutil.PadTo([]byte{1, 2}, 96),
				},
				{
					AttestingIndices: []uint64{3},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
						BeaconBlockRoot: make([]byte, 32),
					},
					Signature: bytesutil.PadTo([]byte{1, 3}, 96),
				},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 3, 7},
				Data: &ethpb.AttestationData{
					Source:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
					Target:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
					BeaconBlockRoot: make([]byte, 32),
				},
				Signature: bytesutil.PadTo([]byte{1, 4}, 96),
			},
			slashingsFound: 2,
		},
		{
			name: "same source and target, different block root, should report a slashing ",
			savedAtts: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0, 2, 4, 8},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
						BeaconBlockRoot: bytesutil.PadTo([]byte("good block root"), 32),
					},
					Signature: bytesutil.PadTo([]byte{1, 2}, 96),
				},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 4},
				Data: &ethpb.AttestationData{
					Source:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
					Target:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
					BeaconBlockRoot: bytesutil.PadTo([]byte("bad block root"), 32),
				},
				Signature: make([]byte, 96),
			},
			slashingsFound: 1,
		},
		{
			name: "same attestation should not report double",
			savedAtts: []*ethpb.IndexedAttestation{
				{
					AttestingIndices: []uint64{0},
					Data: &ethpb.AttestationData{
						Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
						Target:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
						BeaconBlockRoot: bytesutil.PadTo([]byte("good block root"), 32),
					},
					Signature: bytesutil.PadTo([]byte{1, 2}, 96),
				},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0},
				Data: &ethpb.AttestationData{
					Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
					Target:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
					BeaconBlockRoot: bytesutil.PadTo([]byte("good block root"), 32),
				},
				Signature: make([]byte, 96),
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
				cfg:                &Config{SlasherDB: db},
				minMaxSpanDetector: attestations.NewSpanDetector(db),
			}
			require.NoError(t, db.SaveIndexedAttestations(ctx, tt.savedAtts))
			for _, att := range tt.savedAtts {
				require.NoError(t, ds.minMaxSpanDetector.UpdateSpans(ctx, att))
			}

			slashings, err := ds.DetectAttesterSlashings(ctx, tt.incomingAtt)
			require.NoError(t, err)
			require.Equal(t, tt.slashingsFound, len(slashings), "Unexpected amount of slashings found")
			savedSlashings, err := db.AttesterSlashings(ctx, status.Active)
			require.NoError(t, err)
			require.Equal(t, tt.slashingsFound, len(savedSlashings), "Did not save slashing to db")

			for _, ss := range slashings {
				slashingAtt1 := ss.Attestation_1
				slashingAtt2 := ss.Attestation_2
				if !isDoubleVote(slashingAtt1, slashingAtt2) {
					t.Fatalf(
						"Expected slashing to be valid, received atts with target epoch %d and %d but not valid",
						slashingAtt2.Data.Target.Epoch,
						slashingAtt1.Data.Target.Epoch,
					)
				}
			}
		})
	}
}

func TestDetect_updateHighestAttestation(t *testing.T) {
	tests := []struct {
		name         string
		savedHighest *slashpb.HighestAttestation
		incomingAtt  *ethpb.IndexedAttestation
		expected     *slashpb.HighestAttestation
	}{
		{
			name: "update only target to higher",
			savedHighest: &slashpb.HighestAttestation{
				ValidatorId:        1,
				HighestSourceEpoch: 1,
				HighestTargetEpoch: 2,
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 3, 7},
				Data: &ethpb.AttestationData{
					Source:          &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					Target:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
					BeaconBlockRoot: make([]byte, 32),
				},
				Signature: bytesutil.PadTo([]byte{1, 2}, 96),
			},
			expected: &slashpb.HighestAttestation{
				ValidatorId:        1,
				HighestSourceEpoch: 1,
				HighestTargetEpoch: 4,
			},
		},
		{
			name: "update target and source to higher",
			savedHighest: &slashpb.HighestAttestation{
				ValidatorId:        1,
				HighestSourceEpoch: 1,
				HighestTargetEpoch: 2,
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 3, 7},
				Data: &ethpb.AttestationData{
					Source:          &ethpb.Checkpoint{Epoch: 3, Root: make([]byte, 32)},
					Target:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
					BeaconBlockRoot: make([]byte, 32),
				},
				Signature: bytesutil.PadTo([]byte{1, 2}, 96),
			},
			expected: &slashpb.HighestAttestation{
				ValidatorId:        1,
				HighestSourceEpoch: 3,
				HighestTargetEpoch: 4,
			},
		},
		{
			name: "no update",
			savedHighest: &slashpb.HighestAttestation{
				ValidatorId:        1,
				HighestSourceEpoch: 1,
				HighestTargetEpoch: 2,
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 3, 7},
				Data: &ethpb.AttestationData{
					Source:          &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					Target:          &ethpb.Checkpoint{Epoch: 2, Root: make([]byte, 32)},
					BeaconBlockRoot: make([]byte, 32),
				},
				Signature: bytesutil.PadTo([]byte{1, 2}, 96),
			},
			expected: &slashpb.HighestAttestation{
				ValidatorId:        1,
				HighestSourceEpoch: 1,
				HighestTargetEpoch: 2,
			},
		},
		{
			name: "update target to higher when source is lower(should be a slashable attestation)",
			savedHighest: &slashpb.HighestAttestation{
				ValidatorId:        1,
				HighestSourceEpoch: 5,
				HighestTargetEpoch: 6,
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 3, 7},
				Data: &ethpb.AttestationData{
					Source:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
					Target:          &ethpb.Checkpoint{Epoch: 8, Root: make([]byte, 32)},
					BeaconBlockRoot: make([]byte, 32),
				},
				Signature: bytesutil.PadTo([]byte{1, 2}, 96),
			},
			expected: &slashpb.HighestAttestation{
				ValidatorId:        1,
				HighestSourceEpoch: 5,
				HighestTargetEpoch: 8,
			},
		},
		{
			name: "update source to higher when target is same",
			savedHighest: &slashpb.HighestAttestation{
				ValidatorId:        1,
				HighestSourceEpoch: 3,
				HighestTargetEpoch: 6,
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 3, 7},
				Data: &ethpb.AttestationData{
					Source:          &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
					Target:          &ethpb.Checkpoint{Epoch: 6, Root: make([]byte, 32)},
					BeaconBlockRoot: make([]byte, 32),
				},
				Signature: bytesutil.PadTo([]byte{1, 2}, 96),
			},
			expected: &slashpb.HighestAttestation{
				ValidatorId:        1,
				HighestSourceEpoch: 4,
				HighestTargetEpoch: 6,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testDB.SetupSlasherDB(t, false)
			ctx := context.Background()
			ds := Service{
				ctx:               ctx,
				cfg:               &Config{SlasherDB: db},
				proposalsDetector: proposals.NewProposeDetector(db),
			}
			require.NoError(t, db.SaveHighestAttestation(ctx, tt.savedHighest))

			// Update and assert.
			require.NoError(t, ds.UpdateHighestAttestation(ctx, tt.incomingAtt))
			h, err := db.HighestAttestation(ctx, tt.savedHighest.ValidatorId)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.HighestSourceEpoch, h.HighestSourceEpoch)
			assert.Equal(t, tt.expected.HighestTargetEpoch, h.HighestTargetEpoch)
		})
	}
}

func TestDetect_detectProposerSlashing(t *testing.T) {
	type testStruct struct {
		name        string
		blk         *ethpb.SignedBeaconBlockHeader
		incomingBlk *ethpb.SignedBeaconBlockHeader
		slashing    *ethpb.ProposerSlashing
	}
	s0, err := helpers.StartSlot(0)
	require.NoError(t, err)
	sigBlk1slot0, err := testDetect.SignedBlockHeader(s0, 0)
	require.NoError(t, err)
	sigBlk2slot0, err := testDetect.SignedBlockHeader(s0, 0)
	require.NoError(t, err)
	s1, err := helpers.StartSlot(1)
	require.NoError(t, err)
	sigBlk1epoch1, err := testDetect.SignedBlockHeader(s1, 0)
	require.NoError(t, err)
	tests := []testStruct{
		{
			name:        "same block sig dont slash",
			blk:         sigBlk1slot0,
			incomingBlk: sigBlk1slot0,
			slashing:    nil,
		},
		{
			name:        "block from different epoch dont slash",
			blk:         sigBlk1slot0,
			incomingBlk: sigBlk1epoch1,
			slashing:    nil,
		},
		{
			name:        "different sig from same slot slash",
			blk:         sigBlk1slot0,
			incomingBlk: sigBlk2slot0,
			slashing:    &ethpb.ProposerSlashing{Header_1: sigBlk2slot0, Header_2: sigBlk1slot0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testDB.SetupSlasherDB(t, false)
			ctx := context.Background()
			ds := Service{
				ctx:               ctx,
				cfg:               &Config{SlasherDB: db},
				proposalsDetector: proposals.NewProposeDetector(db),
			}
			require.NoError(t, db.SaveBlockHeader(ctx, tt.blk))

			slashing, err := ds.proposalsDetector.DetectDoublePropose(ctx, tt.incomingBlk)
			require.NoError(t, err)
			assert.DeepEqual(t, tt.slashing, slashing)
			savedSlashings, err := db.ProposalSlashingsByStatus(ctx, status.Active)
			require.NoError(t, err)
			if tt.slashing != nil {
				require.Equal(t, 1, len(savedSlashings), "Did not save slashing to db")
			}

			if slashing != nil && !isDoublePropose(slashing.Header_1, slashing.Header_2) {
				t.Fatalf(
					"Expected slashing to be valid, received atts with target epoch %v and %v but not valid",
					slashing.Header_1,
					slashing.Header_2,
				)
			}
		})
	}
}
func TestDetect_detectProposerSlashingNoUpdate(t *testing.T) {
	type testStruct struct {
		name        string
		blk         *ethpb.SignedBeaconBlockHeader
		noUpdtaeBlk *ethpb.BeaconBlockHeader
		slashable   bool
	}
	s0, err := helpers.StartSlot(0)
	require.NoError(t, err)
	sigBlk1slot0, err := testDetect.SignedBlockHeader(s0, 0)
	require.NoError(t, err)
	blk1slot0, err := testDetect.BlockHeader(s0, 0)
	require.NoError(t, err)
	blk2slot0, err := testDetect.BlockHeader(s0, 0)
	require.NoError(t, err)
	diffRoot := [32]byte{1, 1, 1}
	blk2slot0.ParentRoot = diffRoot[:]
	blk3slot0, err := testDetect.BlockHeader(s0, 0)
	require.NoError(t, err)
	blk3slot0.StateRoot = diffRoot[:]
	blk4slot0, err := testDetect.BlockHeader(s0, 0)
	require.NoError(t, err)
	blk4slot0.BodyRoot = diffRoot[:]
	tests := []testStruct{
		{
			name:        "same block don't slash",
			blk:         sigBlk1slot0,
			noUpdtaeBlk: blk1slot0,
			slashable:   false,
		},
		{
			name:        "diff parent root slash",
			blk:         sigBlk1slot0,
			noUpdtaeBlk: blk2slot0,
			slashable:   true,
		},
		{
			name:        "diff state root slash",
			blk:         sigBlk1slot0,
			noUpdtaeBlk: blk3slot0,
			slashable:   true,
		},
		{
			name:        "diff body root slash",
			blk:         sigBlk1slot0,
			noUpdtaeBlk: blk4slot0,
			slashable:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testDB.SetupSlasherDB(t, false)
			ctx := context.Background()
			ds := Service{
				ctx:               ctx,
				cfg:               &Config{SlasherDB: db},
				proposalsDetector: proposals.NewProposeDetector(db),
			}
			require.NoError(t, db.SaveBlockHeader(ctx, tt.blk))

			slashble, err := ds.proposalsDetector.DetectDoubleProposeNoUpdate(ctx, tt.noUpdtaeBlk)
			require.NoError(t, err)
			assert.Equal(t, tt.slashable, slashble)
		})
	}
}

func TestServer_MapResultsToAtts(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	ctx := context.Background()
	ds := Service{
		ctx: ctx,
		cfg: &Config{SlasherDB: db},
	}
	// 3 unique results, but 7 validators in total.
	results := []*types.DetectionResult{
		// 3 For the same slashable epoch and same sigs.
		{
			ValidatorIndex: 1,
			SlashableEpoch: 5,
			Kind:           types.DoubleVote,
			SigBytes:       [2]byte{5, 5},
		},
		{
			ValidatorIndex: 2,
			SlashableEpoch: 5,
			Kind:           types.DoubleVote,
			SigBytes:       [2]byte{5, 5},
		},
		{
			ValidatorIndex: 3,
			SlashableEpoch: 5,
			Kind:           types.DoubleVote,
			SigBytes:       [2]byte{5, 5},
		},
		// Different signature.
		{
			ValidatorIndex: 5,
			SlashableEpoch: 5,
			Kind:           types.DoubleVote,
			SigBytes:       [2]byte{3, 5},
		},
		// Different slashable epoch.
		{
			ValidatorIndex: 5,
			SlashableEpoch: 4,
			Kind:           types.DoubleVote,
			SigBytes:       [2]byte{5, 5},
		},
		// Different both.
		{
			ValidatorIndex: 8,
			SlashableEpoch: 6,
			Kind:           types.DoubleVote,
			SigBytes:       [2]byte{2, 1},
		},
		{
			ValidatorIndex: 7,
			SlashableEpoch: 6,
			Kind:           types.DoubleVote,
			SigBytes:       [2]byte{2, 1},
		},
	}
	expectedResultsToAtts := map[[32]byte][]*ethpb.IndexedAttestation{
		resultHash(results[0]): {
			createIndexedAttForResult(results[0]),
			createIndexedAttForResult(results[1]),
			createIndexedAttForResult(results[2]),
		},
		resultHash(results[3]): {
			createIndexedAttForResult(results[3]),
		},
		resultHash(results[4]): {
			createIndexedAttForResult(results[4]),
		},
		resultHash(results[5]): {
			createIndexedAttForResult(results[6]),
			createIndexedAttForResult(results[5]),
		},
	}
	for _, atts := range expectedResultsToAtts {
		require.NoError(t, ds.cfg.SlasherDB.SaveIndexedAttestations(ctx, atts))
	}

	resultsToAtts, err := ds.mapResultsToAtts(ctx, results)
	require.NoError(t, err)
	if !reflect.DeepEqual(expectedResultsToAtts, resultsToAtts) {
		t.Error("Expected map:")
		for key, value := range resultsToAtts {
			t.Errorf("Key %#x: %d atts", key, len(value))
			t.Errorf("%+v", value)
		}
		t.Error("To equal:")
		for key, value := range expectedResultsToAtts {
			t.Errorf("Key %#x: %d atts", key, len(value))
			t.Errorf("%+v", value)
		}
	}
}

func createIndexedAttForResult(result *types.DetectionResult) *ethpb.IndexedAttestation {
	return &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{result.ValidatorIndex},
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: []byte("text block root"),
			Target: &ethpb.Checkpoint{
				Epoch: result.SlashableEpoch,
			},
		},
		Signature: append(result.SigBytes[:], []byte{uint8(result.ValidatorIndex), 4, 5, 6, 7, 8}...),
	}
}
