package doublylinkedtree

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice"
	forkchoicetypes "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestForkChoice_UpdateBalancesPositiveChange(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, indexToHash(2), indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 3, indexToHash(3), indexToHash(2), params.BeaconConfig().ZeroHash, 0, 0))

	f.votes = []Vote{
		{indexToHash(1), indexToHash(1), 0},
		{indexToHash(2), indexToHash(2), 0},
		{indexToHash(3), indexToHash(3), 0},
	}

	// Each node gets one unique vote. The weight should look like 103 <- 102 <- 101 because
	// they get propagated back.
	require.NoError(t, f.updateBalances([]uint64{10, 20, 30}))
	s := f.store
	assert.Equal(t, uint64(10), s.nodeByRoot[indexToHash(1)].balance)
	assert.Equal(t, uint64(20), s.nodeByRoot[indexToHash(2)].balance)
	assert.Equal(t, uint64(30), s.nodeByRoot[indexToHash(3)].balance)
}

func TestForkChoice_UpdateBalancesNegativeChange(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, indexToHash(2), indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 3, indexToHash(3), indexToHash(2), params.BeaconConfig().ZeroHash, 0, 0))
	s := f.store
	s.nodeByRoot[indexToHash(1)].balance = 100
	s.nodeByRoot[indexToHash(2)].balance = 100
	s.nodeByRoot[indexToHash(3)].balance = 100

	f.balances = []uint64{100, 100, 100}
	f.votes = []Vote{
		{indexToHash(1), indexToHash(1), 0},
		{indexToHash(2), indexToHash(2), 0},
		{indexToHash(3), indexToHash(3), 0},
	}

	require.NoError(t, f.updateBalances([]uint64{10, 20, 30}))
	assert.Equal(t, uint64(10), s.nodeByRoot[indexToHash(1)].balance)
	assert.Equal(t, uint64(20), s.nodeByRoot[indexToHash(2)].balance)
	assert.Equal(t, uint64(30), s.nodeByRoot[indexToHash(3)].balance)
}

func TestForkChoice_UpdateBalancesUnderflow(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, indexToHash(2), indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 3, indexToHash(3), indexToHash(2), params.BeaconConfig().ZeroHash, 0, 0))
	s := f.store
	s.nodeByRoot[indexToHash(1)].balance = 100
	s.nodeByRoot[indexToHash(2)].balance = 100
	s.nodeByRoot[indexToHash(3)].balance = 100

	f.balances = []uint64{125, 125, 125}
	f.votes = []Vote{
		{indexToHash(1), indexToHash(1), 0},
		{indexToHash(2), indexToHash(2), 0},
		{indexToHash(3), indexToHash(3), 0},
	}

	require.NoError(t, f.updateBalances([]uint64{10, 20, 30}))
	assert.Equal(t, uint64(0), s.nodeByRoot[indexToHash(1)].balance)
	assert.Equal(t, uint64(0), s.nodeByRoot[indexToHash(2)].balance)
	assert.Equal(t, uint64(5), s.nodeByRoot[indexToHash(3)].balance)
}

func TestForkChoice_IsCanonical(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, indexToHash(2), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 3, indexToHash(3), indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 4, indexToHash(4), indexToHash(2), params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 5, indexToHash(5), indexToHash(4), params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 6, indexToHash(6), indexToHash(5), params.BeaconConfig().ZeroHash, 1, 1))

	require.Equal(t, true, f.IsCanonical(params.BeaconConfig().ZeroHash))
	require.Equal(t, false, f.IsCanonical(indexToHash(1)))
	require.Equal(t, true, f.IsCanonical(indexToHash(2)))
	require.Equal(t, false, f.IsCanonical(indexToHash(3)))
	require.Equal(t, true, f.IsCanonical(indexToHash(4)))
	require.Equal(t, true, f.IsCanonical(indexToHash(5)))
	require.Equal(t, true, f.IsCanonical(indexToHash(6)))
}

func TestForkChoice_IsCanonicalReorg(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, [32]byte{'1'}, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, [32]byte{'2'}, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 3, [32]byte{'3'}, [32]byte{'1'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 4, [32]byte{'4'}, [32]byte{'2'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 5, [32]byte{'5'}, [32]byte{'4'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 6, [32]byte{'6'}, [32]byte{'5'}, params.BeaconConfig().ZeroHash, 1, 1))

	f.store.nodesLock.Lock()
	f.store.nodeByRoot[[32]byte{'3'}].balance = 10
	require.NoError(t, f.store.treeRootNode.applyWeightChanges(ctx))
	require.Equal(t, uint64(10), f.store.nodeByRoot[[32]byte{'1'}].weight)
	require.Equal(t, uint64(0), f.store.nodeByRoot[[32]byte{'2'}].weight)

	require.NoError(t, f.store.treeRootNode.updateBestDescendant(ctx, 1, 1))
	require.DeepEqual(t, [32]byte{'3'}, f.store.treeRootNode.bestDescendant.root)
	f.store.nodesLock.Unlock()

	h, err := f.store.head(ctx, [32]byte{'1'})
	require.NoError(t, err)
	require.DeepEqual(t, [32]byte{'3'}, h)
	require.DeepEqual(t, h, f.store.headNode.root)

	require.Equal(t, true, f.IsCanonical(params.BeaconConfig().ZeroHash))
	require.Equal(t, true, f.IsCanonical([32]byte{'1'}))
	require.Equal(t, false, f.IsCanonical([32]byte{'2'}))
	require.Equal(t, true, f.IsCanonical([32]byte{'3'}))
	require.Equal(t, false, f.IsCanonical([32]byte{'4'}))
	require.Equal(t, false, f.IsCanonical([32]byte{'5'}))
	require.Equal(t, false, f.IsCanonical([32]byte{'6'}))
}

func TestForkChoice_AncestorRoot(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, indexToHash(2), indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 5, indexToHash(3), indexToHash(2), params.BeaconConfig().ZeroHash, 1, 1))
	f.store.treeRootNode = f.store.nodeByRoot[indexToHash(1)]
	f.store.treeRootNode.parent = nil

	r, err := f.AncestorRoot(ctx, indexToHash(3), 6)
	assert.NoError(t, err)
	assert.Equal(t, bytesutil.ToBytes32(r), indexToHash(3))

	_, err = f.AncestorRoot(ctx, indexToHash(3), 0)
	assert.ErrorContains(t, ErrNilNode.Error(), err)

	root, err := f.AncestorRoot(ctx, indexToHash(3), 5)
	require.NoError(t, err)
	hash3 := indexToHash(3)
	require.DeepEqual(t, hash3[:], root)
	root, err = f.AncestorRoot(ctx, indexToHash(3), 1)
	require.NoError(t, err)
	hash1 := indexToHash(1)
	require.DeepEqual(t, hash1[:], root)
}

func TestForkChoice_AncestorEqualSlot(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'1'}, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 101, [32]byte{'3'}, [32]byte{'1'}, params.BeaconConfig().ZeroHash, 1, 1))

	r, err := f.AncestorRoot(ctx, [32]byte{'3'}, 100)
	require.NoError(t, err)
	root := bytesutil.ToBytes32(r)
	require.Equal(t, root, [32]byte{'1'})
}

func TestForkChoice_AncestorLowerSlot(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'1'}, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 200, [32]byte{'3'}, [32]byte{'1'}, params.BeaconConfig().ZeroHash, 1, 1))

	r, err := f.AncestorRoot(ctx, [32]byte{'3'}, 150)
	require.NoError(t, err)
	root := bytesutil.ToBytes32(r)
	require.Equal(t, root, [32]byte{'1'})
}

func TestForkChoice_RemoveEquivocating(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)
	// Insert a block it will be head
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1))
	head, err := f.Head(ctx, params.BeaconConfig().ZeroHash, []uint64{})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'a'}, head)

	// Insert two extra blocks
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 3, [32]byte{'c'}, [32]byte{'a'}, [32]byte{'C'}, 1, 1))
	head, err = f.Head(ctx, params.BeaconConfig().ZeroHash, []uint64{})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'c'}, head)

	// Insert two attestations for block b, one for c it becomes head
	f.ProcessAttestation(ctx, []uint64{1, 2}, [32]byte{'b'}, 1)
	f.ProcessAttestation(ctx, []uint64{3}, [32]byte{'c'}, 1)
	head, err = f.Head(ctx, params.BeaconConfig().ZeroHash, []uint64{100, 200, 200, 300})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'b'}, head)

	// Process b's slashing, c is now head
	f.InsertSlashedIndex(ctx, 1)
	require.Equal(t, uint64(200), f.store.nodeByRoot[[32]byte{'b'}].balance)
	head, err = f.Head(ctx, params.BeaconConfig().ZeroHash, []uint64{100, 200, 200, 300})
	require.Equal(t, uint64(200), f.store.nodeByRoot[[32]byte{'b'}].weight)
	require.Equal(t, uint64(300), f.store.nodeByRoot[[32]byte{'c'}].weight)
	require.NoError(t, err)
	require.Equal(t, [32]byte{'c'}, head)

	// Process b's slashing again, should be a noop
	f.InsertSlashedIndex(ctx, 1)
	require.Equal(t, uint64(200), f.store.nodeByRoot[[32]byte{'b'}].balance)
	head, err = f.Head(ctx, params.BeaconConfig().ZeroHash, []uint64{100, 200, 200, 300})
	require.Equal(t, uint64(200), f.store.nodeByRoot[[32]byte{'b'}].weight)
	require.Equal(t, uint64(300), f.store.nodeByRoot[[32]byte{'c'}].weight)
	require.NoError(t, err)
	require.Equal(t, [32]byte{'c'}, head)

	// Process index where index == vote length. Should not panic.
	f.InsertSlashedIndex(ctx, types.ValidatorIndex(len(f.balances)))
	f.InsertSlashedIndex(ctx, types.ValidatorIndex(len(f.votes)))
	require.Equal(t, true, len(f.store.slashedIndices) > 0)
}

func indexToHash(i uint64) [32]byte {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], i)
	return hash.Hash(b[:])
}

func TestStore_UpdateCheckpoints(t *testing.T) {
	f := setup(1, 1)
	jr := [32]byte{'j'}
	fr := [32]byte{'f'}
	jc := &ethpb.Checkpoint{Root: jr[:], Epoch: 3}
	fc := &ethpb.Checkpoint{Root: fr[:], Epoch: 2}
	require.NoError(t, f.UpdateJustifiedCheckpoint(jc))
	require.NoError(t, f.UpdateFinalizedCheckpoint(fc))
	require.Equal(t, f.store.justifiedEpoch, jc.Epoch)
	require.Equal(t, f.store.finalizedEpoch, fc.Epoch)
}

func TestStore_CommonAncestor(t *testing.T) {
	ctx := context.Background()
	f := setup(0, 0)

	//  /-- b -- d -- e
	// a
	//  \-- c -- f
	//        \-- g
	//        \ -- h -- i -- j
	require.NoError(t, f.InsertOptimisticBlock(ctx, 0, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, [32]byte{'c'}, [32]byte{'a'}, [32]byte{'C'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 3, [32]byte{'d'}, [32]byte{'b'}, [32]byte{}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 4, [32]byte{'e'}, [32]byte{'d'}, [32]byte{}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 5, [32]byte{'f'}, [32]byte{'c'}, [32]byte{}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 6, [32]byte{'g'}, [32]byte{'c'}, [32]byte{}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 7, [32]byte{'h'}, [32]byte{'c'}, [32]byte{}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 8, [32]byte{'i'}, [32]byte{'h'}, [32]byte{}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 9, [32]byte{'j'}, [32]byte{'i'}, [32]byte{}, 1, 1))

	tests := []struct {
		name     string
		r1       [32]byte
		r2       [32]byte
		wantRoot [32]byte
	}{
		{
			name:     "Common ancestor between c and b is a",
			r1:       [32]byte{'c'},
			r2:       [32]byte{'b'},
			wantRoot: [32]byte{'a'},
		},
		{
			name:     "Common ancestor between c and d is a",
			r1:       [32]byte{'c'},
			r2:       [32]byte{'d'},
			wantRoot: [32]byte{'a'},
		},
		{
			name:     "Common ancestor between c and e is a",
			r1:       [32]byte{'c'},
			r2:       [32]byte{'e'},
			wantRoot: [32]byte{'a'},
		},
		{
			name:     "Common ancestor between g and f is c",
			r1:       [32]byte{'g'},
			r2:       [32]byte{'f'},
			wantRoot: [32]byte{'c'},
		},
		{
			name:     "Common ancestor between f and h is c",
			r1:       [32]byte{'f'},
			r2:       [32]byte{'h'},
			wantRoot: [32]byte{'c'},
		},
		{
			name:     "Common ancestor between g and h is c",
			r1:       [32]byte{'g'},
			r2:       [32]byte{'h'},
			wantRoot: [32]byte{'c'},
		},
		{
			name:     "Common ancestor between b and h is a",
			r1:       [32]byte{'b'},
			r2:       [32]byte{'h'},
			wantRoot: [32]byte{'a'},
		},
		{
			name:     "Common ancestor between e and h is a",
			r1:       [32]byte{'e'},
			r2:       [32]byte{'h'},
			wantRoot: [32]byte{'a'},
		},
		{
			name:     "Common ancestor between i and f is c",
			r1:       [32]byte{'i'},
			r2:       [32]byte{'f'},
			wantRoot: [32]byte{'c'},
		},
		{
			name:     "Common ancestor between e and h is a",
			r1:       [32]byte{'j'},
			r2:       [32]byte{'g'},
			wantRoot: [32]byte{'c'},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotRoot, err := f.CommonAncestorRoot(ctx, tc.r1, tc.r2)
			require.NoError(t, err)
			require.Equal(t, tc.wantRoot, gotRoot)
		})
	}

	// a -- b -- c -- d
	f = setup(0, 0)
	require.NoError(t, f.InsertOptimisticBlock(ctx, 0, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 1, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 2, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 3, [32]byte{'d'}, [32]byte{'c'}, [32]byte{}, 1, 1))
	tests = []struct {
		name     string
		r1       [32]byte
		r2       [32]byte
		wantRoot [32]byte
	}{
		{
			name:     "Common ancestor between a and b is a",
			r1:       [32]byte{'a'},
			r2:       [32]byte{'b'},
			wantRoot: [32]byte{'a'},
		},
		{
			name:     "Common ancestor between b and d is b",
			r1:       [32]byte{'d'},
			r2:       [32]byte{'b'},
			wantRoot: [32]byte{'b'},
		},
		{
			name:     "Common ancestor between d and a is a",
			r1:       [32]byte{'d'},
			r2:       [32]byte{'a'},
			wantRoot: [32]byte{'a'},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotRoot, err := f.CommonAncestorRoot(ctx, tc.r1, tc.r2)
			require.NoError(t, err)
			require.Equal(t, tc.wantRoot, gotRoot)
		})
	}

	// Equal inputs should return the same root.
	r, err := f.CommonAncestorRoot(ctx, [32]byte{'b'}, [32]byte{'b'})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'b'}, r)
	// Requesting finalized root (last node) should return the same root.
	r, err = f.CommonAncestorRoot(ctx, [32]byte{'a'}, [32]byte{'a'})
	require.NoError(t, err)
	require.Equal(t, [32]byte{'a'}, r)
	// Requesting unknown root
	_, err = f.CommonAncestorRoot(ctx, [32]byte{'a'}, [32]byte{'z'})
	require.ErrorIs(t, err, ErrNilNode)
	_, err = f.CommonAncestorRoot(ctx, [32]byte{'z'}, [32]byte{'a'})
	require.ErrorIs(t, err, ErrNilNode)
	require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'y'}, [32]byte{'z'}, [32]byte{}, 1, 1))
	// broken link
	_, err = f.CommonAncestorRoot(ctx, [32]byte{'y'}, [32]byte{'a'})
	require.ErrorIs(t, err, forkchoice.ErrUnknownCommonAncestor)
}

func TestStore_InsertOptimisticChain(t *testing.T) {
	f := setup(1, 1)
	blks := make([]*forkchoicetypes.BlockAndCheckpoint, 0)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = 1
	pr := [32]byte{}
	blk.Block.ParentRoot = pr[:]
	root, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := wrapper.WrappedSignedBeaconBlock(blk)
	require.NoError(t, err)
	blks = append(blks, &forkchoicetypes.BlockAndCheckpoint{Block: wsb.Block(), JustifiedEpoch: 1,
		FinalizedEpoch: 1})
	for i := uint64(2); i < 11; i++ {
		blk := util.NewBeaconBlock()
		blk.Block.Slot = types.Slot(i)
		copiedRoot := root
		blk.Block.ParentRoot = copiedRoot[:]
		wsb, err = wrapper.WrappedSignedBeaconBlock(blk)
		require.NoError(t, err)
		blks = append(blks, &forkchoicetypes.BlockAndCheckpoint{Block: wsb.Block(), JustifiedEpoch: 1,
			FinalizedEpoch: 1})
		root, err = blk.Block.HashTreeRoot()
		require.NoError(t, err)
	}
	args := make([]*forkchoicetypes.BlockAndCheckpoint, 10)
	for i := 0; i < len(blks); i++ {
		args[i] = blks[10-i-1]
	}
	require.NoError(t, f.InsertOptimisticChain(context.Background(), args))

	f = setup(1, 1)
	require.NoError(t, f.InsertOptimisticChain(context.Background(), args[2:]))
}
