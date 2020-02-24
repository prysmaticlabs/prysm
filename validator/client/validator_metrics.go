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

var validatorBalancesGaugeVec = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: "validator",
		Name:      "balance",
		Help:      "current validator balance.",
	},
	[]string{
		// validator pubkey
		"pkey",
	},
)

// LogValidatorGainsAndLosses logs important metrics related to this validator client's
// responsibilities throughout the beacon chain's lifecycle. It logs absolute accrued rewards
// and penalties over time, percentage gain/loss, and gives the end user a better idea
// of how the validator performs with respect to the rest.
func (v *validator) LogValidatorGainsAndLosses(ctx context.Context, slot uint64) error {
	if slot%params.BeaconConfig().SlotsPerEpoch != 0 || slot <= params.BeaconConfig().SlotsPerEpoch {
		// Do nothing if we are not at the start of a new epoch and before the first epoch.
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

	missingValidators := make(map[[48]byte]bool)
	for _, val := range resp.MissingValidators {
		missingValidators[bytesutil.ToBytes48(val)] = true
	}

	included := 0
	votedSource := 0
	votedTarget := 0
	votedHead := 0

	reported := 0
	for i, pkey := range pubKeys {
		pubKey := fmt.Sprintf("%#x", pkey[:8])
		log := log.WithField("pubKey", pubKey)
		if missingValidators[bytesutil.ToBytes48(pkey)] {
			log.Info("Validator not in beacon chain")
			if v.emitAccountMetrics {
				validatorBalancesGaugeVec.WithLabelValues(pubKey).Set(0)
			}
			continue
		}
		if slot < params.BeaconConfig().SlotsPerEpoch {
			v.prevBalance[bytesutil.ToBytes48(pkey)] = params.BeaconConfig().MaxEffectiveBalance
		}

		if v.prevBalance[bytesutil.ToBytes48(pkey)] > 0 && len(resp.BalancesAfterEpochTransition) > i {
			newBalance := float64(resp.BalancesAfterEpochTransition[i]) / float64(params.BeaconConfig().GweiPerEth)
			prevBalance := float64(resp.BalancesBeforeEpochTransition[i]) / float64(params.BeaconConfig().GweiPerEth)
			percentNet := (newBalance - prevBalance) / prevBalance
			log.WithFields(logrus.Fields{
				"epoch":                (slot / params.BeaconConfig().SlotsPerEpoch) - 1,
				"correctlyVotedSource": resp.CorrectlyVotedSource[i],
				"correctlyVotedTarget": resp.CorrectlyVotedTarget[i],
				"correctlyVotedHead":   resp.CorrectlyVotedHead[i],
				"inclusionSlot":        resp.InclusionSlots[i],
				"inclusionDistance":    resp.InclusionDistances[i],
				"oldBalance":           prevBalance,
				"newBalance":           newBalance,
				"percentChange":        fmt.Sprintf("%.5f%%", percentNet*100),
			}).Info("Previous epoch voting summary")
			if v.emitAccountMetrics {
				validatorBalancesGaugeVec.WithLabelValues(pubKey).Set(newBalance)
			}
		}

		if reported < len(resp.InclusionSlots) && resp.InclusionSlots[i] != ^uint64(0) {
			included++
		}
		if reported < len(resp.CorrectlyVotedSource) && resp.CorrectlyVotedSource[i] {
			votedSource++
		}
		if reported < len(resp.CorrectlyVotedTarget) && resp.CorrectlyVotedTarget[i] {
			votedTarget++
		}
		if reported < len(resp.CorrectlyVotedHead) && resp.CorrectlyVotedHead[i] {
			votedHead++
		}
		if reported < len(resp.BalancesAfterEpochTransition) {
			v.prevBalance[bytesutil.ToBytes48(pkey)] = resp.BalancesBeforeEpochTransition[i]
		}

		reported++
	}

	log.WithFields(logrus.Fields{
		"epoch":                          (slot / params.BeaconConfig().SlotsPerEpoch) - 1,
		"attestationInclusionPercentage": fmt.Sprintf("%.2f", float64(included)/float64(len(resp.InclusionSlots))),
		"correctlyVotedSourcePercentage": fmt.Sprintf("%.2f", float64(votedSource)/float64(len(resp.CorrectlyVotedSource))),
		"correctlyVotedTargetPercentage": fmt.Sprintf("%.2f", float64(votedTarget)/float64(len(resp.CorrectlyVotedTarget))),
		"correctlyVotedHeadPercentage":   fmt.Sprintf("%.2f", float64(votedHead)/float64(len(resp.CorrectlyVotedHead))),
	}).Info("Previous epoch aggregated voting summary")

	return nil
}
