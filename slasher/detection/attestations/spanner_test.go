package attestations

import (
	"context"
	"reflect"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

func TestSpanDetector_DetectSlashingForValidator(t *testing.T) {
	type testStruct struct {
		name                     string
		sourceEpoch              uint64
		targetEpoch              uint64
		slashableEpoch           uint64
		shouldSlash              bool
		spansByEpochForValidator map[uint64][2]uint16
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
			spansByEpochForValidator: map[uint64][2]uint16{
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
			spansByEpochForValidator: map[uint64][2]uint16{
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
			spansByEpochForValidator: map[uint64][2]uint16{
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
			spansByEpochForValidator: map[uint64][2]uint16{
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
			spansByEpochForValidator: map[uint64][2]uint16{
				3: {1, 0},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			numEpochsToTrack := 100
			sd := &SpanDetector{
				spans: make([]map[uint64][2]uint16, numEpochsToTrack),
			}
			// We only care about validator index 0 for these tests for simplicity.
			validatorIndex := uint64(0)
			for k, v := range tt.spansByEpochForValidator {
				sd.spans[k] = map[uint64][2]uint16{
					validatorIndex: v,
				}
			}
			ctx := context.Background()
			res, err := sd.DetectSlashingForValidator(ctx, validatorIndex, tt.sourceEpoch, tt.targetEpoch)
			if err != nil {
				t.Fatal(err)
			}
			if !tt.shouldSlash && res != nil {
				t.Fatalf("Did not want validator to be slashed but found slashable offense: %v", res)
			}
			if tt.shouldSlash {
				want := &DetectionResult{
					Kind:           SurroundVote,
					SlashableEpoch: tt.slashableEpoch,
				}
				if !reflect.DeepEqual(res, want) {
					t.Errorf("Wanted: %v, received %v", want, res)
				}
			}
		})
	}
}

func TestSpanDetector_SpansForValidatorByEpoch(t *testing.T) {
	numEpochsToTrack := 2
	sd := &SpanDetector{
		spans: make([]map[uint64][2]uint16, numEpochsToTrack),
	}
	epoch := uint64(1)
	validatorIndex := uint64(40)
	sd.spans[epoch] = map[uint64][2]uint16{
		validatorIndex: {3, 7},
	}
	want := [2]uint16{3, 7}
	ctx := context.Background()
	res, err := sd.SpansForValidatorByEpoch(ctx, validatorIndex, epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, res) {
		t.Errorf("Wanted %v, received %v", want, res)
	}
	validatorIndex = uint64(0)
	if _, err = sd.SpansForValidatorByEpoch(
		ctx,
		validatorIndex,
		epoch,
	); err != nil && !strings.Contains(err.Error(), "validator index 0 not found") {
		t.Errorf("Wanted validator index not found error, received %v", err)
	}
	validatorIndex = uint64(40)
	epoch = uint64(3)
	if _, err = sd.SpansForValidatorByEpoch(
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
		spans: make([]map[uint64][2]uint16, numEpochsToTrack),
	}
	epoch := uint64(1)
	validatorIndex := uint64(40)
	want := map[uint64][2]uint16{
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
		spans: make([]map[uint64][2]uint16, numEpochsToTrack),
	}
	epoch := uint64(1)
	validatorIndex := uint64(40)
	sd.spans[epoch] = map[uint64][2]uint16{
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
	want := make(map[uint64][2]uint16)
	if res := sd.ValidatorSpansByEpoch(ctx, epoch); !reflect.DeepEqual(res, want) {
		t.Errorf("Wanted %v for epoch after deleting, received %v", want, res)
	}
}

func TestNewSpanDetector_UpdateSpans(t *testing.T) {
	type testStruct struct {
		name      string
		att       *ethpb.IndexedAttestation
		numEpochs uint64
		want      []map[uint64][2]uint16
	}
	tests := []testStruct{
		{
			att: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 1, 2},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 0,
					},
					Target: &ethpb.Checkpoint{
						Epoch: 0,
					},
				},
			},
			numEpochs: 2,
			want: []map[uint64][2]uint16{
				// Epoch 0.
				{
					0: {1, 0},
					1: {1, 0},
					2: {1, 0},
				},
				// Epoch 1.
				{
					0: {1, 0},
					1: {1, 0},
					2: {1, 0},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &SpanDetector{
				spans: make([]map[uint64][2]uint16, tt.numEpochs),
			}
			ctx := context.Background()
			if err := sd.UpdateSpans(ctx, tt.att); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(sd.spans, tt.want) {
				t.Errorf("Wanted spans %v, received %v", tt.want, sd.spans)
			}
		})
	}
}
