package attestations

import (
	"context"
	"fmt"
	"sync"

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
	spans []map[uint64][2]uint16
	lock  sync.RWMutex
}

// NewSpanDetector creates a new instance of a struct tracking
// several epochs of min-max spans for each validator in
// the beacon state.
func NewSpanDetector() *SpanDetector {
	return &SpanDetector{
		spans: make([]map[uint64][2]uint16, 256),
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
	numSpans := uint64(len(s.spans))
	if sp := s.spans[sourceEpoch%numSpans]; sp != nil {
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

// SpanForEpochByValidator returns the specific min-max span for a
// validator index in a given epoch.
func (s *SpanDetector) SpanForEpochByValidator(ctx context.Context, valIdx uint64, epoch uint64) ([2]uint16, error) {
	ctx, span := trace.StartSpan(ctx, "detection.SpanForEpochByValidator")
	defer span.End()
	s.lock.RLock()
	defer s.lock.RUnlock()
	numSpans := uint64(len(s.spans))
	if span := s.spans[epoch%numSpans]; span != nil {
		if minMaxSpan, ok := span[valIdx]; ok {
			return minMaxSpan, nil
		}
		return [2]uint16{}, fmt.Errorf("validator index %d not found in span map", valIdx)
	}
	return [2]uint16{}, fmt.Errorf("no data found for epoch %d", epoch)
}

// ValidatorSpansByEpoch returns a list of all validator spans in a given epoch.
func (s *SpanDetector) ValidatorSpansByEpoch(ctx context.Context, epoch uint64) map[uint64][2]uint16 {
	ctx, span := trace.StartSpan(ctx, "detection.ValidatorSpansByEpoch")
	defer span.End()
	s.lock.RLock()
	defer s.lock.RUnlock()
	numSpans := uint64(len(s.spans))
	return s.spans[epoch%numSpans]
}

// DeleteValidatorSpansByEpoch deletes a min-max span for a validator
// index from a min-max span in a given epoch.
func (s *SpanDetector) DeleteValidatorSpansByEpoch(ctx context.Context, validatorIdx uint64, epoch uint64) error {
	ctx, span := trace.StartSpan(ctx, "detection.DeleteValidatorSpansByEpoch")
	defer span.End()
	s.lock.Lock()
	defer s.lock.Unlock()
	numSpans := uint64(len(s.spans))
	if val := s.spans[epoch%numSpans]; val != nil {
		delete(val, validatorIdx)
		return nil
	}
	return fmt.Errorf("no span map found at epoch %d", epoch)
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
		s.updateMinSpan(source, target, valIdx)
		s.updateMaxSpan(source, target, valIdx)
	}
	return nil
}

// Updates a min span for a validator index given a source and target epoch
// for an attestation produced by the validator. Used for catching surrounding votes.
func (s *SpanDetector) updateMinSpan(source uint64, target uint64, valIdx uint64) {
	numSpans := uint64(len(s.spans))
	if source < 1 {
		return
	}
	for epoch := source - 1; epoch >= 0; epoch-- {
		newMinSpan := uint16(target - epoch)
		if sp := s.spans[epoch%numSpans]; sp == nil {
			s.spans[epoch%numSpans] = make(map[uint64][2]uint16)
		}
		minSpan := s.spans[epoch%numSpans][valIdx][0]
		maxSpan := s.spans[epoch%numSpans][valIdx][1]
		if minSpan == 0 || minSpan > newMinSpan {
			s.spans[epoch%numSpans][valIdx] = [2]uint16{newMinSpan, maxSpan}
		} else {
			break
		}
	}
}

// Updates a max span for a validator index given a source and target epoch
// for an attestation produced by the validator. Used for catching surrounded votes.
func (s *SpanDetector) updateMaxSpan(source uint64, target uint64, valIdx uint64) {
	numSpans := uint64(len(s.spans))
	for epoch := source + 1; epoch < target; epoch++ {
		if sp := s.spans[epoch%numSpans]; sp == nil {
			s.spans[epoch%numSpans] = make(map[uint64][2]uint16)
		}
		minSpan := s.spans[epoch%numSpans][valIdx][0]
		maxSpan := s.spans[epoch%numSpans][valIdx][1]
		newMaxSpan := uint16(target - epoch)
		if newMaxSpan > maxSpan {
			s.spans[epoch%numSpans][valIdx] = [2]uint16{minSpan, newMaxSpan}
		} else {
			break
		}
	}
}
