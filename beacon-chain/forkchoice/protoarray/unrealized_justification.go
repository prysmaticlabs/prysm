package protoarray

import (
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/epoch/precompute"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
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
func (f *ForkChoice) updateUnrealizedCheckpoints() {
	f.store.nodesLock.Lock()
	defer f.store.nodesLock.Unlock()
	f.store.checkpointsLock.Lock()
	defer f.store.checkpointsLock.Unlock()
	for _, node := range f.store.nodes {
		node.justifiedEpoch = node.unrealizedJustifiedEpoch
		node.finalizedEpoch = node.unrealizedFinalizedEpoch
		if node.justifiedEpoch > f.store.justifiedCheckpoint.Epoch {
			if node.justifiedEpoch > f.store.bestJustifiedCheckpoint.Epoch {
				f.store.bestJustifiedCheckpoint = f.store.unrealizedJustifiedCheckpoint
			}
			f.store.prevJustifiedCheckpoint = f.store.justifiedCheckpoint
			f.store.justifiedCheckpoint = f.store.unrealizedJustifiedCheckpoint
		}
		if node.finalizedEpoch > f.store.finalizedCheckpoint.Epoch {
			f.store.justifiedCheckpoint = f.store.unrealizedJustifiedCheckpoint
			f.store.finalizedCheckpoint = f.store.unrealizedFinalizedCheckpoint
		}
	}
}

func (s *Store) pullTips(state state.BeaconState, node *Node, jc, fc *ethpb.Checkpoint) (*ethpb.Checkpoint, *ethpb.Checkpoint) {
	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	if node.parent == NonExistentNode { // Nothing to do if the parent is nil.
		return jc, fc
	}

	currentEpoch := slots.ToEpoch(slots.CurrentSlot(s.genesisTime))
	stateSlot := state.Slot()
	stateEpoch := slots.ToEpoch(stateSlot)

	parent := s.nodes[node.parent]
	currJustified := parent.unrealizedJustifiedEpoch == currentEpoch
	prevJustified := parent.unrealizedJustifiedEpoch+1 == currentEpoch
	tooEarlyForCurr := slots.SinceEpochStarts(stateSlot)*3 < params.BeaconConfig().SlotsPerEpoch*2
	if currJustified || (stateEpoch == currentEpoch && prevJustified && tooEarlyForCurr) {
		node.unrealizedJustifiedEpoch = parent.unrealizedJustifiedEpoch
		node.unrealizedFinalizedEpoch = parent.unrealizedFinalizedEpoch
		return jc, fc
	}

	uj, uf, err := precompute.UnrealizedCheckpoints(state)
	if err != nil {
		log.WithError(err).Debug("could not compute unrealized checkpoints")
		uj, uf = jc, fc
	}

	// Update store's unrealized checkpoints.
	s.checkpointsLock.Lock()
	if uj.Epoch > s.unrealizedJustifiedCheckpoint.Epoch {
		s.unrealizedJustifiedCheckpoint = &forkchoicetypes.Checkpoint{
			Epoch: uj.Epoch, Root: bytesutil.ToBytes32(uj.Root),
		}
	}
	if uf.Epoch > s.unrealizedFinalizedCheckpoint.Epoch {
		s.unrealizedJustifiedCheckpoint = &forkchoicetypes.Checkpoint{
			Epoch: uj.Epoch, Root: bytesutil.ToBytes32(uj.Root),
		}
		s.unrealizedFinalizedCheckpoint = &forkchoicetypes.Checkpoint{
			Epoch: uf.Epoch, Root: bytesutil.ToBytes32(uf.Root),
		}
	}
	s.checkpointsLock.Unlock()

	// Update node's checkpoints.
	node.unrealizedJustifiedEpoch, node.unrealizedFinalizedEpoch = uj.Epoch, uf.Epoch
	if stateEpoch < currentEpoch {
		jc, fc = uj, uf
		node.justifiedEpoch = uj.Epoch
		node.finalizedEpoch = uf.Epoch
	}

	return jc, fc
}
