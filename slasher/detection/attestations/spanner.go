package attestations

import (
	"context"
	"fmt"
	"sync"

	"github.com/prysmaticlabs/prysm/slasher/db"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// DetectionKind defines an enum type that
// gives us information on the type of slashable offense
// found when analyzing validator min-max spans.
type DetectionKind int

const (
	// DoubleVote denotes a slashable offense in which
	// a validator cast two conflicting attestations within
	// the same target epoch.
	DoubleVote DetectionKind = iota
	// SurroundVote denotes a slashable offense in which
	// a validator surrounded or was surrounded by a previous
	// attestation created by the same validator.
	SurroundVote
)

// DetectionResult tells us the kind of slashable
// offense found from detecting on min-max spans +
// the slashable epoch for the offense.
type DetectionResult struct {
	Kind           DetectionKind
	SlashableEpoch uint64
}

// SpanDetector defines a struct which can detect slashable
// attestation offenses by tracking validator min-max
// spans from validators.
type SpanDetector struct {
	// Slice of epochs for valindex => min-max span.
	db   db.Database
	lock sync.RWMutex
}

// NewSpanDetector creates a new instance of a struct tracking
// several epochs of min-max spans for each validator in
// the beacon state.
func NewSpanDetector(db db.Database) *SpanDetector {
	return &SpanDetector{
		db: db,
	}
}

// DetectSlashingForValidator uses a validator index and its corresponding
// min-max spans during an epoch to detect an epoch in which the validator
// committed a slashable attestation.
func (s *SpanDetector) DetectSlashingForValidator(
	ctx context.Context,
	validatorIdx uint64,
	attData *ethpb.AttestationData,
) (*DetectionResult, error) {
	ctx, span := trace.StartSpan(ctx, "detection.DetectSlashingForValidator")
	defer span.End()
	sourceEpoch := attData.Source.Epoch
	targetEpoch := attData.Target.Epoch
	if (targetEpoch - sourceEpoch) > params.BeaconConfig().WeakSubjectivityPeriod {
		return nil, fmt.Errorf(
			"attestation span was greater than weak subjectivity period %d, received: %d",
			params.BeaconConfig().WeakSubjectivityPeriod,
			targetEpoch-sourceEpoch,
		)
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	distance := uint16(targetEpoch - sourceEpoch)
	sp, err := s.db.EpochSpansMap(ctx, sourceEpoch)
	if err != nil {
		return nil, err
	}
	if sp != nil {
		minSpan := sp[validatorIdx][0]
		if minSpan > 0 && minSpan < distance {
			return &DetectionResult{
				Kind:           SurroundVote,
				SlashableEpoch: sourceEpoch + uint64(minSpan),
			}, nil
		}

		maxSpan := sp[validatorIdx][1]
		if maxSpan > distance {
			return &DetectionResult{
				Kind:           SurroundVote,
				SlashableEpoch: sourceEpoch + uint64(maxSpan),
			}, nil
		}
	}
	return nil, nil
}

// UpdateSpans given an indexed attestation for all of its attesting indices.
func (s *SpanDetector) UpdateSpans(ctx context.Context, att *ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "detection.UpdateSpans")
	defer span.End()
	s.lock.Lock()
	defer s.lock.Unlock()
	source := att.Data.Source.Epoch
	target := att.Data.Target.Epoch
	// Update spansForEpoch[valIdx] using the source/target data for
	// each validator in attesting indices.
	for i := 0; i < len(att.AttestingIndices); i++ {
		valIdx := att.AttestingIndices[i]
		// Update min and max spans.
		s.updateMinSpan(ctx, source, target, valIdx)
		s.updateMaxSpan(ctx, source, target, valIdx)
	}
	return nil
}

// Updates a min span for a validator index given a source and target epoch
// for an attestation produced by the validator. Used for catching surrounding votes.
func (s *SpanDetector) updateMinSpan(ctx context.Context, source uint64, target uint64, valIdx uint64) error {
	if source < 1 {
		return nil
	}
	for epoch := source - 1; epoch >= 0; epoch-- {
		newMinSpan := uint16(target - epoch)
		sp, err := s.db.EpochSpanByValidatorIndex(ctx, valIdx, epoch)
		if err != nil {
			return err
		}
		minSpan := sp[0]
		maxSpan := sp[1]
		if minSpan == 0 || minSpan > newMinSpan {
			if err := s.db.SaveValidatorEpochSpans(ctx, valIdx, epoch, [2]uint16{newMinSpan, maxSpan}); err != nil {
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
		sp, err := s.db.EpochSpanByValidatorIndex(ctx, valIdx, epoch)
		if err != nil {
			return err
		}
		minSpan := sp[0]
		maxSpan := sp[1]
		newMaxSpan := uint16(target - epoch)
		if newMaxSpan > maxSpan {
			if err := s.db.SaveValidatorEpochSpans(ctx, valIdx, epoch, [2]uint16{minSpan, newMaxSpan}); err != nil {
				return err
			}
		} else {
			break
		}
	}
	return nil
}
