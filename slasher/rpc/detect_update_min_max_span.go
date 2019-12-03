package rpc

import (
	"context"
	"fmt"

	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Detector is a function type used to implement the slashable surrounding/surrounded
// vote detection methods.
type detectFn = func(attestationEpochSpan uint64, recorderEpochSpan *ethpb.MinMaxEpochSpan, sourceEpoch uint64) uint64

// detectMax is a function for maxDetector used to detect surrounding attestations.
func detectMax(
	attestationEpochSpan uint64,
	recorderEpochSpan *ethpb.MinMaxEpochSpan,
	attestationSourceEpoch uint64) uint64 {

	maxSpan := uint64(recorderEpochSpan.MaxEpochSpan)
	if maxSpan > attestationEpochSpan {
		return maxSpan + attestationSourceEpoch
	}
	return 0
}

// detectMin is a function for minDetecter used to detect surrounded attestations.
func detectMin(attestationEpochSpan uint64,
	recorderEpochSpan *ethpb.MinMaxEpochSpan,
	attestationSourceEpoch uint64) uint64 {

	minSpan := uint64(recorderEpochSpan.MinEpochSpan)

	if minSpan > 0 && minSpan < attestationEpochSpan {
		return minSpan + attestationSourceEpoch
	}
	return 0
}

// DetectAndUpdateMaxEpochSpan is used to detect and update the max span of an incoming attestation.
// This is used for detecting surrounding votes.
// The max span is the span between the current attestation's source epoch and the furthest attestation's
// target epoch that has a lower (earlier) source epoch.
// Logic for this detection method was designed by https://github.com/protolambda
// Detailed here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func (ss *Server) DetectAndUpdateMaxEpochSpan(ctx context.Context, source uint64, target uint64, validatorIdx uint64, spanMap *ethpb.EpochSpanMap) (uint64, *ethpb.EpochSpanMap, error) {
	if target < source {
		return 0, nil, fmt.Errorf(
			"target: %d < source: %d ",
			target,
			source,
		)
	}
	targetEpoch, span, spanMap, err := ss.detectSlashingByEpochSpan(source, target, spanMap, detectMax)
	if err != nil {
		return 0, nil, err
	}
	if targetEpoch > 0 {
		return targetEpoch, spanMap, nil
	}
	for i := uint64(1); i < target-source; i++ {
		val := uint32(span - i)
		if _, ok := spanMap.EpochSpanMap[source+i]; !ok {
			spanMap.EpochSpanMap[source+i] = &ethpb.MinMaxEpochSpan{}
		}
		if spanMap.EpochSpanMap[source+i].MaxEpochSpan < val {
			spanMap.EpochSpanMap[source+i].MaxEpochSpan = val
		} else {
			break
		}
	}
	if err := ss.SlasherDB.SaveValidatorSpansMap(validatorIdx, spanMap); err != nil {
		return 0, nil, err
	}
	return 0, spanMap, nil
}

// DetectAndUpdateMinEpochSpan is used to detect surrounded votes and update the min epoch span
// of an incoming attestation.
// The min span is the span between the current attestations target epoch and the
// closest attestation's target distance.
//
// Logic is following the detection method designed by https://github.com/protolambda
// Detailed here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func (ss *Server) DetectAndUpdateMinEpochSpan(ctx context.Context, source uint64, target uint64, validatorIdx uint64, spanMap *ethpb.EpochSpanMap) (uint64, *ethpb.EpochSpanMap, error) {
	if target < source {
		return 0, nil, fmt.Errorf(
			"target: %d < source: %d ",
			target,
			source,
		)
	}
	targetEpoch, _, spanMap, err := ss.detectSlashingByEpochSpan(source, target, spanMap, detectMin)
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
			spanMap.EpochSpanMap[i] = &ethpb.MinMaxEpochSpan{}
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
func (ss *Server) detectSlashingByEpochSpan(source, target uint64, spanMap *ethpb.EpochSpanMap, detector detectFn) (uint64, uint64, *ethpb.EpochSpanMap, error) {
	span := target - source
	if span > params.BeaconConfig().WeakSubjectivityPeriod {
		return 0, span, nil, fmt.Errorf("target: %d - source: %d > weakSubjectivityPeriod",
			params.BeaconConfig().WeakSubjectivityPeriod,
			span,
		)
	}
	if _, ok := spanMap.EpochSpanMap[source]; ok {
		return detector(span, spanMap.EpochSpanMap[source], source), span, spanMap, nil
	}
	return 0, span, spanMap, nil
}
