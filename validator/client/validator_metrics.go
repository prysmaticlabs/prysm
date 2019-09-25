package client

import (
	"context"
	"encoding/hex"
	"fmt"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

// LogValidatorGainsAndLosses logs important metrics related to this validator client's
// responsibilities throughout the beacon chain's lifecycle. It logs absolute accrued rewards
// and penalties over time, percentage gain/loss, and gives the end user a better idea
// of how the validator performs with respect to the rest.
func (v *validator) LogValidatorGainsAndLosses(ctx context.Context, slot uint64) error {
	if slot%params.BeaconConfig().SlotsPerEpoch != 0 {
		// Do nothing if we are not at the start of a new epoch.
		return nil
	}
	if !v.logValidatorBalances {
		return nil
	}

	req := &pb.ValidatorPerformanceRequest{
		Slot:       slot,
		PublicKeys: v.pubkeys,
	}
	resp, err := v.validatorClient.ValidatorPerformance(ctx, req)
	if err != nil {
		return err
	}

	log.WithFields(logrus.Fields{
		"slot":  slot,
		"epoch": slot / params.BeaconConfig().SlotsPerEpoch,
	}).Info("Start of a new epoch!")
	log.WithFields(logrus.Fields{
		"totalValidators":     resp.TotalValidators,
		"numActiveValidators": resp.TotalActiveValidators,
	}).Info("Validator registry information")
	log.Info("Generating validator performance report from the previous epoch...")
	avgBalance := resp.AverageActiveValidatorBalance / float32(params.BeaconConfig().GweiPerEth)
	log.WithField(
		"averageEthBalance", fmt.Sprintf("%f", avgBalance),
	).Info("Average eth balance per active validator in the beacon chain")

	missingValidators := make(map[[48]byte]bool)
	for _, val := range resp.MissingValidators {
		missingValidators[bytesutil.ToBytes48(val)] = true
	}
	for i, pkey := range v.pubkeys {
		tpk := hex.EncodeToString(pkey)[:12]
		if missingValidators[bytesutil.ToBytes48(pkey)] {
			log.WithField("pubKey", fmt.Sprintf("%#x", tpk)).Info("Validator not able to be retrieved from beacon node")
			continue
		}
		if slot < params.BeaconConfig().SlotsPerEpoch {
			v.prevBalance[bytesutil.ToBytes48(pkey)] = params.BeaconConfig().MaxEffectiveBalance
		}
		newBalance := float64(resp.Balances[i]) / float64(params.BeaconConfig().GweiPerEth)

		if v.prevBalance[bytesutil.ToBytes48(pkey)] > 0 {
			prevBalance := float64(v.prevBalance[bytesutil.ToBytes48(pkey)]) / float64(params.BeaconConfig().GweiPerEth)
			percentNet := (newBalance - prevBalance) / prevBalance
			log.WithFields(logrus.Fields{
				"prevBalance":   prevBalance,
				"newBalance":    newBalance,
				"delta":         fmt.Sprintf("%.8f", newBalance-prevBalance),
				"percentChange": fmt.Sprintf("%.5f%%", percentNet*100),
				"pubKey":        tpk,
			}).Info("Net gains/losses in eth")
		}
		v.prevBalance[bytesutil.ToBytes48(pkey)] = resp.Balances[i]
	}

	return nil
}
