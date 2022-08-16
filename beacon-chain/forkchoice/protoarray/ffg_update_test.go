package protoarray

import (
	"context"
	"testing"

	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	v3 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v3"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

// prepareForkchoiceState prepares a beacon State with the given data to mock
// insert into forkchoice
func prepareForkchoiceState(
	_ context.Context,
	slot types.Slot,
	blockRoot [32]byte,
	parentRoot [32]byte,
	payloadHash [32]byte,
	justifiedEpoch types.Epoch,
	finalizedEpoch types.Epoch,
) (state.BeaconState, [32]byte, error) {
	blockHeader := &ethpb.BeaconBlockHeader{
		ParentRoot: parentRoot[:],
	}

	executionHeader := &enginev1.ExecutionPayloadHeader{
		BlockHash: payloadHash[:],
	}

	justifiedCheckpoint := &ethpb.Checkpoint{
		Epoch: justifiedEpoch,
	}

	finalizedCheckpoint := &ethpb.Checkpoint{
		Epoch: finalizedEpoch,
	}

	base := &ethpb.BeaconStateBellatrix{
		Slot:                         slot,
		RandaoMixes:                  make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		BlockRoots:                   make([][]byte, 1),
		CurrentJustifiedCheckpoint:   justifiedCheckpoint,
		FinalizedCheckpoint:          finalizedCheckpoint,
		LatestExecutionPayloadHeader: executionHeader,
		LatestBlockHeader:            blockHeader,
	}

	base.BlockRoots[0] = append(base.BlockRoots[0], blockRoot[:]...)
	st, err := v3.InitializeFromProto(base)
	return st, blockRoot, err
}
func TestFFGUpdates_OneBranch(t *testing.T) {
	balances := []uint64{1, 1}
	f := setup(0, 0)
	ctx := context.Background()

	// The head should always start at the finalized block.
	r, err := f.Head(context.Background(), balances)
	require.NoError(t, err)
	assert.Equal(t, params.BeaconConfig().ZeroHash, r, "Incorrect head with genesis")

	// Define the following tree:
	//            0 <- justified: 0, finalized: 0
	//            |
	//            1 <- justified: 0, finalized: 0
	//            |
	//            2 <- justified: 1, finalized: 0
	//            |
	//            3 <- justified: 2, finalized: 1
	st, blkRoot, err := prepareForkchoiceState(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(context.Background(), 2, indexToHash(2), indexToHash(1), params.BeaconConfig().ZeroHash, 1, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(context.Background(), 3, indexToHash(3), indexToHash(2), params.BeaconConfig().ZeroHash, 2, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkRoot))

	// With starting justified epoch at 0, the head should be 3:
	//            0 <- start
	//            |
	//            1
	//            |
	//            2
	//            |
	//            3 <- head
	r, err = f.Head(context.Background(), balances)
	require.NoError(t, err)
	assert.Equal(t, indexToHash(3), r, "Incorrect head for with justified epoch at 0")

	// With starting justified epoch at 1, the head should be 2:
	//            0
	//            |
	//            1 <- start
	//            |
	//            2 <- head
	//            |
	//            3
	f.store.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Root: indexToHash(1), Epoch: 1}
	f.store.finalizedCheckpoint = &forkchoicetypes.Checkpoint{Root: indexToHash(0), Epoch: 0}
	r, err = f.Head(context.Background(), balances)
	require.NoError(t, err)
	assert.Equal(t, indexToHash(2), r, "Incorrect head with justified epoch at 1")

	// With starting justified epoch at 2, the head should be 3:
	//            0
	//            |
	//            1
	//            |
	//            2 <- start
	//            |
	//            3 <- head
	f.store.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Root: indexToHash(3), Epoch: 2}
	r, err = f.Head(context.Background(), balances)
	require.NoError(t, err)
	assert.Equal(t, indexToHash(3), r, "Incorrect head with justified epoch at 2")
}

func TestFFGUpdates_TwoBranches(t *testing.T) {
	balances := []uint64{1, 1}
	f := setup(0, 0)
	ctx := context.Background()

	r, err := f.Head(context.Background(), balances)
	require.NoError(t, err)
	assert.Equal(t, params.BeaconConfig().ZeroHash, r, "Incorrect head with genesis")

	// Define the following tree:
	//                                0
	//                               / \
	//  justified: 0, finalized: 0 -> 1   2 <- justified: 0, finalized: 0
	//                              |   |
	//  justified: 1, finalized: 0 -> 3   4 <- justified: 0, finalized: 0
	//                              |   |
	//  justified: 1, finalized: 0 -> 5   6 <- justified: 0, finalized: 0
	//                              |   |
	//  justified: 1, finalized: 0 -> 7   8 <- justified: 1, finalized: 0
	//                              |   |
	//  justified: 2, finalized: 0 -> 9  10 <- justified: 2, finalized: 0
	// Left branch.
	st, blkRoot, err := prepareForkchoiceState(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(context.Background(), 2, indexToHash(3), indexToHash(1), params.BeaconConfig().ZeroHash, 1, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(context.Background(), 3, indexToHash(5), indexToHash(3), params.BeaconConfig().ZeroHash, 1, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(context.Background(), 4, indexToHash(7), indexToHash(5), params.BeaconConfig().ZeroHash, 1, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(context.Background(), 4, indexToHash(9), indexToHash(7), params.BeaconConfig().ZeroHash, 2, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkRoot))
	// Right branch.
	st, blkRoot, err = prepareForkchoiceState(context.Background(), 1, indexToHash(2), params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(context.Background(), 2, indexToHash(4), indexToHash(2), params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(context.Background(), 3, indexToHash(6), indexToHash(4), params.BeaconConfig().ZeroHash, 0, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(context.Background(), 4, indexToHash(8), indexToHash(6), params.BeaconConfig().ZeroHash, 1, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(context.Background(), 4, indexToHash(10), indexToHash(8), params.BeaconConfig().ZeroHash, 2, 0)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, blkRoot))

	// With start at 0, the head should be 10:
	//           0  <-- start
	//          / \
	//         1   2
	//         |   |
	//         3   4
	//         |   |
	//         5   6
	//         |   |
	//         7   8
	//         |   |
	//         9  10 <-- head
	r, err = f.Head(context.Background(), balances)
	require.NoError(t, err)
	assert.Equal(t, indexToHash(10), r, "Incorrect head with justified epoch at 0")

	// Add a vote to 1:
	//                 0
	//                / \
	//    +1 vote -> 1   2
	//               |   |
	//               3   4
	//               |   |
	//               5   6
	//               |   |
	//               7   8
	//               |   |
	//               9  10
	f.ProcessAttestation(context.Background(), []uint64{0}, indexToHash(1), 0)

	// With the additional vote to the left branch, the head should be 9:
	//           0  <-- start
	//          / \
	//         1   2
	//         |   |
	//         3   4
	//         |   |
	//         5   6
	//         |   |
	//         7   8
	//         |   |
	// head -> 9  10
	r, err = f.Head(context.Background(), balances)
	require.NoError(t, err)
	assert.Equal(t, indexToHash(9), r, "Incorrect head with justified epoch at 0")

	// Add a vote to 2:
	//                 0
	//                / \
	//               1   2 <- +1 vote
	//               |   |
	//               3   4
	//               |   |
	//               5   6
	//               |   |
	//               7   8
	//               |   |
	//               9  10
	f.ProcessAttestation(context.Background(), []uint64{1}, indexToHash(2), 0)

	// With the additional vote to the right branch, the head should be 10:
	//           0  <-- start
	//          / \
	//         1   2
	//         |   |
	//         3   4
	//         |   |
	//         5   6
	//         |   |
	//         7   8
	//         |   |
	//         9  10 <-- head
	r, err = f.Head(context.Background(), balances)
	require.NoError(t, err)
	assert.Equal(t, indexToHash(10), r, "Incorrect head with justified epoch at 0")

	f.store.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: 1, Root: indexToHash(1)}
	r, err = f.Head(context.Background(), balances)
	require.NoError(t, err)
	assert.Equal(t, indexToHash(7), r, "Incorrect head with justified epoch at 0")
}

func setup(justifiedEpoch, finalizedEpoch types.Epoch) *ForkChoice {
	f := New()
	f.store.nodesIndices[params.BeaconConfig().ZeroHash] = 0
	f.store.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: justifiedEpoch, Root: params.BeaconConfig().ZeroHash}
	f.store.bestJustifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: justifiedEpoch, Root: params.BeaconConfig().ZeroHash}
	f.store.finalizedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: finalizedEpoch, Root: params.BeaconConfig().ZeroHash}
	f.store.nodes = append(f.store.nodes, &Node{
		slot:           0,
		root:           params.BeaconConfig().ZeroHash,
		parent:         NonExistentNode,
		justifiedEpoch: justifiedEpoch,
		finalizedEpoch: finalizedEpoch,
		bestChild:      NonExistentNode,
		bestDescendant: NonExistentNode,
		weight:         0,
	})
	return f
}
