package attestations

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"

	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// SlashingDetector is a function type used to implement the slashable surrounding/surrounded
// vote detection methods.
type detectFn = func(attestationEpochSpan uint64, recorderEpochSpan *slashpb.MinMaxEpochSpan, sourceEpoch uint64) uint64

// detectSurroundedAtt is a function for maxDetector used to detect surrounding attestations.
func detectSurroundedAtt(
	attestationEpochSpan uint64,
	recorderEpochSpan *slashpb.MinMaxEpochSpan,
	attestationSourceEpoch uint64,
) uint64 {
	maxSpan := uint64(recorderEpochSpan.MaxEpochSpan)
	if maxSpan > attestationEpochSpan {
		return maxSpan + attestationSourceEpoch
	}
	return 0
}

// detectSurroundAtt is a function for minDetecter used to detect surrounded attestations.
func detectSurroundAtt(
	attestationEpochSpan uint64,
	recorderEpochSpan *slashpb.MinMaxEpochSpan,
	attestationSourceEpoch uint64,
) uint64 {
	minSpan := uint64(recorderEpochSpan.MinEpochSpan)
	if minSpan > 0 && minSpan < attestationEpochSpan {
		return minSpan + attestationSourceEpoch
	}
	return 0
}

// DetectSurroundedAttestations is used to detect and update the max span of an incoming attestation.
// This is used for detecting surrounding votes.
// The max span is the span between the current attestation's source epoch and the furthest attestation's
// target epoch that has a lower (earlier) source epoch.
// Logic for this detection method was designed by https://github.com/protolambda
// Detailed here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func (d *AttDetector) DetectSurroundedAttestations(
	ctx context.Context,
	source uint64,
	target uint64,
	validatorIdx uint64,
	spanMap *slashpb.EpochSpanMap,
) (uint64, *slashpb.EpochSpanMap, error) {
	if target < source {
		return 0, nil, fmt.Errorf(
			"target: %d < source: %d ",
			target,
			source,
		)
	}
	targetEpoch, span, spanMap, err := d.detectSlashingByEpochSpan(source, target, spanMap, detectSurroundedAtt)
	if err != nil {
		return 0, nil, err
	}
	if targetEpoch > 0 {
		return targetEpoch, spanMap, nil
	}

	for i := uint64(1); i < target-source; i++ {
		val := uint32(span - i)
		if _, ok := spanMap.EpochSpanMap[source+i]; !ok {
			spanMap.EpochSpanMap[source+i] = &slashpb.MinMaxEpochSpan{}
		}
		if spanMap.EpochSpanMap[source+i].MaxEpochSpan < val {
			spanMap.EpochSpanMap[source+i].MaxEpochSpan = val
		} else {
			break
		}
	}
	if err := d.slashingDetector.SlasherDB.SaveValidatorSpansMap(validatorIdx, spanMap); err != nil {
		return 0, nil, err
	}
	return 0, spanMap, nil
}

// DetectSurroundAttestation is used to detect surrounded votes and update the min epoch span
// of an incoming attestation.
// The min span is the span between the current attestations target epoch and the
// closest attestation's target distance.
//
// Logic is following the detection method designed by https://github.com/protolambda
// Detailed here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
func (d *AttDetector) DetectSurroundAttestation(
	ctx context.Context,
	source uint64,
	target uint64,
	validatorIdx uint64,
	spanMap *slashpb.EpochSpanMap,
) (uint64, *slashpb.EpochSpanMap, error) {
	if target < source {
		return 0, nil, fmt.Errorf(
			"target: %d < source: %d ",
			target,
			source,
		)
	}
	targetEpoch, _, spanMap, err := d.detectSlashingByEpochSpan(source, target, spanMap, detectSurroundAtt)
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
func (d *AttDetector) detectSlashingByEpochSpan(
	source,
	target uint64,
	spanMap *slashpb.EpochSpanMap,
	detector detectFn,
) (uint64, uint64, *slashpb.EpochSpanMap, error) {
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

// DetectSurroundVotes is a method used to return the attestation that were detected
// by min max surround detection method.
func (d *AttDetector) DetectSurroundVotes(ctx context.Context, validatorIdx uint64, req *ethpb.IndexedAttestation) ([]*ethpb.AttesterSlashing, error) {
	spanMap, err := d.slashingDetector.SlasherDB.ValidatorSpansMap(validatorIdx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get validator spans map")
	}
	minTargetEpoch, spanMap, err := d.DetectSurroundAttestation(ctx, req.Data.Source.Epoch, req.Data.Target.Epoch, validatorIdx, spanMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update min spans")
	}
	maxTargetEpoch, spanMap, err := d.DetectSurroundedAttestations(ctx, req.Data.Source.Epoch, req.Data.Target.Epoch, validatorIdx, spanMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update max spans")
	}
	if err := d.slashingDetector.SlasherDB.SaveValidatorSpansMap(validatorIdx, spanMap); err != nil {
		return nil, errors.Wrap(err, "failed to save validator spans map")
	}

	var as []*ethpb.AttesterSlashing
	if minTargetEpoch > 0 {
		attestations, err := d.slashingDetector.SlasherDB.IdxAttsForTargetFromID(minTargetEpoch, validatorIdx)
		if err != nil {
			return nil, err
		}
		for _, ia := range attestations {
			if ia.Data == nil {
				continue
			}
			if ia.Data.Source.Epoch > req.Data.Source.Epoch && ia.Data.Target.Epoch < req.Data.Target.Epoch {
				as = append(as, &ethpb.AttesterSlashing{
					Attestation_1: req,
					Attestation_2: ia,
				})
			}
		}
	}
	if maxTargetEpoch > 0 {
		attestations, err := d.slashingDetector.SlasherDB.IdxAttsForTargetFromID(maxTargetEpoch, validatorIdx)
		if err != nil {
			return nil, err
		}
		for _, ia := range attestations {
			if ia.Data.Source.Epoch < req.Data.Source.Epoch && ia.Data.Target.Epoch > req.Data.Target.Epoch {
				as = append(as, &ethpb.AttesterSlashing{
					Attestation_1: req,
					Attestation_2: ia,
				})
			}
		}
	}
	return as, nil
}
