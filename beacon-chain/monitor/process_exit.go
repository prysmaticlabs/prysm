package monitor

import (
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/sirupsen/logrus"
)

// processExitsFromBlock logs the event of one of our tracked validators' exit was
// included in a block
func (s *Service) processExitsFromBlock(blk block.BeaconBlock) {
	s.RLock()
	defer s.RUnlock()
	for _, exit := range blk.Body().VoluntaryExits() {
		idx := exit.Exit.ValidatorIndex
		if s.trackedIndex(idx) {
			log.WithFields(logrus.Fields{
				"ValidatorIndex": idx,
				"Slot":           blk.Slot(),
			}).Info("Voluntary exit was included")
		}
	}
}

// processExit logs the event of one of our tracked validators' exit was processed
func (s *Service) processExit(exit *ethpb.SignedVoluntaryExit) {
	idx := exit.Exit.ValidatorIndex
	s.RLock()
	defer s.RUnlock()
	if s.trackedIndex(idx) {
		log.WithFields(logrus.Fields{
			"ValidatorIndex": idx,
		}).Info("Voluntary exit was processed")
	}
}
