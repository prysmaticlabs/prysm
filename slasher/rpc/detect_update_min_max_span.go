package rpc

import (
	"context"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Detector is an interface used to implement the slashing surround an surrounded detection
// methods.
type Detector interface {
	Detect(attestationEpochSpan uint64, recorderEpochSpans *ethpb.MinMaxEpochSpan, attestationSourceEpoch uint64) uint64
}

// maxDetector is a detector used to detect surround attestations.
type maxDetector struct{}

// minDetector is a detector used to detect surrounded attestations.
type minDetector struct{}

func (d maxDetector) Detect(attestationEpochSpan uint64, recorderEpochSpans *ethpb.MinMaxEpochSpan, attestationSourceEpoch uint64) uint64 {
	maxSpan := uint64(recorderEpochSpans.MaxEpochSpan)
	if maxSpan > attestationEpochSpan {
		return maxSpan + attestationSourceEpoch
	}
	return 0
}

func (d minDetector) Detect(attestationEpochSpan uint64, recorderEpochSpans *ethpb.MinMaxEpochSpan, attestationSourceEpoch uint64) uint64 {
	minSpan := uint64(recorderEpochSpans.MinEpochSpan)
	if minSpan < attestationEpochSpan {
		return minSpan + attestationSourceEpoch
	}
	return 0
}

// DetectAndUpdateMaxEpochSpan is used to detect and update the max span of an incoming attestation.
// max span is the span between the current attestation source epoch and the furthest attestation
// target that its source epoch is lower then it.
// distance.
// logic is following the detection method designed by https://github.com/protolambda
// from here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func (ss *Server) DetectAndUpdateMaxEpochSpan(ctx context.Context, source uint64, target uint64, validatorIdx uint64) (uint64, error) {
	targetEpoch, span, spanMap, err := ss.detectSlashingByEpochSpan(source, target, validatorIdx, maxDetector{})
	if err != nil {
		return 0, err
	}
	if targetEpoch > 0 {
		return targetEpoch, nil
	}
	for i := uint64(1); i < target-source; i++ {
		val := uint32(span - i - 1)
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
		return 0, err
	}
	return 0, nil
}

// DetectAndUpdateMinEpochSpan is used to detect surround and update the min epoch span
// of an incoming attestation.
// min span is the span between the current attestation and the closest attestation target
// distance.
// logic is following the detection method designed by https://github.com/protolambda
// from here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func (ss *Server) DetectAndUpdateMinEpochSpan(ctx context.Context, source uint64, target uint64, validatorIdx uint64) (uint64, error) {
	targetEpoch, _, spanMap, err := ss.detectSlashingByEpochSpan(source, target, validatorIdx, minDetector{})
	if err != nil {
		return 0, err
	}
	if targetEpoch > 0 {
		return targetEpoch, nil
	}
	if source == 0 {
		return 0, nil
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
	if err := ss.SlasherDB.SaveValidatorSpansMap(validatorIdx, spanMap); err != nil {
		return 0, errors.Wrap(err, "could not save validator spans")
	}
	return 0, nil
}

// detectSlashingByEpochSpan is used to detect if a slashable event is present
// in the db by checking either the closest attestation target or the furthest
// attestation target. this method receives the detection func in order to be
// compatible for both cases.
func (ss *Server) detectSlashingByEpochSpan(source, target, validatorIdx uint64, detector interface{}) (uint64, uint64, *ethpb.EpochSpanMap, error) {
	d, ok := detector.(Detector)
	if !ok {
		return 0, 0, nil, fmt.Errorf("detector interface should be of type Detector. got: %v",
			reflect.TypeOf(detector),
		)
	}
	span := target - source + 1
	if span > params.BeaconConfig().WeakSubjectivityPeriod {
		return 0, span, nil, fmt.Errorf("%d target - source: %d > weakSubjectivityPeriod",
			params.BeaconConfig().WeakSubjectivityPeriod,
			span,
		)
	}
	spanMap, err := ss.SlasherDB.ValidatorSpansMap(validatorIdx)
	if err != nil {
		return 0, span, nil, errors.Wrapf(err, "could not retrieve span map for validatorIdx: %d", validatorIdx)
	}
	if _, ok := spanMap.EpochSpanMap[source]; ok {
		return d.Detect(span, spanMap.EpochSpanMap[source], source), span, spanMap, nil
	}
	return 0, span, spanMap, nil
}
