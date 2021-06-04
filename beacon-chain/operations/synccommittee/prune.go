package synccommittee

import (
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
)

// pruneSyncCommitteeStore prunes sync committee store on every slot interval.
func (s *Service) pruneSyncCommitteeStore() {
	ticker := time.NewTicker(time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// Chain has not started. There's nothing to do.
			if s.genesisTime == 0 {
				continue
			}
			s.pruneExpiredSyncCommitteeSignatures()
			s.pruneExpiredSyncCommitteeContributions()
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting routine")
			return

		}
	}
}

// This prunes expired sync committee signatures from the store.
func (s *Service) pruneExpiredSyncCommitteeSignatures() {
	currentSlot := helpers.CurrentSlot(s.genesisTime)
	s.store.signatureLock.Lock()
	defer s.store.signatureLock.Unlock()

	// Delete the sync committee signatures from 2 slots back.
	// Doesn't matter when in current slot the deletion happen,
	// and this is the simplest and safest approach.
	expiredSlot := currentSlot - 2
	delete(s.store.signatureCache, expiredSlot)
}

// This prunes expired sync committee contributions from the store.
func (s *Service) pruneExpiredSyncCommitteeContributions() {
	currentSlot := helpers.CurrentSlot(s.genesisTime)

	s.store.contributionLock.Lock()
	defer s.store.contributionLock.Unlock()

	// Delete the sync committee signatures from 2 slots back.
	// Doesn't matter when in current slot the deletion happen,
	// and this is the simplest and safest approach.
	expiredSlot := currentSlot - 2
	delete(s.store.contributionCache, expiredSlot)
}
