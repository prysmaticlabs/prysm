package synccommittee

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/copyutil"
)

// SaveSyncCommitteeContribution saves a sync committee contribution in cache.
// The cache does not filter out duplicate contribution, it will be up to the caller.
func (s *Store) SaveSyncCommitteeContribution(sig *ethpb.SyncCommitteeContribution) error {
	if sig == nil {
		return nilContributionErr
	}

	copied := copyutil.CopySyncCommitteeContribution(sig)
	slot := copied.Slot
	s.contributionLock.Lock()
	defer s.contributionLock.Unlock()

	sigs, ok := s.contributionCache[slot]
	if !ok {
		s.contributionCache[slot] = []*ethpb.SyncCommitteeContribution{copied}
		return nil
	}

	s.contributionCache[slot] = append(sigs, copied)

	return nil
}

// SyncCommitteeContributions returns sync committee contributions in cache by slot.
func (s *Store) SyncCommitteeContributions(slot types.Slot) []*ethpb.SyncCommitteeContribution {
	s.contributionLock.RLock()
	defer s.contributionLock.RUnlock()

	sigs, ok := s.contributionCache[slot]
	if !ok {
		return nil
	}
	return sigs
}

// DeleteSyncCommitteeContributions deletes sync committee contributions in cache by slot.
func (s *Store) DeleteSyncCommitteeContributions(slot types.Slot) {
	s.contributionLock.Lock()
	defer s.contributionLock.Unlock()
	delete(s.contributionCache, slot)
}
