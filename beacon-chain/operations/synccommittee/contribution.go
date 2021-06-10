package synccommittee

import (
	"strconv"

	"github.com/hashicorp/vault/sdk/queue"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/copyutil"
)

// To give two slots tolerance for objects that arrive earlier.
// This account for previous slot, current slot, two future slots.
const syncCommitteeMaxQueueSize = 4

// SaveSyncCommitteeContribution saves a sync committee contribution in to a priority queue.
// The priority queue is capped at syncCommitteeMaxQueueSize contributions.
func (s *Store) SaveSyncCommitteeContribution(cont *ethpb.SyncCommitteeContribution) error {
	if cont == nil {
		return nilContributionErr
	}

	contributions, err := s.SyncCommitteeContributions(cont.Slot)
	if err != nil {
		return err
	}

	s.contributionLock.Lock()
	defer s.contributionLock.Unlock()
	copied := copyutil.CopySyncCommitteeContribution(cont)

	// Contributions exist in the queue. Append instead of insert new.
	if contributions != nil {
		contributions = append(contributions, copied)
		if err := s.contributionCache.Push(&queue.Item{
			Key:      syncCommitteeKey(cont.Slot),
			Value:    contributions,
			Priority: int64(cont.Slot),
		}); err != nil {
			return err
		}
		return nil
	}

	// Contribution does not exist. Insert new.
	if err := s.contributionCache.Push(&queue.Item{
		Key:      syncCommitteeKey(cont.Slot),
		Value:    []*ethpb.SyncCommitteeContribution{copied},
		Priority: int64(cont.Slot),
	}); err != nil {
		return err
	}

	// Trim contributions in queue down to syncCommitteeMaxQueueSize.
	if s.contributionCache.Len() > syncCommitteeMaxQueueSize {
		if _, err := s.contributionCache.Pop(); err != nil {
			return err
		}
	}

	return nil
}

// SyncCommitteeContributions returns sync committee contributions by slot from the priority queue.
// Upon retrieval, the contribution is removed from the queue.
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

	contributions, ok := item.Value.([]*ethpb.SyncCommitteeContribution)
	if !ok {
		return nil, errors.New("not typed []ethpb.SyncCommitteeContribution")
	}

	return contributions, nil
}

func syncCommitteeKey(slot types.Slot) string {
	return strconv.FormatUint(uint64(slot), 10)
}
