package protoarray

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	forkchoicetypes "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/time/slots"
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
	var uj, uf *ethpb.Checkpoint

	currentSlot := slots.CurrentSlot(s.genesisTime)
	currentEpoch := slots.ToEpoch(currentSlot)
	stateSlot := state.Slot()
	stateEpoch := slots.ToEpoch(stateSlot)
	if node.parent == NonExistentNode {
		return jc, fc
	}
	parent := s.nodes[node.parent]
	currJustified := parent.unrealizedJustifiedEpoch == currentEpoch
	prevJustified := parent.unrealizedJustifiedEpoch+1 == currentEpoch
	tooEarlyForCurr := slots.SinceEpochStarts(stateSlot)*3 < params.BeaconConfig().SlotsPerEpoch*2
	if currJustified || (stateEpoch == currentEpoch && prevJustified && tooEarlyForCurr) {
		node.unrealizedJustifiedEpoch = parent.unrealizedJustifiedEpoch
		node.unrealizedFinalizedEpoch = parent.unrealizedFinalizedEpoch
		return jc, fc
	}

	uj, uf, err := precompute.UnrealizedCheckpoints(ctx, state)
	if err != nil {
		log.WithError(err).Debug("could not compute unrealized checkpoints")
		uj, uf = jc, fc
	}
	node.unrealizedJustifiedEpoch, node.unrealizedFinalizedEpoch = uj.Epoch, uf.Epoch
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

	if stateEpoch < currentEpoch {
		jc, fc = uj, uf
		node.justifiedEpoch = uj.Epoch
		node.finalizedEpoch = uf.Epoch
	}
	s.checkpointsLock.Unlock()
	return jc, fc
}
func TestStore_PullTips_Heuristics(t *testing.T) {
	resetCfg := features.InitWithReset(&features.Flags{
		PullTips: true,
	})
	defer resetCfg()
	ctx := context.Background()
	t.Run("Current epoch is justified", func(tt *testing.T) {
		f := setup(1, 1)
		st, root, err := prepareForkchoiceState(ctx, 65, [32]byte{'p'}, [32]byte{}, [32]byte{}, 1, 1)
		require.NoError(tt, err)
		require.NoError(tt, f.InsertNode(ctx, st, root))
		f.store.nodes[1].unrealizedJustifiedEpoch = types.Epoch(2)
		driftGenesisTime(f, 66, 0)

		st, root, err = prepareForkchoiceState(ctx, 66, [32]byte{'h'}, [32]byte{'p'}, [32]byte{}, 1, 1)
		require.NoError(tt, err)
		require.NoError(tt, f.InsertNode(ctx, st, root))
		require.Equal(tt, types.Epoch(2), f.store.nodes[2].unrealizedJustifiedEpoch)
		require.Equal(tt, types.Epoch(1), f.store.nodes[2].unrealizedFinalizedEpoch)
	})

	t.Run("Previous Epoch is justified and too early for current", func(tt *testing.T) {
		f := setup(1, 1)
		st, root, err := prepareForkchoiceState(ctx, 95, [32]byte{'p'}, [32]byte{}, [32]byte{}, 1, 1)
		require.NoError(tt, err)
		require.NoError(tt, f.InsertNode(ctx, st, root))
		f.store.nodes[1].unrealizedJustifiedEpoch = types.Epoch(2)
		driftGenesisTime(f, 96, 0)

		st, root, err = prepareForkchoiceState(ctx, 96, [32]byte{'h'}, [32]byte{'p'}, [32]byte{}, 1, 1)
		require.NoError(tt, err)
		require.NoError(tt, f.InsertNode(ctx, st, root))
		require.Equal(tt, types.Epoch(2), f.store.nodes[2].unrealizedJustifiedEpoch)
		require.Equal(tt, types.Epoch(1), f.store.nodes[2].unrealizedFinalizedEpoch)
	})
	t.Run("Previous Epoch is justified and not too early for current", func(tt *testing.T) {
		f := setup(1, 1)
		st, root, err := prepareForkchoiceState(ctx, 95, [32]byte{'p'}, [32]byte{}, [32]byte{}, 1, 1)
		require.NoError(tt, err)
		require.NoError(tt, f.InsertNode(ctx, st, root))
		f.store.nodes[1].unrealizedJustifiedEpoch = types.Epoch(2)
		driftGenesisTime(f, 127, 0)

		st, root, err = prepareForkchoiceState(ctx, 127, [32]byte{'h'}, [32]byte{'p'}, [32]byte{}, 1, 1)
		require.NoError(tt, err)
		require.NoError(tt, f.InsertNode(ctx, st, root))
		// Check that the justification point is not the parent's.
		// This tests that the heuristics in pullTips did not apply and
		// the test continues to compute a bogus unrealized
		// justification
		require.Equal(tt, types.Epoch(1), f.store.nodes[2].unrealizedJustifiedEpoch)
	})
	t.Run("Block from previous Epoch", func(tt *testing.T) {
		f := setup(1, 1)
		st, root, err := prepareForkchoiceState(ctx, 94, [32]byte{'p'}, [32]byte{}, [32]byte{}, 1, 1)
		require.NoError(tt, err)
		require.NoError(tt, f.InsertNode(ctx, st, root))
		f.store.nodes[1].unrealizedJustifiedEpoch = types.Epoch(2)
		driftGenesisTime(f, 96, 0)

		st, root, err = prepareForkchoiceState(ctx, 95, [32]byte{'h'}, [32]byte{'p'}, [32]byte{}, 1, 1)
		require.NoError(tt, err)
		require.NoError(tt, f.InsertNode(ctx, st, root))
		// Check that the justification point is not the parent's.
		// This tests that the heuristics in pullTips did not apply and
		// the test continues to compute a bogus unrealized
		// justification
		require.Equal(tt, types.Epoch(1), f.store.nodes[2].unrealizedJustifiedEpoch)
	})
	t.Run("Previous Epoch is not justified", func(tt *testing.T) {
		f := setup(1, 1)
		st, root, err := prepareForkchoiceState(ctx, 128, [32]byte{'p'}, [32]byte{}, [32]byte{}, 2, 1)
		require.NoError(tt, err)
		require.NoError(tt, f.InsertNode(ctx, st, root))
		driftGenesisTime(f, 129, 0)

		st, root, err = prepareForkchoiceState(ctx, 129, [32]byte{'h'}, [32]byte{'p'}, [32]byte{}, 2, 1)
		require.NoError(tt, err)
		require.NoError(tt, f.InsertNode(ctx, st, root))
		// Check that the justification point is not the parent's.
		// This tests that the heuristics in pullTips did not apply and
		// the test continues to compute a bogus unrealized
		// justification
		require.Equal(tt, types.Epoch(2), f.store.nodes[2].unrealizedJustifiedEpoch)
	})
}
