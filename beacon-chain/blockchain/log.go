package blockchain

import (
	"encoding/hex"

	"github.com/sirupsen/logrus"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

var log = logrus.WithField("prefix", "blockchain")

// logs state transition related data every slot.
func logStateTransitionData(b *ethpb.BeaconBlock, r []byte) {
	log.WithFields(logrus.Fields{
		"slot": b.Slot,
		"root": hex.EncodeToString(r),
		"attestations": len(b.Body.Attestations),
		"deposits":     len(b.Body.Deposits),
	}).Info("Finished state transition and updated fork choice store for block")
}
