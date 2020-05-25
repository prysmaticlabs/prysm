// Package attestations defines an implementation of a
// slashable attestation detector using min-max surround vote checking.
package attestations

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	db "github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/iface"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var (
	latestMinSpanDistanceObserved = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "latest_min_span_distance_observed",
		Help: "The latest distance between target - source observed for min spans",
	})
	latestMaxSpanDistanceObserved = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "latest_max_span_distance_observed",
		Help: "The latest distance between target - source observed for max spans",
	})
)

// We look back 128 epochs when updating min/max spans
// for incoming attestations.
// TODO(#5040): Remove lookback and handle min spans properly.
const epochLookback = 128

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

// DetectSlashingsForAttestation uses a validator index and its corresponding
// min-max spans during an epoch to detect an epoch in which the validator
// committed a slashable attestation.
func (s *SpanDetector) DetectSlashingsForAttestationNew(
	ctx context.Context,
	att *ethpb.IndexedAttestation,
) ([]*types.DetectionResult, error) {
	ctx, traceSpan := trace.StartSpan(ctx, "spanner.DetectSlashingsForAttestation")
	defer traceSpan.End()
	sourceEpoch := att.Data.Source.Epoch
	targetEpoch := att.Data.Target.Epoch
	if (targetEpoch - sourceEpoch) > params.BeaconConfig().WeakSubjectivityPeriod {
		return nil, fmt.Errorf(
			"attestation span was greater than weak subjectivity period %d, received: %d",
			params.BeaconConfig().WeakSubjectivityPeriod,
			targetEpoch-sourceEpoch,
		)
	}

	sourceSpans, _, _, err := s.slasherDB.EpochSpans(ctx, sourceEpoch)
	if err != nil {
		return nil, err
	}
	targetSpans, _, _, err := s.slasherDB.EpochSpans(ctx, targetEpoch)
	if err != nil {
		return nil, err
	}

	var detections []*types.DetectionResult
	distance := uint16(targetEpoch - sourceEpoch)
	for _, idx := range att.AttestingIndices {
		if ctx.Err() != nil {
			return nil, errors.Wrap(ctx.Err(), "could not detect slashings")
		}
		span, err := s.slasherDB.GetValidatorSpan(ctx, sourceSpans, idx)
		if err != nil {
			return nil, err
		}
		minSpan := span.MinSpan
		if minSpan > 0 && minSpan < distance {
			slashableEpoch := sourceEpoch + uint64(minSpan)
			slashableSpans, _, _, err := s.slasherDB.EpochSpans(ctx, slashableEpoch)
			if err != nil {
				return nil, err
			}
			targetSpan, err := s.slasherDB.GetValidatorSpan(ctx, slashableSpans, idx)
			if err != nil {
				return nil, err
			}
			detections = append(detections, &types.DetectionResult{
				ValidatorIndex: idx,
				Kind:           types.SurroundVote,
				SlashableEpoch: slashableEpoch,
				SigBytes:       targetSpan.SigBytes,
			})
			continue
		}

		maxSpan := span.MaxSpan
		if maxSpan > distance {
			slashableEpoch := sourceEpoch + uint64(maxSpan)
			slashableSpans, _, _, err := s.slasherDB.EpochSpans(ctx, slashableEpoch)
			if err != nil {
				return nil, err
			}
			targetSpan, err := s.slasherDB.GetValidatorSpan(ctx, slashableSpans, idx)
			if err != nil {
				return nil, err
			}
			detections = append(detections, &types.DetectionResult{
				ValidatorIndex: idx,
				Kind:           types.SurroundVote,
				SlashableEpoch: slashableEpoch,
				SigBytes:       targetSpan.SigBytes,
			})
			continue
		}
		targetSpan, err := s.slasherDB.GetValidatorSpan(ctx, targetSpans, idx)
		if err != nil {
			return nil, err
		}
		// Check if the validator has attested for this epoch or not.
		if targetSpan.HasAttested {
			detections = append(detections, &types.DetectionResult{
				ValidatorIndex: idx,
				Kind:           types.DoubleVote,
				SlashableEpoch: targetEpoch,
				SigBytes:       targetSpan.SigBytes,
			})
			continue
		}
	}

	return detections, nil
}

func (s *SpanDetector) DetectSlashingsForAttestation(
	ctx context.Context,
	att *ethpb.IndexedAttestation,
) ([]*types.DetectionResult, error) {
	ctx, traceSpan := trace.StartSpan(ctx, "spanner.DetectSlashingsForAttestation")
	defer traceSpan.End()
	sourceEpoch := att.Data.Source.Epoch
	targetEpoch := att.Data.Target.Epoch
	if (targetEpoch - sourceEpoch) > params.BeaconConfig().WeakSubjectivityPeriod {
		return nil, fmt.Errorf(
			"attestation span was greater than weak subjectivity period %d, received: %d",
			params.BeaconConfig().WeakSubjectivityPeriod,
			targetEpoch-sourceEpoch,
		)
	}

	spanMap, _, err := s.slasherDB.EpochSpansMap(ctx, sourceEpoch)
	if err != nil {
		return nil, err
	}
	targetSpanMap, _, err := s.slasherDB.EpochSpansMap(ctx, targetEpoch)
	if err != nil {
		return nil, err
	}

	var detections []*types.DetectionResult
	distance := uint16(targetEpoch - sourceEpoch)
	for _, idx := range att.AttestingIndices {
		if ctx.Err() != nil {
			return nil, errors.Wrap(ctx.Err(), "could not detect slashings")
		}
		span := spanMap[idx]
		minSpan := span.MinSpan
		if minSpan > 0 && minSpan < distance {
			slashableEpoch := sourceEpoch + uint64(minSpan)
			targetSpan, err := s.slasherDB.EpochSpanByValidatorIndex(ctx, idx, slashableEpoch)
			if err != nil {
				return nil, err
			}
			detections = append(detections, &types.DetectionResult{
				ValidatorIndex: idx,
				Kind:           types.SurroundVote,
				SlashableEpoch: slashableEpoch,
				SigBytes:       targetSpan.SigBytes,
			})
			continue
		}

		maxSpan := span.MaxSpan
		if maxSpan > distance {
			slashableEpoch := sourceEpoch + uint64(maxSpan)
			targetSpan, err := s.slasherDB.EpochSpanByValidatorIndex(ctx, idx, slashableEpoch)
			if err != nil {
				return nil, err
			}
			detections = append(detections, &types.DetectionResult{
				ValidatorIndex: idx,
				Kind:           types.SurroundVote,
				SlashableEpoch: slashableEpoch,
				SigBytes:       targetSpan.SigBytes,
			})
			continue
		}

		targetSpan := targetSpanMap[idx]
		// Check if the validator has attested for this epoch or not.
		if targetSpan.HasAttested {
			detections = append(detections, &types.DetectionResult{
				ValidatorIndex: idx,
				Kind:           types.DoubleVote,
				SlashableEpoch: targetEpoch,
				SigBytes:       targetSpan.SigBytes,
			})
			continue
		}
	}

	return detections, nil
}

// UpdateSpans given an indexed attestation for all of its attesting indices.
func (s *SpanDetector) UpdateSpans(ctx context.Context, att *ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "spanner.UpdateSpans")
	defer span.End()
	// Save the signature for the received attestation so we can have more detail to find it in the DB.
	if err := s.saveSigBytes(ctx, att); err != nil {
		return err
	}
	// Update min and max spans.
	if err := s.updateMinSpan(ctx, att); err != nil {
		return err
	}
	if err := s.updateMaxSpan(ctx, att); err != nil {
		return err
	}
	return nil
}

// UpdateSpans given an indexed attestation for all of its attesting indices.
func (s *SpanDetector) UpdateSpansNew(ctx context.Context, att *ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "spanner.UpdateSpans")
	defer span.End()
	// Save the signature for the received attestation so we can have more detail to find it in the DB.
	if err := s.saveSigBytesNew(ctx, att); err != nil {
		return err
	}
	log.Info("saveSigBytesNew")
	// Update min and max spans.
	if err := s.updateMinSpanNew(ctx, att); err != nil {
		return err
	}
	log.Info("updateMinSpanNew")

	if err := s.updateMaxSpanNew(ctx, att); err != nil {
		return err
	}
	log.Info("updateMaxSpanNew")
	return nil
}

// saveSigBytes saves the first 2 bytes of the signature for the att we're updating the spans to.
// Later used to help us find the violating attestation in the DB.
func (s *SpanDetector) saveSigBytesNew(ctx context.Context, att *ethpb.IndexedAttestation) error {
	ctx, traceSpan := trace.StartSpan(ctx, "spanner.saveSigBytes")
	defer traceSpan.End()
	target := att.Data.Target.Epoch
	spanMap, _, _, err := s.slasherDB.EpochSpans(ctx, target)
	if err != nil {
		return err
	}

	// We loop through the indices, instead of constantly locking/unlocking the cache for equivalent accesses.
	for _, idx := range att.AttestingIndices {
		if ctx.Err() != nil {
			return errors.Wrap(ctx.Err(), "could not save signature bytes")
		}
		span, err := s.slasherDB.GetValidatorSpan(ctx, spanMap, idx)
		if err != nil {
			return err
		}
		// If the validator has already attested for this target epoch,
		// then we do not need to update the values of the span sig bytes.
		if span.HasAttested {
			return nil
		}

		sigBytes := [2]byte{0, 0}
		if len(att.Signature) > 1 {
			sigBytes = [2]byte{att.Signature[0], att.Signature[1]}
		}
		// Save the signature bytes into the span for this epoch.
		span = types.Span{
			MinSpan:     span.MinSpan,
			MaxSpan:     span.MaxSpan,
			HasAttested: true,
			SigBytes:    sigBytes,
		}
		spanMap, err = s.slasherDB.SetValidatorSpan(ctx, spanMap, idx, span)
		if err != nil {
			return err
		}
	}
	return s.slasherDB.SaveEpochSpans(ctx, target, spanMap)
}

// saveSigBytes saves the first 2 bytes of the signature for the att we're updating the spans to.
// Later used to help us find the violating attestation in the DB.
func (s *SpanDetector) saveSigBytes(ctx context.Context, att *ethpb.IndexedAttestation) error {
	ctx, traceSpan := trace.StartSpan(ctx, "spanner.saveSigBytes")
	defer traceSpan.End()
	target := att.Data.Target.Epoch
	spanMap, _, err := s.slasherDB.EpochSpansMap(ctx, target)
	if err != nil {
		return err
	}

	// We loop through the indices, instead of constantly locking/unlocking the cache for equivalent accesses.
	for _, idx := range att.AttestingIndices {
		if ctx.Err() != nil {
			return errors.Wrap(ctx.Err(), "could not save signature bytes")
		}
		span := spanMap[idx]
		// If the validator has already attested for this target epoch,
		// then we do not need to update the values of the span sig bytes.
		if span.HasAttested {
			return nil
		}

		sigBytes := [2]byte{0, 0}
		if len(att.Signature) > 1 {
			sigBytes = [2]byte{att.Signature[0], att.Signature[1]}
		}
		// Save the signature bytes into the span for this epoch.
		spanMap[idx] = types.Span{
			MinSpan:     span.MinSpan,
			MaxSpan:     span.MaxSpan,
			HasAttested: true,
			SigBytes:    sigBytes,
		}
	}
	return s.slasherDB.SaveEpochSpansMap(ctx, target, spanMap)
}

// Updates a min span for a validator index given a source and target epoch
// for an attestation produced by the validator. Used for catching surrounding votes.
func (s *SpanDetector) updateMinSpanNew(ctx context.Context, att *ethpb.IndexedAttestation) error {
	ctx, traceSpan := trace.StartSpan(ctx, "spanner.updateMinSpan")
	defer traceSpan.End()
	source := att.Data.Source.Epoch
	target := att.Data.Target.Epoch
	if source < 1 {
		return nil
	}
	valIndices := make([]uint64, len(att.AttestingIndices))
	copy(valIndices, att.AttestingIndices)
	latestMinSpanDistanceObserved.Set(float64(att.Data.Target.Epoch - att.Data.Source.Epoch))

	// the for loop tries to update min span using cache for as long as there
	// is a relevant cached epoch. when there is no such epoch in cache batch
	// db read and write is used.
	epoch := source - 1
	log.Infof("before loop epoch: %d len(valIndices) %d", epoch, len(valIndices))
	untilEpoch := uint64(0)
	if epoch >= epochLookback {
		untilEpoch = epoch - epochLookback
		//lastEpochToCache = epoch - epochLookback
	}
	if featureconfig.Get().DisableLookback {
		untilEpoch = 0
	}

	for ; epoch >= untilEpoch; epoch-- {
		if ctx.Err() != nil {
			return errors.Wrap(ctx.Err(), "could not update min spans")
		}
		spans, _, _, err := s.slasherDB.EpochSpans(ctx, epoch)
		log.Infof("getting EpochSpans %d", epoch)
		if err != nil {
			return err
		}
		valNum := 0
		for _, idx := range valIndices {
			span, err := s.slasherDB.GetValidatorSpan(ctx, spans, idx)
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
				spans, err = s.slasherDB.SetValidatorSpan(ctx, spans, idx, span)
				if err != nil {
					return err
				}
				valIndices[valNum] = idx
				valNum++
			}
		}
		valIndices = valIndices[:valNum]
		if err := s.slasherDB.SaveEpochSpans(ctx, epoch, spans); err != nil {
			return err
		}
		log.Infof("after SaveEpochSpans %d", epoch)
		if len(valIndices) == 0 || epoch == 0 {
			log.Infof("epoch: %d len(indices) %d", epoch, len(valIndices))
			break
		}
	}
	return nil
}

// Updates a min span for a validator index given a source and target epoch
// for an attestation produced by the validator. Used for catching surrounding votes.
func (s *SpanDetector) updateMinSpan(ctx context.Context, att *ethpb.IndexedAttestation) error {
	ctx, traceSpan := trace.StartSpan(ctx, "spanner.updateMinSpan")
	defer traceSpan.End()
	source := att.Data.Source.Epoch
	target := att.Data.Target.Epoch
	if source < 1 {
		return nil
	}
	valIndices := make([]uint64, len(att.AttestingIndices))
	copy(valIndices, att.AttestingIndices)
	latestMinSpanDistanceObserved.Set(float64(att.Data.Target.Epoch - att.Data.Source.Epoch))

	// the for loop tries to update min span using cache for as long as there
	// is a relevant cached epoch. when there is no such epoch in cache batch
	// db read and write is used.
	spanMap := make(map[uint64]types.Span)
	epochsSpansMap := make(map[uint64]map[uint64]types.Span)
	epoch := source - 1
	untilEpoch := epoch - epochLookback
	if int(untilEpoch) < 0 || featureconfig.Get().DisableLookback {
		untilEpoch = 0
	}
	useCache := true
	useDb := false
	var err error
	for ; epoch >= untilEpoch; epoch-- {
		if ctx.Err() != nil {
			return errors.Wrap(ctx.Err(), "could not update min spans")
		}
		if featureconfig.Get().DisableLookback {
			if useCache {
				spanMap, useCache, err = s.slasherDB.EpochSpansMap(ctx, epoch)
			}
			// Should happen once when cache is exhausted.
			if !useCache && !useDb && featureconfig.Get().DisableLookback {
				epochsSpansMap, err = s.slasherDB.EpochsSpanByValidatorsIndices(ctx, valIndices, epoch)
				useDb = true
			}
			if err != nil {
				return err
			}
			if useDb && featureconfig.Get().DisableLookback {
				spanMap = epochsSpansMap[epoch]
				if spanMap == nil {
					spanMap = make(map[uint64]types.Span)
					epochsSpansMap[epoch] = spanMap
				}
			}

			indices := valIndices[:0]
			for _, idx := range valIndices {
				span := spanMap[idx]
				newMinSpan := uint16(target - epoch)
				if span.MinSpan == 0 || span.MinSpan > newMinSpan {
					span = types.Span{
						MinSpan:     newMinSpan,
						MaxSpan:     span.MaxSpan,
						SigBytes:    span.SigBytes,
						HasAttested: span.HasAttested,
					}
					spanMap[idx] = span
					indices = append(indices, idx)
				}
			}
			copy(valIndices, indices)
			if useCache {
				if err := s.slasherDB.SaveEpochSpansMap(ctx, epoch, spanMap); err != nil {
					return err
				}
			}
			if len(indices) == 0 || epoch == 0 {
				if useDb {
					// should happen once when finishing update to all epochs and all indices.
					if err := s.slasherDB.SaveEpochsSpanByValidatorsIndices(ctx, epochsSpansMap); err != nil {
						return err
					}
				}
				break
			}
		} else {
			spanMap, _, err := s.slasherDB.EpochSpansMap(ctx, epoch)
			if err != nil {
				return err
			}
			indices := valIndices[:0]
			for _, idx := range valIndices {
				span := spanMap[idx]
				newMinSpan := uint16(target - epoch)
				if span.MinSpan == 0 || span.MinSpan > newMinSpan {
					span = types.Span{
						MinSpan:     newMinSpan,
						MaxSpan:     span.MaxSpan,
						SigBytes:    span.SigBytes,
						HasAttested: span.HasAttested,
					}
					spanMap[idx] = span
					indices = append(indices, idx)
				}
			}
			if err := s.slasherDB.SaveEpochSpansMap(ctx, epoch, spanMap); err != nil {
				return err
			}
			if len(indices) == 0 {
				break
			}
			if epoch == 0 {
				break
			}
		}
	}
	return nil
}

// Updates a max span for a validator index given a source and target epoch
// for an attestation produced by the validator. Used for catching surrounded votes.
func (s *SpanDetector) updateMaxSpanNew(ctx context.Context, att *ethpb.IndexedAttestation) error {
	ctx, traceSpan := trace.StartSpan(ctx, "spanner.updateMaxSpan")
	defer traceSpan.End()
	source := att.Data.Source.Epoch
	target := att.Data.Target.Epoch
	latestMaxSpanDistanceObserved.Set(float64(target - source))
	valIndices := make([]uint64, len(att.AttestingIndices))
	copy(valIndices, att.AttestingIndices)
	for epoch := source + 1; epoch < target; epoch++ {
		if ctx.Err() != nil {
			return errors.Wrap(ctx.Err(), "could not update max spans")
		}
		spans, _, _, err := s.slasherDB.EpochSpans(ctx, epoch)
		if err != nil {
			return err
		}
		indices := valIndices[:0]
		for _, idx := range valIndices {
			span, err := s.slasherDB.GetValidatorSpan(ctx, spans, idx)
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
				spans, err = s.slasherDB.SetValidatorSpan(ctx, spans, idx, span)
				if err != nil {
					return err
				}
				indices = append(indices, idx)
			}
		}
		if err := s.slasherDB.SaveEpochSpans(ctx, epoch, spans); err != nil {
			return err
		}
		if len(indices) == 0 {
			break
		}
	}
	return nil
}

func (s *SpanDetector) updateMaxSpan(ctx context.Context, att *ethpb.IndexedAttestation) error {
	ctx, traceSpan := trace.StartSpan(ctx, "spanner.updateMaxSpan")
	defer traceSpan.End()
	source := att.Data.Source.Epoch
	target := att.Data.Target.Epoch
	latestMaxSpanDistanceObserved.Set(float64(target - source))
	valIndices := make([]uint64, len(att.AttestingIndices))
	copy(valIndices, att.AttestingIndices)
	for epoch := source + 1; epoch < target; epoch++ {
		if ctx.Err() != nil {
			return errors.Wrap(ctx.Err(), "could not update max spans")
		}
		spanMap, _, err := s.slasherDB.EpochSpansMap(ctx, epoch)
		if err != nil {
			return err
		}
		indices := valIndices[:0]
		for _, idx := range valIndices {
			span := spanMap[idx]
			newMaxSpan := uint16(target - epoch)
			if newMaxSpan > span.MaxSpan {
				span = types.Span{
					MinSpan:     span.MinSpan,
					MaxSpan:     newMaxSpan,
					SigBytes:    span.SigBytes,
					HasAttested: span.HasAttested,
				}
				spanMap[idx] = span
				indices = append(indices, idx)
			}
		}
		if err := s.slasherDB.SaveEpochSpansMap(ctx, epoch, spanMap); err != nil {
			return err
		}
		if len(indices) == 0 {
			break
		}
	}
	return nil
}
