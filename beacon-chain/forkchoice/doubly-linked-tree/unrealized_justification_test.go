package doublylinkedtree

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestStore_SetUnrealizedEpochs(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1))
	f.store.nodesLock.RLock()
	require.Equal(t, types.Epoch(1), f.store.nodeByRoot[[32]byte{'b'}].unrealizedJustifiedEpoch)
	require.Equal(t, types.Epoch(1), f.store.nodeByRoot[[32]byte{'b'}].unrealizedFinalizedEpoch)
	f.store.nodesLock.RUnlock()
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'b'}, 2))
	require.NoError(t, f.store.setUnrealizedFinalizedEpoch([32]byte{'b'}, 2))
	f.store.nodesLock.RLock()
	require.Equal(t, types.Epoch(2), f.store.nodeByRoot[[32]byte{'b'}].unrealizedJustifiedEpoch)
	require.Equal(t, types.Epoch(2), f.store.nodeByRoot[[32]byte{'b'}].unrealizedFinalizedEpoch)
	f.store.nodesLock.RUnlock()

	require.ErrorIs(t, errInvalidUnrealizedJustifiedEpoch, f.store.setUnrealizedJustifiedEpoch([32]byte{'b'}, 0))
	require.ErrorIs(t, errInvalidUnrealizedFinalizedEpoch, f.store.setUnrealizedFinalizedEpoch([32]byte{'b'}, 0))
}

func TestStore_UpdateUnrealizedCheckpoints(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1))

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
	require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1))
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'b'}, 2))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1))
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'c'}, 2))

	// Add an attestation to c, it is head
	f.ProcessAttestation(ctx, []uint64{0}, [32]byte{'c'}, 1)
	headRoot, err := f.Head(ctx, [32]byte{}, []uint64{100})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'c'}, headRoot)

	// D is head even though its weight is lower.
	hr := [32]byte{'d'}
	require.NoError(t, f.InsertOptimisticBlock(ctx, 103, hr, [32]byte{'b'}, [32]byte{'D'}, 2, 1))
	require.NoError(t, f.UpdateJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: 2, Root: hr[:]}))
	headRoot, err = f.Head(ctx, [32]byte{}, []uint64{100})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'d'}, headRoot)
	require.Equal(t, uint64(0), f.store.nodeByRoot[[32]byte{'d'}].weight)
	require.Equal(t, uint64(100), f.store.nodeByRoot[[32]byte{'c'}].weight)

	// Update unrealized justification, c becomes head
	f.UpdateUnrealizedCheckpoints()
	headRoot, err = f.Head(ctx, [32]byte{}, []uint64{100})
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
	require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, [32]byte{'D'}, 0, 0))

	// Epoch 2 Blocks
	require.NoError(t, f.InsertOptimisticBlock(ctx, 104, [32]byte{'e'}, [32]byte{'d'}, [32]byte{'E'}, 0, 0))
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'e'}, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, [32]byte{'F'}, 0, 0))
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'f'}, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 106, [32]byte{'g'}, [32]byte{'f'}, [32]byte{'G'}, 0, 0))
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'g'}, 2))
	require.NoError(t, f.store.setUnrealizedFinalizedEpoch([32]byte{'g'}, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 107, [32]byte{'h'}, [32]byte{'g'}, [32]byte{'H'}, 0, 0))
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'h'}, 2))
	require.NoError(t, f.store.setUnrealizedFinalizedEpoch([32]byte{'h'}, 1))
	// Add an attestation for h
	f.ProcessAttestation(ctx, []uint64{0}, [32]byte{'h'}, 1)

	// Epoch 3
	// Current Head is H
	headRoot, err := f.Head(ctx, [32]byte{}, []uint64{100})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'h'}, headRoot)
	require.Equal(t, types.Epoch(0), f.JustifiedEpoch())

	// Insert Block I, it becomes Head
	hr := [32]byte{'i'}
	require.NoError(t, f.InsertOptimisticBlock(ctx, 108, hr, [32]byte{'f'}, [32]byte{'I'}, 1, 0))
	require.NoError(t, f.UpdateJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: 1, Root: hr[:]}))
	headRoot, err = f.Head(ctx, [32]byte{}, []uint64{100})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'i'}, headRoot)
	require.Equal(t, types.Epoch(1), f.JustifiedEpoch())
	require.Equal(t, types.Epoch(0), f.FinalizedEpoch())

	// Realized Justified checkpoints, H becomes head
	f.UpdateUnrealizedCheckpoints()
	headRoot, err = f.Head(ctx, [32]byte{}, []uint64{100})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'h'}, headRoot)
	require.Equal(t, types.Epoch(2), f.JustifiedEpoch())
	require.Equal(t, types.Epoch(1), f.FinalizedEpoch())
}

//    Epoch  1       |         Epoch 2
//                   |
//              -- D (late)
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
	require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 0, 0))

	// Epoch 2 blocks
	require.NoError(t, f.InsertOptimisticBlock(ctx, 104, [32]byte{'e'}, [32]byte{'c'}, [32]byte{'E'}, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, [32]byte{'F'}, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 106, [32]byte{'g'}, [32]byte{'f'}, [32]byte{'G'}, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 107, [32]byte{'h'}, [32]byte{'g'}, [32]byte{'H'}, 0, 0))

	// Insert an attestation to H, H is head
	f.ProcessAttestation(ctx, []uint64{0}, [32]byte{'h'}, 1)
	headRoot, err := f.Head(ctx, [32]byte{}, []uint64{100})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'h'}, headRoot)
	require.Equal(t, types.Epoch(0), f.JustifiedEpoch())

	// D arrives late, D is head
	require.NoError(t, f.InsertOptimisticBlock(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, [32]byte{'D'}, 0, 0))
	require.NoError(t, f.store.setUnrealizedJustifiedEpoch([32]byte{'d'}, 1))
	f.UpdateUnrealizedCheckpoints()
	headRoot, err = f.Head(ctx, [32]byte{}, []uint64{100})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'d'}, headRoot)
	require.Equal(t, types.Epoch(1), f.JustifiedEpoch())
	require.Equal(t, uint64(0), f.store.nodeByRoot[[32]byte{'d'}].weight)
	require.Equal(t, uint64(100), f.store.nodeByRoot[[32]byte{'h'}].weight)
}
