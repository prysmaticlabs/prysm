package protoarray

import (
	"context"
	"testing"

	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestStore_SetUnrealizedEpochs(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	state, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	f.store.nodesLock.RLock()
	require.Equal(t, types.Epoch(1), f.store.nodes[2].unrealizedJustifiedEpoch)
	require.Equal(t, types.Epoch(1), f.store.nodes[2].unrealizedFinalizedEpoch)
	f.store.nodesLock.RUnlock()
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'b'}, 2))
	require.NoError(t, f.store.setUnrealizedFinalizedEpoch([32]byte{'b'}, 2))
	f.store.nodesLock.RLock()
	require.Equal(t, types.Epoch(2), f.store.nodes[2].unrealizedJustifiedEpoch)
	require.Equal(t, types.Epoch(2), f.store.nodes[2].unrealizedFinalizedEpoch)
	f.store.nodesLock.RUnlock()

	require.ErrorIs(t, errInvalidUnrealizedJustifiedEpoch, f.store.setUnrealizedJustifiedEpoch([32]byte{'b'}, 0))
	require.ErrorIs(t, errInvalidUnrealizedFinalizedEpoch, f.store.setUnrealizedFinalizedEpoch([32]byte{'b'}, 0))
}

func TestStore_UpdateUnrealizedCheckpoints(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	state, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))

}

//
//  Epoch 2    |   Epoch 3
//             |
//           C |
//         /   |
//  A <-- B    |
//         \   |
//           ---- D
//
//  B is the first block that justifies A.
//
func TestStore_LongFork(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	state, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'b'}, 2))
	state, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'c'}, 2))

	// Add an attestation to c, it is head
	f.ProcessAttestation(ctx, []uint64{0}, [32]byte{'c'}, 1)
	headRoot, err := f.Head(ctx, []uint64{100})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'c'}, headRoot)

	// D is head even though its weight is lower.
	ha := [32]byte{'a'}
	state, blkRoot, err = prepareForkchoiceState(ctx, 103, [32]byte{'d'}, [32]byte{'b'}, [32]byte{'D'}, 2, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	require.NoError(t, f.UpdateJustifiedCheckpoint(&forkchoicetypes.Checkpoint{Epoch: 2, Root: ha}))
	headRoot, err = f.Head(ctx, []uint64{100})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'d'}, headRoot)
	require.Equal(t, uint64(0), f.store.nodes[4].weight)
	require.Equal(t, uint64(100), f.store.nodes[3].weight)

	// Update unrealized justification, c becomes head
	f.updateUnrealizedCheckpoints()
	headRoot, err = f.Head(ctx, []uint64{100})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'c'}, headRoot)
}

//
//
//        Epoch 1                Epoch 2               Epoch 3
//                        |                      |
//                        |                      |
//   A <-- B <-- C <-- D <-- E <-- F <-- G <-- H |
//                        |        \             |
//                        |         --------------- I
//                        |                      |
//
//   E justifies A. G justifies E.
//
func TestStore_NoDeadLock(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()

	// Epoch 1 blocks
	state, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, [32]byte{'D'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))

	// Epoch 2 Blocks
	state, blkRoot, err = prepareForkchoiceState(ctx, 104, [32]byte{'e'}, [32]byte{'d'}, [32]byte{'E'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'e'}, 1))
	state, blkRoot, err = prepareForkchoiceState(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, [32]byte{'F'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'f'}, 1))
	state, blkRoot, err = prepareForkchoiceState(ctx, 106, [32]byte{'g'}, [32]byte{'f'}, [32]byte{'G'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'g'}, 2))
	require.NoError(t, f.store.setUnrealizedFinalizedEpoch([32]byte{'g'}, 1))
	f.store.unrealizedJustifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 2}
	f.store.unrealizedFinalizedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 1}
	state, blkRoot, err = prepareForkchoiceState(ctx, 107, [32]byte{'h'}, [32]byte{'g'}, [32]byte{'H'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'h'}, 2))
	require.NoError(t, f.store.setUnrealizedFinalizedEpoch([32]byte{'h'}, 1))
	// Add an attestation for h
	f.ProcessAttestation(ctx, []uint64{0}, [32]byte{'h'}, 1)

	// Epoch 3
	// Current Head is H
	headRoot, err := f.Head(ctx, []uint64{100})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'h'}, headRoot)
	require.Equal(t, types.Epoch(0), f.JustifiedCheckpoint().Epoch)

	// Insert Block I, it becomes Head
	hr := [32]byte{'i'}
	state, blkRoot, err = prepareForkchoiceState(ctx, 108, hr, [32]byte{'f'}, [32]byte{'I'}, 1, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	ha := [32]byte{'a'}
	require.NoError(t, f.UpdateJustifiedCheckpoint(&forkchoicetypes.Checkpoint{Epoch: 1, Root: ha}))
	headRoot, err = f.Head(ctx, []uint64{100})
	require.NoError(t, err)
	require.Equal(t, hr, headRoot)
	require.Equal(t, types.Epoch(1), f.JustifiedCheckpoint().Epoch)
	require.Equal(t, types.Epoch(0), f.FinalizedCheckpoint().Epoch)

	// Realized Justified checkpoints, H becomes head
	f.updateUnrealizedCheckpoints()
	headRoot, err = f.Head(ctx, []uint64{100})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'h'}, headRoot)
	require.Equal(t, types.Epoch(2), f.JustifiedCheckpoint().Epoch)
	require.Equal(t, types.Epoch(1), f.FinalizedCheckpoint().Epoch)
}

//    Epoch  1       |         Epoch 2
//                   |
//              -- D (late) --
//             /     |
//  A <- B <- C      |
//             \     |
//              -- -- -- E <- F <- G <- H
//                   |
//
// D justifies and comes late.
//
func TestStore_ForkNextEpoch(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()

	// Epoch 1 blocks (D does not arrive)
	state, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))

	// Epoch 2 blocks
	state, blkRoot, err = prepareForkchoiceState(ctx, 104, [32]byte{'e'}, [32]byte{'c'}, [32]byte{'E'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, [32]byte{'F'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 106, [32]byte{'g'}, [32]byte{'f'}, [32]byte{'G'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 107, [32]byte{'h'}, [32]byte{'g'}, [32]byte{'H'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))

	// Insert an attestation to H, H is head
	f.ProcessAttestation(ctx, []uint64{0}, [32]byte{'h'}, 1)
	headRoot, err := f.Head(ctx, []uint64{100})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'h'}, headRoot)
	require.Equal(t, types.Epoch(0), f.JustifiedCheckpoint().Epoch)

	// D arrives late, D is head
	state, blkRoot, err = prepareForkchoiceState(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, [32]byte{'D'}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'d'}, 1))
	f.store.unrealizedJustifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 1}
	f.updateUnrealizedCheckpoints()
	headRoot, err = f.Head(ctx, []uint64{100})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'d'}, headRoot)
	require.Equal(t, types.Epoch(1), f.JustifiedCheckpoint().Epoch)
	// nodes[8] = D since it's late!
	require.Equal(t, uint64(0), f.store.nodes[8].weight)
	require.Equal(t, uint64(100), f.store.nodes[7].weight)
}

func TestStore_PullTips_Heuristics(t *testing.T) {
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
