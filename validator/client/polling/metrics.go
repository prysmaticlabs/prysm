package polling

import (
	"context"
	"fmt"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/client/metrics"
	"github.com/sirupsen/logrus"
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
			metrics.ValidatorBalancesGaugeVec.WithLabelValues(fmtKey).Set(0)
		}
	}

	included := 0
	votedSource := 0
	votedTarget := 0
	votedHead := 0
	prevEpoch := uint64(0)
	if slot >= params.BeaconConfig().SlotsPerEpoch {
		prevEpoch = (slot / params.BeaconConfig().SlotsPerEpoch) - 1
	}
	gweiPerEth := float64(params.BeaconConfig().GweiPerEth)
	for i, pubKey := range resp.PublicKeys {
		pubKeyBytes := bytesutil.ToBytes48(pubKey)
		if slot < params.BeaconConfig().SlotsPerEpoch {
			v.prevBalance[pubKeyBytes] = params.BeaconConfig().MaxEffectiveBalance
		}

		fmtKey := fmt.Sprintf("%#x", pubKey)
		truncatedKey := fmt.Sprintf("%#x", pubKey[:8])
		if v.prevBalance[pubKeyBytes] > 0 {
			newBalance := float64(resp.BalancesAfterEpochTransition[i]) / gweiPerEth
			prevBalance := float64(resp.BalancesBeforeEpochTransition[i]) / gweiPerEth
			percentNet := (newBalance - prevBalance) / prevBalance
			log.WithFields(logrus.Fields{
				"pubKey":               truncatedKey,
				"epoch":                prevEpoch,
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
				metrics.ValidatorBalancesGaugeVec.WithLabelValues(fmtKey).Set(newBalance)
			}
		}

		if resp.InclusionSlots[i] != ^uint64(0) {
			included++
		}
		if resp.CorrectlyVotedSource[i] {
			votedSource++
		}
		if resp.CorrectlyVotedTarget[i] {
			votedTarget++
		}
		if resp.CorrectlyVotedHead[i] {
			votedHead++
		}
		v.prevBalance[pubKeyBytes] = resp.BalancesBeforeEpochTransition[i]
	}

	log.WithFields(logrus.Fields{
		"epoch":                          prevEpoch,
		"attestationInclusionPercentage": fmt.Sprintf("%.0f%%", (float64(included)/float64(len(resp.InclusionSlots)))*100),
		"correctlyVotedSourcePercentage": fmt.Sprintf("%.0f%%", (float64(votedSource)/float64(len(resp.CorrectlyVotedSource)))*100),
		"correctlyVotedTargetPercentage": fmt.Sprintf("%.0f%%", (float64(votedTarget)/float64(len(resp.CorrectlyVotedTarget)))*100),
		"correctlyVotedHeadPercentage":   fmt.Sprintf("%.0f%%", (float64(votedHead)/float64(len(resp.CorrectlyVotedHead)))*100),
	}).Info("Previous epoch aggregated voting summary")

	return nil
}
