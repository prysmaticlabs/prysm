package monitor

import (
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/attestation"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
)

// processAttestation logs in the event that one of our tracked validators'
// appears in the attesting indices and this attestation was not included
// before.
func (s *Service) processAttestation(state state.BeaconState, att *ethpb.Attestation, included bool) {
	committee, err := helpers.BeaconCommitteeFromState(s.ctx, state, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		log.Error("Could not get beacon committee")
		return
	}
	attestingIndices, err := attestation.AttestingIndices(att.AggregationBits, committee)
	if err != nil {
		log.Error("Could not get attesting indices")
		return
	}
	for _, idx := range attestingIndices {
		if s.TrackedIndex(types.ValidatorIndex(idx)) &&
			s.latestPerformance[types.ValidatorIndex(idx)].attestedSlot != att.Data.Slot {
			latestPerf := s.latestPerformance[types.ValidatorIndex(idx)]
			aggregatedPerf := s.aggregatedPerformance[types.ValidatorIndex(idx)]

			balance, err := state.BalanceAtIndex(types.ValidatorIndex(idx))
			if err != nil {
				log.Error("Could not get balance")
				return
			}
			balanceChg := balance - latestPerf.balance

			logFields := logrus.Fields{
				"ValidatorIndex": idx,
				"Slot":           att.Data.Slot,
				"Source":         fmt.Sprintf("%#x", bytesutil.Trunc(att.Data.Source.Root)),
				"Target":         fmt.Sprintf("%#x", bytesutil.Trunc(att.Data.Target.Root)),
				"Head":           fmt.Sprintf("%#x", bytesutil.Trunc(att.Data.BeaconBlockRoot)),
				"NewBalance":     balance,
				"BalanceChange":  balanceChg,
			}
			var logMessage string
			if !included {
				logMessage = "Attestation processed"
			} else {
				aggregatedPerf.totalAttestedCount++
				aggregatedPerf.totalRequestedCount++

				latestPerf.balanceChange = balanceChg
				latestPerf.balance = balance
				latestPerf.attestedSlot = att.Data.Slot
				latestPerf.inclusionSlot = state.Slot()
				inclusionSlotGauge.WithLabelValues(fmt.Sprintf("%d", idx)).Set(float64(latestPerf.inclusionSlot))
				aggregatedPerf.totalDistance += uint64(latestPerf.inclusionSlot - latestPerf.attestedSlot)

				if state.Version() == version.Altair {
					targetIdx := params.BeaconConfig().TimelyTargetFlagIndex
					sourceIdx := params.BeaconConfig().TimelySourceFlagIndex
					headIdx := params.BeaconConfig().TimelyHeadFlagIndex

					var participation []byte
					if slots.ToEpoch(latestPerf.inclusionSlot) ==
						slots.ToEpoch(latestPerf.attestedSlot) {
						participation, err = state.CurrentEpochParticipation()
						if err != nil {
							log.WithError(err).Error("Could not get current epoch participation")
							return
						}
					} else {
						participation, err = state.PreviousEpochParticipation()
						if err != nil {
							log.WithError(err).Error("Could not get previous epoch participation")
							return
						}
					}
					flags := participation[idx]
					latestPerf.timelySource = ((flags >> sourceIdx) & 1) == 1
					latestPerf.timelyHead = ((flags >> headIdx) & 1) == 1
					latestPerf.timelyTarget = ((flags >> targetIdx) & 1) == 1

					if latestPerf.timelySource {
						aggregatedPerf.totalCorrectSource++
						timelySourceCounter.WithLabelValues(fmt.Sprintf("%d", idx)).Inc()
					}
					if latestPerf.timelyHead {
						aggregatedPerf.totalCorrectHead++
						timelyHeadCounter.WithLabelValues(fmt.Sprintf("%d", idx)).Inc()
					}
					if latestPerf.timelyTarget {
						aggregatedPerf.totalCorrectTarget++
						timelyTargetCounter.WithLabelValues(fmt.Sprintf("%d", idx)).Inc()
					}
				}
				logFields["CorrectHead"] = latestPerf.timelyHead
				logFields["CorrectSource"] = latestPerf.timelySource
				logFields["CorrectTarget"] = latestPerf.timelyTarget
				logFields["Inclusion Slot"] = latestPerf.inclusionSlot

				logMessage = "Attestation Included"

				// Only update performance on included attestations
				s.latestPerformance[types.ValidatorIndex(idx)] = latestPerf
				s.aggregatedPerformance[types.ValidatorIndex(idx)] = aggregatedPerf
			}
			log.WithFields(logFields).Info(logMessage)
		}
	}
}

// processUnaggregatedAttestation logs when the beacon node sees an unaggregated attestation from one of our
// tracked validators
func (s *Service) processUnaggregatedAttestation(att *ethpb.Attestation) {
	var root [32]byte
	copy(root[:], att.Data.BeaconBlockRoot)
	state, err := s.config.StateGen.StateByRootIfCached(s.ctx, root)
	if err != nil {
		log.WithError(err).Error("Could not query cache for state")
		return
	}
	if state == nil {
		log.Debug("Skipping Unnagregated Attestation due to state not found in cache")
		return
	}
	s.processAttestation(state, att, false)
}

// processAggregatedAttestation logs when we see an aggregation from one of our tracked validators or an aggregated
// attestation from one of our tracked validators
func (s *Service) processAggregatedAttestation(att *ethpb.AggregateAttestationAndProof) {
	if s.TrackedIndex(att.AggregatorIndex) {
		log.WithFields(logrus.Fields{
			"ValidatorIndex": att.AggregatorIndex,
		}).Info("Aggregated attestation processed")
		aggregatedPerf := s.aggregatedPerformance[att.AggregatorIndex]
		aggregatedPerf.totalAggregations++
		s.aggregatedPerformance[att.AggregatorIndex] = aggregatedPerf

		aggregationCounter.WithLabelValues(fmt.Sprintf("%d", att.AggregatorIndex)).Inc()
	}

	var root [32]byte
	copy(root[:], att.Aggregate.Data.BeaconBlockRoot)
	state, err := s.config.StateGen.StateByRootIfCached(s.ctx, root)
	if err != nil {
		log.WithError(err).Error("Could not query cache for state")
		return
	}
	if state == nil {
		log.Debug("Skipping Agregated Attestation due to state not found in cache")
		return
	}
	s.processAttestation(state, att.Aggregate, false)
}
