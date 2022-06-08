package protoarray

import (
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
)

func (s *Store) setUnrealizedJustifiedEpoch(root [32]byte, epoch types.Epoch) error {
	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()
	index, ok := s.nodesIndices[root]
	if !ok {
		return ErrUnknownNodeRoot
	}
	if index >= uint64(len(s.nodes)) {
		return errInvalidNodeIndex
	}
	node := s.nodes[index]
	if node == nil {
		return errInvalidNodeIndex
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
	index, ok := s.nodesIndices[root]
	if !ok {
		return ErrUnknownNodeRoot
	}
	if index >= uint64(len(s.nodes)) {
		return errInvalidNodeIndex
	}
	node := s.nodes[index]
	if node == nil {
		return errInvalidNodeIndex
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
	for _, node := range f.store.nodes {
		node.justifiedEpoch = node.unrealizedJustifiedEpoch
		node.finalizedEpoch = node.unrealizedFinalizedEpoch
		if node.justifiedEpoch > f.store.justifiedEpoch {
			f.store.justifiedEpoch = node.justifiedEpoch
		}
		if node.finalizedEpoch > f.store.finalizedEpoch {
			f.store.finalizedEpoch = node.finalizedEpoch
		}
	}
}
