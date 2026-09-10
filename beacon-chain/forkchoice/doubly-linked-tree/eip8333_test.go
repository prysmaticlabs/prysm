package doublylinkedtree

import (
	"testing"

	forkchoicetypes "github.com/OffchainLabs/prysm/v7/beacon-chain/forkchoice/types"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func setupEip8333(t *testing.T, forkEpoch primitives.Epoch) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.HezeForkEpoch = forkEpoch
	params.OverrideBeaconConfig(cfg)
}

func TestStore_TargetRootForEpoch_EIP8333(t *testing.T) {
	setupEip8333(t, 2)
	ctx := t.Context()
	f := setup(1, 1)
	spe := params.BeaconConfig().SlotsPerEpoch
	zeroHash := params.BeaconConfig().ZeroHash

	// blkA at slot 32 is the first block of pre-activation epoch 1 and remains its own target.
	st, blkA, err := prepareForkchoiceState(ctx, spe, [32]byte{'a'}, zeroHash, zeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkA))
	target, err := f.TargetRootForEpoch(blkA.Root(), 1)
	require.NoError(t, err)
	require.Equal(t, blkA.Root(), target)

	// blkB at slot 63 is the boundary block of activation epoch 2.
	st, blkB, err := prepareForkchoiceState(ctx, 2*spe-1, [32]byte{'b'}, blkA.Root(), zeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkB))
	target, err = f.TargetRootForEpoch(blkB.Root(), 1)
	require.NoError(t, err)
	require.Equal(t, blkA.Root(), target)

	// blkC at slot 64, the first slot of the activation epoch, is no longer its own target.
	st, blkC, err := prepareForkchoiceState(ctx, 2*spe, [32]byte{'c'}, blkB.Root(), zeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkC))
	target, err = f.TargetRootForEpoch(blkC.Root(), 2)
	require.NoError(t, err)
	require.Equal(t, blkB.Root(), target)

	// blkD at slot 65 inherits the boundary target.
	st, blkD, err := prepareForkchoiceState(ctx, 2*spe+1, [32]byte{'d'}, blkC.Root(), zeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkD))
	target, err = f.TargetRootForEpoch(blkD.Root(), 2)
	require.NoError(t, err)
	require.Equal(t, blkB.Root(), target)

	// Pre-activation epochs resolve under the previous anchoring across the fork boundary.
	target, err = f.TargetRootForEpoch(blkD.Root(), 1)
	require.NoError(t, err)
	require.Equal(t, blkA.Root(), target)

	// Dependent roots are unchanged by the new anchoring.
	dependent, err := f.DependentRootForEpoch(blkD.Root(), 2)
	require.NoError(t, err)
	require.Equal(t, blkB.Root(), dependent)
	dependent, err = f.DependentRootForEpoch(blkD.Root(), 1)
	require.NoError(t, err)
	require.Equal(t, zeroHash, dependent)

	// blkE at slot 101 with both the boundary and first slots of epoch 3 empty:
	// the checkpoint falls back to the most recent earlier block.
	st, blkE, err := prepareForkchoiceState(ctx, 3*spe+5, [32]byte{'e'}, blkD.Root(), zeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkE))
	target, err = f.TargetRootForEpoch(blkE.Root(), 3)
	require.NoError(t, err)
	require.Equal(t, blkD.Root(), target)

	// Epochs after the node's epoch resolve to the node itself.
	target, err = f.TargetRootForEpoch(blkE.Root(), 4)
	require.NoError(t, err)
	require.Equal(t, blkE.Root(), target)

	// Walking back from a post-activation node resolves earlier epochs correctly.
	target, err = f.TargetRootForEpoch(blkE.Root(), 2)
	require.NoError(t, err)
	require.Equal(t, blkB.Root(), target)
}

func TestForkChoice_IsViableForCheckpoint_EIP8333(t *testing.T) {
	setupEip8333(t, 2)
	ctx := t.Context()
	f := setup(1, 1)
	spe := params.BeaconConfig().SlotsPerEpoch
	zeroHash := params.BeaconConfig().ZeroHash

	// blkA at slot 32 sits at pre-activation epoch 1's checkpoint slot.
	st, blkA, err := prepareForkchoiceState(ctx, spe, [32]byte{'a'}, zeroHash, zeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkA))
	viable, err := f.IsViableForCheckpoint(&forkchoicetypes.Checkpoint{Root: blkA.Root(), Epoch: 1})
	require.NoError(t, err)
	require.Equal(t, true, viable)

	// blkB at slot 63 sits exactly at the post-activation checkpoint slot for epoch 2.
	st, blkB, err := prepareForkchoiceState(ctx, 2*spe-1, [32]byte{'b'}, blkA.Root(), zeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkB))
	viable, err = f.IsViableForCheckpoint(&forkchoicetypes.Checkpoint{Root: blkB.Root(), Epoch: 2})
	require.NoError(t, err)
	require.Equal(t, true, viable)

	// blkC at the epoch start slot 64 is past the checkpoint slot and no longer viable for epoch 2.
	st, blkC, err := prepareForkchoiceState(ctx, 2*spe, [32]byte{'c'}, blkB.Root(), zeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkC))
	viable, err = f.IsViableForCheckpoint(&forkchoicetypes.Checkpoint{Root: blkC.Root(), Epoch: 2})
	require.NoError(t, err)
	require.Equal(t, false, viable)

	// blkB remains viable for epoch 2 with a descendant past the checkpoint slot.
	viable, err = f.IsViableForCheckpoint(&forkchoicetypes.Checkpoint{Root: blkB.Root(), Epoch: 2})
	require.NoError(t, err)
	require.Equal(t, true, viable)

	// blkC is viable for epoch 3 while it has no descendants at or before slot 95.
	viable, err = f.IsViableForCheckpoint(&forkchoicetypes.Checkpoint{Root: blkC.Root(), Epoch: 3})
	require.NoError(t, err)
	require.Equal(t, true, viable)
}

func TestStore_PruneIncompatibleChildren_EIP8333(t *testing.T) {
	buildTree := func(t *testing.T, f *ForkChoice) (a, b, x, c [32]byte) {
		ctx := t.Context()
		spe := params.BeaconConfig().SlotsPerEpoch
		zeroHash := params.BeaconConfig().ZeroHash
		st, blkA, err := prepareForkchoiceState(ctx, spe, [32]byte{'a'}, zeroHash, zeroHash, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, st, blkA))
		// blkB at slot 60 is the finalized checkpoint block for epoch 2, its trailing slots are empty.
		st, blkB, err := prepareForkchoiceState(ctx, 2*spe-4, [32]byte{'b'}, blkA.Root(), zeroHash, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, st, blkB))
		// blkX at slot 62 competes with the checkpoint position and conflicts with finalization.
		st, blkX, err := prepareForkchoiceState(ctx, 2*spe-2, [32]byte{'x'}, blkB.Root(), zeroHash, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, st, blkX))
		// blkC at the epoch start slot 64 descends from the checkpoint block.
		st, blkC, err := prepareForkchoiceState(ctx, 2*spe, [32]byte{'c'}, blkB.Root(), zeroHash, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, st, blkC))
		return blkA.Root(), blkB.Root(), blkX.Root(), blkC.Root()
	}

	t.Run("activated keeps the epoch start child", func(t *testing.T) {
		setupEip8333(t, 2)
		f := setup(1, 1)
		a, b, x, c := buildTree(t, f)
		f.store.finalizedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 2, Root: b}
		f.store.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 2, Root: b}
		require.NoError(t, f.store.prune(t.Context()))
		require.Equal(t, false, f.HasNode(a))
		require.Equal(t, true, f.HasNode(b))
		require.Equal(t, false, f.HasNode(x), "a competing block at or before the checkpoint slot conflicts with finalization")
		require.Equal(t, true, f.HasNode(c), "the epoch start block is compatible with the boundary checkpoint")
	})
	t.Run("not activated prunes the epoch start child", func(t *testing.T) {
		params.SetupTestConfigCleanup(t)
		f := setup(1, 1)
		a, b, x, c := buildTree(t, f)
		f.store.finalizedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 2, Root: b}
		f.store.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 2, Root: b}
		require.NoError(t, f.store.prune(t.Context()))
		require.Equal(t, false, f.HasNode(a))
		require.Equal(t, true, f.HasNode(b))
		require.Equal(t, false, f.HasNode(x))
		require.Equal(t, false, f.HasNode(c), "under the previous anchoring the epoch start block competes with the checkpoint")
	})
}
