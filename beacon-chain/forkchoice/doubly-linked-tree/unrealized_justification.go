package doublylinkedtree

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	forkchoicetypes "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
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
			if node.justifiedEpoch > f.store.bestJustifiedCheckpoint.Epoch {
				f.store.bestJustifiedCheckpoint = f.store.unrealizedJustifiedCheckpoint
			}
			f.store.justifiedCheckpoint = f.store.unrealizedJustifiedCheckpoint
		}
		if node.finalizedEpoch > f.store.finalizedCheckpoint.Epoch {
			f.store.justifiedCheckpoint = f.store.unrealizedJustifiedCheckpoint
			f.store.finalizedCheckpoint = f.store.unrealizedFinalizedCheckpoint
		}
	}
}

func (s *Store) pullTips(ctx context.Context, state state.BeaconState, node *Node, jc, fc *ethpb.Checkpoint) (*ethpb.Checkpoint, *ethpb.Checkpoint) {
	s.checkpointsLock.Lock()
	defer s.checkpointsLock.Unlock()

	var uj, uf *ethpb.Checkpoint
	currentSlot := slots.CurrentSlot(s.genesisTime)
	currentEpoch := slots.ToEpoch(currentSlot)
	stateSlot := state.Slot()
	stateEpoch := slots.ToEpoch(stateSlot)
	if node.parent == nil {
		return jc, fc
	}
	currJustified := node.parent.unrealizedJustifiedEpoch == currentEpoch
	prevJustified := node.parent.unrealizedJustifiedEpoch+1 == currentEpoch
	tooEarlyForCurr := slots.SinceEpochStarts(stateSlot)*3 < params.BeaconConfig().SlotsPerEpoch*2
	if currJustified || (stateEpoch == currentEpoch && prevJustified && tooEarlyForCurr) {
		node.unrealizedJustifiedEpoch = node.parent.unrealizedJustifiedEpoch
		node.unrealizedFinalizedEpoch = node.parent.unrealizedFinalizedEpoch
		return jc, fc
	}

	uj, uf, err := precompute.UnrealizedCheckpoints(ctx, state)
	if err != nil {
		log.WithError(err).Debug("could not compute unrealized checkpoints")
		uj, uf = jc, fc
	}
	node.unrealizedJustifiedEpoch, node.unrealizedFinalizedEpoch = uj.Epoch, uf.Epoch
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

	if stateEpoch < currentEpoch {
		jc, fc = uj, uf
		node.justifiedEpoch = uj.Epoch
		node.finalizedEpoch = uf.Epoch
	}
	return jc, fc
}
