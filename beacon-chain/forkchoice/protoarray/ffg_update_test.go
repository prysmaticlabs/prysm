package protoarray

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestFFGUpdates_OneBranch(t *testing.T) {
	balances := []uint64{1, 1}
	f := setup(0, 0)

	// The head should always start at the finalized block.
	r, err := f.Head(context.Background(), 0, params.BeaconConfig().ZeroHash, balances, 0)
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
	require.NoError(t, f.ProcessBlock(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, [32]byte{}, 0, 0))
	require.NoError(t, f.ProcessBlock(context.Background(), 2, indexToHash(2), indexToHash(1), [32]byte{}, 1, 0))
	require.NoError(t, f.ProcessBlock(context.Background(), 3, indexToHash(3), indexToHash(2), [32]byte{}, 2, 1))

	// With starting justified epoch at 0, the head should be 3:
	//            0 <- start
	//            |
	//            1
	//            |
	//            2
	//            |
	//            3 <- head
	r, err = f.Head(context.Background(), 0, params.BeaconConfig().ZeroHash, balances, 0)
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
	r, err = f.Head(context.Background(), 1, indexToHash(2), balances, 0)
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
	r, err = f.Head(context.Background(), 2, indexToHash(3), balances, 1)
	require.NoError(t, err)
	assert.Equal(t, indexToHash(3), r, "Incorrect head with justified epoch at 2")
}

func TestFFGUpdates_TwoBranches(t *testing.T) {
	balances := []uint64{1, 1}
	f := setup(0, 0)

	r, err := f.Head(context.Background(), 0, params.BeaconConfig().ZeroHash, balances, 0)
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
	require.NoError(t, f.ProcessBlock(context.Background(), 1, indexToHash(1), params.BeaconConfig().ZeroHash, [32]byte{}, 0, 0))
	require.NoError(t, f.ProcessBlock(context.Background(), 2, indexToHash(3), indexToHash(1), [32]byte{}, 1, 0))
	require.NoError(t, f.ProcessBlock(context.Background(), 3, indexToHash(5), indexToHash(3), [32]byte{}, 1, 0))
	require.NoError(t, f.ProcessBlock(context.Background(), 4, indexToHash(7), indexToHash(5), [32]byte{}, 1, 0))
	require.NoError(t, f.ProcessBlock(context.Background(), 4, indexToHash(9), indexToHash(7), [32]byte{}, 2, 0))
	// Right branch.
	require.NoError(t, f.ProcessBlock(context.Background(), 1, indexToHash(2), params.BeaconConfig().ZeroHash, [32]byte{}, 0, 0))
	require.NoError(t, f.ProcessBlock(context.Background(), 2, indexToHash(4), indexToHash(2), [32]byte{}, 0, 0))
	require.NoError(t, f.ProcessBlock(context.Background(), 3, indexToHash(6), indexToHash(4), [32]byte{}, 0, 0))
	require.NoError(t, f.ProcessBlock(context.Background(), 4, indexToHash(8), indexToHash(6), [32]byte{}, 1, 0))
	require.NoError(t, f.ProcessBlock(context.Background(), 4, indexToHash(10), indexToHash(8), [32]byte{}, 2, 0))

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
	r, err = f.Head(context.Background(), 0, params.BeaconConfig().ZeroHash, balances, 0)
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
	r, err = f.Head(context.Background(), 0, params.BeaconConfig().ZeroHash, balances, 0)
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
	r, err = f.Head(context.Background(), 0, params.BeaconConfig().ZeroHash, balances, 0)
	require.NoError(t, err)
	assert.Equal(t, indexToHash(10), r, "Incorrect head with justified epoch at 0")

	r, err = f.Head(context.Background(), 1, indexToHash(1), balances, 0)
	require.NoError(t, err)
	assert.Equal(t, indexToHash(7), r, "Incorrect head with justified epoch at 0")
}

func setup(justifiedEpoch uint64, finalizedEpoch uint64) *ForkChoice {
	f := New(0, 0, params.BeaconConfig().ZeroHash)
	f.store.nodesIndices[params.BeaconConfig().ZeroHash] = 0
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
