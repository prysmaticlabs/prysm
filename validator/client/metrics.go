package client

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var (
	// ValidatorStatusesGaugeVec used to track validator statuses by public key.
	ValidatorStatusesGaugeVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "validator",
			Name:      "statuses",
			Help:      "validator statuses: 0 UNKNOWN, 1 DEPOSITED, 2 PENDING, 3 ACTIVE, 4 EXITING, 5 SLASHING, 6 EXITED",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorAggSuccessVec used to count successful aggregations.
	ValidatorAggSuccessVec = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator",
			Name:      "successful_aggregations",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorAggFailVec used to count failed aggregations.
	ValidatorAggFailVec = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator",
			Name:      "failed_aggregations",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorProposeSuccessVec used to count successful proposals.
	ValidatorProposeSuccessVec = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator",
			Name:      "successful_proposals",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorProposeFailVec used to count failed proposals.
	ValidatorProposeFailVec = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator",
			Name:      "failed_proposals",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorProposeFailVecSlasher used to count failed proposals by slashing protection.
	ValidatorProposeFailVecSlasher = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "validator_proposals_rejected_total",
			Help: "Count the block proposals rejected by slashing protection.",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorBalancesGaugeVec used to keep track of validator balances by public key.
	ValidatorBalancesGaugeVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "validator",
			Name:      "balance",
			Help:      "current validator balance.",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorInclusionDistancesGaugeVec used to keep track of validator inclusion distances by public key.
	ValidatorInclusionDistancesGaugeVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "validator",
			Name:      "inclusion_distance",
			Help:      "Inclusion distance of last attestation. Deprecated after Altair hard fork.",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorAttestedSlotsGaugeVec used to keep track of validator attested slots by public key.
	ValidatorAttestedSlotsGaugeVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "validator",
			Name:      "last_attested_slot",
			Help:      "Last attested slot.",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorCorrectlyVotedSourceGaugeVec used to keep track of validator's accuracy on voting source by public key.
	ValidatorCorrectlyVotedSourceGaugeVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "validator",
			Name:      "correctly_voted_source",
			Help:      "True if correctly voted source in last attestation.",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorCorrectlyVotedTargetGaugeVec used to keep track of validator's accuracy on voting target by public key.
	ValidatorCorrectlyVotedTargetGaugeVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "validator",
			Name:      "correctly_voted_target",
			Help:      "True if correctly voted target in last attestation.",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorCorrectlyVotedHeadGaugeVec used to keep track of validator's accuracy on voting head by public key.
	ValidatorCorrectlyVotedHeadGaugeVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "validator",
			Name:      "correctly_voted_head",
			Help:      "True if correctly voted head in last attestation.",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorAttestSuccessVec used to count successful attestations.
	ValidatorAttestSuccessVec = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator",
			Name:      "successful_attestations",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorAttestFailVec used to count failed attestations.
	ValidatorAttestFailVec = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator",
			Name:      "failed_attestations",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorAttestFailVecSlasher used to count failed attestations by slashing protection.
	ValidatorAttestFailVecSlasher = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "validator_attestations_rejected_total",
			Help: "Count the attestations rejected by slashing protection.",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorNextAttestationSlotGaugeVec used to track validator statuses by public key.
	ValidatorNextAttestationSlotGaugeVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "validator",
			Name:      "next_attestation_slot",
			Help:      "validator next scheduled attestation slot",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorNextProposalSlotGaugeVec used to track validator statuses by public key.
	ValidatorNextProposalSlotGaugeVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "validator",
			Name:      "next_proposal_slot",
			Help:      "validator next scheduled proposal slot",
		},
		[]string{
			"pubkey",
		},
	)
	// ValidatorInactivityScoreGaugeVec used to track validator inactivity scores.
	ValidatorInactivityScoreGaugeVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "validator",
			Name:      "inactivity_score",
			Help:      "Validator inactivity score. 0 is optimum number. New in Altair hardfork",
		},
		[]string{
			"pubkey",
		},
	)
)

// LogValidatorGainsAndLosses logs important metrics related to this validator client's
// responsibilities throughout the beacon chain's lifecycle. It logs absolute accrued rewards
// and penalties over time, percentage gain/loss, and gives the end user a better idea
// of how the validator performs with respect to the rest.
func (v *validator) LogValidatorGainsAndLosses(ctx context.Context, slot types.Slot) error {
	if !core.IsEpochEnd(slot) || slot <= params.BeaconConfig().SlotsPerEpoch {
		// Do nothing unless we are at the end of the epoch, and not in the first epoch.
		return nil
	}
	if !v.logValidatorBalances {
		return nil
	}

	var pks [][48]byte
	var err error
	pks, err = v.keyManager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return err
	}
	pubKeys := bytesutil.FromBytes48Array(pks)

	req := &ethpb.ValidatorPerformanceRequest{
		PublicKeys: pubKeys,
	}
	resp, err := v.beaconClient.GetValidatorPerformance(ctx, req)
	if err != nil {
		return err
	}

	if v.emitAccountMetrics {
		for _, missingPubKey := range resp.MissingValidators {
			fmtKey := fmt.Sprintf("%#x", missingPubKey)
			ValidatorBalancesGaugeVec.WithLabelValues(fmtKey).Set(0)
		}
	}

	prevEpoch := types.Epoch(0)
	if slot >= params.BeaconConfig().SlotsPerEpoch {
		prevEpoch = types.Epoch(slot/params.BeaconConfig().SlotsPerEpoch) - 1
		if uint64(v.voteStats.startEpoch) == ^uint64(0) { // Handles unknown first epoch.
			v.voteStats.startEpoch = prevEpoch
		}
	}
	gweiPerEth := float64(params.BeaconConfig().GweiPerEth)
	v.prevBalanceLock.Lock()
	for i, pubKey := range resp.PublicKeys {
		pubKeyBytes := bytesutil.ToBytes48(pubKey)
		if slot < params.BeaconConfig().SlotsPerEpoch {
			v.prevBalance[pubKeyBytes] = params.BeaconConfig().MaxEffectiveBalance
		}
		if _, ok := v.startBalances[pubKeyBytes]; !ok {
			v.startBalances[pubKeyBytes] = resp.BalancesBeforeEpochTransition[i]
		}

		fmtKey := fmt.Sprintf("%#x", pubKey)
		truncatedKey := fmt.Sprintf("%#x", bytesutil.Trunc(pubKey))
		if v.prevBalance[pubKeyBytes] > 0 {
			newBalance := float64(resp.BalancesAfterEpochTransition[i]) / gweiPerEth
			prevBalance := float64(resp.BalancesBeforeEpochTransition[i]) / gweiPerEth
			startBalance := float64(v.startBalances[pubKeyBytes]) / gweiPerEth
			percentNet := (newBalance - prevBalance) / prevBalance
			percentSinceStart := (newBalance - startBalance) / startBalance

			log := log.WithFields(logrus.Fields{
				"pubKey":                  truncatedKey,
				"epoch":                   prevEpoch,
				"correctlyVotedSource":    resp.CorrectlyVotedSource[i],
				"correctlyVotedTarget":    resp.CorrectlyVotedTarget[i],
				"correctlyVotedHead":      resp.CorrectlyVotedHead[i],
				"startBalance":            startBalance,
				"oldBalance":              prevBalance,
				"newBalance":              newBalance,
				"percentChange":           fmt.Sprintf("%.5f%%", percentNet*100),
				"percentChangeSinceStart": fmt.Sprintf("%.5f%%", percentSinceStart*100),
			})

			// These fields are deprecated after Altair.
			if core.SlotToEpoch(slot) < params.BeaconConfig().AltairForkEpoch {
				log = log.WithFields(logrus.Fields{
					"inclusionSlot":     resp.InclusionSlots[i],
					"inclusionDistance": resp.InclusionDistances[i],
				})
			}

			log.Info("Previous epoch voting summary")
			if v.emitAccountMetrics {
				ValidatorBalancesGaugeVec.WithLabelValues(fmtKey).Set(newBalance)
				if resp.CorrectlyVotedSource[i] {
					ValidatorCorrectlyVotedSourceGaugeVec.WithLabelValues(fmtKey).Set(1)
				} else {
					ValidatorCorrectlyVotedSourceGaugeVec.WithLabelValues(fmtKey).Set(0)
				}
				if resp.CorrectlyVotedTarget[i] {
					ValidatorCorrectlyVotedTargetGaugeVec.WithLabelValues(fmtKey).Set(1)
				} else {
					ValidatorCorrectlyVotedTargetGaugeVec.WithLabelValues(fmtKey).Set(0)
				}
				if resp.CorrectlyVotedHead[i] {
					ValidatorCorrectlyVotedHeadGaugeVec.WithLabelValues(fmtKey).Set(1)
				} else {
					ValidatorCorrectlyVotedHeadGaugeVec.WithLabelValues(fmtKey).Set(0)
				}

				// Phase0 specific metrics
				if core.SlotToEpoch(slot) < params.BeaconConfig().AltairForkEpoch {
					ValidatorInclusionDistancesGaugeVec.WithLabelValues(fmtKey).Set(float64(resp.InclusionDistances[i]))
				} else { // Altair specific metrics.
					// Reset phase0 fields that no longer apply
					ValidatorInclusionDistancesGaugeVec.DeleteLabelValues(fmtKey)

					ValidatorInactivityScoreGaugeVec.WithLabelValues(fmtKey).Set(float64(resp.InactivityScores[i]))
				}
			}
		}
		v.prevBalance[pubKeyBytes] = resp.BalancesBeforeEpochTransition[i]
	}
	v.prevBalanceLock.Unlock()

	v.UpdateLogAggregateStats(resp, slot)
	return nil
}

// UpdateLogAggregateStats updates and logs the voteStats struct of a validator using the RPC response obtained from LogValidatorGainsAndLosses.
func (v *validator) UpdateLogAggregateStats(resp *ethpb.ValidatorPerformanceResponse, slot types.Slot) {
	summary := &v.voteStats
	currentEpoch := types.Epoch(slot / params.BeaconConfig().SlotsPerEpoch)
	var included uint64
	var correctSource, correctTarget, correctHead, inactivityScore int

	for i := range resp.PublicKeys {
		// In phase0, we consider attestations included if the inclusion slot is not max uint64.
		// In altair, we consider attestations included if correctlyVotedTarget is true.
		if core.SlotToEpoch(slot) < params.BeaconConfig().AltairForkEpoch {
			if uint64(resp.InclusionSlots[i]) != ^uint64(0) {
				included++
				summary.includedAttestedCount++
				summary.totalDistance += resp.InclusionDistances[i]
			}
		} else if resp.CorrectlyVotedTarget[i] {
			included++
			summary.includedAttestedCount++
		}

		if resp.CorrectlyVotedSource[i] {
			correctSource++
			summary.correctSources++
		}
		if resp.CorrectlyVotedTarget[i] {
			correctTarget++
			summary.correctTargets++
		}
		if resp.CorrectlyVotedHead[i] {
			correctHead++
			summary.correctHeads++
		}

		// Altair metrics
		if core.SlotToEpoch(slot) > params.BeaconConfig().AltairForkEpoch {
			inactivityScore += int(resp.InactivityScores[i])
		}
	}

	// Return early if no attestation got included from previous epoch.
	// This happens when validators joined half way through epoch and already passed its assigned slot.
	if included == 0 {
		return
	}

	summary.totalAttestedCount += uint64(len(resp.CorrectlyVotedTarget))
	summary.totalSources += included
	summary.totalTargets += included
	summary.totalHeads += included

	epochSummaryFields := logrus.Fields{
		"epoch":                   currentEpoch - 1,
		"attestationInclusionPct": fmt.Sprintf("%.0f%%", (float64(included)/float64(len(resp.CorrectlyVotedTarget)))*100),
		"correctlyVotedSourcePct": fmt.Sprintf("%.0f%%", (float64(correctSource)/float64(included))*100),
		"correctlyVotedTargetPct": fmt.Sprintf("%.0f%%", (float64(correctTarget)/float64(included))*100),
		"correctlyVotedHeadPct":   fmt.Sprintf("%.0f%%", (float64(correctHead)/float64(included))*100),
	}

	// Altair summary fields.
	if core.SlotToEpoch(slot) > params.BeaconConfig().AltairForkEpoch {
		epochSummaryFields["averageInactivityScore"] = fmt.Sprintf("%.0f", float64(inactivityScore)/float64(len(resp.CorrectlyVotedTarget)))
	}

	log.WithFields(epochSummaryFields).Info("Previous epoch aggregated voting summary")

	var totalStartBal, totalPrevBal uint64
	for i, val := range v.startBalances {
		totalStartBal += val
		totalPrevBal += v.prevBalance[i]
	}

	launchSummaryFields := logrus.Fields{
		"numberOfEpochs":           fmt.Sprintf("%d", currentEpoch-summary.startEpoch),
		"attestationsInclusionPct": fmt.Sprintf("%.0f%%", (float64(summary.includedAttestedCount)/float64(summary.totalAttestedCount))*100),
		"correctlyVotedSourcePct":  fmt.Sprintf("%.0f%%", (float64(summary.correctSources)/float64(summary.totalSources))*100),
		"correctlyVotedTargetPct":  fmt.Sprintf("%.0f%%", (float64(summary.correctTargets)/float64(summary.totalTargets))*100),
		"correctlyVotedHeadPct":    fmt.Sprintf("%.0f%%", (float64(summary.correctHeads)/float64(summary.totalHeads))*100),
		"pctChangeCombinedBalance": fmt.Sprintf("%.5f%%", (float64(totalPrevBal)-float64(totalStartBal))/float64(totalStartBal)*100),
	}

	// Add phase0 specific fields
	if core.SlotToEpoch(slot) < params.BeaconConfig().AltairForkEpoch {
		launchSummaryFields["averageInclusionDistance"] = fmt.Sprintf("%.2f slots", float64(summary.totalDistance)/float64(summary.includedAttestedCount))
	}

	log.WithFields(launchSummaryFields).Info("Vote summary since launch")
}
