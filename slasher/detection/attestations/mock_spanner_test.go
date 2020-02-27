package attestations

import (
	"context"
	"sync"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/iface"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

var _ = iface.SpanDetector(&MockSpanDetector{})

// MockSpanDetector defines a struct which can detect slashable
// attestation offenses by tracking validator min-max
// spans from validators.
type MockSpanDetector struct {
	// Slice of epochs for valindex => min-max span.
	spans []map[uint64][2]uint16
	lock  sync.RWMutex
}

// DetectSlashingForValidator uses a validator index and its corresponding
// min-max spans during an epoch to detect an epoch in which the validator
// committed a slashable attestation.
func (s *MockSpanDetector) DetectSlashingForValidator(
	ctx context.Context,
	validatorIdx uint64,
	attData *ethpb.AttestationData,
) (*types.DetectionResult, error) {
	if attData.Target.Epoch == 0 {
		return nil, nil
	} else if attData.Target.Epoch > 5 {
		return &types.DetectionResult{
			Kind:           types.SurroundVote,
			SlashableEpoch: attData.Target.Epoch,
		}, nil
	} else {
		return &types.DetectionResult{
			Kind:           types.DoubleVote,
			SlashableEpoch: attData.Target.Epoch,
		}, nil
	}
}

// SpanForEpochByValidator returns the specific min-max span for a
// validator index in a given epoch.
func (s *MockSpanDetector) SpanForEpochByValidator(ctx context.Context, valIdx uint64, epoch uint64) ([3]uint16, error) {
	return [3]uint16{0, 0, 0}, nil
}

// ValidatorSpansByEpoch returns a list of all validator spans in a given epoch.
func (s *MockSpanDetector) ValidatorSpansByEpoch(ctx context.Context, epoch uint64) map[uint64][3]uint16 {
	return make(map[uint64][3]uint16, 0)
}

// DeleteValidatorSpansByEpoch mocks the delete spans by epoch function.
func (s *MockSpanDetector) DeleteValidatorSpansByEpoch(ctx context.Context, validatorIdx uint64, epoch uint64) error {
	return nil
}

// UpdateSpans is a mock for updating the spans for a given attestation..
func (s *MockSpanDetector) UpdateSpans(ctx context.Context, att *ethpb.IndexedAttestation) error {
	return nil
}
