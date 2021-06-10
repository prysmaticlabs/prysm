package synccommittee

import (
	"strconv"

	"github.com/hashicorp/vault/sdk/queue"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/copyutil"
)

const syncCommitteeMaxQueueSize = 5

// SaveSyncCommitteeContribution saves a sync committee contribution in to a priority queue.
// The priority queue capped at  5 contributions.
func (s *Store) SaveSyncCommitteeContribution(sig *ethpb.SyncCommitteeContribution) error {
	if sig == nil {
		return nilContributionErr
	}

	copied := copyutil.CopySyncCommitteeContribution(sig)
	s.contributionLock.Lock()
	defer s.contributionLock.Unlock()

	// Handle case where key exists.

	item := &queue.Item{
		Key:      syncCommitteeKey(sig.Slot),
		Value:    []*ethpb.SyncCommitteeContribution{copied},
		Priority: int64(sig.Slot),
	}

	s.contributionCache.Push(item)

	if s.contributionCache.Len() > 5 {
		if _, err := s.contributionCache.Pop(); err != nil {
			return err
		}
	}

	return nil
}

// SyncCommitteeContributions returns sync committee contributions in cache by slot from the priority queue.
func (s *Store) SyncCommitteeContributions(slot types.Slot) ([]*ethpb.SyncCommitteeContribution, error) {
	s.contributionLock.RLock()
	defer s.contributionLock.RUnlock()

	item, err := s.contributionCache.PopByKey(syncCommitteeKey(slot))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}

	contribution, ok := item.Value.([]*ethpb.SyncCommitteeContribution)
	if !ok {
		return nil, errors.New("not typed []ethpb.SyncCommitteeContribution")
	}

	return contribution, nil
}

func syncCommitteeKey(slot types.Slot) string {
	return strconv.FormatUint(uint64(slot), 10)
}
