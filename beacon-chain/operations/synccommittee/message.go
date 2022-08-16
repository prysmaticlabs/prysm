package synccommittee

import (
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/container/queue"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// SaveSyncCommitteeMessage saves a sync committee message in to a priority queue.
// The priority queue capped at syncCommitteeMaxQueueSize contributions.
func (s *Store) SaveSyncCommitteeMessage(msg *ethpb.SyncCommitteeMessage) error {
	if msg == nil {
		return errNilMessage
	}

	s.messageLock.Lock()
	defer s.messageLock.Unlock()

	item, err := s.messageCache.PopByKey(syncCommitteeKey(msg.Slot))
	if err != nil {
		return err
	}

	copied := ethpb.CopySyncCommitteeMessage(msg)
	// Messages exist in the queue. Append instead of insert new.
	if item != nil {
		messages, ok := item.Value.([]*ethpb.SyncCommitteeMessage)
		if !ok {
			return errors.New("not typed []ethpb.SyncCommitteeMessage")
		}

		messages = append(messages, copied)
		savedSyncCommitteeMessageTotal.Inc()
		return s.messageCache.Push(&queue.Item{
			Key:      syncCommitteeKey(msg.Slot),
			Value:    messages,
			Priority: int64(msg.Slot),
		})
	}

	// Message does not exist. Insert new.
	if err := s.messageCache.Push(&queue.Item{
		Key:      syncCommitteeKey(msg.Slot),
		Value:    []*ethpb.SyncCommitteeMessage{copied},
		Priority: int64(msg.Slot),
	}); err != nil {
		return err
	}
	savedSyncCommitteeMessageTotal.Inc()

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

	item := s.messageCache.RetrieveByKey(syncCommitteeKey(slot))
	if item == nil {
		return nil, nil
	}

	messages, ok := item.Value.([]*ethpb.SyncCommitteeMessage)
	if !ok {
		return nil, errors.New("not typed []ethpb.SyncCommitteeMessage")
	}

	return messages, nil
}
