package attestations

import (
	"context"
	"fmt"
	"sync"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	db2 "github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/iface"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"go.opencensus.io/trace"
)

var _ = iface.SpanDetector(&SpanDetector{})

// SpanDetector defines a struct which can detect slashable
// attestation offenses by tracking validator min-max
// spans from validators and attestation data roots.
type SpanDetector struct {
	// Slice of epochs for valindex => min-max span + double vote filter
	db   db2.Database
	lock sync.RWMutex
}

// NewSpanDetector creates a new instance of a struct tracking
// several epochs of min-max spans for each validator in
// the beacon state.
func NewSpanDetector(db db2.Database) *SpanDetector {
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
	s.lock.RLock()
	defer s.lock.RUnlock()
	distance := uint16(targetEpoch - sourceEpoch)
	sp, err := s.db.EpochSpanByValidatorIndex(ctx, sourceEpoch, validatorIdx)
	if err != nil {
		return nil, err
	}

	minSpan := sp.MinSpan
	if minSpan > 0 && minSpan < distance {
		slashableEpoch := sourceEpoch + uint64(minSpan)
		span, err := s.db.EpochSpanByValidatorIndex(ctx, slashableEpoch, validatorIdx)
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
		span, err := s.db.EpochSpanByValidatorIndex(ctx, slashableEpoch, validatorIdx)
		if err != nil {
			return nil, err
		}
		return &types.DetectionResult{
			Kind:           types.SurroundVote,
			SlashableEpoch: slashableEpoch,
			SigBytes:       span.SigBytes,
		}, nil
	}
	sp, err = s.db.EpochSpanByValidatorIndex(ctx, targetEpoch, validatorIdx)
	if err != nil {
		return nil, err
	}

	// Check if the validator has attested for this epoch or not.
	if !sp.HasAttested {
		return nil, nil
	}

	return &types.DetectionResult{
		Kind:           types.DoubleVote,
		SlashableEpoch: targetEpoch,
		SigBytes:       sp.SigBytes,
	}, nil

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
		// Save the signature for the received attestation so we can have more detail to find it in the DB.
		s.saveSigBytes(att, valIdx)
		// Update min and max spans.
		s.updateMinSpan(source, target, valIdx)
		s.updateMaxSpan(source, target, valIdx)
	}
	return nil
}

// saveSigBytes saves the first 2 bytes of the signature for the att we're updating the spans to.
// Later used to help us find the violating attestation in the DB.
func (s *SpanDetector) saveSigBytes(att *ethpb.IndexedAttestation, valIdx uint64) {
	numSpans := uint64(len(s.spans))
	target := att.Data.Target.Epoch

	// Check if there is already info saved in span[2].
	if sp := s.spans[target%numSpans]; sp == nil {
		s.spans[target%numSpans] = make(map[uint64]types.Span)
	}
	// If the validator has already attested for this target epoch,
	if s.spans[target%numSpans][valIdx].HasAttested {
		return
	}

	sigBytes := [2]byte{0, 0}
	if len(att.Signature) > 1 {
		sigBytes = [2]byte{att.Signature[0], att.Signature[1]}
	}
	// Save the signature bytes into the span for this epoch.
	span := s.spans[target%numSpans][valIdx]
	s.spans[target%numSpans][valIdx] = types.Span{
		MinSpan:     span.MinSpan,
		MaxSpan:     span.MaxSpan,
		SigBytes:    sigBytes,
		HasAttested: true,
	}
}

// Updates a min span for a validator index given a source and target epoch
// for an attestation produced by the validator. Used for catching surrounding votes.
func (s *SpanDetector) updateMinSpan(source uint64, target uint64, valIdx uint64) {
	numSpans := uint64(len(s.spans))
	if source < 1 {
		return
	}
	for epochInt := int64(source - 1); epochInt >= 0; epochInt-- {
		epoch := uint64(epochInt)
		if sp := s.spans[epoch%numSpans]; sp == nil {
			s.spans[epoch%numSpans] = make(map[uint64]types.Span)
		}
		newMinSpan := uint16(target - epoch)
		span := s.spans[epoch%numSpans][valIdx]
		if span.MinSpan == 0 || span.MinSpan > newMinSpan {
			s.spans[epoch%numSpans][valIdx] = types.Span{
				MinSpan:     newMinSpan,
				MaxSpan:     span.MaxSpan,
				SigBytes:    span.SigBytes,
				HasAttested: span.HasAttested,
			}
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
			s.spans[epoch%numSpans] = make(map[uint64]types.Span)
		}
		span := s.spans[epoch%numSpans][valIdx]
		newMaxSpan := uint16(target - epoch)
		if newMaxSpan > span.MaxSpan {
			s.spans[epoch%numSpans][valIdx] = types.Span{
				MinSpan:     span.MinSpan,
				MaxSpan:     newMaxSpan,
				SigBytes:    span.SigBytes,
				HasAttested: span.HasAttested,
			}
		} else {
			break
		}
	}
}
