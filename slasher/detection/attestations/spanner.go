package attestations

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/iface"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"github.com/prysmaticlabs/prysm/slasher/detection/filter"
	"go.opencensus.io/trace"
)

var _ = iface.SpanDetector(&SpanDetector{})

// SpanDetector defines a struct which can detect slashable
// attestation offenses by tracking validator min-max
// spans from validators and attestation data roots.
type SpanDetector struct {
	// Slice of epochs for valindex => min-max span + double vote filter
	spans []map[uint64][3]uint16
	lock  sync.RWMutex
}

// NewSpanDetector creates a new instance of a struct tracking
// several epochs of min-max spans for each validator in
// the beacon state.
func NewSpanDetector() *SpanDetector {
	return &SpanDetector{
		spans: make([]map[uint64][3]uint16, 256),
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
			return &types.DetectionResult{
				Kind:           types.SurroundVote,
				SlashableEpoch: sourceEpoch + uint64(minSpan),
			}, nil
		}

		maxSpan := sp[validatorIdx][1]
		if maxSpan > distance {
			return &types.DetectionResult{
				Kind:           types.SurroundVote,
				SlashableEpoch: sourceEpoch + uint64(maxSpan),
			}, nil
		}
	}

	if sp := s.spans[targetEpoch%numSpans]; sp != nil {
		filterNum := sp[validatorIdx][2]
		if filterNum == 0 {
			return nil, nil
		}
		filterBytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(filterBytes, filterNum)
		attFilter := filter.BloomFilter(filterBytes)

		attDataRoot, err := ssz.HashTreeRoot(attData)
		if err != nil {
			return nil, err
		}
		found, err := attFilter.Contains(attDataRoot[:])
		if err != nil {
			return nil, err
		}
		if !found {
			return &types.DetectionResult{
				Kind:           types.DoubleVote,
				SlashableEpoch: targetEpoch,
			}, nil
		}
	}

	return nil, nil
}

// SpanForEpochByValidator returns the specific min-max span for a
// validator index in a given epoch.
func (s *SpanDetector) SpanForEpochByValidator(ctx context.Context, valIdx uint64, epoch uint64) ([3]uint16, error) {
	ctx, span := trace.StartSpan(ctx, "detection.SpanForEpochByValidator")
	defer span.End()
	s.lock.RLock()
	defer s.lock.RUnlock()
	numSpans := uint64(len(s.spans))
	if span := s.spans[epoch%numSpans]; span != nil {
		if minMaxSpan, ok := span[valIdx]; ok {
			return minMaxSpan, nil
		}
		return [3]uint16{}, fmt.Errorf("validator index %d not found in span map", valIdx)
	}
	return [3]uint16{}, fmt.Errorf("no data found for epoch %d", epoch)
}

// ValidatorSpansByEpoch returns a list of all validator spans in a given epoch.
func (s *SpanDetector) ValidatorSpansByEpoch(ctx context.Context, epoch uint64) map[uint64][3]uint16 {
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
		// Mark the double vote filter.
		if err := s.markAttFilter(att.Data, valIdx); err != nil {
			return errors.Wrap(err, "failed to update attestation filter")
		}
		// Update min and max spans.
		s.updateMinSpan(source, target, valIdx)
		s.updateMaxSpan(source, target, valIdx)
	}
	return nil
}

// markAttFilter sets the third uint16 in the target epochs span to a bloom filter
// with the attestation data root as the key in set. After creating the []byte for the bloom filter,
// it encoded into a uint16 to keep the data structure for the spanner simple and clean.
// A bloom filter is used to prevent collision when using such a small data size.
func (s *SpanDetector) markAttFilter(attData *ethpb.AttestationData, valIdx uint64) error {
	numSpans := uint64(len(s.spans))
	target := attData.Target.Epoch

	// Check if there is an existing bloom filter, if so, don't modify it.
	if sp := s.spans[target%numSpans]; sp == nil {
		s.spans[target%numSpans] = make(map[uint64][3]uint16)
	}
	filterNum := s.spans[target%numSpans][valIdx][2]
	if filterNum != 0 {
		return nil
	}

	// Generate the attestation data root and use it as the only key in the bloom filter.
	attDataRoot, err := ssz.HashTreeRoot(attData)
	if err != nil {
		return err
	}
	attFilter, err := filter.NewBloomFilter(attDataRoot[:])
	if err != nil {
		return err
	}
	filterNum = binary.LittleEndian.Uint16(attFilter)

	// Set the bloom filter back into the span for the epoch.
	minSpan := s.spans[target%numSpans][valIdx][0]
	maxSpan := s.spans[target%numSpans][valIdx][1]
	s.spans[target%numSpans][valIdx] = [3]uint16{minSpan, maxSpan, filterNum}
	return nil
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
			s.spans[epoch%numSpans] = make(map[uint64][3]uint16)
		}
		newMinSpan := uint16(target - epoch)
		minSpan := s.spans[epoch%numSpans][valIdx][0]
		maxSpan := s.spans[epoch%numSpans][valIdx][1]
		attFilter := s.spans[epoch%numSpans][valIdx][2]
		if minSpan == 0 || minSpan > newMinSpan {
			fmt.Printf("epoch %d, valIdx %d: %v\n", epoch%numSpans, valIdx, [3]uint16{newMinSpan, maxSpan, attFilter})
			s.spans[epoch%numSpans][valIdx] = [3]uint16{newMinSpan, maxSpan, attFilter}
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
			s.spans[epoch%numSpans] = make(map[uint64][3]uint16)
		}
		minSpan := s.spans[epoch%numSpans][valIdx][0]
		maxSpan := s.spans[epoch%numSpans][valIdx][1]
		attFilter := s.spans[epoch%numSpans][valIdx][2]
		newMaxSpan := uint16(target - epoch)
		if newMaxSpan > maxSpan {
			s.spans[epoch%numSpans][valIdx] = [3]uint16{minSpan, newMaxSpan, attFilter}
		} else {
			break
		}
	}
}
