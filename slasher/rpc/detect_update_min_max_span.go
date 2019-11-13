package rpc

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

type detect func(span uint64, spans *ethpb.MinMaxSpan, source uint64) uint64

func detectMax(span uint64, spans *ethpb.MinMaxSpan, source uint64) uint64 {
	maxSpan := uint64(spans.MaxSpan)
	if maxSpan > span {
		return maxSpan + source
	}
	return 0
}

func detectMin(span uint64, spans *ethpb.MinMaxSpan, source uint64) uint64 {
	minSpan := uint64(spans.MinSpan)
	if minSpan < span {
		return minSpan + source
	}
	return 0
}

func (ss *Server) detectSpan(source, target, validatorIdx uint64, detectionFunc detect) (uint64, uint64, *ethpb.EpochSpanMap, error) {
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
		return detectionFunc(span, spanMap.EpochSpanMap[source], source), span, spanMap, nil
	}
	return 0, span, spanMap, nil
}

// DetectAndUpdateMaxSpan is used to detect and update the max span of an incoming attestation.
// logic is following the detection method designed by https://github.com/protolambda
// from here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func (ss *Server) DetectAndUpdateMaxSpan(ctx context.Context, source uint64, target uint64, validatorIdx uint64) (uint64, error) {
	targetEpoch, span, spanMap, err := ss.detectSpan(source, target, validatorIdx, detectMax)
	if err != nil {
		return 0, err
	}
	if targetEpoch > 0 {
		return targetEpoch, nil
	}
	for i := uint64(1); i < target-source; i++ {
		val := uint32(span - i - 1)
		if _, ok := spanMap.EpochSpanMap[source+i]; !ok {
			spanMap.EpochSpanMap[source+i] = &ethpb.MinMaxSpan{}
		}
		if spanMap.EpochSpanMap[source+i].MaxSpan < val {
			spanMap.EpochSpanMap[source+i].MaxSpan = val
		} else {
			break
		}
	}
	if err := ss.SlasherDB.SaveValidatorSpansMap(validatorIdx, spanMap); err != nil {
		return 0, err
	}
	return 0, nil
}

// DetectAndUpdateMinSpan is used to detect surround and update the min span
// of an incoming attestation.
// logic is following the detection method designed by https://github.com/protolambda
// from here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func (ss *Server) DetectAndUpdateMinSpan(ctx context.Context, source uint64, target uint64, validatorIdx uint64) (uint64, error) {
	targetEpoch, _, spanMap, err := ss.detectSpan(source, target, validatorIdx, detectMin)
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
			spanMap.EpochSpanMap[i] = &ethpb.MinMaxSpan{}
		}
		if spanMap.EpochSpanMap[i].MinSpan == 0 || spanMap.EpochSpanMap[i].MinSpan > val {
			spanMap.EpochSpanMap[i].MinSpan = val
		} else {
			break
		}
	}
	if err := ss.SlasherDB.SaveValidatorSpansMap(validatorIdx, spanMap); err != nil {
		return 0, errors.Wrap(err, "could not save validator spans")
	}
	return 0, nil
}
