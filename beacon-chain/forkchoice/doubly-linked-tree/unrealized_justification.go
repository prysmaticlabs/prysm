package doublylinkedtree

import (
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
)

func (s *Store) setUnrealizedJustifiedEpoch(root [32]byte, epoch types.Epoch) error {
	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	node, ok := s.nodeByRoot[root]
	if !ok || node == nil {
		return errors.Wrap(ErrNilNode, "could not set unrealized justified epoch")
	}
	if epoch < node.unrealizedJustifiedEpoch {
		return errInvalidUnrealizedJustifiedEpoch
	}
	node.unrealizedJustifiedEpoch = epoch
	return nil
}

func (s *Store) setUnrealizedFinalizedEpoch(root [32]byte, epoch types.Epoch) error {
	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	node, ok := s.nodeByRoot[root]
	if !ok || node == nil {
		return errors.Wrap(ErrNilNode, "could not set unrealized finalized epoch")
	}
	if epoch < node.unrealizedFinalizedEpoch {
		return errInvalidUnrealizedFinalizedEpoch
	}
	node.unrealizedFinalizedEpoch = epoch
	return nil
}

// UpdateUnrealizedCheckpoints "realizes" the unrealized justified and finalized
// epochs stored within nodes. It should be called at the beginning of each
// epoch
func (f *ForkChoice) UpdateUnrealizedCheckpoints() {
	f.store.nodesLock.Lock()
	defer f.store.nodesLock.Unlock()
	for _, node := range f.store.nodeByRoot {
		node.justifiedEpoch = node.unrealizedJustifiedEpoch
		node.finalizedEpoch = node.unrealizedFinalizedEpoch
		if node.justifiedEpoch > f.store.justifiedCheckpoint.Epoch {
			f.store.justifiedCheckpoint.Epoch = node.justifiedEpoch
		}
		if node.finalizedEpoch > f.store.finalizedCheckpoint.Epoch {
			f.store.finalizedCheckpoint.Epoch = node.finalizedEpoch
		}
	}
}
