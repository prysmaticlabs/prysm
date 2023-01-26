package sync

import (
	"time"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

const broadcastBLSChangesRateLimit = 1000

// This routine broadcasts all known BLS changes at the Capella fork.
func (s *Service) broadcastBLSChanges(currSlot types.Slot) {
	capellaSlotStart, err := slots.EpochStart(params.BeaconConfig().CapellaForkEpoch)
	if err != nil {
		// only possible error is an overflow, so we exit early from the method
		return
	}
	if currSlot == capellaSlotStart {
		changes, err := s.cfg.blsToExecPool.PendingBLSToExecChanges()
		if err != nil {
			log.WithError(err).Error("could not get BLS to execution changes")
		}
		go func() {
			st := slots.NewSlotTicker(time.Now(), params.BeaconConfig().SecondsPerSlot)
			for {
				select {
				case <-s.ctx.Done():
					return
				case <-st.C():
					limit := broadcastBLSChangesRateLimit
					if len(changes) < broadcastBLSChangesRateLimit {
						limit = len(changes)
					}
					for _, ch := range changes[:limit] {
						if err := s.cfg.p2p.Broadcast(s.ctx, ch); err != nil {
							log.WithError(err).Error("could not broadcast BLS to execution changes.")
						}
						changes = changes[limit:]
						if len(changes) == 0 {
							return
						}
					}
				}
			}
		}()
	}
}
