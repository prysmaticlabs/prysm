package protoarray

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/testing/require"
)

// We test the algorithm to check the optimistic status of a node. The
// status for this test is the following branching diagram
//
//                       -- E -- F
//                      /
//                  -- C -- D
//                 /
// 0 -- 1 -- A -- B      -- J -- K
//                 \    /
//                  -- G -- H -- I
//
// Here nodes 0, 1, A, B, C, D are fully validated and nodes
// E, F, G, H, J, K are optimistic.
// Synced Tips are nodes B, C, D
// nodes 0 and 1 are outside the Fork Choice Store.

func TestOptimistic(t *testing.T) {
	root0 := bytesutil.ToBytes32([]byte("hello0"))
	root1 := bytesutil.ToBytes32([]byte("hello1"))

	nodeA := &Node{
		slot:      types.Slot(100),
		root:      bytesutil.ToBytes32([]byte("helloA")),
		bestChild: 1,
	}
	nodeB := &Node{
		slot:      types.Slot(101),
		root:      bytesutil.ToBytes32([]byte("helloB")),
		bestChild: 2,
		parent:    0,
	}
	nodeC := &Node{
		slot:      types.Slot(102),
		root:      bytesutil.ToBytes32([]byte("helloC")),
		bestChild: 3,
		parent:    1,
	}
	nodeD := &Node{
		slot:      types.Slot(103),
		root:      bytesutil.ToBytes32([]byte("helloD")),
		bestChild: NonExistentNode,
		parent:    2,
	}
	nodeE := &Node{
		slot:      types.Slot(103),
		root:      bytesutil.ToBytes32([]byte("helloE")),
		bestChild: 5,
		parent:    2,
	}
	nodeF := &Node{
		slot:      types.Slot(104),
		root:      bytesutil.ToBytes32([]byte("helloF")),
		bestChild: NonExistentNode,
		parent:    4,
	}
	nodeG := &Node{
		slot:      types.Slot(102),
		root:      bytesutil.ToBytes32([]byte("helloG")),
		bestChild: 7,
		parent:    1,
	}
	nodeH := &Node{
		slot:      types.Slot(103),
		root:      bytesutil.ToBytes32([]byte("helloH")),
		bestChild: 8,
		parent:    6,
	}
	nodeI := &Node{
		slot:      types.Slot(104),
		root:      bytesutil.ToBytes32([]byte("helloI")),
		bestChild: NonExistentNode,
		parent:    7,
	}
	nodeJ := &Node{
		slot:      types.Slot(103),
		root:      bytesutil.ToBytes32([]byte("helloJ")),
		bestChild: 10,
		parent:    6,
	}
	nodeK := &Node{
		slot:      types.Slot(104),
		root:      bytesutil.ToBytes32([]byte("helloK")),
		bestChild: NonExistentNode,
		parent:    9,
	}
	nodes := []*Node{
		nodeA,
		nodeB,
		nodeC,
		nodeD,
		nodeE,
		nodeF,
		nodeG,
		nodeH,
		nodeI,
		nodeJ,
		nodeK,
	}
	ni := map[[32]byte]uint64{
		nodeA.root: 0,
		nodeB.root: 1,
		nodeC.root: 2,
		nodeD.root: 3,
		nodeE.root: 4,
		nodeF.root: 5,
		nodeG.root: 6,
		nodeH.root: 7,
		nodeI.root: 8,
		nodeJ.root: 9,
		nodeK.root: 10,
	}

	s := &Store{
		nodes:        nodes,
		nodesIndices: ni,
	}

	tips := map[[32]byte]types.Slot{
		nodeB.root: nodeB.slot,
		nodeC.root: nodeC.slot,
		nodeD.root: nodeD.slot,
	}
	st := &optimisticStore{
		validatedTips: tips,
	}
	f := &ForkChoice{
		store:      s,
		syncedTips: st,
	}
	ctx := context.Background()
	// We test the implementation of boundarySyncedTips
	min, max := f.boundarySyncedTips()
	require.Equal(t, min, types.Slot(101), "minimum tip slot is different")
	require.Equal(t, max, types.Slot(103), "maximum tip slot is different")

	// We test first nodes outside the Fork Choice store
	_, err := f.IsOptimistic(ctx, root0)
	require.ErrorIs(t, ErrUnknownNodeRoot, err)

	_, err = f.IsOptimistic(ctx, root1)
	require.ErrorIs(t, ErrUnknownNodeRoot, err)

	// We check all nodes in the Fork Choice store.
	op, err := f.IsOptimistic(ctx, nodeA.root)
	require.NoError(t, err)
	require.Equal(t, op, false)

	op, err = f.IsOptimistic(ctx, nodeB.root)
	require.NoError(t, err)
	require.Equal(t, op, false)

	op, err = f.IsOptimistic(ctx, nodeC.root)
	require.NoError(t, err)
	require.Equal(t, op, false)

	op, err = f.IsOptimistic(ctx, nodeD.root)
	require.NoError(t, err)
	require.Equal(t, op, false)

	op, err = f.IsOptimistic(ctx, nodeE.root)
	require.NoError(t, err)
	require.Equal(t, op, true)

	op, err = f.IsOptimistic(ctx, nodeF.root)
	require.NoError(t, err)
	require.Equal(t, op, true)

	op, err = f.IsOptimistic(ctx, nodeG.root)
	require.NoError(t, err)
	require.Equal(t, op, true)

	op, err = f.IsOptimistic(ctx, nodeH.root)
	require.NoError(t, err)
	require.Equal(t, op, true)

	op, err = f.IsOptimistic(ctx, nodeI.root)
	require.NoError(t, err)
	require.Equal(t, op, true)

	op, err = f.IsOptimistic(ctx, nodeJ.root)
	require.NoError(t, err)
	require.Equal(t, op, true)

	op, err = f.IsOptimistic(ctx, nodeK.root)
	require.NoError(t, err)
	require.Equal(t, op, true)

	// request a write Lock to synced Tips regression #10289
	f.syncedTips.Lock()
	defer f.syncedTips.Unlock()
}

// This tests the algorithm to update syncedTips
// We start with the following diagram
//
//                E -- F
//               /
//         C -- D
//        /      \
//  A -- B        G -- H -- I
//        \        \
//         J        -- K -- L
//
// And every block in the Fork choice is optimistic. Synced_Tips contains a
// single block that is outside of Fork choice
//
func TestSetOptimisticToValid(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)

	require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'j'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 104, [32]byte{'e'}, [32]byte{'d'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 104, [32]byte{'g'}, [32]byte{'d'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'h'}, [32]byte{'g'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'k'}, [32]byte{'g'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 106, [32]byte{'i'}, [32]byte{'h'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 106, [32]byte{'l'}, [32]byte{'k'}, params.BeaconConfig().ZeroHash, 1, 1))
	tests := []struct {
		root      [32]byte                // the root of the new VALID block
		tips      map[[32]byte]types.Slot // the old synced tips
		newTips   map[[32]byte]types.Slot // the updated synced tips
		wantedErr error
	}{
		{
			[32]byte{'i'},
			map[[32]byte]types.Slot{[32]byte{'z'}: 90},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
				[32]byte{'d'}: 103,
				[32]byte{'g'}: 104,
				[32]byte{'i'}: 106,
			},
			nil,
		},
		{
			[32]byte{'i'},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
				[32]byte{'d'}: 103,
			},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
				[32]byte{'d'}: 103,
				[32]byte{'g'}: 104,
				[32]byte{'i'}: 106,
			},
			nil,
		},
		{
			[32]byte{'i'},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
				[32]byte{'d'}: 103,
				[32]byte{'e'}: 103,
			},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
				[32]byte{'e'}: 104,
				[32]byte{'g'}: 104,
				[32]byte{'i'}: 106,
			},
			nil,
		},
		{
			[32]byte{'j'},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
				[32]byte{'f'}: 105,
				[32]byte{'g'}: 104,
				[32]byte{'i'}: 106,
			},
			map[[32]byte]types.Slot{
				[32]byte{'f'}: 105,
				[32]byte{'g'}: 104,
				[32]byte{'i'}: 106,
				[32]byte{'j'}: 102,
			},
			nil,
		},
		{
			[32]byte{'g'},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
				[32]byte{'f'}: 105,
				[32]byte{'g'}: 104,
				[32]byte{'i'}: 106,
			},
			map[[32]byte]types.Slot{
				[32]byte{'f'}: 105,
				[32]byte{'g'}: 104,
				[32]byte{'i'}: 106,
				[32]byte{'j'}: 102,
			},
			errInvalidBestChildIndex,
		},
		{
			[32]byte{'p'},
			map[[32]byte]types.Slot{},
			map[[32]byte]types.Slot{},
			errInvalidNodeIndex,
		},
	}
	for _, tc := range tests {
		f.syncedTips.Lock()
		f.syncedTips.validatedTips = tc.tips
		f.syncedTips.Unlock()
		err := f.SetOptimisticToValid(context.Background(), tc.root)
		if tc.wantedErr != nil {
			require.ErrorIs(t, err, tc.wantedErr)
		} else {
			require.NoError(t, err)
			f.syncedTips.RLock()
			require.DeepEqual(t, f.syncedTips.validatedTips, tc.newTips)
			f.syncedTips.RUnlock()
		}
	}
}

// We test the algorithm to update a node from SYNCING to INVALID
// We start with the same diagram as above:
//
//                         E(2) -- F(1)
//                        /
//             C(7) -- D(6)
//            /           \
//  A(10) -- B(9)          G(3) -- H(1) -- I(0)
//            \               \
//             J(1)             -- K(1) -- L(0)
//
// And every block in the Fork choice is optimistic. Synced_Tips contains a
// single block that is outside of Fork choice. The numbers in parentheses are
// the weights of the nodes before removal
//
func TestSetOptimisticToInvalid(t *testing.T) {
	tests := []struct {
		root              [32]byte                // the root of the new INVALID block
		tips              map[[32]byte]types.Slot // the old synced tips
		wantedParentTip   bool
		newBestChild      uint64
		newBestDescendant uint64
		newParentWeight   uint64
		returnedRoots     [][32]byte
	}{
		{
			[32]byte{'j'},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
				[32]byte{'d'}: 103,
				[32]byte{'g'}: 104,
			},
			false,
			3,
			4,
			8,
			[][32]byte{[32]byte{'j'}},
		},
		{
			[32]byte{'j'},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
			},
			true,
			3,
			4,
			8,
			[][32]byte{[32]byte{'j'}},
		},
		{
			[32]byte{'i'},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
				[32]byte{'d'}: 103,
				[32]byte{'g'}: 104,
				[32]byte{'h'}: 105,
			},
			true,
			NonExistentNode,
			NonExistentNode,
			1,
			[][32]byte{[32]byte{'i'}},
		},
		{
			[32]byte{'i'},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
				[32]byte{'d'}: 103,
				[32]byte{'g'}: 104,
			},
			false,
			NonExistentNode,
			NonExistentNode,
			1,
			[][32]byte{[32]byte{'i'}},
		},
	}
	for _, tc := range tests {
		ctx := context.Background()
		f := setup(1, 1)

		require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'j'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, params.BeaconConfig().ZeroHash, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 104, [32]byte{'e'}, [32]byte{'d'}, params.BeaconConfig().ZeroHash, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 104, [32]byte{'g'}, [32]byte{'d'}, params.BeaconConfig().ZeroHash, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, params.BeaconConfig().ZeroHash, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'h'}, [32]byte{'g'}, params.BeaconConfig().ZeroHash, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'k'}, [32]byte{'g'}, params.BeaconConfig().ZeroHash, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 106, [32]byte{'i'}, [32]byte{'h'}, params.BeaconConfig().ZeroHash, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 106, [32]byte{'l'}, [32]byte{'k'}, params.BeaconConfig().ZeroHash, 1, 1))
		weights := []uint64{10, 10, 9, 7, 1, 6, 2, 3, 1, 1, 1, 0, 0}
		f.syncedTips.Lock()
		f.syncedTips.validatedTips = tc.tips
		f.syncedTips.Unlock()
		f.store.nodesLock.Lock()
		for i, node := range f.store.nodes {
			node.weight = weights[i]
		}
		// Make j be the best child and descendant of b
		nodeB := f.store.nodes[2]
		nodeB.bestChild = 4
		nodeB.bestDescendant = 4
		idx := f.store.nodesIndices[tc.root]
		node := f.store.nodes[idx]
		parentIndex := node.parent
		require.NotEqual(t, NonExistentNode, parentIndex)
		parent := f.store.nodes[parentIndex]
		f.store.nodesLock.Unlock()
		roots, err := f.SetOptimisticToInvalid(context.Background(), tc.root)
		require.NoError(t, err)
		require.DeepEqual(t, tc.returnedRoots, roots)
		f.syncedTips.RLock()
		_, parentSyncedTip := f.syncedTips.validatedTips[parent.root]
		f.syncedTips.RUnlock()
		require.Equal(t, tc.wantedParentTip, parentSyncedTip)
		require.Equal(t, tc.newBestChild, parent.bestChild)
		require.Equal(t, tc.newBestDescendant, parent.bestDescendant)
		require.Equal(t, tc.newParentWeight, parent.weight)
	}
}

// This tests the algorithm to find the tip of a given node
// We start with the following diagram
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
func TestFindSyncedTip(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)

	require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'j'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 104, [32]byte{'e'}, [32]byte{'d'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 104, [32]byte{'g'}, [32]byte{'d'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'h'}, [32]byte{'g'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'k'}, [32]byte{'g'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 106, [32]byte{'i'}, [32]byte{'h'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 106, [32]byte{'l'}, [32]byte{'k'}, params.BeaconConfig().ZeroHash, 1, 1))
	tests := []struct {
		root   [32]byte                // the root of the block
		tips   map[[32]byte]types.Slot // the synced tips
		wanted [32]byte                // the root of expected tip
	}{
		{
			[32]byte{'i'},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
				[32]byte{'d'}: 103,
				[32]byte{'g'}: 104,
			},
			[32]byte{'g'},
		},
		{
			[32]byte{'g'},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
				[32]byte{'d'}: 103,
				[32]byte{'h'}: 104,
				[32]byte{'k'}: 106,
			},
			[32]byte{'d'},
		},
		{
			[32]byte{'e'},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
				[32]byte{'d'}: 103,
				[32]byte{'g'}: 103,
			},
			[32]byte{'d'},
		},
		{
			[32]byte{'j'},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
				[32]byte{'f'}: 105,
				[32]byte{'g'}: 104,
				[32]byte{'i'}: 106,
			},
			[32]byte{'b'},
		},
		{
			[32]byte{'g'},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
				[32]byte{'f'}: 105,
				[32]byte{'g'}: 104,
				[32]byte{'i'}: 106,
			},
			[32]byte{'g'},
		},
	}
	for _, tc := range tests {
		f.store.nodesLock.RLock()
		node := f.store.nodes[f.store.nodesIndices[tc.root]]
		syncedTips := &optimisticStore{
			validatedTips: tc.tips,
		}
		syncedTips.RLock()
		idx, err := f.store.findSyncedTip(ctx, node, syncedTips)
		require.NoError(t, err)
		require.Equal(t, tc.wanted, f.store.nodes[idx].root)

		f.store.nodesLock.RUnlock()
		syncedTips.RUnlock()
	}
}

// This is a regression test (10341)
func TestIsOptimistic_DeadLock(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)
	require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 90, [32]byte{'b'}, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 101, [32]byte{'c'}, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'d'}, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 103, [32]byte{'e'}, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1))
	tips := map[[32]byte]types.Slot{
		[32]byte{'a'}: 100,
		[32]byte{'d'}: 102,
	}
	f.syncedTips.validatedTips = tips
	_, err := f.IsOptimistic(ctx, [32]byte{'a'})
	require.NoError(t, err)

	// Acquire a write lock, this should not hang
	f.store.nodesLock.Lock()
	f.store.nodesLock.Unlock()
	_, err = f.IsOptimistic(ctx, [32]byte{'e'})
	require.NoError(t, err)

	// Acquire a write lock, this should not hang
	f.store.nodesLock.Lock()
	f.store.nodesLock.Unlock()
	_, err = f.IsOptimistic(ctx, [32]byte{'b'})
	require.NoError(t, err)

	// Acquire a write lock, this should not hang
	f.store.nodesLock.Lock()
	f.store.nodesLock.Unlock()

	_, err = f.IsOptimistic(ctx, [32]byte{'c'})
	require.NoError(t, err)

	// Acquire a write lock, this should not hang
	f.store.nodesLock.Lock()
	f.store.nodesLock.Unlock()

}
