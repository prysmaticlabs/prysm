package forkchoice

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "forkchoice")

// logs epoch related data during epoch boundary.
func logEpochData(beaconState *pb.BeaconState) {
	log.WithFields(logrus.Fields{
		"epoch":                  beaconState.Slot / params.BeaconConfig().SlotsPerEpoch,
		"previousJustifiedEpoch": beaconState.PreviousJustifiedCheckpoint.Epoch,
		"justifiedEpoch":         beaconState.CurrentJustifiedCheckpoint.Epoch,
		"finalizedEpoch":         beaconState.FinalizedCheckpoint.Epoch,
		"depositIndex":           beaconState.Eth1DepositIndex,
		"numValidators":          len(beaconState.Validators),
	}).Info("Starting next epoch")
}
