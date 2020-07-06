package client

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
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
			// Validator pubkey.
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
			// validator pubkey
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
			// validator pubkey
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
			// validator pubkey
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
			// validator pubkey
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
			// validator pubkey
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
			// validator pubkey
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
			// validator pubkey
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
			// validator pubkey
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
			// validator pubkey
			"pubkey",
		},
	)
)

// LogValidatorGainsAndLosses logs important metrics related to this validator client's
// responsibilities throughout the beacon chain's lifecycle. It logs absolute accrued rewards
// and penalties over time, percentage gain/loss, and gives the end user a better idea
// of how the validator performs with respect to the rest.
func (v *validator) LogValidatorGainsAndLosses(ctx context.Context, slot uint64) error {
	if slot%params.BeaconConfig().SlotsPerEpoch != 0 || slot <= params.BeaconConfig().SlotsPerEpoch {
		// Do nothing unless we are at the start of the epoch, and not in the first epoch.
		return nil
	}
	if !v.logValidatorBalances {
		return nil
	}

	pks, err := v.keyManager.FetchValidatingKeys()
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
			fmtKey := fmt.Sprintf("%#x", missingPubKey[:])
			ValidatorBalancesGaugeVec.WithLabelValues(fmtKey).Set(0)
		}
	}

	prevEpoch := uint64(0)
	if slot >= params.BeaconConfig().SlotsPerEpoch {
		prevEpoch = (slot / params.BeaconConfig().SlotsPerEpoch) - 1
		if v.voteStats.startEpoch == 0 {
			//prevEpoch may be 0 so we store startEpoch (+1). Thus a value of 0 tells us it is not set (default)
			v.voteStats.startEpoch = prevEpoch + 1
		}
	}
	gweiPerEth := float64(params.BeaconConfig().GweiPerEth)
	for i, pubKey := range resp.PublicKeys {
		pubKeyBytes := bytesutil.ToBytes48(pubKey)
		if slot < params.BeaconConfig().SlotsPerEpoch {
			v.prevBalance[pubKeyBytes] = params.BeaconConfig().MaxEffectiveBalance
		}
		if _, ok := v.startBalances[pubKeyBytes]; ok == false {
			v.startBalances[pubKeyBytes] = resp.BalancesBeforeEpochTransition[i]
		}

		fmtKey := fmt.Sprintf("%#x", pubKey)
		truncatedKey := fmt.Sprintf("%#x", pubKey[:8])
		if v.prevBalance[pubKeyBytes] > 0 {
			newBalance := float64(resp.BalancesAfterEpochTransition[i]) / gweiPerEth
			prevBalance := float64(resp.BalancesBeforeEpochTransition[i]) / gweiPerEth
			startBalance := float64(v.startBalances[pubKeyBytes]) / gweiPerEth
			percentNet := (newBalance - prevBalance) / prevBalance
			percentSinceStart := (newBalance - startBalance) / startBalance
			log.WithFields(logrus.Fields{
				"pubKey":                  truncatedKey,
				"epoch":                   prevEpoch,
				"correctlyVotedSource":    resp.CorrectlyVotedSource[i],
				"correctlyVotedTarget":    resp.CorrectlyVotedTarget[i],
				"correctlyVotedHead":      resp.CorrectlyVotedHead[i],
				"inclusionSlot":           resp.InclusionSlots[i],
				"inclusionDistance":       resp.InclusionDistances[i],
				"startBalance":            startBalance,
				"oldBalance":              prevBalance,
				"newBalance":              newBalance,
				"percentChange":           fmt.Sprintf("%.5f%%", percentNet*100),
				"percentChangeSinceStart": fmt.Sprintf("%.5f%%", percentSinceStart*100),
			}).Info("Previous epoch voting summary")
			if v.emitAccountMetrics {
				ValidatorBalancesGaugeVec.WithLabelValues(fmtKey).Set(newBalance)
			}
		}
		v.prevBalance[pubKeyBytes] = resp.BalancesBeforeEpochTransition[i]
	}

	v.UpdateLogAggregateStats(resp, slot)
	return nil
}

//UpdateLogAggregateStats updates and logs the voteStats struct of a validator using the RPC response obtained during LogValidatorGainsAndLosses
func (v *validator) UpdateLogAggregateStats(resp *ethpb.ValidatorPerformanceResponse, slot uint64) {
	summary := &v.voteStats
	currentEpoch := slot / params.BeaconConfig().SlotsPerEpoch
	var included, correctSource, correctTarget, correctHead int

	for i := range resp.PublicKeys {
		if resp.InclusionSlots[i] != ^uint64(0) {
			included++
			summary.includedAttestedCount++
			summary.totalDistance += resp.InclusionDistances[i]
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
	}

	summary.totalAttestedCount += uint64(len(resp.InclusionSlots))
	summary.totalSources += uint64(included)
	summary.totalTargets += uint64(included)
	summary.totalHeads += uint64(included)

	log.WithFields(logrus.Fields{
		"epoch":                   currentEpoch - 1,
		"attestationInclusionPct": fmt.Sprintf("%.0f%%", (float64(included)/float64(len(resp.InclusionSlots)))*100),
		"correctlyVotedSourcePct": fmt.Sprintf("%.0f%%", (float64(correctSource)/float64(included))*100),
		"correctlyVotedTargetPct": fmt.Sprintf("%.0f%%", (float64(correctTarget)/float64(included))*100),
		"correctlyVotedHeadPct":   fmt.Sprintf("%.0f%%", (float64(correctHead)/float64(included))*100),
	}).Info("Previous epoch aggregated voting summary")

	var totalStartBal, totalPrevBal uint64
	for i, val := range v.startBalances {
		totalStartBal += val
		totalPrevBal += v.prevBalance[i]
	}

	log.WithFields(logrus.Fields{
		"numberOfEpochs":           fmt.Sprintf("%d", currentEpoch-summary.startEpoch+1), //+1 to include the original epoch
		"attestationsInclusionPct": fmt.Sprintf("%.0f%%", (float64(summary.includedAttestedCount)/float64(summary.totalAttestedCount))*100),
		"averageInclusionDistance": fmt.Sprintf("%.2f slots", float64(summary.totalDistance)/float64(summary.includedAttestedCount)),
		"correctlyVotedSourcePct":  fmt.Sprintf("%.0f%%", (float64(summary.correctSources)/float64(summary.totalSources))*100),
		"correctlyVotedTargetPct":  fmt.Sprintf("%.0f%%", (float64(summary.correctTargets)/float64(summary.totalTargets))*100),
		"correctlyVotedHeadPct":    fmt.Sprintf("%.0f%%", (float64(summary.correctHeads)/float64(summary.totalHeads))*100),
		"pctChangeCombinedBalance": fmt.Sprintf("%.5f%%", (float64(totalPrevBal)-float64(totalStartBal))/float64(totalStartBal)*100),
	}).Info("Vote summary since launch")
}
