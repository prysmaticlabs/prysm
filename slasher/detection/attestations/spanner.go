package attestations

import (
	"context"
	"fmt"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	db "github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/iface"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"go.opencensus.io/trace"
)

var _ = iface.SpanDetector(&SpanDetector{})

// SpanDetector defines a struct which can detect slashable
// attestation offenses by tracking validator min-max
// spans from validators and attestation data roots.
type SpanDetector struct {
	slasherDB db.Database
}

// NewSpanDetector creates a new instance of a struct tracking
// several epochs of min-max spans for each validator in
// the beacon state.
func NewSpanDetector(db db.Database) *SpanDetector {
	return &SpanDetector{
		slasherDB: db,
	}
}

// DetectSlashingForValidator uses a validator index and its corresponding
// min-max spans during an epoch to detect an epoch in which the validator
// committed a slashable attestation.
func (s *SpanDetector) DetectSlashingForValidator(
	ctx context.Context,
	validatorIdx uint64,
	attData *ethpb.AttestationData,
) (*types.DetectionResult, error) {
	ctx, traceSpan := trace.StartSpan(ctx, "detection.DetectSlashingForValidator")
	defer traceSpan.End()
	sourceEpoch := attData.Source.Epoch
	targetEpoch := attData.Target.Epoch
	if (targetEpoch - sourceEpoch) > params.BeaconConfig().WeakSubjectivityPeriod {
		return nil, fmt.Errorf(
			"attestation span was greater than weak subjectivity period %d, received: %d",
			params.BeaconConfig().WeakSubjectivityPeriod,
			targetEpoch-sourceEpoch,
		)
	}
	distance := uint16(targetEpoch - sourceEpoch)
	sp, err := s.slasherDB.EpochSpanByValidatorIndex(ctx, validatorIdx, sourceEpoch)
	if err != nil {
		return nil, err
	}

	minSpan := sp.MinSpan
	if minSpan > 0 && minSpan < distance {
		slashableEpoch := sourceEpoch + uint64(minSpan)
		span, err := s.slasherDB.EpochSpanByValidatorIndex(ctx, validatorIdx, slashableEpoch)
		if err != nil {
			return nil, err
		}
		return &types.DetectionResult{
			Kind:           types.SurroundVote,
			SlashableEpoch: sourceEpoch + uint64(minSpan),
			SigBytes:       span.SigBytes,
		}, nil
	}

	maxSpan := sp.MaxSpan
	if maxSpan > distance {
		slashableEpoch := sourceEpoch + uint64(maxSpan)
		span, err := s.slasherDB.EpochSpanByValidatorIndex(ctx, validatorIdx, slashableEpoch)
		if err != nil {
			return nil, err
		}
		return &types.DetectionResult{
			Kind:           types.SurroundVote,
			SlashableEpoch: slashableEpoch,
			SigBytes:       span.SigBytes,
		}, nil
	}
	sp, err = s.slasherDB.EpochSpanByValidatorIndex(ctx, validatorIdx, targetEpoch)
	if err != nil {
		return nil, err
	}

	// Check if the validator has attested for this epoch or not.
	if sp.HasAttested {
		return &types.DetectionResult{
			Kind:           types.DoubleVote,
			SlashableEpoch: targetEpoch,
			SigBytes:       sp.SigBytes,
		}, nil
	}

	return nil, nil
}

// UpdateSpans given an indexed attestation for all of its attesting indices.
func (s *SpanDetector) UpdateSpans(ctx context.Context, att *ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "detection.UpdateSpans")
	defer span.End()
	source := att.Data.Source.Epoch
	target := att.Data.Target.Epoch
	for i := 0; i < len(att.AttestingIndices); i++ {
		valIdx := att.AttestingIndices[i]
		// Save the signature for the received attestation so we can have more detail to find it in the DB.
		err := s.saveSigBytes(ctx, att, valIdx)
		if err != nil {
			return err
		}
		// Update min and max spans.
		err = s.updateMinSpan(ctx, source, target, valIdx)
		if err != nil {
			return err
		}
		err = s.updateMaxSpan(ctx, source, target, valIdx)
		if err != nil {
			return err
		}
	}
	return nil
}

// saveSigBytes saves the first 2 bytes of the signature for the att we're updating the spans to.
// Later used to help us find the violating attestation in the DB.
func (s *SpanDetector) saveSigBytes(ctx context.Context, att *ethpb.IndexedAttestation, valIdx uint64) error {
	target := att.Data.Target.Epoch
	sp, err := s.slasherDB.EpochSpanByValidatorIndex(ctx, valIdx, target)
	if err != nil {
		return err
	}

	// If the validator has already attested for this target epoch,
	// then we do not need to update the values of the span sig bytes.
	if sp.HasAttested {
		return nil
	}

	sigBytes := [2]byte{0, 0}
	if len(att.Signature) > 1 {
		sigBytes = [2]byte{att.Signature[0], att.Signature[1]}
	}
	// Save the signature bytes into the span for this epoch.
	sp.HasAttested = true
	sp.SigBytes = sigBytes
	return s.slasherDB.SaveValidatorEpochSpans(ctx, valIdx, target, sp)
}

// Updates a min span for a validator index given a source and target epoch
// for an attestation produced by the validator. Used for catching surrounding votes.
func (s *SpanDetector) updateMinSpan(ctx context.Context, source uint64, target uint64, valIdx uint64) error {
	if source < 1 {
		return nil
	}
	for epochInt := int64(source - 1); epochInt >= 0; epochInt-- {
		epoch := uint64(epochInt)
		span, err := s.slasherDB.EpochSpanByValidatorIndex(ctx, valIdx, epoch)
		if err != nil {
			return err
		}
		newMinSpan := uint16(target - epoch)
		if span.MinSpan == 0 || span.MinSpan > newMinSpan {
			span = types.Span{
				MinSpan:     newMinSpan,
				MaxSpan:     span.MaxSpan,
				SigBytes:    span.SigBytes,
				HasAttested: span.HasAttested,
			}
			if err := s.slasherDB.SaveValidatorEpochSpans(ctx, valIdx, epoch, span); err != nil {
				return err
			}
		} else {
			break
		}
	}
	return nil
}

// Updates a max span for a validator index given a source and target epoch
// for an attestation produced by the validator. Used for catching surrounded votes.
func (s *SpanDetector) updateMaxSpan(ctx context.Context, source uint64, target uint64, valIdx uint64) error {
	for epoch := source + 1; epoch < target; epoch++ {
		span, err := s.slasherDB.EpochSpanByValidatorIndex(ctx, valIdx, epoch)
		if err != nil {
			return err
		}
		newMaxSpan := uint16(target - epoch)
		if newMaxSpan > span.MaxSpan {
			span = types.Span{
				MinSpan:     span.MinSpan,
				MaxSpan:     newMaxSpan,
				SigBytes:    span.SigBytes,
				HasAttested: span.HasAttested,
			}
			if err := s.slasherDB.SaveValidatorEpochSpans(ctx, valIdx, epoch, span); err != nil {
				return err
			}
		} else {
			break
		}
	}
	return nil
}
