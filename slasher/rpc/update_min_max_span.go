package rpc

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/params"

	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// UpdateMaxSpan is used to update the max span of an incoming attestation after the slashing detection phase.
// logic is following protolambda detection method.
// from here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func (ss *Server) UpdateMaxSpan(source uint64, target uint64, validatorIdx uint64) error {
	spanMap, err := ss.SlasherDb.ValidatorSpansMap(validatorIdx)
	if err != nil {
		return errors.Wrapf(err, "couldnt retrieve span map for validatorIdx: %v", validatorIdx)
	}
	if spanMap.EpochSpanMap == nil {
		spanMap.EpochSpanMap = make(map[uint64]*ethpb.MinMaxSpan)
	}
	for i := uint64(1); i < target-source; i++ {
		diff := target - source
		if diff > params.BeaconConfig().WeakSubjectivityPeriod {
			return fmt.Errorf("attestation detection supports only weak subjectivity period: %v target - source: %v > weakSubjectivityPeriod", params.BeaconConfig().WeakSubjectivityPeriod, diff)
		}
		val := uint32(diff - i)
		if spanMap.EpochSpanMap[source+i] == nil {
			spanMap.EpochSpanMap[source+i] = &ethpb.MinMaxSpan{MinSpan: 0, MaxSpan: 0}
		}
		if spanMap.EpochSpanMap[source+i].MaxSpan < val {
			spanMap.EpochSpanMap[source+i].MaxSpan = val
		} else {
			break
		}
	}
	if err := ss.SlasherDb.SaveValidatorSpansMap(validatorIdx, spanMap); err != nil {
		errors.Wrap(err, "Got error while trying to save validator spans")
	}
	return nil
}

// UpdateMinSpan is used to update the min span of an incoming attestation after the slashing detection phase.
// logic is following protolambda detection method.
// from here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func (ss *Server) UpdateMinSpan(source uint64, target uint64, validatorIdx uint64) error {
	spanMap, err := ss.SlasherDb.ValidatorSpansMap(validatorIdx)
	if err != nil {
		return errors.Wrapf(err, "couldn't retrieve span map for validatorIdx: %v", validatorIdx)
	}
	if spanMap.EpochSpanMap == nil {
		spanMap.EpochSpanMap = make(map[uint64]*ethpb.MinMaxSpan)
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
	if err := ss.SlasherDb.SaveValidatorSpansMap(validatorIdx, spanMap); err != nil {
		errors.Wrap(err, "Got error while trying to save validator spans")
	}
	return nil
}
