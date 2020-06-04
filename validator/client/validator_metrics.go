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
		"pubkey",
	},
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

	missingValidators := make(map[[48]byte]bool)
	for _, val := range resp.MissingValidators {
		missingValidators[bytesutil.ToBytes48(val)] = true
	}

	included := 0
	votedSource := 0
	votedTarget := 0
	votedHead := 0

	reported := 0
	for _, pkey := range pubKeys {
		pubKey := fmt.Sprintf("%#x", pkey[:8])
		log := log.WithField("pubKey", pubKey)
		fmtKey := fmt.Sprintf("%#x", pkey[:])
		if missingValidators[bytesutil.ToBytes48(pkey)] {
			if v.emitAccountMetrics {
				validatorBalancesGaugeVec.WithLabelValues(fmtKey).Set(0)
			}
			continue
		}
		if slot < params.BeaconConfig().SlotsPerEpoch {
			v.prevBalance[bytesutil.ToBytes48(pkey)] = params.BeaconConfig().MaxEffectiveBalance
		}

		if v.prevBalance[bytesutil.ToBytes48(pkey)] > 0 && len(resp.BalancesAfterEpochTransition) > reported {
			newBalance := float64(resp.BalancesAfterEpochTransition[reported]) / float64(params.BeaconConfig().GweiPerEth)
			prevBalance := float64(resp.BalancesBeforeEpochTransition[reported]) / float64(params.BeaconConfig().GweiPerEth)
			percentNet := (newBalance - prevBalance) / prevBalance
			log.WithFields(logrus.Fields{
				"epoch":                (slot / params.BeaconConfig().SlotsPerEpoch) - 1,
				"correctlyVotedSource": resp.CorrectlyVotedSource[reported],
				"correctlyVotedTarget": resp.CorrectlyVotedTarget[reported],
				"correctlyVotedHead":   resp.CorrectlyVotedHead[reported],
				"inclusionSlot":        resp.InclusionSlots[reported],
				"inclusionDistance":    resp.InclusionDistances[reported],
				"oldBalance":           prevBalance,
				"newBalance":           newBalance,
				"percentChange":        fmt.Sprintf("%.5f%%", percentNet*100),
			}).Info("Previous epoch voting summary")
			if v.emitAccountMetrics {
				validatorBalancesGaugeVec.WithLabelValues(fmtKey).Set(newBalance)
			}
		}

		if reported < len(resp.InclusionSlots) && resp.InclusionSlots[reported] != ^uint64(0) {
			included++
		}
		if reported < len(resp.CorrectlyVotedSource) && resp.CorrectlyVotedSource[reported] {
			votedSource++
		}
		if reported < len(resp.CorrectlyVotedTarget) && resp.CorrectlyVotedTarget[reported] {
			votedTarget++
		}
		if reported < len(resp.CorrectlyVotedHead) && resp.CorrectlyVotedHead[reported] {
			votedHead++
		}
		if reported < len(resp.BalancesAfterEpochTransition) {
			v.prevBalance[bytesutil.ToBytes48(pkey)] = resp.BalancesBeforeEpochTransition[reported]
		}

		reported++
	}

	log.WithFields(logrus.Fields{
		"epoch":                          (slot / params.BeaconConfig().SlotsPerEpoch) - 1,
		"attestationInclusionPercentage": fmt.Sprintf("%.0f%%", (float64(included)/float64(len(resp.InclusionSlots)))*100),
		"correctlyVotedSourcePercentage": fmt.Sprintf("%.0f%%", (float64(votedSource)/float64(len(resp.CorrectlyVotedSource)))*100),
		"correctlyVotedTargetPercentage": fmt.Sprintf("%.0f%%", (float64(votedTarget)/float64(len(resp.CorrectlyVotedTarget)))*100),
		"correctlyVotedHeadPercentage":   fmt.Sprintf("%.0f%%", (float64(votedHead)/float64(len(resp.CorrectlyVotedHead)))*100),
	}).Info("Previous epoch aggregated voting summary")

	return nil
}
