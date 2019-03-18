package client

import (
	"context"
	"fmt"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
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
	epoch := slot / params.BeaconConfig().SlotsPerEpoch
	if epoch == params.BeaconConfig().GenesisEpoch {
		v.prevBalance = params.BeaconConfig().MaxDepositAmount
	}
	req := &pb.ValidatorPerformanceRequest{
		Slot:      slot,
		PublicKey: v.key.PublicKey.Marshal(),
	}
	resp, err := v.validatorClient.ValidatorPerformance(ctx, req)
	if err != nil {
		return err
	}
	newBalance := float64(resp.Balance) / float64(params.BeaconConfig().GweiPerEth)
	log.WithFields(logrus.Fields{
		"slot":  slot - params.BeaconConfig().GenesisSlot,
		"epoch": (slot / params.BeaconConfig().SlotsPerEpoch) - params.BeaconConfig().GenesisEpoch,
	}).Info("Start of a new epoch!")
	log.WithFields(logrus.Fields{
		"totalValidators":     resp.TotalValidators,
		"numActiveValidators": resp.TotalActiveValidators,
	}).Infof("Validator registry information")
	log.Info("Generating validator performance report from the previous epoch...")
	log.WithFields(logrus.Fields{
		"ethBalance": newBalance,
	}).Info("New validator balance")
	avgBalance := resp.AverageValidatorBalance / float32(params.BeaconConfig().GweiPerEth)
	if v.prevBalance > 0 {
		prevBalance := float64(v.prevBalance) / float64(params.BeaconConfig().GweiPerEth)
		percentNet := (newBalance - prevBalance) / prevBalance
		log.WithField("prevEthBalance", prevBalance).Info("Previous validator balance")
		log.WithFields(logrus.Fields{
			"eth":           fmt.Sprintf("%f", newBalance-prevBalance),
			"percentChange": fmt.Sprintf("%.2f%%", percentNet*100),
		}).Info("Net gains/losses in eth")
	}
	log.WithField(
		"averageEthBalance", fmt.Sprintf("%f", avgBalance),
	).Info("Average eth balance per validator in the beacon chain")
	v.prevBalance = resp.Balance
	return nil
}
