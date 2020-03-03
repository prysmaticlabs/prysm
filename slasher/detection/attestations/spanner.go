package attestations

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/prysmaticlabs/prysm/slasher/db"

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
	//s.lock.RLock()
	//defer s.lock.RUnlock()
	distance := uint16(targetEpoch - sourceEpoch)
	sp, err := s.db.EpochSpansMap(ctx, sourceEpoch)
	if err != nil {
		return nil, err
	}
	if sp != nil {
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

// UpdateSpans given an indexed attestation for all of its attesting indices.
func (s *SpanDetector) UpdateSpans(ctx context.Context, att *ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "detection.UpdateSpans")
	defer span.End()
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
		s.updateMinSpan(ctx, source, target, valIdx)
		s.updateMaxSpan(ctx, source, target, valIdx)
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
func (s *SpanDetector) updateMinSpan(ctx context.Context, source uint64, target uint64, valIdx uint64) error {
	if source < 1 {
		return nil
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	for epoch := source - 1; epoch >= 0; epoch-- {
		newMinSpan := uint16(target - epoch)
		sp, err := s.db.EpochSpanByValidatorIndex(ctx, valIdx, epoch)
		if err != nil {
			return err
		}
		minSpan := sp[0]
		maxSpan := sp[1]
		attFilter := sp[2]
		if minSpan == 0 || minSpan > newMinSpan {
			fmt.Printf("epoch %d, valIdx %d: %v\n", epoch, valIdx, [3]uint16{newMinSpan, maxSpan, attFilter})
			if err := s.db.SaveValidatorEpochSpans(ctx, valIdx, epoch, [3]uint16{newMinSpan, maxSpan, attFilter}); err != nil {
				return err
			}
		} else {
			break
		}
		if epoch == 0 {
			break
		}
	}
	return nil
}

// Updates a max span for a validator index given a source and target epoch
// for an attestation produced by the validator. Used for catching surrounded votes.
func (s *SpanDetector) updateMaxSpan(ctx context.Context, source uint64, target uint64, valIdx uint64) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for epoch := source + 1; epoch < target; epoch++ {
		sp, err := s.db.EpochSpanByValidatorIndex(ctx, valIdx, epoch)
		if err != nil {
			return err
		}
		minSpan := sp[0]
		maxSpan := sp[1]
		attFilter := sp[2]

		newMaxSpan := uint16(target - epoch)
		if newMaxSpan > maxSpan {
			if err := s.db.SaveValidatorEpochSpans(ctx, valIdx, epoch, [2]uint16{minSpan, newMaxSpan, attFilter}); err != nil {
				return err
			}
		} else {
			break
		}
	}
	return nil
}
