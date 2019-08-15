package forkchoice

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "forkchoice")

// logs epoch related data during epoch boundary.
func logEpochData(beaconState *pb.BeaconState) {
	log.WithField(
		"previousJustifiedEpoch", beaconState.PreviousJustifiedCheckpoint.Epoch,
	).Info("Previous justified epoch")
	log.WithField(
		"justifiedEpoch", beaconState.CurrentJustifiedCheckpoint.Epoch,
	).Info("Justified epoch")
	log.WithField(
		"finalizedEpoch", beaconState.FinalizedCheckpoint.Epoch,
	).Info("Finalized epoch")
	log.WithField(
		"Deposit Index", beaconState.Eth1DepositIndex,
	).Info("ETH1 Deposit Index")
	log.WithField(
		"numValidators", len(beaconState.Validators),
	).Info("Validator registry length")
}
