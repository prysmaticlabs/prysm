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
	slot0 := types.Slot(98)
	root1 := bytesutil.ToBytes32([]byte("hello1"))
	slot1 := types.Slot(99)

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
	op, err := f.Optimistic(ctx, root0, slot0)
	require.NoError(t, err)
	require.Equal(t, op, false)

	op, err = f.Optimistic(ctx, root1, slot1)
	require.NoError(t, err)
	require.Equal(t, op, false)

	// We check all nodes in the Fork Choice store.
	op, err = f.Optimistic(ctx, nodeA.root, nodeA.slot)
	require.NoError(t, err)
	require.Equal(t, op, false)

	op, err = f.Optimistic(ctx, nodeB.root, nodeB.slot)
	require.NoError(t, err)
	require.Equal(t, op, false)

	op, err = f.Optimistic(ctx, nodeC.root, nodeC.slot)
	require.NoError(t, err)
	require.Equal(t, op, false)

	op, err = f.Optimistic(ctx, nodeD.root, nodeD.slot)
	require.NoError(t, err)
	require.Equal(t, op, false)

	op, err = f.Optimistic(ctx, nodeE.root, nodeE.slot)
	require.NoError(t, err)
	require.Equal(t, op, true)

	op, err = f.Optimistic(ctx, nodeF.root, nodeF.slot)
	require.NoError(t, err)
	require.Equal(t, op, true)

	op, err = f.Optimistic(ctx, nodeG.root, nodeG.slot)
	require.NoError(t, err)
	require.Equal(t, op, true)

	op, err = f.Optimistic(ctx, nodeH.root, nodeH.slot)
	require.NoError(t, err)
	require.Equal(t, op, true)

	op, err = f.Optimistic(ctx, nodeI.root, nodeI.slot)
	require.NoError(t, err)
	require.Equal(t, op, true)

	op, err = f.Optimistic(ctx, nodeJ.root, nodeJ.slot)
	require.NoError(t, err)
	require.Equal(t, op, true)

	op, err = f.Optimistic(ctx, nodeK.root, nodeK.slot)
	require.NoError(t, err)
	require.Equal(t, op, true)
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
func TestUpdateSyncTipsWithValidRoots(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)

	require.NoError(t, f.ProcessBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{}, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{}, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{}, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 102, [32]byte{'j'}, [32]byte{'b'}, [32]byte{}, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, [32]byte{}, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 104, [32]byte{'e'}, [32]byte{'d'}, [32]byte{}, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 104, [32]byte{'g'}, [32]byte{'d'}, [32]byte{}, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, [32]byte{}, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 105, [32]byte{'h'}, [32]byte{'g'}, [32]byte{}, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 105, [32]byte{'k'}, [32]byte{'g'}, [32]byte{}, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 106, [32]byte{'i'}, [32]byte{'h'}, [32]byte{}, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 106, [32]byte{'l'}, [32]byte{'k'}, [32]byte{}, 1, 1))
	tests := []struct {
		root      [32]byte                // the root of the new VALID block
		tips      map[[32]byte]types.Slot // the old synced tips
		newtips   map[[32]byte]types.Slot // the updated synced tips
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
		err := f.UpdateSyncedTipsWithValidRoot(context.Background(), tc.root)
		if tc.wantedErr != nil {
			require.ErrorIs(t, err, tc.wantedErr)
		} else {
			require.NoError(t, err)
			f.syncedTips.RLock()
			require.DeepEqual(t, f.syncedTips.validatedTips, tc.newtips)
			f.syncedTips.RUnlock()
		}
	}
}

// We test the algorithm to update a node from SYNCING to INVALID
// We start with the same diagram as above:
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
func TestUpdateSyncTipsWithInvalidRoot(t *testing.T) {
	tests := []struct {
		root              [32]byte                // the root of the new INVALID block
		tips              map[[32]byte]types.Slot // the old synced tips
		wantedParentTip   bool
		newBestChild      uint64
		newBestDescendant uint64
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
		},
		{
			[32]byte{'j'},
			map[[32]byte]types.Slot{
				[32]byte{'b'}: 101,
			},
			true,
			3,
			4,
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
		},
	}
	for _, tc := range tests {
		ctx := context.Background()
		f := setup(1, 1)

		require.NoError(t, f.ProcessBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{}, 1, 1))
		require.NoError(t, f.ProcessBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{}, 1, 1))
		require.NoError(t, f.ProcessBlock(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{}, 1, 1))
		require.NoError(t, f.ProcessBlock(ctx, 102, [32]byte{'j'}, [32]byte{'b'}, [32]byte{}, 1, 1))
		require.NoError(t, f.ProcessBlock(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, [32]byte{}, 1, 1))
		require.NoError(t, f.ProcessBlock(ctx, 104, [32]byte{'e'}, [32]byte{'d'}, [32]byte{}, 1, 1))
		require.NoError(t, f.ProcessBlock(ctx, 104, [32]byte{'g'}, [32]byte{'d'}, [32]byte{}, 1, 1))
		require.NoError(t, f.ProcessBlock(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, [32]byte{}, 1, 1))
		require.NoError(t, f.ProcessBlock(ctx, 105, [32]byte{'h'}, [32]byte{'g'}, [32]byte{}, 1, 1))
		require.NoError(t, f.ProcessBlock(ctx, 105, [32]byte{'k'}, [32]byte{'g'}, [32]byte{}, 1, 1))
		require.NoError(t, f.ProcessBlock(ctx, 106, [32]byte{'i'}, [32]byte{'h'}, [32]byte{}, 1, 1))
		require.NoError(t, f.ProcessBlock(ctx, 106, [32]byte{'l'}, [32]byte{'k'}, [32]byte{}, 1, 1))
		f.syncedTips.Lock()
		f.syncedTips.validatedTips = tc.tips
		f.syncedTips.Unlock()
		f.store.nodesLock.Lock()
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
		err := f.UpdateSyncedTipsWithInvalidRoot(context.Background(), tc.root)
		require.NoError(t, err)
		f.syncedTips.RLock()
		_, parentSyncedTip := f.syncedTips.validatedTips[parent.root]
		f.syncedTips.RUnlock()
		require.Equal(t, tc.wantedParentTip, parentSyncedTip)
		require.Equal(t, tc.newBestChild, parent.bestChild)
		require.Equal(t, tc.newBestDescendant, parent.bestDescendant)
	}
}
