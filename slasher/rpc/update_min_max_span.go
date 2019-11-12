package rpc

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// UpdateMaxSpan is used to update the max span of an incoming attestation after the slashing detection phase.
// logic is following the detection method designed by https://github.com/protolambda
// from here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func (ss *Server) UpdateMaxSpan(ctx context.Context, source uint64, target uint64, validatorIdx uint64) error {
	diff := target - source
	if diff > params.BeaconConfig().WeakSubjectivityPeriod {
		return fmt.Errorf("%d target - source: %d > weakSubjectivityPeriod",
			params.BeaconConfig().WeakSubjectivityPeriod,
			diff,
		)
	}
	spanMap, err := ss.SlasherDB.ValidatorSpansMap(validatorIdx)
	if err != nil {
		return errors.Wrapf(err, "could not retrieve span map for validatorIdx: %v", validatorIdx)
	}
	for i := uint64(1); i < target-source; i++ {
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
	if err := ss.SlasherDB.SaveValidatorSpansMap(validatorIdx, spanMap); err != nil {
		return err
	}
	return nil
}

// UpdateMinSpan is used to update the min span of an incoming attestation after the slashing detection phase.
// logic is following the detection method designed by https://github.com/protolambda
// from here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func (ss *Server) UpdateMinSpan(ctx context.Context, source uint64, target uint64, validatorIdx uint64) error {
	diff := target - source
	if diff > params.BeaconConfig().WeakSubjectivityPeriod {
		return fmt.Errorf("%d target - source: %d > weakSubjectivityPeriod",
			params.BeaconConfig().WeakSubjectivityPeriod,
			diff,
		)
	}
	spanMap, err := ss.SlasherDB.ValidatorSpansMap(validatorIdx)
	if err != nil {
		return errors.Wrapf(err, "could not retrieve span map for validatorIdx: %d", validatorIdx)
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
		errors.Wrap(err, "could not save validator spans")
	}
	return nil
}
