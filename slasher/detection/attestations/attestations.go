package attestations

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// Detector is a function type used to implement the slashable surrounding/surrounded
// vote detection methods.
type detectFn = func(attestationEpochSpan uint64, recorderEpochSpan *slashpb.MinMaxEpochSpan, sourceEpoch uint64) uint64

// detectMax is a function for maxDetector used to detect surrounding attestations.
func detectMax(attEpochSpan uint64, recorderEpochSpan *slashpb.MinMaxEpochSpan, attSourceEpoch uint64) uint64 {
	maxSpan := uint64(recorderEpochSpan.MaxEpochSpan)
	if maxSpan > attEpochSpan {
		return maxSpan + attSourceEpoch
	}
	return 0
}

// detectMin is a function for minDetector used to detect surrounded attestations.
func detectMin(attEpochSpan uint64, recorderEpochSpan *slashpb.MinMaxEpochSpan, attSourceEpoch uint64) uint64 {
	minSpan := uint64(recorderEpochSpan.MinEpochSpan)
	if minSpan > 0 && minSpan < attEpochSpan {
		return minSpan + attSourceEpoch
	}
	return 0
}

// DetectAndUpdateSpans runs detection and updating for both min and max epoch spans, this is used for
// attestation slashing detection.
// Detailed here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func DetectAndUpdateSpans(
	ctx context.Context,
	att *ethpb.IndexedAttestation,
	spanMap *slashpb.EpochSpanMap,
) (*slashpb.EpochSpanMap, uint64, uint64, error) {
	ctx, span := trace.StartSpan(ctx, "Detection.DetectAndUpdateSpans")
	defer span.End()
	minTargetEpoch, spanMap, err := detectAndUpdateMinEpochSpan(ctx, att.Data.Source.Epoch, att.Data.Target.Epoch, spanMap)
	if err != nil {
		return nil, 0, 0, errors.Wrap(err, "failed to update min spans")
	}
	maxTargetEpoch, spanMap, err := detectAndUpdateMaxEpochSpan(ctx, att.Data.Source.Epoch, att.Data.Target.Epoch, spanMap)
	if err != nil {
		return nil, 0, 0, errors.Wrap(err, "failed to update max spans")
	}
	return spanMap, minTargetEpoch, maxTargetEpoch, nil
}

// detectAndUpdateMaxEpochSpan is used to detect and update the max span of an incoming attestation.
// This is used for detecting surrounding votes.
// The max span is the span between the current attestation's source epoch and the furthest attestation's
// target epoch that has a lower (earlier) source epoch.
// Logic for this detection method was designed by https://github.com/protolambda
// Detailed here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func detectAndUpdateMaxEpochSpan(
	ctx context.Context,
	source uint64,
	target uint64,
	spanMap *slashpb.EpochSpanMap,
) (uint64, *slashpb.EpochSpanMap, error) {
	ctx, span := trace.StartSpan(ctx, "Detection.detectAndUpdateMaxEpochSpan")
	defer span.End()
	if target < source {
		return 0, nil, fmt.Errorf("target: %d < source: %d ", target, source)
	}
	targetEpoch, minMaxSpan, spanMap, err := detectSlashingByEpochSpan(ctx, source, target, spanMap, detectMax)
	if err != nil {
		return 0, nil, err
	}
	if targetEpoch > 0 {
		return targetEpoch, spanMap, nil
	}

	for i := uint64(1); i < target-source; i++ {
		val := uint32(minMaxSpan - i)
		if _, ok := spanMap.EpochSpanMap[source+i]; !ok {
			spanMap.EpochSpanMap[source+i] = &slashpb.MinMaxEpochSpan{}
		}
		if spanMap.EpochSpanMap[source+i].MaxEpochSpan < val {
			spanMap.EpochSpanMap[source+i].MaxEpochSpan = val
		} else {
			break
		}
	}
	return 0, spanMap, nil
}

// detectAndUpdateMinEpochSpan is used to detect surrounded votes and update the min epoch span
// of an incoming attestation.
// The min span is the span between the current attestations target epoch and the
// closest attestation's target distance.
//
// Logic is following the detection method designed by https://github.com/protolambda
// Detailed here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func detectAndUpdateMinEpochSpan(
	ctx context.Context,
	source uint64,
	target uint64,
	spanMap *slashpb.EpochSpanMap,
) (uint64, *slashpb.EpochSpanMap, error) {
	ctx, span := trace.StartSpan(ctx, "Detection.detectAndUpdateMinEpochSpan")
	defer span.End()
	if target < source {
		return 0, nil, fmt.Errorf(
			"target: %d < source: %d ",
			target,
			source,
		)
	}
	targetEpoch, _, spanMap, err := detectSlashingByEpochSpan(ctx, source, target, spanMap, detectMin)
	if err != nil {
		return 0, nil, err
	}
	if targetEpoch > 0 {
		return targetEpoch, spanMap, nil
	}
	if source == 0 {
		return 0, spanMap, nil
	}

	for i := source - 1; i > 0; i-- {
		val := uint32(target - (i))
		if _, ok := spanMap.EpochSpanMap[i]; !ok {
			spanMap.EpochSpanMap[i] = &slashpb.MinMaxEpochSpan{}
		}
		if spanMap.EpochSpanMap[i].MinEpochSpan == 0 || spanMap.EpochSpanMap[i].MinEpochSpan > val {
			spanMap.EpochSpanMap[i].MinEpochSpan = val
		} else {
			break
		}
	}
	return 0, spanMap, nil
}

// detectSlashingByEpochSpan is used to detect if a slashable event is present
// in the db by checking either the closest attestation target or the furthest
// attestation target. This method receives a detector function in order to be used
// for both surrounding and surrounded vote cases.
func detectSlashingByEpochSpan(
	ctx context.Context,
	source uint64,
	target uint64,
	spanMap *slashpb.EpochSpanMap,
	detector detectFn,
) (uint64, uint64, *slashpb.EpochSpanMap, error) {
	ctx, span := trace.StartSpan(ctx, "Detection.detectSlashingByEpochSpan")
	defer span.End()
	minMaxSpan := target - source
	if minMaxSpan > params.BeaconConfig().WeakSubjectivityPeriod {
		return 0, minMaxSpan, nil, fmt.Errorf(
			"target: %d - source: %d > weakSubjectivityPeriod",
			params.BeaconConfig().WeakSubjectivityPeriod,
			minMaxSpan,
		)
	}
	if _, ok := spanMap.EpochSpanMap[source]; ok {
		return detector(minMaxSpan, spanMap.EpochSpanMap[source], source), minMaxSpan, spanMap, nil
	}
	return 0, minMaxSpan, spanMap, nil
}
