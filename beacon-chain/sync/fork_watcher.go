package sync

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/p2putils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
)

// Is a background routine that observes for new incoming forks. Depending on the epoch
// it will be in charge of subscribing/unsubscribing the relevant topics at the fork boundaries.
func (s *Service) forkWatcher() {
	slotTicker := slotutil.NewSlotTicker(s.cfg.Chain.GenesisTime(), params.BeaconConfig().SecondsPerSlot)
	for {
		select {
		case currSlot := <-slotTicker.C():
			currEpoch := helpers.SlotToEpoch(currSlot)
			genRoot := s.cfg.Chain.GenesisValidatorRoot()
			isNextForkEpoch, err := p2putils.IsForkNextEpoch(s.cfg.Chain.GenesisTime(), genRoot[:])
			if err != nil {
				log.WithError(err).Error("Could not retrieve next fork epoch")
				continue
			}
			// In preparation for the upcoming fork
			// in the following epoch, the node
			// will subscribe the new topics in advance.
			if isNextForkEpoch {
				nextEpoch := currEpoch + 1
				if nextEpoch == params.BeaconConfig().AltairForkEpoch {
					digest, err := p2putils.ForkDigestFromEpoch(nextEpoch, genRoot[:])
					if err != nil {
						log.WithError(err).Error("Could not retrieve fork digest")
						continue
					}
					s.registerSubscribers(nextEpoch, digest)
					s.registerRPCHandlersAltair()
				}
			}
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting goroutine")
			slotTicker.Done()
			return
		}
	}
}
