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
type detectFn = func(span *slashpb.MinMaxEpochSpan, sourceEpoch uint64, attDistance uint64) uint64

// detectMax is a function for maxDetector used to detect surrounding attestations.
func detectMax(span *slashpb.MinMaxEpochSpan, attSourceEpoch uint64, attDistance uint64) uint64 {
	maxSpan := uint64(span.MaxEpochSpan)
	if maxSpan > attDistance {
		return maxSpan + attSourceEpoch
	}
	return 0
}

// detectMin is a function for minDetector used to detect surrounded attestations.
func detectMin(span *slashpb.MinMaxEpochSpan, attSourceEpoch uint64, attDistance uint64) uint64 {
	minSpan := uint64(span.MinEpochSpan)
	if minSpan > 0 && minSpan < attDistance {
		return minSpan + attSourceEpoch
	}
	return 0
}

// DetectAndUpdateSpans runs detection and updating for both min and max epoch spans, this is used for
// attestation slashing detection.
// Detailed here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func DetectAndUpdateSpans(
	ctx context.Context,
	spanMap *slashpb.EpochSpanMap,
	att *ethpb.IndexedAttestation,
) (*slashpb.EpochSpanMap, uint64, error) {
	ctx, span := trace.StartSpan(ctx, "Detection.DetectAndUpdateSpans")
	defer span.End()
	spanMap, minTargetEpoch, err := detectAndUpdateMinEpochSpan(ctx, spanMap, att.Data.Source.Epoch, att.Data.Target.Epoch)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to update min spans")
	}
	spanMap, maxTargetEpoch, err := detectAndUpdateMaxEpochSpan(ctx, spanMap, att.Data.Source.Epoch, att.Data.Target.Epoch)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to update max spans")
	}

	slashableEpoch := minTargetEpoch
	if slashableEpoch == 0 {
		slashableEpoch = maxTargetEpoch
	}
	return spanMap, slashableEpoch, nil
}

// detectAndUpdateMaxEpochSpan is used to detect and update the max span of an incoming attestation.
// This is used for detecting surrounding votes.
// The max span is the span between the current attestation's source epoch and the furthest attestation's
// target epoch that has a lower (earlier) source epoch.
// Logic for this detection method was designed by https://github.com/protolambda
// Detailed here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func detectAndUpdateMaxEpochSpan(
	ctx context.Context,
	spanMap *slashpb.EpochSpanMap,
	source uint64,
	target uint64,
) (*slashpb.EpochSpanMap, uint64, error) {
	ctx, span := trace.StartSpan(ctx, "Detection.detectAndUpdateMaxEpochSpan")
	defer span.End()
	if source > target {
		return nil, 0, fmt.Errorf(
			"source cannot be greater than target, received source %d, target %d",
			source,
			target,
		)
	}
	spanMap, distance, targetEpoch, err := detectSlashingByEpochSpan(ctx, spanMap, source, target, detectMax)
	if err != nil {
		return nil, 0, err
	}
	if targetEpoch > 0 {
		return spanMap, targetEpoch, nil
	}

	for i := uint64(1); i < target-source; i++ {
		val := uint32(distance - i)
		if _, ok := spanMap.EpochSpanMap[source+i]; !ok {
			spanMap.EpochSpanMap[source+i] = &slashpb.MinMaxEpochSpan{}
		}
		if spanMap.EpochSpanMap[source+i].MaxEpochSpan < val {
			spanMap.EpochSpanMap[source+i].MaxEpochSpan = val
		} else {
			break
		}
	}
	return spanMap, 0, nil
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
	spanMap *slashpb.EpochSpanMap,
	source uint64,
	target uint64,
) (*slashpb.EpochSpanMap, uint64, error) {
	ctx, span := trace.StartSpan(ctx, "Detection.detectAndUpdateMinEpochSpan")
	defer span.End()
	if source > target {
		return nil, 0, fmt.Errorf(
			"source cannot be greater than target, received source %d, target %d",
			source,
			target,
		)
	}
	spanMap, _, targetEpoch, err := detectSlashingByEpochSpan(ctx, spanMap, source, target, detectMin)
	if err != nil {
		return nil, 0, err
	}
	if targetEpoch > 0 {
		return spanMap, targetEpoch, nil
	}
	if source == 0 {
		return spanMap, 0, nil
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
	return spanMap, 0, nil
}

// detectSlashingByEpochSpan is used to detect if a slashable event is present
// in the db by checking either the closest attestation target or the furthest
// attestation target. This method receives a detector function in order to be used
// for both surrounding and surrounded vote cases.
func detectSlashingByEpochSpan(
	ctx context.Context,
	spanMap *slashpb.EpochSpanMap,
	source uint64,
	target uint64,
	detector detectFn,
) (*slashpb.EpochSpanMap, uint64, uint64, error) {
	ctx, span := trace.StartSpan(ctx, "Detection.detectSlashingByEpochSpan")
	defer span.End()
	distance := target - source
	if distance > params.BeaconConfig().WeakSubjectivityPeriod {
		return nil, distance, 0, fmt.Errorf(
			"attestation span was greater than waek subjectivity period, received: %d",
			distance,
		)
	}
	if _, ok := spanMap.EpochSpanMap[source]; ok {
		return spanMap, distance, detector(spanMap.EpochSpanMap[source], source, distance), nil
	}
	return spanMap, distance, 0, nil
}
