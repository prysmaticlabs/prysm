package attestations

import (
	"context"
	"reflect"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

func TestSpanDetector_DetectSlashingForValidator_Double(t *testing.T) {
	type testStruct struct {
		name        string
		att         *ethpb.IndexedAttestation
		incomingAtt *ethpb.IndexedAttestation
		slashCount  uint64
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
			name: "att with different target, should not slash",
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
			name: "same att with different aggregates, should not slash",
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
			slashCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			numEpochsToTrack := 100
			sd := &SpanDetector{
				spans: make([]map[uint64][3]uint16, numEpochsToTrack),
			}

			ctx := context.Background()
			if err := sd.UpdateSpans(ctx, tt.att); err != nil {
				t.Fatal(err)
			}
			if err := sd.UpdateSpans(ctx, tt.incomingAtt); err != nil {
				t.Fatal(err)
			}

			slashTotal := uint64(0)
			for _, valIdx := range sliceutil.IntersectionUint64(tt.att.AttestingIndices, tt.incomingAtt.AttestingIndices) {
				res, err := sd.DetectSlashingForValidator(ctx, valIdx, tt.incomingAtt.Data)
				if err != nil {
					t.Fatal(err)
				}
				var want *types.DetectionResult
				if tt.slashCount > 0 {
					slashTotal++
					want = &types.DetectionResult{
						Kind:           types.DoubleVote,
						SlashableEpoch: tt.incomingAtt.Data.Target.Epoch,
					}
				}
				if !reflect.DeepEqual(res, want) {
					t.Errorf("Wanted: %v, received %v", want, res)
				}
			}
			if slashTotal != tt.slashCount {
				t.Fatalf("Unexpected amount of slashings found, received %d, expected %d", slashTotal, tt.slashCount)
			}
		})
	}
}

func TestSpanDetector_DetectSlashingForValidator_Surround(t *testing.T) {
	type testStruct struct {
		name                     string
		sourceEpoch              uint64
		targetEpoch              uint64
		slashableEpoch           uint64
		shouldSlash              bool
		spansByEpochForValidator map[uint64][3]uint16
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
			spansByEpochForValidator: map[uint64][3]uint16{
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
			spansByEpochForValidator: map[uint64][3]uint16{
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
			spansByEpochForValidator: map[uint64][3]uint16{
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
			spansByEpochForValidator: map[uint64][3]uint16{
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
			spansByEpochForValidator: map[uint64][3]uint16{
				3: {1, 0},
			},
		},
		// Proto Max Span Tests from the eth2-surround repo.
		{
			name:        "Proto max span test #1",
			sourceEpoch: 8,
			targetEpoch: 18,
			shouldSlash: false,
			spansByEpochForValidator: map[uint64][3]uint16{
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
			spansByEpochForValidator: map[uint64][3]uint16{
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
			spansByEpochForValidator: map[uint64][3]uint16{
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
			spansByEpochForValidator: map[uint64][3]uint16{
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
			spansByEpochForValidator: map[uint64][3]uint16{
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
			spansByEpochForValidator: map[uint64][3]uint16{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			numEpochsToTrack := 100
			sd := &SpanDetector{
				spans: make([]map[uint64][3]uint16, numEpochsToTrack),
			}
			// We only care about validator index 0 for these tests for simplicity.
			validatorIndex := uint64(0)
			for k, v := range tt.spansByEpochForValidator {
				sd.spans[k] = map[uint64][3]uint16{
					validatorIndex: v,
				}
			}
			ctx := context.Background()
			attData := &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{
					Epoch: tt.sourceEpoch,
				},
				Target: &ethpb.Checkpoint{
					Epoch: tt.targetEpoch,
				},
			}
			res, err := sd.DetectSlashingForValidator(ctx, validatorIndex, attData)
			if err != nil {
				t.Fatal(err)
			}
			if !tt.shouldSlash && res != nil {
				t.Fatalf("Did not want validator to be slashed but found slashable offense: %v", res)
			}
			if tt.shouldSlash {
				want := &types.DetectionResult{
					Kind:           types.SurroundVote,
					SlashableEpoch: tt.slashableEpoch,
				}
				if !reflect.DeepEqual(res, want) {
					t.Errorf("Wanted: %v, received %v", want, res)
				}
			}
		})
	}
}

func TestSpanDetector_DetectSlashingForValidator_MultipleValidators(t *testing.T) {
	type testStruct struct {
		name            string
		sourceEpochs    []uint64
		targetEpochs    []uint64
		slashableEpochs []uint64
		shouldSlash     []bool
		spansByEpoch    []map[uint64][3]uint16
	}
	tests := []testStruct{
		{
			name:            "3 of 5 validators slashed",
			sourceEpochs:    []uint64{0, 2, 4, 5, 1},
			targetEpochs:    []uint64{10, 3, 5, 9, 8},
			slashableEpochs: []uint64{6, 0, 7, 8, 0},
			// Detections - surrounding, none, surrounded, surrounding, none.
			shouldSlash: []bool{true, false, true, true, false},
			// Atts in map: (src, epoch) - 0: (2, 6), 1: (1, 2), 2: (1, 7), 3: (6, 8), 4: (0, 3)
			spansByEpoch: []map[uint64][3]uint16{
				// Epoch 0.
				{
					0: {6, 0},
					1: {2, 0},
					2: {7, 0},
					3: {8, 0},
				},
				// Epoch 1.
				{
					0: {5, 0},
					3: {7, 0},
					4: {0, 1},
				},
				// Epoch 2.
				{
					2: {0, 5},
					3: {6, 0},
					4: {0, 2},
				},
				// Epoch 3.
				{
					0: {0, 3},
					2: {0, 4},
					3: {5, 0},
				},
				// Epoch 4.
				{
					0: {0, 2},
					2: {0, 3},
					3: {4, 0},
				},
				// Epoch 5.
				{
					0: {0, 1},
					2: {0, 2},
					3: {3, 0},
				},
				// Epoch 6.
				{
					2: {0, 1},
				},
				// Epoch 7.
				{
					3: {0, 1},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			numEpochsToTrack := 100
			sd := &SpanDetector{
				spans: make([]map[uint64][3]uint16, numEpochsToTrack),
			}
			for i := 0; i < len(tt.spansByEpoch); i++ {
				sd.spans[i] = tt.spansByEpoch[i]
			}
			ctx := context.Background()
			for valIdx := uint64(0); valIdx < uint64(len(tt.shouldSlash)); valIdx++ {
				attData := &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: tt.sourceEpochs[valIdx],
					},
					Target: &ethpb.Checkpoint{
						Epoch: tt.targetEpochs[valIdx],
					},
				}
				res, err := sd.DetectSlashingForValidator(ctx, valIdx, attData)
				if err != nil {
					t.Fatal(err)
				}
				if !tt.shouldSlash[valIdx] && res != nil {
					t.Fatalf("Did not want validator to be slashed but found slashable offense: %v", res)
				}
				if tt.shouldSlash[valIdx] {
					want := &types.DetectionResult{
						Kind:           types.SurroundVote,
						SlashableEpoch: tt.slashableEpochs[valIdx],
					}
					if !reflect.DeepEqual(res, want) {
						t.Errorf("Wanted: %v, received %v", want, res)
					}
				}
			}
		})
	}
}

func TestSpanDetector_SpanForEpochByValidator(t *testing.T) {
	numEpochsToTrack := 2
	sd := &SpanDetector{
		spans: make([]map[uint64][3]uint16, numEpochsToTrack),
	}
	epoch := uint64(1)
	validatorIndex := uint64(40)
	sd.spans[epoch] = map[uint64][3]uint16{
		validatorIndex: {3, 7},
	}
	want := [3]uint16{3, 7}
	ctx := context.Background()
	res, err := sd.SpanForEpochByValidator(ctx, validatorIndex, epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, res) {
		t.Errorf("Wanted %v, received %v", want, res)
	}
	validatorIndex = uint64(0)
	if _, err = sd.SpanForEpochByValidator(
		ctx,
		validatorIndex,
		epoch,
	); err != nil && !strings.Contains(err.Error(), "validator index 0 not found") {
		t.Errorf("Wanted validator index not found error, received %v", err)
	}
	validatorIndex = uint64(40)
	epoch = uint64(3)
	if _, err = sd.SpanForEpochByValidator(
		ctx,
		validatorIndex,
		epoch,
	); err != nil && !strings.Contains(err.Error(), "no data found for epoch") {
		t.Errorf("Wanted no data found for epoch error, received %v", err)
	}
}

func TestSpanDetector_ValidatorSpansByEpoch(t *testing.T) {
	numEpochsToTrack := 2
	sd := &SpanDetector{
		spans: make([]map[uint64][3]uint16, numEpochsToTrack),
	}
	epoch := uint64(1)
	validatorIndex := uint64(40)
	want := map[uint64][3]uint16{
		validatorIndex: {3, 7},
	}
	sd.spans[epoch] = want
	res := sd.ValidatorSpansByEpoch(context.Background(), epoch)
	if !reflect.DeepEqual(res, want) {
		t.Errorf("Wanted %v, received %v", want, res)
	}
}

func TestSpanDetector_DeleteValidatorSpansByEpoch(t *testing.T) {
	numEpochsToTrack := 2
	sd := &SpanDetector{
		spans: make([]map[uint64][3]uint16, numEpochsToTrack),
	}
	epoch := uint64(1)
	validatorIndex := uint64(40)
	sd.spans[epoch] = map[uint64][3]uint16{
		validatorIndex: {3, 7},
	}
	ctx := context.Background()
	if err := sd.DeleteValidatorSpansByEpoch(
		ctx,
		validatorIndex,
		0, /* epoch */
	); err != nil && !strings.Contains(err.Error(), "no span map found at epoch 0") {
		t.Errorf("Wanted error when deleting epoch 0, received: %v", err)
	}
	if err := sd.DeleteValidatorSpansByEpoch(ctx, validatorIndex, epoch); err != nil {
		t.Fatal(err)
	}
	want := make(map[uint64][3]uint16)
	if res := sd.ValidatorSpansByEpoch(ctx, epoch); !reflect.DeepEqual(res, want) {
		t.Errorf("Wanted %v for epoch after deleting, received %v", want, res)
	}
}

func TestNewSpanDetector_UpdateSpans(t *testing.T) {
	type testStruct struct {
		name string
		att  *ethpb.IndexedAttestation
		want []map[uint64][3]uint16
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
			},
			want: []map[uint64][3]uint16{
				// Epoch 0.
				{
					0: {4, 0, 0},
					1: {4, 0, 0},
					2: {4, 0, 0},
				},
				// Epoch 1.
				{
					0: {3, 0, 0},
					1: {3, 0, 0},
					2: {3, 0, 0},
				},
				// Epoch 2.
				nil,
				// Epoch 3.
				{
					0: {0, 1, 0},
					1: {0, 1, 0},
					2: {0, 1, 0},
				},
				// Epoch 4.
				{
					0: {0, 0, 4188},
					1: {0, 0, 4188},
					2: {0, 0, 4188},
				},
				nil,
				nil,
				nil,
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
			},
			want: []map[uint64][3]uint16{
				// Epoch 0.
				nil,
				// Epoch 1.
				{
					0: {0, 4, 0},
					1: {0, 4, 0},
					2: {0, 4, 0},
				},
				// Epoch 2.
				{
					0: {0, 3, 0},
					1: {0, 3, 0},
					2: {0, 3, 0},
				},
				// Epoch 3.
				{
					0: {0, 2, 0},
					1: {0, 2, 0},
					2: {0, 2, 0},
				},
				// Epoch 4.
				{
					0: {0, 1, 0},
					1: {0, 1, 0},
					2: {0, 1, 0},
				},
				// Epoch 5.
				{
					0: {0, 0, 2074},
					1: {0, 0, 2074},
					2: {0, 0, 2074},
				},
				nil,
				nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &SpanDetector{
				spans: make([]map[uint64][3]uint16, 8),
			}
			ctx := context.Background()
			if err := sd.UpdateSpans(ctx, tt.att); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(sd.spans, tt.want) {
				t.Errorf("Wanted and received:\n%v \n%v", tt.want, sd.spans)
			}
		})
	}
}
