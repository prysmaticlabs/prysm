package blockchain

import (
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "blockchain")

// logs state transition related data every slot.
func logStateTransitionData(b *ethpb.BeaconBlock, r []byte) {
	log.WithFields(logrus.Fields{
		"slot":         b.Slot,
		"attestations": len(b.Body.Attestations),
		"deposits":     len(b.Body.Deposits),
	}).Info("Finished applying state transition")
}
