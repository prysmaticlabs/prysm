package synccommittee

import (
	"github.com/hashicorp/vault/sdk/queue"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/copyutil"
)

// SaveSyncCommitteeMessage saves a sync committee message in to a priority queue.
// The priority queue capped at syncCommitteeMaxQueueSize contributions.
func (s *Store) SaveSyncCommitteeMessage(msg *ethpb.SyncCommitteeMessage) error {
	if msg == nil {
		return nilMessageErr
	}

	messages, err := s.SyncCommitteeMessages(msg.Slot)
	if err != nil {
		return err
	}

	s.messageLock.Lock()
	defer s.messageLock.Unlock()
	copied := copyutil.CopySyncCommitteeMessage(msg)

	// Messages exist in the queue. Append instead of insert new.
	if messages != nil {
		messages = append(messages, copied)
		s.messageCache.Push(&queue.Item{
			Key:      syncCommitteeKey(msg.Slot),
			Value:    messages,
			Priority: int64(msg.Slot),
		})
		return nil
	}

	// Message does not exist. Insert new.
	s.messageCache.Push(&queue.Item{
		Key:      syncCommitteeKey(msg.Slot),
		Value:    []*ethpb.SyncCommitteeMessage{copied},
		Priority: int64(msg.Slot),
	})

	// Trim messages in queue down to syncCommitteeMaxQueueSize.
	if s.messageCache.Len() > syncCommitteeMaxQueueSize {
		if _, err := s.messageCache.Pop(); err != nil {
			return err
		}
	}

	return nil
}

// SyncCommitteeMessages returns sync committee messages by slot from the priority queue.
// Upon retrieval, the message is removed from the queue.
func (s *Store) SyncCommitteeMessages(slot types.Slot) ([]*ethpb.SyncCommitteeMessage, error) {
	s.messageLock.RLock()
	defer s.messageLock.RUnlock()

	item, err := s.messageCache.PopByKey(syncCommitteeKey(slot))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}

	messages, ok := item.Value.([]*ethpb.SyncCommitteeMessage)
	if !ok {
		return nil, errors.New("not typed []ethpb.SyncCommitteeMessage")
	}

	return messages, nil
}
