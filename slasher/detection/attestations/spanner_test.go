package attestations

import (
	"context"
	"reflect"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
	dbtypes "github.com/prysmaticlabs/prysm/slasher/db/types"
	slashertypes "github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func indexedAttestation(source, target types.Epoch, indices []uint64) *ethpb.IndexedAttestation {
	return &ethpb.IndexedAttestation{
		AttestingIndices: indices,
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{
				Epoch: source,
				Root:  []byte("good source"),
			},
			Target: &ethpb.Checkpoint{
				Epoch: target,
				Root:  []byte("good target"),
			},
		},
		Signature: []byte{1, 2},
	}
}

func TestSpanDetector_DetectSlashingsForAttestation_Double(t *testing.T) {
	type testStruct struct {
		name        string
		att         *ethpb.IndexedAttestation
		incomingAtt *ethpb.IndexedAttestation
		slashCount  int
	}
	tests := []testStruct{
		{
			name: "att with different target root, same target epoch, should slash",
			att: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 2},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 0,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 2,
						Root:  []byte("good target"),
					},
				},
				Signature: []byte{1, 2},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{2},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 0,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 2,
						Root:  []byte("bad target"),
					},
				},
			},
			slashCount: 1,
		},
		{
			name: "att with different source, same target, should slash",
			att: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 2},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 0,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 2,
						Root:  []byte("good target"),
					},
				},
				Signature: []byte{1, 2},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{2},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 1,
						Root:  []byte("bad source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 2,
						Root:  []byte("bad target"),
					},
				},
			},
			slashCount: 1,
		},
		{
			name: "att with different committee index, rest is the same, should slash",
			att: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 2},
				Data: &ethpb.AttestationData{
					CommitteeIndex: 4,
					Source: &ethpb.Checkpoint{
						Epoch: 0,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 2,
						Root:  []byte("good target"),
					},
				},
				Signature: []byte{1, 2},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{2},
				Data: &ethpb.AttestationData{
					CommitteeIndex: 3,
					Source: &ethpb.Checkpoint{
						Epoch: 1,
						Root:  []byte("bad source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 2,
						Root:  []byte("bad target"),
					},
				},
				Signature: []byte{1, 2},
			},
			slashCount: 1,
		},
		{
			name: "att with same target and source, different block root, should slash",
			att: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 2, 4, 6},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 0,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 2,
						Root:  []byte("good target"),
					},
					BeaconBlockRoot: []byte("good block root"),
				},
				Signature: []byte{1, 2},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{2, 4, 6},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 0,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 2,
						Root:  []byte("good target"),
					},
					BeaconBlockRoot: []byte("bad block root"),
				},
			},
			slashCount: 3,
		},
		{
			name: "att with different target, should not detect possible double",
			att: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 2, 4, 6},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 0,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 2,
						Root:  []byte("good target"),
					},
					BeaconBlockRoot: []byte("good block root"),
				},
				Signature: []byte{1, 2},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{2, 4, 6},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 0,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 1,
						Root:  []byte("really good target"),
					},
					BeaconBlockRoot: []byte("really good block root"),
				},
			},
			slashCount: 0,
		},
		{
			name: "same att with different aggregates, should detect possible double",
			att: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{1, 2, 4, 6},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 0,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 2,
						Root:  []byte("good target"),
					},
					BeaconBlockRoot: []byte("good block root"),
				},
				Signature: []byte{1, 2},
			},
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{2, 3, 4, 16},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 0,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 2,
						Root:  []byte("good target"),
					},
					BeaconBlockRoot: []byte("good block root"),
				},
			},
			slashCount: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testDB.SetupSlasherDB(t, false)
			ctx := context.Background()

			sd := &SpanDetector{
				slasherDB: db,
			}

			require.NoError(t, sd.UpdateSpans(ctx, tt.att))

			res, err := sd.DetectSlashingsForAttestation(ctx, tt.incomingAtt)
			require.NoError(t, err)

			var want []*slashertypes.DetectionResult
			if tt.slashCount > 0 {
				for _, indice := range sliceutil.IntersectionUint64(tt.att.AttestingIndices, tt.incomingAtt.AttestingIndices) {
					want = append(want, &slashertypes.DetectionResult{
						ValidatorIndex: indice,
						Kind:           slashertypes.DoubleVote,
						SlashableEpoch: tt.incomingAtt.Data.Target.Epoch,
						SigBytes:       [2]byte{1, 2},
					})
				}
			}
			assert.DeepEqual(t, want, res)
			require.Equal(t, tt.slashCount, len(res), "Unexpected amount of slashings found")
		})
	}
}

func TestSpanDetector_DetectSlashingsForAttestation_Surround(t *testing.T) {
	type testStruct struct {
		name                     string
		sourceEpoch              types.Epoch
		targetEpoch              types.Epoch
		slashableEpoch           types.Epoch
		shouldSlash              bool
		spansByEpochForValidator map[types.Epoch][3]uint16
	}
	tests := []testStruct{
		{
			name:           "Should slash if max span > distance",
			sourceEpoch:    3,
			targetEpoch:    6,
			slashableEpoch: 7,
			shouldSlash:    true,
			// Given a distance of (6 - 3) = 3, we want the validator at epoch 3 to have
			// committed a slashable offense by having a max span of 4 > distance.
			spansByEpochForValidator: map[types.Epoch][3]uint16{
				3: {0, 4},
			},
		},
		{
			name:        "Should NOT slash if max span < distance",
			sourceEpoch: 3,
			targetEpoch: 6,
			// Given a distance of (6 - 3) = 3, we want the validator at epoch 3 to NOT
			// have committed slashable offense by having a max span of 1 < distance.
			shouldSlash: false,
			spansByEpochForValidator: map[types.Epoch][3]uint16{
				3: {0, 1},
			},
		},
		{
			name:        "Should NOT slash if max span == distance",
			sourceEpoch: 3,
			targetEpoch: 6,
			// Given a distance of (6 - 3) = 3, we want the validator at epoch 3 to NOT
			// have committed slashable offense by having a max span of 3 == distance.
			shouldSlash: false,
			spansByEpochForValidator: map[types.Epoch][3]uint16{
				3: {0, 3},
			},
		},
		{
			name:        "Should NOT slash if min span == 0",
			sourceEpoch: 3,
			targetEpoch: 6,
			// Given a min span of 0 and no max span slashing, we want validator to NOT
			// have committed a slashable offense if min span == 0.
			shouldSlash: false,
			spansByEpochForValidator: map[types.Epoch][3]uint16{
				3: {0, 1},
			},
		},
		{
			name:        "Should slash if min span > 0 and min span < distance",
			sourceEpoch: 3,
			targetEpoch: 6,
			// Given a distance of (6 - 3) = 3, we want the validator at epoch 3 to have
			// committed a slashable offense by having a min span of 1 < distance.
			shouldSlash:    true,
			slashableEpoch: 4,
			spansByEpochForValidator: map[types.Epoch][3]uint16{
				3: {1, 0},
			},
		},
		// Proto Max Span Tests from the eth2-surround repo.
		{
			name:        "Proto max span test #1",
			sourceEpoch: 8,
			targetEpoch: 18,
			shouldSlash: false,
			spansByEpochForValidator: map[types.Epoch][3]uint16{
				0: {4, 0},
				1: {2, 0},
				2: {1, 0},
				4: {0, 2},
				5: {0, 1},
			},
		},
		{
			name:           "Proto max span test #2",
			sourceEpoch:    4,
			targetEpoch:    12,
			shouldSlash:    false,
			slashableEpoch: 0,
			spansByEpochForValidator: map[types.Epoch][3]uint16{
				4:  {14, 2},
				5:  {13, 1},
				6:  {12, 0},
				7:  {11, 0},
				9:  {0, 9},
				10: {0, 8},
				11: {0, 7},
				12: {0, 6},
				13: {0, 5},
				14: {0, 4},
				15: {0, 3},
				16: {0, 2},
				17: {0, 1},
			},
		},
		{
			name:           "Proto max span test #3",
			sourceEpoch:    10,
			targetEpoch:    15,
			shouldSlash:    true,
			slashableEpoch: 18,
			spansByEpochForValidator: map[types.Epoch][3]uint16{
				4:  {14, 2},
				5:  {13, 7},
				6:  {12, 6},
				7:  {11, 5},
				8:  {0, 4},
				9:  {0, 9},
				10: {0, 8},
				11: {0, 7},
				12: {0, 6},
				13: {0, 5},
				14: {0, 4},
				15: {0, 3},
				16: {0, 2},
				17: {0, 1},
			},
		},
		// Proto Min Span Tests from the eth2-surround repo.
		{
			name:        "Proto min span test #1",
			sourceEpoch: 4,
			targetEpoch: 6,
			shouldSlash: false,
			spansByEpochForValidator: map[types.Epoch][3]uint16{
				1: {5, 0},
				2: {4, 0},
				3: {3, 0},
			},
		},
		{
			name:        "Proto min span test #2",
			sourceEpoch: 11,
			targetEpoch: 15,
			shouldSlash: false,
			spansByEpochForValidator: map[types.Epoch][3]uint16{
				1:  {5, 0},
				2:  {4, 0},
				3:  {3, 0},
				4:  {14, 0},
				5:  {13, 1},
				6:  {12, 0},
				7:  {11, 0},
				8:  {10, 0},
				9:  {9, 0},
				10: {8, 0},
				11: {7, 0},
				12: {6, 0},
				14: {0, 4},
				15: {0, 3},
				16: {0, 2},
				17: {0, 1},
			},
		},
		{
			name:           "Proto min span test #3",
			sourceEpoch:    9,
			targetEpoch:    19,
			shouldSlash:    true,
			slashableEpoch: 14,
			spansByEpochForValidator: map[types.Epoch][3]uint16{
				0:  {5, 0},
				1:  {4, 0},
				2:  {3, 0},
				3:  {11, 0},
				4:  {10, 1},
				5:  {9, 0},
				6:  {8, 0},
				7:  {7, 0},
				8:  {6, 0},
				9:  {5, 0},
				10: {7, 0},
				11: {6, 3},
				12: {0, 2},
				13: {0, 1},
				14: {0, 3},
				15: {0, 2},
				16: {0, 1},
				17: {0, 0},
			},
		},
		{
			name:           "Should slash if max span > distance && source > target",
			sourceEpoch:    6,
			targetEpoch:    3,
			slashableEpoch: 7,
			shouldSlash:    true,
			// Given a distance of (6 - 3) = 3, we want the validator at epoch 3 to have
			// committed a slashable offense by having a max span of 4 > distance.
			spansByEpochForValidator: map[types.Epoch][3]uint16{
				3: {0, 4},
			},
		},
		{
			name:        "Should not slash if max span = distance && source > target",
			sourceEpoch: 6,
			targetEpoch: 3,
			shouldSlash: false,
			// Given a distance of (6 - 3) = 3, we want the validator at epoch 3 to NOT
			// have committed slashable offense by having a max span of 1 < distance.
			spansByEpochForValidator: map[types.Epoch][3]uint16{
				3: {0, 1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testDB.SetupSlasherDB(t, false)
			ctx := context.Background()

			sd := &SpanDetector{
				slasherDB: db,
			}
			// We only care about validator index 0 for these tests for simplicity.
			validatorIndex := uint64(0)
			for k, v := range tt.spansByEpochForValidator {
				epochStore, err := slashertypes.NewEpochStore([]byte{})
				require.NoError(t, err)
				span := slashertypes.Span{
					MinSpan: v[0],
					MaxSpan: v[1],
				}
				epochStore, err = epochStore.SetValidatorSpan(validatorIndex, span)
				require.NoError(t, err)
				require.NoError(t, sd.slasherDB.SaveEpochSpans(ctx, k, epochStore, dbtypes.UseDB), "Failed to save to slasherDB")
			}

			att := &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: tt.sourceEpoch,
					},
					Target: &ethpb.Checkpoint{
						Epoch: tt.targetEpoch,
					},
				},
				AttestingIndices: []uint64{0},
			}
			res, err := sd.DetectSlashingsForAttestation(ctx, att)
			require.NoError(t, err)
			require.Equal(t, false, !tt.shouldSlash && res != nil, "Did not want validator to be slashed but found slashable offense: %v", res)
			if tt.shouldSlash {
				want := []*slashertypes.DetectionResult{
					{
						Kind:           slashertypes.SurroundVote,
						SlashableEpoch: tt.slashableEpoch,
					},
				}
				assert.DeepEqual(t, want, res)
			}
		})
	}
}

func TestSpanDetector_DetectSlashingsForAttestation_MultipleValidators(t *testing.T) {
	type testStruct struct {
		name            string
		incomingAtt     *ethpb.IndexedAttestation
		slashableEpochs []types.Epoch
		shouldSlash     []bool
		atts            []*ethpb.IndexedAttestation
	}
	tests := []testStruct{
		{
			name: "3 of 4 validators slashed, differing histories",
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 1, 2, 3},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 3,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 6,
						Root:  []byte("good target"),
					},
				},
				Signature: []byte{1, 2},
			},
			slashableEpochs: []types.Epoch{6, 7, 5, 0},
			// Detections - double, surround, surrounded, none.
			shouldSlash: []bool{true, true, true, false},
			// Atts in map: (src, epoch) - 0: (3, 6), 1: (2, 7), 2: (4, 5), 3: (5, 7)
			atts: []*ethpb.IndexedAttestation{
				indexedAttestation(3, 6, []uint64{0}),
				indexedAttestation(2, 7, []uint64{1}),
				indexedAttestation(4, 5, []uint64{2}),
				indexedAttestation(5, 7, []uint64{3}),
			},
		},
		{
			name: "3 of 4 validators slashed, differing surrounds",
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 1, 2, 3},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 5,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 7,
						Root:  []byte("good target"),
					},
				},
				Signature: []byte{1, 2},
			},
			slashableEpochs: []types.Epoch{8, 9, 10, 0},
			// Detections - surround, surround, surround, none.
			shouldSlash: []bool{true, true, true, false},
			// Atts in map: (src, epoch) - 0: (1, 8), 1: (3, 9), 2: (2, 10), 3: (4, 6)
			atts: []*ethpb.IndexedAttestation{
				indexedAttestation(1, 8, []uint64{0}),
				indexedAttestation(3, 9, []uint64{1}),
				indexedAttestation(2, 10, []uint64{2}),
				indexedAttestation(4, 6, []uint64{3}),
			},
		},
		{
			name: "3 of 4 validators slashed, differing surrounded",
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 1, 2, 3},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 2,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 9,
						Root:  []byte("good target"),
					},
				},
				Signature: []byte{1, 2},
			},
			slashableEpochs: []types.Epoch{8, 8, 7, 0},
			// Detections - surround, surround, surround, none.
			shouldSlash: []bool{true, true, true, false},
			// Atts in map: (src, epoch) - 0: (5, 8), 1: (3, 8), 2: (4, 7), 3: (1, 5)
			atts: []*ethpb.IndexedAttestation{
				indexedAttestation(5, 8, []uint64{0}),
				indexedAttestation(3, 8, []uint64{1}),
				indexedAttestation(4, 7, []uint64{2}),
				indexedAttestation(1, 5, []uint64{3}),
			},
		},
		{
			name: "3 of 4 validators slashed, differing doubles",
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 1, 2, 3},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 2,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 7,
						Root:  []byte("good target"),
					},
				},
				Signature: []byte{1, 2},
			},
			slashableEpochs: []types.Epoch{7, 7, 7, 0},
			// Detections - surround, surround, surround, none.
			shouldSlash: []bool{true, true, true, false},
			// Atts in map: (src, epoch) - 0: (2, 7), 1: (3, 7), 2: (6, 7), 3: (1, 5)
			atts: []*ethpb.IndexedAttestation{
				indexedAttestation(2, 7, []uint64{0}),
				indexedAttestation(3, 7, []uint64{1}),
				indexedAttestation(6, 7, []uint64{2}),
				indexedAttestation(1, 5, []uint64{3}),
			},
		},
		{
			name: "3 of 4 validators slashed, differing surrounds source > target",
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 1, 2, 3},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 7,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 5,
						Root:  []byte("good target"),
					},
				},
				Signature: []byte{1, 2},
			},
			slashableEpochs: []types.Epoch{8, 9, 10, 0},
			// Detections - surround, surround, surround, none.
			shouldSlash: []bool{true, true, true, false},
			// Atts in map: (src, epoch) - 0: (1, 8), 1: (3, 9), 2: (2, 10), 3: (4, 6)
			atts: []*ethpb.IndexedAttestation{
				indexedAttestation(1, 8, []uint64{0}),
				indexedAttestation(3, 9, []uint64{1}),
				indexedAttestation(2, 10, []uint64{2}),
				indexedAttestation(4, 6, []uint64{3}),
			},
		},
		{
			name: "3 of 4 validators slashed, differing surrounded source > target",
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 1, 2, 3},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 9,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 2,
						Root:  []byte("good target"),
					},
				},
				Signature: []byte{1, 2},
			},
			slashableEpochs: []types.Epoch{8, 8, 7, 0},
			// Detections - surround, surround, surround, none.
			shouldSlash: []bool{true, true, true, false},
			// Atts in map: (src, epoch) - 0: (5, 8), 1: (3, 8), 2: (4, 7), 3: (1, 5)
			atts: []*ethpb.IndexedAttestation{
				indexedAttestation(5, 8, []uint64{0}),
				indexedAttestation(3, 8, []uint64{1}),
				indexedAttestation(4, 7, []uint64{2}),
				indexedAttestation(1, 5, []uint64{3}),
			},
		},
		{
			name: "3 of 4 validators slashed, differing surrounded source > target in update",
			incomingAtt: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 1, 2, 3},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 2,
						Root:  []byte("good source"),
					},
					Target: &ethpb.Checkpoint{
						Epoch: 9,
						Root:  []byte("good target"),
					},
				},
				Signature: []byte{1, 2},
			},
			slashableEpochs: []types.Epoch{8, 8, 7, 0},
			// Detections - surround, surround, surround, none.
			shouldSlash: []bool{true, true, true, false},
			// Atts in map: (src, epoch) - 0: (5, 8), 1: (3, 8), 2: (4, 7), 3: (1, 5)
			atts: []*ethpb.IndexedAttestation{
				indexedAttestation(8, 5, []uint64{0}),
				indexedAttestation(8, 3, []uint64{1}),
				indexedAttestation(7, 4, []uint64{2}),
				indexedAttestation(5, 1, []uint64{3}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testDB.SetupSlasherDB(t, false)
			ctx := context.Background()
			defer func() {
				assert.NoError(t, db.ClearDB())
			}()
			defer func() {
				assert.NoError(t, db.Close())
			}()

			spanDetector := &SpanDetector{
				slasherDB: db,
			}
			for _, att := range tt.atts {
				require.NoError(t, spanDetector.UpdateSpans(ctx, att), "Failed to save to slasherDB")
			}
			res, err := spanDetector.DetectSlashingsForAttestation(ctx, tt.incomingAtt)
			require.NoError(t, err)
			var want []*slashertypes.DetectionResult
			for i := 0; i < len(tt.incomingAtt.AttestingIndices); i++ {
				if tt.shouldSlash[i] {
					if tt.slashableEpochs[i] == tt.incomingAtt.Data.Target.Epoch {
						want = append(want, &slashertypes.DetectionResult{
							ValidatorIndex: uint64(i),
							Kind:           slashertypes.DoubleVote,
							SlashableEpoch: tt.slashableEpochs[i],
							SigBytes:       [2]byte{1, 2},
						})
					} else {
						want = append(want, &slashertypes.DetectionResult{
							ValidatorIndex: uint64(i),
							Kind:           slashertypes.SurroundVote,
							SlashableEpoch: tt.slashableEpochs[i],
							SigBytes:       [2]byte{1, 2},
						})
					}
				}
			}
			if !reflect.DeepEqual(want, res) {
				for i, ww := range want {
					t.Errorf("Wanted   %d: %+v\n", i, ww)
				}
				for i, rr := range res {
					t.Errorf("Received %d: %+v\n", i, rr)
				}
				t.Errorf("Wanted: %v, received %v", want, res)
			}
		})
	}
}

func TestNewSpanDetector_UpdateSpans(t *testing.T) {
	type testStruct struct {
		name string
		att  *ethpb.IndexedAttestation
		want []map[uint64]slashertypes.Span
	}
	tests := []testStruct{
		{
			name: "Distance of 2 should update min spans accordingly",
			att: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 1, 2},
				Data: &ethpb.AttestationData{
					CommitteeIndex: 0,
					Source: &ethpb.Checkpoint{
						Epoch: 2,
					},
					Target: &ethpb.Checkpoint{
						Epoch: 4,
					},
				},
				Signature: []byte{1, 2},
			},
			want: []map[uint64]slashertypes.Span{
				// Epoch 0.
				{
					0: {MinSpan: 4, MaxSpan: 0, SigBytes: [2]byte{0, 0}, HasAttested: false},
					1: {MinSpan: 4, MaxSpan: 0, SigBytes: [2]byte{0, 0}, HasAttested: false},
					2: {MinSpan: 4, MaxSpan: 0, SigBytes: [2]byte{0, 0}, HasAttested: false},
				},
				// Epoch 1.
				{
					0: {MinSpan: 3, MaxSpan: 0, SigBytes: [2]byte{0, 0}, HasAttested: false},
					1: {MinSpan: 3, MaxSpan: 0, SigBytes: [2]byte{0, 0}, HasAttested: false},
					2: {MinSpan: 3, MaxSpan: 0, SigBytes: [2]byte{0, 0}, HasAttested: false},
				},
				// Epoch 2.
				{},
				// Epoch 3.
				{
					0: {MinSpan: 0, MaxSpan: 1, SigBytes: [2]byte{0, 0}, HasAttested: false},
					1: {MinSpan: 0, MaxSpan: 1, SigBytes: [2]byte{0, 0}, HasAttested: false},
					2: {MinSpan: 0, MaxSpan: 1, SigBytes: [2]byte{0, 0}, HasAttested: false},
				},
				// Epoch 4.
				{
					0: {MinSpan: 0, MaxSpan: 0, SigBytes: [2]byte{1, 2}, HasAttested: true},
					1: {MinSpan: 0, MaxSpan: 0, SigBytes: [2]byte{1, 2}, HasAttested: true},
					2: {MinSpan: 0, MaxSpan: 0, SigBytes: [2]byte{1, 2}, HasAttested: true},
				},
				{},
				{},
				{},
			},
		},
		{
			name: "Distance of 4 should update max spans accordingly",
			att: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 1, 2},
				Data: &ethpb.AttestationData{
					CommitteeIndex: 1,
					Source: &ethpb.Checkpoint{
						Epoch: 0,
					},
					Target: &ethpb.Checkpoint{
						Epoch: 5,
					},
				},
				Signature: []byte{1, 2},
			},
			want: []map[uint64]slashertypes.Span{
				// Epoch 0.
				{},
				// Epoch 1.
				{
					0: {MinSpan: 0, MaxSpan: 4, SigBytes: [2]byte{0, 0}, HasAttested: false},
					1: {MinSpan: 0, MaxSpan: 4, SigBytes: [2]byte{0, 0}, HasAttested: false},
					2: {MinSpan: 0, MaxSpan: 4, SigBytes: [2]byte{0, 0}, HasAttested: false},
				},
				// Epoch 2.
				{
					0: {MinSpan: 0, MaxSpan: 3, SigBytes: [2]byte{0, 0}, HasAttested: false},
					1: {MinSpan: 0, MaxSpan: 3, SigBytes: [2]byte{0, 0}, HasAttested: false},
					2: {MinSpan: 0, MaxSpan: 3, SigBytes: [2]byte{0, 0}, HasAttested: false},
				},
				// Epoch 3.
				{
					0: {MinSpan: 0, MaxSpan: 2, SigBytes: [2]byte{0, 0}, HasAttested: false},
					1: {MinSpan: 0, MaxSpan: 2, SigBytes: [2]byte{0, 0}, HasAttested: false},
					2: {MinSpan: 0, MaxSpan: 2, SigBytes: [2]byte{0, 0}, HasAttested: false},
				},
				// Epoch 4.
				{
					0: {MinSpan: 0, MaxSpan: 1, SigBytes: [2]byte{0, 0}, HasAttested: false},
					1: {MinSpan: 0, MaxSpan: 1, SigBytes: [2]byte{0, 0}, HasAttested: false},
					2: {MinSpan: 0, MaxSpan: 1, SigBytes: [2]byte{0, 0}, HasAttested: false},
				},
				// Epoch 5.
				{
					0: {MinSpan: 0, MaxSpan: 0, SigBytes: [2]byte{1, 2}, HasAttested: true},
					1: {MinSpan: 0, MaxSpan: 0, SigBytes: [2]byte{1, 2}, HasAttested: true},
					2: {MinSpan: 0, MaxSpan: 0, SigBytes: [2]byte{1, 2}, HasAttested: true},
				},
				{},
				{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testDB.SetupSlasherDB(t, false)
			ctx := context.Background()
			defer func() {
				assert.NoError(t, db.ClearDB())
			}()
			defer func() {
				assert.NoError(t, db.Close())
			}()

			sd := &SpanDetector{
				slasherDB: db,
			}
			require.NoError(t, sd.UpdateSpans(ctx, tt.att))
			for epoch := range tt.want {
				sm, err := sd.slasherDB.EpochSpans(ctx, types.Epoch(epoch), dbtypes.UseDB)
				require.NoError(t, err, "Failed to read from slasherDB")
				resMap, err := sm.ToMap()
				require.NoError(t, err)
				assert.DeepEqual(t, tt.want[epoch], resMap)
			}
		})
	}
}

func TestSpanDetector_UpdateMinSpansCheckCacheSize(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{DisableLookback: true})
	defer resetCfg()

	att := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Data: &ethpb.AttestationData{
			CommitteeIndex: 0,
			Source: &ethpb.Checkpoint{
				Epoch: 150,
			},
			Target: &ethpb.Checkpoint{
				Epoch: 152,
			},
		},
		Signature: []byte{1, 2},
	}

	db := testDB.SetupSlasherDB(t, false)
	ctx := context.Background()
	defer func() {
		assert.NoError(t, db.ClearDB())
	}()
	defer func() {
		assert.NoError(t, db.Close())
	}()

	sd := &SpanDetector{
		slasherDB: db,
	}
	require.NoError(t, sd.updateMinSpan(ctx, att))
	require.Equal(t, int(epochLookback), db.CacheLength(ctx), "Unexpected cache length")
}
