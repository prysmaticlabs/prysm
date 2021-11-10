package monitor

import (
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/sirupsen/logrus"
)

// processExitsFromBlock logs the event of one of our tracked validators' exit was
// included in a block
func (s *Service) processExitsFromBlock(blk block.BeaconBlock) {
	for _, exit := range blk.Body().VoluntaryExits() {
		idx := exit.Exit.ValidatorIndex
		if s.TrackedIndex(idx) {
			log.WithFields(logrus.Fields{
				"ValidatorIndex": idx,
				"Slot":           blk.Slot(),
			}).Info("VoluntaryExit was included")
		}
	}
}

// processExit logs the event of one of our tracked validators' exit was processed
func (s *Service) processExit(exit *ethpb.SignedVoluntaryExit) {
	idx := exit.Exit.ValidatorIndex
	if s.TrackedIndex(idx) {
		log.WithFields(logrus.Fields{
			"ValidatorIndex": idx,
		}).Info("VoluntaryExit was processed")
	}
}
