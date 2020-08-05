package attestations

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/iface"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

var _ = iface.SpanDetector(&MockSpanDetector{})

// MockSpanDetector defines a struct which can detect slashable
// attestation offenses by tracking validator min-max
// spans from validators.
type MockSpanDetector struct{}

// DetectSlashingsForAttestation mocks a detected slashing, if the sent attestation data
// has a source epoch of 0, nothing will be detected. If the sent attestation data has a target
// epoch equal to or greater than 6, it will "detect" a surrounded vote for the target epoch + 1.
// If the target epoch is greater than 12, it will "detect" a surrounding vote for target epoch - 1.
// Lastly, if it has a target epoch less than 6, it will "detect" a double vote for the target epoch.
func (s *MockSpanDetector) DetectSlashingsForAttestation(
	ctx context.Context,
	att *ethpb.IndexedAttestation,
) ([]*types.DetectionResult, error) {
	var detections []*types.DetectionResult
	switch {
	// If the source epoch is 0, don't find a slashing.
	case att.Data.Source.Epoch == 0:
		return nil, nil
	// If the target epoch is > 12, it will "detect" a surrounded saved attestation.
	case att.Data.Target.Epoch > 12:
		detections = append(detections, &types.DetectionResult{
			Kind:           types.SurroundVote,
			SlashableEpoch: att.Data.Target.Epoch - 1,
			SigBytes:       [2]byte{1, 2},
		})
		return detections, nil
	// If the target epoch is >= 6 < 12, it will "detect" a surrounding saved attestation.
	case att.Data.Target.Epoch >= 6:
		detections = append(detections, &types.DetectionResult{
			Kind:           types.SurroundVote,
			SlashableEpoch: att.Data.Target.Epoch + 1,
			SigBytes:       [2]byte{1, 2},
		})
		return detections, nil
	// If the target epoch is less than 6, it will "detect" a double vote.
	default:
		detections = append(detections, &types.DetectionResult{
			Kind:           types.DoubleVote,
			SlashableEpoch: att.Data.Target.Epoch,
			SigBytes:       [2]byte{1, 2},
		})
	}
	return detections, nil
}

// SpanForEpochByValidator returns the specific min-max span for a
// validator index in a given epoch.
func (s *MockSpanDetector) SpanForEpochByValidator(ctx context.Context, valIdx uint64, epoch uint64) (types.Span, error) {
	return types.Span{MinSpan: 0, MaxSpan: 0, SigBytes: [2]byte{}, HasAttested: false}, nil
}

// ValidatorSpansByEpoch returns a list of all validator spans in a given epoch.
func (s *MockSpanDetector) ValidatorSpansByEpoch(ctx context.Context, epoch uint64) map[uint64]types.Span {
	return make(map[uint64]types.Span, 0)
}

// DeleteValidatorSpansByEpoch mocks the delete spans by epoch function.
func (s *MockSpanDetector) DeleteValidatorSpansByEpoch(ctx context.Context, validatorIdx uint64, epoch uint64) error {
	return nil
}

// UpdateSpans is a mock for updating the spans for a given attestation..
func (s *MockSpanDetector) UpdateSpans(ctx context.Context, att *ethpb.IndexedAttestation) error {
	return nil
}
