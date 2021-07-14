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
					digestExists := false
					for _, t := range s.subHandler.allTopics() {
						retDigest, err := digestFromTopic(t)
						if err != nil {
							log.WithError(err).Error("Could not retrieve digest")
							continue
						}
						if retDigest == digest {
							digestExists = true
							break
						}
					}
					if digestExists {
						continue
					}
					s.registerSubscribers(nextEpoch, digest)
					s.registerRPCHandlersAltair()
				}
			}
			// This routine takes care of the de-registration of
			// old gossip pubsub handlers. Once we are at the epoch
			// after the fork, we de-register from all the outdated topics.
			currFork, err := p2putils.Fork(currEpoch)
			if err != nil {
				log.WithError(err)
				continue
			}
			epochAfterFork := currFork.Epoch + 1
			nonGenesisFork := currFork.Epoch > 1
			// If we are in the epoch after the fork, we start de-registering.
			if epochAfterFork == currEpoch && nonGenesisFork {
				// Look at the previous fork's digest.
				epochBeforeFork := currFork.Epoch - 1
				prevDigest, err := p2putils.ForkDigestFromEpoch(epochBeforeFork, genRoot[:])
				if err != nil {
					log.WithError(err)
					continue
				}
				// Run through all our current active topics and see
				// if there are any subscriptions to be removed.
				for _, t := range s.subHandler.allTopics() {
					retDigest, err := digestFromTopic(t)
					if err != nil {
						log.WithError(err).Error("Could not retrieve digest")
						continue
					}
					if retDigest == prevDigest {
						if err := s.cfg.P2P.PubSub().UnregisterTopicValidator(t); err != nil {
							log.WithError(err).Error("Could not unregister topic validator")
						}
						s.subHandler.removeTopic(t)
					}
				}
			}
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting goroutine")
			slotTicker.Done()
			return
		}
	}
}
