package attestations

import (
	"time"

	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

// prune prunes attestations on every prune interval.
func (s *Service) pruneExpired() {
	ticker := time.NewTicker(s.cfg.pruneInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			expirySlot, err := s.expirySlot()
			if err != nil {
				log.WithError(err).Error("Could not get expiry slot")
				continue
			}
			numExpired := s.cfg.Cache.PruneBefore(expirySlot)
			s.updateMetrics(numExpired)
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting routine")
			return
		}
	}
}

// Attestations for a slot before the returned slot are considered expired.
func (s *Service) expirySlot() (primitives.Slot, error) {
	currSlot := slots.CurrentSlot(s.genesisTime)
	currEpoch := slots.ToEpoch(currSlot)
	if currEpoch == 0 {
		return 0, nil
	}
	if currEpoch < params.BeaconConfig().DenebForkEpoch {
		currSlot.SubSlot(params.BeaconConfig().SlotsPerEpoch).Add(1)
	}
	return slots.EpochStart(currEpoch - 1)
}
