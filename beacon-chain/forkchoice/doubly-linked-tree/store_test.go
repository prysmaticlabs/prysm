package doublylinkedtree

import (
	"context"
	"testing"
	"time"

	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestStore_JustifiedEpoch(t *testing.T) {
	j := types.Epoch(100)
	f := setup(j, j)
	require.Equal(t, j, f.JustifiedCheckpoint().Epoch)
}

func TestStore_FinalizedEpoch(t *testing.T) {
	j := types.Epoch(50)
	f := setup(j, j)
	require.Equal(t, j, f.FinalizedCheckpoint().Epoch)
}

func TestStore_NodeCount(t *testing.T) {
	f := setup(0, 0)
	state, blkRoot, err := prepareForkchoiceState(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(context.Background(), state, blkRoot))
	require.Equal(t, 2, f.NodeCount())
}

func TestStore_NodeByRoot(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	state, blkRoot, err := prepareForkchoiceState(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(context.Background(), 2, indexToHash(2), indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	node0 := f.store.treeRootNode
	node1 := node0.children[0]
	node2 := node1.children[0]

	expectedRoots := map[[32]byte]*Node{
		params.BeaconConfig().ZeroHash: node0,
		indexToHash(1):                 node1,
		indexToHash(2):                 node2,
	}

	require.Equal(t, 3, f.NodeCount())
	for root, node := range f.store.nodeByRoot {
		v, ok := expectedRoots[root]
		require.Equal(t, ok, true)
		require.Equal(t, v, node)
	}
}

func TestForkChoice_HasNode(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	state, blkRoot, err := prepareForkchoiceState(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	require.Equal(t, true, f.HasNode(indexToHash(1)))
}

func TestStore_Head_UnknownJustifiedRoot(t *testing.T) {
	f := setup(0, 0)

	f.store.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 1, Root: [32]byte{'a'}}
	_, err := f.store.head(context.Background())
	assert.ErrorContains(t, errUnknownJustifiedRoot.Error(), err)
}

func TestStore_Head_Itself(t *testing.T) {
	f := setup(0, 0)
	state, blkRoot, err := prepareForkchoiceState(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(context.Background(), state, blkRoot))

	// Since the justified node does not have a best descendant so the best node
	// is itself.
	f.store.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 0, Root: indexToHash(1)}
	h, err := f.store.head(context.Background())
	require.NoError(t, err)
	assert.Equal(t, indexToHash(1), h)
}

func TestStore_Head_BestDescendant(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	state, blkRoot, err := prepareForkchoiceState(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(context.Background(), 2, indexToHash(2), indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(context.Background(), 3, indexToHash(3), indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(context.Background(), 4, indexToHash(4), indexToHash(2), params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	f.store.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 0, Root: indexToHash(1)}
	h, err := f.store.head(context.Background())
	require.NoError(t, err)
	require.Equal(t, h, indexToHash(4))
}

func TestStore_UpdateBestDescendant_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := setup(0, 0)
	state, blkRoot, err := prepareForkchoiceState(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	cancel()
	state, blkRoot, err = prepareForkchoiceState(ctx, 2, indexToHash(2), indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	err = f.InsertNode(ctx, state, blkRoot)
	require.ErrorContains(t, "context canceled", err)
}

func TestStore_Insert(t *testing.T) {
	// The new node does not have a parent.
	treeRootNode := &Node{slot: 0, root: indexToHash(0)}
	nodeByRoot := map[[32]byte]*Node{indexToHash(0): treeRootNode}
	nodeByPayload := map[[32]byte]*Node{indexToHash(0): treeRootNode}
	jc := &forkchoicetypes.Checkpoint{Epoch: 0}
	fc := &forkchoicetypes.Checkpoint{Epoch: 0}
	s := &Store{nodeByRoot: nodeByRoot, treeRootNode: treeRootNode, nodeByPayload: nodeByPayload, justifiedCheckpoint: jc, finalizedCheckpoint: fc, highestReceivedNode: &Node{}}
	payloadHash := [32]byte{'a'}
	_, err := s.insert(context.Background(), 100, indexToHash(100), indexToHash(0), payloadHash, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, len(s.nodeByRoot), "Did not insert block")
	assert.Equal(t, (*Node)(nil), treeRootNode.parent, "Incorrect parent")
	assert.Equal(t, 1, len(treeRootNode.children), "Incorrect children number")
	assert.Equal(t, payloadHash, treeRootNode.children[0].payloadHash, "Incorrect payload hash")
	child := treeRootNode.children[0]
	assert.Equal(t, types.Epoch(1), child.justifiedEpoch, "Incorrect justification")
	assert.Equal(t, types.Epoch(1), child.finalizedEpoch, "Incorrect finalization")
	assert.Equal(t, indexToHash(100), child.root, "Incorrect root")
}

func TestStore_Prune_MoreThanThreshold(t *testing.T) {
	// Define 100 nodes in store.
	numOfNodes := uint64(100)
	f := setup(0, 0)
	ctx := context.Background()
	state, blkRoot, err := prepareForkchoiceState(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	for i := uint64(2); i < numOfNodes; i++ {
		state, blkRoot, err = prepareForkchoiceState(ctx, types.Slot(i), indexToHash(i), indexToHash(i-1), params.BeaconConfig().ZeroHash, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	}

	s := f.store

	// Finalized root is at index 99 so everything before 99 should be pruned.
	s.finalizedCheckpoint.Root = indexToHash(99)
	require.NoError(t, s.prune(context.Background()))
	assert.Equal(t, 1, len(s.nodeByRoot), "Incorrect nodes count")
}

func TestStore_Prune_MoreThanOnce(t *testing.T) {
	// Define 100 nodes in store.
	numOfNodes := uint64(100)
	f := setup(0, 0)
	ctx := context.Background()
	state, blkRoot, err := prepareForkchoiceState(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	for i := uint64(2); i < numOfNodes; i++ {
		state, blkRoot, err = prepareForkchoiceState(ctx, types.Slot(i), indexToHash(i), indexToHash(i-1), params.BeaconConfig().ZeroHash, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	}

	s := f.store

	// Finalized root is at index 11 so everything before 11 should be pruned.
	s.finalizedCheckpoint.Root = indexToHash(10)
	require.NoError(t, s.prune(context.Background()))
	assert.Equal(t, 90, len(s.nodeByRoot), "Incorrect nodes count")

	// One more time.
	s.finalizedCheckpoint.Root = indexToHash(20)
	require.NoError(t, s.prune(context.Background()))
	assert.Equal(t, 80, len(s.nodeByRoot), "Incorrect nodes count")
}

func TestStore_Prune_ReturnEarly(t *testing.T) {
	// Define 100 nodes in store.
	numOfNodes := uint64(100)
	f := setup(0, 0)
	ctx := context.Background()
	state, blkRoot, err := prepareForkchoiceState(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	for i := uint64(2); i < numOfNodes; i++ {
		state, blkRoot, err = prepareForkchoiceState(ctx, types.Slot(i), indexToHash(i), indexToHash(i-1), params.BeaconConfig().ZeroHash, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	}
	require.NoError(t, f.store.prune(ctx))
	nodeCount := f.NodeCount()
	require.NoError(t, f.store.prune(ctx))
	require.Equal(t, nodeCount, f.NodeCount())
}

// This unit tests starts with a simple branch like this
//
//       - 1
//     /
// -- 0 -- 2
//
// And we finalize 1. As a result only 1 should survive
func TestStore_Prune_NoDanglingBranch(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	state, blkRoot, err := prepareForkchoiceState(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 2, indexToHash(2), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))

	s := f.store
	s.finalizedCheckpoint.Root = indexToHash(1)
	require.NoError(t, s.prune(context.Background()))
	require.Equal(t, len(s.nodeByRoot), 1)
}

// This test starts with the following branching diagram
/// We start with the following diagram
//
//                E -- F
//               /
//         C -- D
//        /      \
//  A -- B        G -- H -- I
//        \        \
//         J        -- K -- L
//
//
func TestStore_tips(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)

	state, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'j'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 104, [32]byte{'e'}, [32]byte{'d'}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 104, [32]byte{'g'}, [32]byte{'d'}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 105, [32]byte{'h'}, [32]byte{'g'}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 105, [32]byte{'k'}, [32]byte{'g'}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 106, [32]byte{'i'}, [32]byte{'h'}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 106, [32]byte{'l'}, [32]byte{'k'}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	expectedMap := map[[32]byte]types.Slot{
		{'f'}: 105,
		{'i'}: 106,
		{'l'}: 106,
		{'j'}: 102,
	}
	roots, slots := f.store.tips()
	for i, r := range roots {
		expectedSlot, ok := expectedMap[r]
		require.Equal(t, true, ok)
		require.Equal(t, slots[i], expectedSlot)
	}
}

func TestStore_PruneMapsNodes(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	state, blkRoot, err := prepareForkchoiceState(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 2, indexToHash(2), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))

	s := f.store
	s.finalizedCheckpoint.Root = indexToHash(1)
	require.NoError(t, s.prune(context.Background()))
	require.Equal(t, len(s.nodeByRoot), 1)

}

func TestStore_HasParent(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	state, blkRoot, err := prepareForkchoiceState(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 2, indexToHash(2), indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 3, indexToHash(3), indexToHash(2), params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	require.Equal(t, false, f.HasParent(params.BeaconConfig().ZeroHash))
	require.Equal(t, true, f.HasParent(indexToHash(1)))
	require.Equal(t, true, f.HasParent(indexToHash(2)))
	require.Equal(t, true, f.HasParent(indexToHash(3)))
	require.Equal(t, false, f.HasParent(indexToHash(4)))
}

func TestForkChoice_HighestReceivedBlockSlotRoot(t *testing.T) {
	f := setup(1, 1)
	s := f.store
	_, err := s.insert(context.Background(), 100, [32]byte{'A'}, [32]byte{}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.Equal(t, types.Slot(100), s.highestReceivedNode.slot)
	require.Equal(t, types.Slot(100), f.HighestReceivedBlockSlot())
	require.Equal(t, [32]byte{'A'}, f.HighestReceivedBlockRoot())
	_, err = s.insert(context.Background(), 1000, [32]byte{'B'}, [32]byte{}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.Equal(t, types.Slot(1000), s.highestReceivedNode.slot)
	require.Equal(t, types.Slot(1000), f.HighestReceivedBlockSlot())
	require.Equal(t, [32]byte{'B'}, f.HighestReceivedBlockRoot())
	_, err = s.insert(context.Background(), 500, [32]byte{'C'}, [32]byte{}, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.Equal(t, types.Slot(1000), s.highestReceivedNode.slot)
	require.Equal(t, types.Slot(1000), f.HighestReceivedBlockSlot())
	require.Equal(t, [32]byte{'B'}, f.HighestReceivedBlockRoot())
}

func TestForkChoice_ReceivedBlocksLastEpoch(t *testing.T) {
	f := setup(1, 1)
	s := f.store
	b := [32]byte{}

	// Make sure it doesn't underflow
	s.genesisTime = uint64(time.Now().Add(time.Duration(-1*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	_, err := s.insert(context.Background(), 1, [32]byte{'a'}, b, b, 1, 1)
	require.NoError(t, err)
	count, err := f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
	require.Equal(t, types.Slot(1), f.HighestReceivedBlockSlot())

	// 64
	// Received block last epoch is 1
	_, err = s.insert(context.Background(), 64, [32]byte{'A'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-64*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
	require.Equal(t, types.Slot(64), f.HighestReceivedBlockSlot())

	// 64 65
	// Received block last epoch is 2
	_, err = s.insert(context.Background(), 65, [32]byte{'B'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-65*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(2), count)
	require.Equal(t, types.Slot(65), f.HighestReceivedBlockSlot())

	// 64 65 66
	// Received block last epoch is 3
	_, err = s.insert(context.Background(), 66, [32]byte{'C'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-66*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(3), count)
	require.Equal(t, types.Slot(66), f.HighestReceivedBlockSlot())

	// 64 65 66
	//       98
	// Received block last epoch is 1
	_, err = s.insert(context.Background(), 98, [32]byte{'D'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-98*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
	require.Equal(t, types.Slot(98), f.HighestReceivedBlockSlot())

	// 64 65 66
	//       98
	//              132
	// Received block last epoch is 1
	_, err = s.insert(context.Background(), 132, [32]byte{'E'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-132*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
	require.Equal(t, types.Slot(132), f.HighestReceivedBlockSlot())

	// 64 65 66
	//       98
	//              132
	//       99
	// Received block last epoch is still 1. 99 is outside the window
	_, err = s.insert(context.Background(), 99, [32]byte{'F'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-132*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
	require.Equal(t, types.Slot(132), f.HighestReceivedBlockSlot())

	// 64 65 66
	//       98
	//              132
	//       99 100
	// Received block last epoch is still 1. 100 is at the same position as 132
	_, err = s.insert(context.Background(), 100, [32]byte{'G'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-132*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
	require.Equal(t, types.Slot(132), f.HighestReceivedBlockSlot())

	// 64 65 66
	//       98
	//              132
	//       99 100 101
	// Received block last epoch is 2. 101 is within the window
	_, err = s.insert(context.Background(), 101, [32]byte{'H'}, b, b, 1, 1)
	require.NoError(t, err)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-132*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(2), count)
	require.Equal(t, types.Slot(132), f.HighestReceivedBlockSlot())

	s.genesisTime = uint64(time.Now().Add(time.Duration(-134*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
	s.genesisTime = uint64(time.Now().Add(time.Duration(-165*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second).Unix())
	count, err = f.ReceivedBlocksLastEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(0), count)
}

func TestStore_Orphan_LateBlock(t *testing.T) {
	ctx := context.Background()
	t.Run("happy case, early block", func(tt *testing.T) {
		f := setup(0, 0)
		f.store.committeeBalance = params.BeaconConfig().MaxEffectiveBalance * 100
		balances := make([]uint64, 100)
		for i := range balances {
			balances[i] = params.BeaconConfig().MaxEffectiveBalance
		}

		// Previous Head
		driftGenesisTime(f, 1, 1)
		root := [32]byte{'a'}
		state, blkRoot, err := prepareForkchoiceState(ctx, 1, root, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		attestingIndices := make([]uint64, 100)
		for i := range attestingIndices {
			attestingIndices[i] = uint64(i)
		}
		f.ProcessAttestation(ctx, attestingIndices, root, 0)
		headRoot, err := f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		pbScore, err := computeProposerBoostScore(balances)
		require.NoError(t, err)
		expectedBalance := 100*params.BeaconConfig().MaxEffectiveBalance + pbScore
		require.Equal(t, expectedBalance, f.store.headNode.balance)
		require.Equal(t, f.store.headNode, f.store.proposerHeadNode)

		// early block
		require.NoError(t, f.NewSlot(ctx, 2))
		driftGenesisTime(f, 2, 1)
		root = [32]byte{'b'}
		state, blkRoot, err = prepareForkchoiceState(ctx, 2, root, [32]byte{'a'}, [32]byte{'B'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, f.store.headNode, f.store.proposerHeadNode)
		expectedBalance = pbScore
		require.Equal(t, expectedBalance, f.store.headNode.balance)

		// At y is still head
		driftGenesisTime(f, 2, params.BeaconConfig().SecondsPerETH1Block-1)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, f.store.headNode, f.store.proposerHeadNode)
		expectedBalance = pbScore
		require.Equal(t, expectedBalance, f.store.headNode.balance)

		// At 0 is still head
		driftGenesisTime(f, 3, 0)
		require.NoError(t, f.NewSlot(ctx, 3))
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, f.store.headNode, f.store.proposerHeadNode)
		expectedBalance = uint64(0)
		require.Equal(t, expectedBalance, f.store.headNode.balance)
	})
	t.Run("vanilla case, late block", func(tt *testing.T) {
		f := setup(0, 0)
		f.store.committeeBalance = params.BeaconConfig().MaxEffectiveBalance * 100
		balances := make([]uint64, 300)
		for i := range balances {
			balances[i] = params.BeaconConfig().MaxEffectiveBalance
		}

		// Previous Head
		driftGenesisTime(f, 1, 1)
		root := [32]byte{'a'}
		state, blkRoot, err := prepareForkchoiceState(ctx, 1, root, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		attestingIndices := make([]uint64, 300)
		for i := range attestingIndices {
			attestingIndices[i] = uint64(i)
		}
		f.ProcessAttestation(ctx, attestingIndices[:100], root, 0)
		headRoot, err := f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		pbScore, err := computeProposerBoostScore(balances)
		require.NoError(t, err)
		expectedBalance := 100*params.BeaconConfig().MaxEffectiveBalance + pbScore
		require.Equal(t, expectedBalance, f.store.headNode.balance)
		require.Equal(t, f.store.headNode, f.store.proposerHeadNode)

		// late block low attestation count
		previousHeadRoot := root
		require.NoError(t, f.NewSlot(ctx, 2))
		driftGenesisTime(f, 2, params.BeaconConfig().OrphanLateBlockFirstThreshold+1)
		root = [32]byte{'b'}
		state, blkRoot, err = prepareForkchoiceState(ctx, 2, root, [32]byte{'a'}, [32]byte{'B'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		f.ProcessAttestation(ctx, attestingIndices[100:101], root, 0)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, previousHeadRoot, f.store.proposerHeadNode.root)
		expectedBalance = 100 * params.BeaconConfig().MaxEffectiveBalance
		require.Equal(t, expectedBalance, f.store.proposerHeadNode.balance)
		expectedBalance = params.BeaconConfig().MaxEffectiveBalance
		require.Equal(t, expectedBalance, f.store.headNode.balance)

		// At y is still not head
		driftGenesisTime(f, 2, params.BeaconConfig().SecondsPerSlot-1)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, expectedBalance, f.store.headNode.balance)
		require.Equal(t, root, headRoot)
		require.Equal(t, previousHeadRoot, f.store.proposerHeadNode.root)

		// At 0 is still not head
		driftGenesisTime(f, 3, 0)
		require.NoError(t, f.NewSlot(ctx, 3))
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, previousHeadRoot, f.store.proposerHeadNode.root)
		require.Equal(t, expectedBalance, f.store.headNode.balance)
	})
	t.Run("late block, voted at y", func(tt *testing.T) {
		f := setup(0, 0)
		f.store.committeeBalance = params.BeaconConfig().MaxEffectiveBalance * 100
		balances := make([]uint64, 300)
		for i := range balances {
			balances[i] = params.BeaconConfig().MaxEffectiveBalance
		}

		// Previous Head
		driftGenesisTime(f, 1, 1)
		root := [32]byte{'a'}
		state, blkRoot, err := prepareForkchoiceState(ctx, 1, root, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		attestingIndices := make([]uint64, 300)
		for i := range attestingIndices {
			attestingIndices[i] = uint64(i)
		}
		f.ProcessAttestation(ctx, attestingIndices[:100], root, 0)
		headRoot, err := f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		pbScore, err := computeProposerBoostScore(balances)
		require.NoError(t, err)
		expectedBalance := 100*params.BeaconConfig().MaxEffectiveBalance + pbScore
		require.Equal(t, expectedBalance, f.store.headNode.balance)
		require.Equal(t, f.store.headNode, f.store.proposerHeadNode)

		// late block high attestation count
		require.NoError(t, f.NewSlot(ctx, 2))
		driftGenesisTime(f, 2, params.BeaconConfig().OrphanLateBlockFirstThreshold+1)
		root = [32]byte{'b'}
		state, blkRoot, err = prepareForkchoiceState(ctx, 2, root, [32]byte{'a'}, [32]byte{'B'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		f.ProcessAttestation(ctx, attestingIndices[100:200], root, 0)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, root, f.store.proposerHeadNode.root)
		expectedBalance = 100 * params.BeaconConfig().MaxEffectiveBalance
		require.Equal(t, expectedBalance, f.store.proposerHeadNode.balance)

		// At y is still head
		driftGenesisTime(f, 2, params.BeaconConfig().SecondsPerSlot-1)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, expectedBalance, f.store.headNode.balance)
		require.Equal(t, root, headRoot)
		require.Equal(t, root, f.store.proposerHeadNode.root)

		// At 0 is still head
		driftGenesisTime(f, 3, 0)
		require.NoError(t, f.NewSlot(ctx, 3))
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, root, f.store.proposerHeadNode.root)
		require.Equal(t, expectedBalance, f.store.headNode.balance)
	})
	t.Run("late block, voted between y and 0", func(tt *testing.T) {
		f := setup(0, 0)
		f.store.committeeBalance = params.BeaconConfig().MaxEffectiveBalance * 100
		balances := make([]uint64, 300)
		for i := range balances {
			balances[i] = params.BeaconConfig().MaxEffectiveBalance
		}

		// Previous Head
		driftGenesisTime(f, 1, 1)
		root := [32]byte{'a'}
		state, blkRoot, err := prepareForkchoiceState(ctx, 1, root, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		attestingIndices := make([]uint64, 300)
		for i := range attestingIndices {
			attestingIndices[i] = uint64(i)
		}
		f.ProcessAttestation(ctx, attestingIndices[:100], root, 0)
		headRoot, err := f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		pbScore, err := computeProposerBoostScore(balances)
		require.NoError(t, err)
		expectedBalance := 100*params.BeaconConfig().MaxEffectiveBalance + pbScore
		require.Equal(t, expectedBalance, f.store.headNode.balance)
		require.Equal(t, f.store.headNode, f.store.proposerHeadNode)

		// late block low attestation count
		previousHeadRoot := root
		require.NoError(t, f.NewSlot(ctx, 2))
		driftGenesisTime(f, 2, params.BeaconConfig().OrphanLateBlockFirstThreshold+1)
		root = [32]byte{'b'}
		state, blkRoot, err = prepareForkchoiceState(ctx, 2, root, [32]byte{'a'}, [32]byte{'B'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		f.ProcessAttestation(ctx, attestingIndices[100:101], root, 0)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, previousHeadRoot, f.store.proposerHeadNode.root)
		expectedBalance = 100 * params.BeaconConfig().MaxEffectiveBalance
		require.Equal(t, expectedBalance, f.store.proposerHeadNode.balance)
		expectedBalance = params.BeaconConfig().MaxEffectiveBalance
		require.Equal(t, expectedBalance, f.store.headNode.balance)

		// At y is still not head
		driftGenesisTime(f, 2, params.BeaconConfig().SecondsPerSlot-1)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, expectedBalance, f.store.headNode.balance)
		require.Equal(t, root, headRoot)
		require.Equal(t, previousHeadRoot, f.store.proposerHeadNode.root)

		// At 0 it became head
		driftGenesisTime(f, 3, 0)
		require.NoError(t, f.NewSlot(ctx, 3))
		f.ProcessAttestation(ctx, attestingIndices[101:200], root, 0)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, root, f.store.proposerHeadNode.root)
		expectedBalance = 100 * params.BeaconConfig().MaxEffectiveBalance
		require.Equal(t, expectedBalance, f.store.headNode.balance)
	})
	t.Run("early block, forks", func(tt *testing.T) {
		// An early block arrives and becomes head, even though it forks
		// a previous block, it should also become the proposer's head
		f := setup(0, 0)
		f.store.committeeBalance = params.BeaconConfig().MaxEffectiveBalance * 100
		balances := make([]uint64, 300)
		for i := range balances {
			balances[i] = params.BeaconConfig().MaxEffectiveBalance
		}

		// Previous Head
		driftGenesisTime(f, 1, 1)
		root := [32]byte{'a'}
		state, blkRoot, err := prepareForkchoiceState(ctx, 1, root, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		attestingIndices := make([]uint64, 300)
		for i := range attestingIndices {
			attestingIndices[i] = uint64(i)
		}
		f.ProcessAttestation(ctx, attestingIndices[:100], root, 0)
		headRoot, err := f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		pbScore, err := computeProposerBoostScore(balances)
		require.NoError(t, err)
		expectedBalance := 100*params.BeaconConfig().MaxEffectiveBalance + pbScore
		require.Equal(t, expectedBalance, f.store.headNode.balance)
		require.Equal(t, f.store.headNode, f.store.proposerHeadNode)

		// Next block comes early gets low attestation count
		require.NoError(t, f.NewSlot(ctx, 2))
		driftGenesisTime(f, 2, params.BeaconConfig().OrphanLateBlockFirstThreshold-1)
		root = [32]byte{'b'}
		state, blkRoot, err = prepareForkchoiceState(ctx, 2, root, [32]byte{'a'}, [32]byte{'B'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		f.ProcessAttestation(ctx, attestingIndices[100:101], root, 0)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, root, f.store.proposerHeadNode.root)
		expectedBalance = params.BeaconConfig().MaxEffectiveBalance + pbScore
		require.Equal(t, expectedBalance, f.store.proposerHeadNode.balance)

		// Next block arrives early and forks the current head but still
		// manages to get enough votes to beat PB
		require.NoError(t, f.NewSlot(ctx, 3))
		driftGenesisTime(f, 3, params.BeaconConfig().OrphanLateBlockFirstThreshold-1)
		root = [32]byte{'c'}
		state, blkRoot, err = prepareForkchoiceState(ctx, 3, root, [32]byte{'a'}, [32]byte{'C'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		f.ProcessAttestation(ctx, attestingIndices[101:200], root, 0)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, root, f.store.proposerHeadNode.root)
		expectedBalance = 99*params.BeaconConfig().MaxEffectiveBalance + pbScore
		require.Equal(t, expectedBalance, f.store.proposerHeadNode.balance)

		// At y is still head
		driftGenesisTime(f, 3, params.BeaconConfig().SecondsPerSlot-1)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, expectedBalance, f.store.headNode.balance)
		require.Equal(t, root, headRoot)
		require.Equal(t, root, f.store.proposerHeadNode.root)

		// At 0 is still head
		driftGenesisTime(f, 4, 0)
		require.NoError(t, f.NewSlot(ctx, 4))
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, root, f.store.proposerHeadNode.root)
		expectedBalance = 99 * params.BeaconConfig().MaxEffectiveBalance
		require.Equal(t, expectedBalance, f.store.headNode.balance)
	})
	t.Run("late block, forks", func(tt *testing.T) {
		// A late block arrives and has low vote count, it orphans the
		// previous block, we orphan it back (no change from normal
		// behaviour)
		f := setup(0, 0)
		f.store.committeeBalance = params.BeaconConfig().MaxEffectiveBalance * 100
		balances := make([]uint64, 300)
		for i := range balances {
			balances[i] = params.BeaconConfig().MaxEffectiveBalance
		}

		// Previous Head
		driftGenesisTime(f, 1, 1)
		root := [32]byte{'a'}
		state, blkRoot, err := prepareForkchoiceState(ctx, 1, root, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		attestingIndices := make([]uint64, 300)
		for i := range attestingIndices {
			attestingIndices[i] = uint64(i)
		}
		f.ProcessAttestation(ctx, attestingIndices[:100], root, 0)
		headRoot, err := f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		pbScore, err := computeProposerBoostScore(balances)
		require.NoError(t, err)
		expectedBalance := 100*params.BeaconConfig().MaxEffectiveBalance + pbScore
		require.Equal(t, expectedBalance, f.store.headNode.balance)
		require.Equal(t, f.store.headNode, f.store.proposerHeadNode)

		// Next block comes early gets high attestation count
		require.NoError(t, f.NewSlot(ctx, 2))
		driftGenesisTime(f, 2, params.BeaconConfig().OrphanLateBlockFirstThreshold-1)
		root = [32]byte{'b'}
		state, blkRoot, err = prepareForkchoiceState(ctx, 2, root, [32]byte{'a'}, [32]byte{'B'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		f.ProcessAttestation(ctx, attestingIndices[100:199], root, 0)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, root, f.store.proposerHeadNode.root)
		expectedBalance = 99*params.BeaconConfig().MaxEffectiveBalance + pbScore
		require.Equal(t, expectedBalance, f.store.proposerHeadNode.balance)

		// Next block arrives late and forks the current head and does
		// not become head
		previousHeadRoot := root
		require.NoError(t, f.NewSlot(ctx, 3))
		driftGenesisTime(f, 3, params.BeaconConfig().OrphanLateBlockFirstThreshold+1)
		root = [32]byte{'c'}
		state, blkRoot, err = prepareForkchoiceState(ctx, 3, root, [32]byte{'a'}, [32]byte{'C'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		f.ProcessAttestation(ctx, attestingIndices[199:200], root, 0)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, previousHeadRoot, headRoot)
		require.Equal(t, previousHeadRoot, f.store.proposerHeadNode.root)
		expectedBalance = 99 * params.BeaconConfig().MaxEffectiveBalance
		require.Equal(t, expectedBalance, f.store.proposerHeadNode.balance)

		// At y is still not head
		driftGenesisTime(f, 3, params.BeaconConfig().SecondsPerSlot-1)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, expectedBalance, f.store.headNode.balance)
		require.Equal(t, previousHeadRoot, headRoot)
		require.Equal(t, previousHeadRoot, f.store.proposerHeadNode.root)

		// At 0 is still not head
		driftGenesisTime(f, 4, 0)
		require.NoError(t, f.NewSlot(ctx, 4))
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, previousHeadRoot, headRoot)
		require.Equal(t, previousHeadRoot, f.store.proposerHeadNode.root)
		expectedBalance = 99 * params.BeaconConfig().MaxEffectiveBalance
		require.Equal(t, expectedBalance, f.store.headNode.balance)
	})

	t.Run("late block, reorg due to attestations", func(tt *testing.T) {
		//  A <-- B
		//  \-------- C
		// In this situation head is C. Then block D arrives late
		//
		//  A <-- B
		//  \-------- C <---- D
		// But there is a reorg due to attestations and B becomes head.
		// The proposer head and head should agree.
		f := setup(0, 0)
		f.store.committeeBalance = params.BeaconConfig().MaxEffectiveBalance * 100
		balances := make([]uint64, 300)
		for i := range balances {
			balances[i] = params.BeaconConfig().MaxEffectiveBalance
		}

		// Previous Head
		driftGenesisTime(f, 1, 1)
		root := [32]byte{'a'}
		state, blkRoot, err := prepareForkchoiceState(ctx, 1, root, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		attestingIndices := make([]uint64, 300)
		for i := range attestingIndices {
			attestingIndices[i] = uint64(i)
		}
		f.ProcessAttestation(ctx, attestingIndices[:100], root, 0)
		headRoot, err := f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		pbScore, err := computeProposerBoostScore(balances)
		require.NoError(t, err)
		expectedBalance := 100*params.BeaconConfig().MaxEffectiveBalance + pbScore
		require.Equal(t, expectedBalance, f.store.headNode.balance)
		require.Equal(t, f.store.headNode, f.store.proposerHeadNode)

		// Next block comes early gets low attestation count
		require.NoError(t, f.NewSlot(ctx, 2))
		driftGenesisTime(f, 2, params.BeaconConfig().OrphanLateBlockFirstThreshold-1)
		root = [32]byte{'b'}
		state, blkRoot, err = prepareForkchoiceState(ctx, 2, root, [32]byte{'a'}, [32]byte{'B'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		f.ProcessAttestation(ctx, attestingIndices[100:101], root, 0)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, root, f.store.proposerHeadNode.root)
		expectedBalance = params.BeaconConfig().MaxEffectiveBalance + pbScore
		require.Equal(t, expectedBalance, f.store.proposerHeadNode.balance)

		// Next block arrives early and forks the current head but
		// becomes head due to PB.
		require.NoError(t, f.NewSlot(ctx, 3))
		driftGenesisTime(f, 3, params.BeaconConfig().OrphanLateBlockFirstThreshold-1)
		root = [32]byte{'c'}
		state, blkRoot, err = prepareForkchoiceState(ctx, 3, root, [32]byte{'a'}, [32]byte{'C'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		f.ProcessAttestation(ctx, attestingIndices[101:103], root, 0)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, root, f.store.proposerHeadNode.root)
		expectedBalance = 2*params.BeaconConfig().MaxEffectiveBalance + pbScore
		require.Equal(t, expectedBalance, f.store.proposerHeadNode.balance)

		// Next block arrives late we don't fork because the previous
		// head had low vote count
		require.NoError(t, f.NewSlot(ctx, 4))
		driftGenesisTime(f, 4, params.BeaconConfig().OrphanLateBlockFirstThreshold+1)
		root = [32]byte{'d'}
		state, blkRoot, err = prepareForkchoiceState(ctx, 3, root, [32]byte{'c'}, [32]byte{'D'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, root, f.store.proposerHeadNode.root)
		expectedBalance = uint64(0)
		require.Equal(t, expectedBalance, f.store.proposerHeadNode.balance)

		// At y there have been numerous attestations for the orphaned
		// block B, there's a reorg to that block
		driftGenesisTime(f, 4, params.BeaconConfig().SecondsPerSlot-1)
		f.ProcessAttestation(ctx, attestingIndices[103:200], [32]byte{'b'}, 0)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, [32]byte{'b'}, headRoot)
		require.Equal(t, headRoot, f.store.proposerHeadNode.root)
		// 97 votes from this round plus the one vote the node held
		expectedBalance = 98 * params.BeaconConfig().MaxEffectiveBalance
		require.Equal(t, expectedBalance, f.store.headNode.balance)

		// At 0 the reorg holds not head
		driftGenesisTime(f, 5, 0)
		require.NoError(t, f.NewSlot(ctx, 5))
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, [32]byte{'b'}, headRoot)
		require.Equal(t, headRoot, f.store.proposerHeadNode.root)
		require.Equal(t, expectedBalance, f.store.headNode.balance)
	})
	t.Run("unlively chain, late block", func(tt *testing.T) {
		f := setup(0, 0)
		f.store.committeeBalance = params.BeaconConfig().MaxEffectiveBalance * 100
		balances := make([]uint64, 300)
		for i := range balances {
			balances[i] = params.BeaconConfig().MaxEffectiveBalance
		}

		// Previous Head
		driftGenesisTime(f, 1, 1)
		root := [32]byte{'a'}
		state, blkRoot, err := prepareForkchoiceState(ctx, 1, root, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		attestingIndices := make([]uint64, 300)
		for i := range attestingIndices {
			attestingIndices[i] = uint64(i)
		}
		f.ProcessAttestation(ctx, attestingIndices[:1], root, 0)
		headRoot, err := f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		pbScore, err := computeProposerBoostScore(balances)
		require.NoError(t, err)
		expectedBalance := params.BeaconConfig().MaxEffectiveBalance + pbScore
		require.Equal(t, expectedBalance, f.store.headNode.balance)
		require.Equal(t, f.store.headNode, f.store.proposerHeadNode)

		// late block low attestation count
		require.NoError(t, f.NewSlot(ctx, 2))
		driftGenesisTime(f, 2, params.BeaconConfig().OrphanLateBlockFirstThreshold+1)
		root = [32]byte{'b'}
		state, blkRoot, err = prepareForkchoiceState(ctx, 2, root, [32]byte{'a'}, [32]byte{'B'}, 0, 0)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		f.ProcessAttestation(ctx, attestingIndices[100:102], root, 0)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, root, f.store.proposerHeadNode.root)
		expectedBalance = 2 * params.BeaconConfig().MaxEffectiveBalance
		require.Equal(t, expectedBalance, f.store.proposerHeadNode.balance)

		// At y is still head
		driftGenesisTime(f, 2, params.BeaconConfig().SecondsPerSlot-1)
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, expectedBalance, f.store.headNode.balance)
		require.Equal(t, root, headRoot)
		require.Equal(t, root, f.store.proposerHeadNode.root)

		// At 0 is still head
		driftGenesisTime(f, 3, 0)
		require.NoError(t, f.NewSlot(ctx, 3))
		headRoot, err = f.Head(ctx, balances)
		require.NoError(t, err)
		require.Equal(t, root, headRoot)
		require.Equal(t, root, f.store.proposerHeadNode.root)
		require.Equal(t, expectedBalance, f.store.headNode.balance)
	})

}
