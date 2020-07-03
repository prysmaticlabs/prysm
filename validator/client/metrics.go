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
				ValidatorBalancesGaugeVec.WithLabelValues(fmtKey).Set(newBalance)
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
