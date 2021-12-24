package light

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/container/queue"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// To give four slots tolerance for updates that arrive later.
const maxUpdateQueueSize = 4

func (s *Service) saveUpdate(root []byte, update *ethpb.LightClientUpdate) error {
	if update == nil {
		errors.New("nil update")
	}

	s.updateCacheLock.Lock()
	defer s.updateCacheLock.Unlock()

	h := update.AttestedHeader
	if err := s.updateCache.Push(&queue.Item{
		Key:      string(root),
		Value:    update,
		Priority: int64(h.Slot),
	}); err != nil {
		return err
	}
	// Trim updates in queue down to syncCommitteeMaxQueueSize.
	if s.updateCache.Len() > maxUpdateQueueSize {
		if _, err := s.updateCache.Pop(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) getUpdate(root []byte) (*ethpb.LightClientUpdate, error) {
	s.updateCacheLock.RLock()
	defer s.updateCacheLock.RUnlock()

	item, err := s.updateCache.PopByKey(string(root))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, errors.New("item is nil")
	}

	update, ok := item.Value.(*ethpb.LightClientUpdate)
	if !ok {
		return nil, errors.New("not typed ethpb.LightClientUpdate")
	}

	return update, nil
}
