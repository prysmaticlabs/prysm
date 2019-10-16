package client

import (
	"context"
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
	if slot%params.BeaconConfig().SlotsPerEpoch != 0 || slot < params.BeaconConfig().SlotsPerEpoch {
		// Do nothing if we are not at the start of a new epoch and before the first epoch.
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

	missingValidators := make(map[[48]byte]bool)
	for _, val := range resp.MissingValidators {
		missingValidators[bytesutil.ToBytes48(val)] = true
	}
	for i, pkey := range v.pubkeys {
		pubKey := fmt.Sprintf("%#x", pkey[:8])
		log := log.WithField("pubKey", pubKey)
		if missingValidators[bytesutil.ToBytes48(pkey)] {
			log.Info("Validator not in beacon chain")
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
				"epoch":         (slot / params.BeaconConfig().SlotsPerEpoch) - 1,
				"prevBalance":   prevBalance,
				"newBalance":    newBalance,
				"percentChange": fmt.Sprintf("%.5f%%", percentNet*100),
			}).Info("New Balance")
		}
		v.prevBalance[bytesutil.ToBytes48(pkey)] = resp.Balances[i]
	}
	return nil
}
