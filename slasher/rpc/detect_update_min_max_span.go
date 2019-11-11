package rpc

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// DetectAndUpdateMaxSpan is used to detect and update the max span of an incoming attestation.
// logic is following the detection method designed by https://github.com/protolambda
// from here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func (ss *Server) DetectAndUpdateMaxSpan(ctx context.Context, source uint64, target uint64, validatorIdx uint64) (surroundedTargetEpoch uint64, err error) {
	span := target - source + 1
	if span > params.BeaconConfig().WeakSubjectivityPeriod {
		return 0, fmt.Errorf("
		    %v target - source: %v > weakSubjectivityPeriod", 
		    params.BeaconConfig().WeakSubjectivityPeriod, 
		    span,
		)
	}
	spanMap, err := ss.SlasherDB.ValidatorSpansMap(validatorIdx)
	if err != nil {
		return 0, errors.Wrapf(err, "could not retrieve span map for validatorIdx: %v", validatorIdx)
	}
	if spanMap.EpochSpanMap == nil {
		spanMap.EpochSpanMap = make(map[uint64]*ethpb.MinMaxSpan)
	} else {
		_, ok := spanMap.EpochSpanMap[source]
		if ok {
			maxSpan := uint64(spanMap.EpochSpanMap[source].MaxSpan)
			if maxSpan > span {
				return maxSpan + source, nil
			}
		}
	}

	for i := uint64(1); i < target-source; i++ {
		val := uint32(span - i - 1)
		if spanMap.EpochSpanMap[source+i] == nil {
			spanMap.EpochSpanMap[source+i] = &ethpb.MinMaxSpan{MinSpan: 0, MaxSpan: 0}
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
func (ss *Server) DetectAndUpdateMinSpan(ctx context.Context, source uint64, target uint64, validatorIdx uint64) (surroundTargetEpoch uint64, err error) {
	span := target - source + 1
	if span > params.BeaconConfig().WeakSubjectivityPeriod {
		return 0, fmt.Errorf("attestation slashing detection supports only weak subjectivity period: %v target - source: %v > weakSubjectivityPeriod", params.BeaconConfig().WeakSubjectivityPeriod, span)
	}
	spanMap, err := ss.SlasherDB.ValidatorSpansMap(validatorIdx)
	if err != nil {
		return 0, errors.Wrapf(err, "could not retrieve span map for validatorIdx: %v", validatorIdx)
	}
	if spanMap.EpochSpanMap == nil {
		spanMap.EpochSpanMap = make(map[uint64]*ethpb.MinMaxSpan)
	} else {
		_, ok := spanMap.EpochSpanMap[source]
		if ok {
			minSpan := uint64(spanMap.EpochSpanMap[source].MinSpan)
			if minSpan < span {
				return minSpan + source, nil
			}
		}

	}
	for i := source - 1; i > 0; i-- {
		val := uint32(target - (i))
		if spanMap.EpochSpanMap[i] == nil {
			spanMap.EpochSpanMap[i] = &ethpb.MinMaxSpan{MinSpan: 0, MaxSpan: 0}
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
