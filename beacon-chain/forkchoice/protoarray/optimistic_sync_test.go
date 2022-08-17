package protoarray

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func slicesEqual(a, b [][32]byte) bool {
	if len(a) != len(b) {
		return false
	}

	mapA := make(map[[32]byte]bool, len(a))
	for _, root := range a {
		mapA[root] = true
	}
	for _, root := range b {
		_, ok := mapA[root]
		if !ok {
			return false
		}
	}
	return true
}

func TestOptimistic_Outside_ForkChoice(t *testing.T) {
	root0 := bytesutil.ToBytes32([]byte("hello0"))

	nodeA := &Node{
		slot:      types.Slot(100),
		root:      bytesutil.ToBytes32([]byte("helloA")),
		bestChild: 1,
		status:    valid,
	}
	nodes := []*Node{
		nodeA,
	}
	ni := map[[32]byte]uint64{
		nodeA.root: 0,
	}

	s := &Store{
		nodes:        nodes,
		nodesIndices: ni,
	}

	f := &ForkChoice{
		store: s,
	}
	_, err := f.IsOptimistic(root0)
	require.ErrorIs(t, ErrUnknownNodeRoot, err)
}

// This tests the algorithm to update optimistic Status
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
// The Chain A -- B -- C -- D -- E is VALID.
//
func TestSetOptimisticToValid(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		root             [32]byte // the root of the new VALID block
		testRoot         [32]byte // root of the node we will test optimistic status
		wantedOptimistic bool     // wanted optimistic status for tested node
		wantedErr        error    // wanted error message
	}{
		{
			[32]byte{'i'},
			[32]byte{'i'},
			false,
			nil,
		},
		{
			[32]byte{'i'},
			[32]byte{'f'},
			true,
			nil,
		},
		{
			[32]byte{'i'},
			[32]byte{'b'},
			false,
			nil,
		},
		{
			[32]byte{'i'},
			[32]byte{'h'},
			false,
			nil,
		},
		{
			[32]byte{'b'},
			[32]byte{'b'},
			false,
			nil,
		},
		{
			[32]byte{'b'},
			[32]byte{'h'},
			true,
			nil,
		},
		{
			[32]byte{'b'},
			[32]byte{'a'},
			false,
			nil,
		},
		{
			[32]byte{'k'},
			[32]byte{'k'},
			false,
			nil,
		},
		{
			[32]byte{'k'},
			[32]byte{'l'},
			true,
			nil,
		},
		{
			[32]byte{'p'},
			[32]byte{},
			false,
			ErrUnknownNodeRoot,
		},
	}
	for _, tc := range tests {
		f := setup(1, 1)

		state, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'j'}, [32]byte{'b'}, [32]byte{'J'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, [32]byte{'D'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 104, [32]byte{'e'}, [32]byte{'d'}, [32]byte{'E'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 104, [32]byte{'g'}, [32]byte{'d'}, [32]byte{'G'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, [32]byte{'F'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 105, [32]byte{'h'}, [32]byte{'g'}, [32]byte{'H'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 105, [32]byte{'k'}, [32]byte{'g'}, [32]byte{'K'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 106, [32]byte{'i'}, [32]byte{'h'}, [32]byte{'I'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 106, [32]byte{'l'}, [32]byte{'k'}, [32]byte{'L'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		require.NoError(t, f.SetOptimisticToValid(context.Background(), [32]byte{'e'}))
		optimistic, err := f.IsOptimistic([32]byte{'b'})
		require.NoError(t, err)
		require.Equal(t, false, optimistic)

		err = f.SetOptimisticToValid(context.Background(), tc.root)
		if tc.wantedErr != nil {
			require.ErrorIs(t, err, tc.wantedErr)
		} else {
			require.NoError(t, err)
			optimistic, err := f.IsOptimistic(tc.testRoot)
			require.NoError(t, err)
			require.Equal(t, tc.wantedOptimistic, optimistic)
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
// And the chain A -- B -- C -- D -- E has been fully validated. The numbers in parentheses are
// the weights of the nodes.
//
func TestSetOptimisticToInvalid(t *testing.T) {
	tests := []struct {
		name              string   // test description
		root              [32]byte // the root of the new INVALID block
		parentRoot        [32]byte // the root of the parent block
		payload           [32]byte // the payload of the last valid hash
		newBestChild      uint64
		newBestDescendant uint64
		newParentWeight   uint64
		returnedRoots     [][32]byte
	}{
		{
			"Remove tip, parent was valid",
			[32]byte{'j'},
			[32]byte{'b'},
			[32]byte{'B'},
			3,
			12,
			8,
			[][32]byte{{'j'}},
		},
		{
			"Remove tip, parent was optimistic",
			[32]byte{'i'},
			[32]byte{'h'},
			[32]byte{'H'},
			NonExistentNode,
			NonExistentNode,
			1,
			[][32]byte{{'i'}},
		},
		{
			"Remove tip, lvh is inner and valid",
			[32]byte{'i'},
			[32]byte{'h'},
			[32]byte{'D'},
			6,
			8,
			3,
			[][32]byte{{'g'}, {'h'}, {'k'}, {'i'}, {'l'}},
		},
		{
			"Remove inner, lvh is inner and optimistic",
			[32]byte{'h'},
			[32]byte{'g'},
			[32]byte{'G'},
			10,
			12,
			2,
			[][32]byte{{'h'}, {'i'}},
		},
		{
			"Remove tip, lvh is inner and optimistic",
			[32]byte{'l'},
			[32]byte{'k'},
			[32]byte{'G'},
			9,
			11,
			2,
			[][32]byte{{'k'}, {'l'}},
		},
		{
			"Remove tip, lvh is not an ancestor",
			[32]byte{'j'},
			[32]byte{'b'},
			[32]byte{'C'},
			5,
			12,
			7,
			[][32]byte{{'j'}},
		},
		{
			"Remove inner, lvh is not an ancestor",
			[32]byte{'g'},
			[32]byte{'d'},
			[32]byte{'J'},
			NonExistentNode,
			NonExistentNode,
			1,
			[][32]byte{{'g'}, {'h'}, {'k'}, {'i'}, {'l'}},
		},
		{
			"Remove not inserted, parent was invalid",
			[32]byte{'z'},
			[32]byte{'j'},
			[32]byte{'B'},
			3,
			12,
			8,
			[][32]byte{{'j'}},
		},
		{
			"Remove not inserted, parent was valid",
			[32]byte{'z'},
			[32]byte{'j'},
			[32]byte{'J'},
			NonExistentNode,
			NonExistentNode,
			1,
			[][32]byte{},
		},
	}
	for _, tc := range tests {
		ctx := context.Background()
		f := setup(1, 1)

		state, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'j'}, [32]byte{'b'}, [32]byte{'J'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, [32]byte{'D'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 104, [32]byte{'e'}, [32]byte{'d'}, [32]byte{'E'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 104, [32]byte{'g'}, [32]byte{'d'}, [32]byte{'G'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, [32]byte{'F'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 105, [32]byte{'h'}, [32]byte{'g'}, [32]byte{'H'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 105, [32]byte{'k'}, [32]byte{'g'}, [32]byte{'K'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 106, [32]byte{'i'}, [32]byte{'h'}, [32]byte{'I'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		state, blkRoot, err = prepareForkchoiceState(ctx, 106, [32]byte{'l'}, [32]byte{'k'}, [32]byte{'L'}, 1, 1)
		require.NoError(t, err)
		require.NoError(t, f.InsertNode(ctx, state, blkRoot))
		weights := []uint64{10, 10, 9, 7, 1, 6, 2, 3, 1, 1, 1, 0, 0}
		f.store.nodesLock.Lock()
		for i, node := range f.store.nodes {
			node.weight = weights[i]
		}
		f.store.nodesLock.Unlock()
		require.NoError(t, f.SetOptimisticToValid(ctx, [32]byte{'e'}))
		roots, err := f.SetOptimisticToInvalid(ctx, tc.root, tc.parentRoot, tc.payload)
		require.NoError(t, err)
		f.store.nodesLock.RLock()
		_, ok := f.store.nodesIndices[tc.root]
		require.Equal(t, false, ok)
		lvh := f.store.nodes[f.store.payloadIndices[tc.payload]]
		require.Equal(t, true, slicesEqual(tc.returnedRoots, roots))
		require.Equal(t, tc.newBestChild, lvh.bestChild)
		require.Equal(t, tc.newBestDescendant, lvh.bestDescendant)
		require.Equal(t, tc.newParentWeight, lvh.weight)
		require.Equal(t, syncing, f.store.nodes[8].status /* F */)
		require.Equal(t, valid, f.store.nodes[5].status /* E */)
		f.store.nodesLock.RUnlock()
	}
}

func TestSetOptimisticToInvalid_InvalidRoots(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)

	state, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	_, err = f.SetOptimisticToInvalid(ctx, [32]byte{'p'}, [32]byte{'p'}, [32]byte{'B'})
	require.ErrorIs(t, ErrUnknownNodeRoot, err)
}

// This is a regression test (10445)
func TestSetOptimisticToInvalid_ProposerBoost(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)

	state, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	f.store.proposerBoostLock.Lock()
	f.store.proposerBoostRoot = [32]byte{'c'}
	f.store.previousProposerBoostScore = 10
	f.store.previousProposerBoostRoot = [32]byte{'b'}
	f.store.proposerBoostLock.Unlock()

	_, err = f.SetOptimisticToInvalid(ctx, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'A'})
	require.NoError(t, err)
	f.store.proposerBoostLock.RLock()
	require.Equal(t, uint64(0), f.store.previousProposerBoostScore)
	require.DeepEqual(t, [32]byte{}, f.store.proposerBoostRoot)
	require.DeepEqual(t, params.BeaconConfig().ZeroHash, f.store.previousProposerBoostRoot)
	f.store.proposerBoostLock.RUnlock()
}

// This is a regression test (10996)
func TestSetOptimisticToInvalid_BogusLVH(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)

	state, root, err := prepareForkchoiceState(ctx, 1, [32]byte{'a'}, [32]byte{}, [32]byte{'A'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, root))

	state, root, err = prepareForkchoiceState(ctx, 2, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, root))

	invalidRoots, err := f.SetOptimisticToInvalid(ctx, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'R'})
	require.NoError(t, err)
	require.Equal(t, 1, len(invalidRoots))
	require.Equal(t, [32]byte{'b'}, invalidRoots[0])
}

// This is a regression test (10996)
func TestSetOptimisticToInvalid_BogusLVH_RotNotImported(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)

	state, root, err := prepareForkchoiceState(ctx, 1, [32]byte{'a'}, [32]byte{}, [32]byte{'A'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, root))

	state, root, err = prepareForkchoiceState(ctx, 2, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, root))

	invalidRoots, err := f.SetOptimisticToInvalid(ctx, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'R'})
	require.NoError(t, err)
	require.Equal(t, 0, len(invalidRoots))
}

// Pow       |      Pos
//
//  CA -- A -- B -- C-----D
//   \          \--------------E
//    \
//     ----------------------F -- G
// B is INVALID
//
func TestSetOptimisticToInvalid_ForkAtMerge(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)

	st, root, err := prepareForkchoiceState(ctx, 100, [32]byte{'r'}, [32]byte{}, [32]byte{}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	st, root, err = prepareForkchoiceState(ctx, 101, [32]byte{'a'}, [32]byte{'r'}, [32]byte{}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	st, root, err = prepareForkchoiceState(ctx, 102, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	st, root, err = prepareForkchoiceState(ctx, 103, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	st, root, err = prepareForkchoiceState(ctx, 104, [32]byte{'d'}, [32]byte{'c'}, [32]byte{'D'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	st, root, err = prepareForkchoiceState(ctx, 105, [32]byte{'e'}, [32]byte{'b'}, [32]byte{'E'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	st, root, err = prepareForkchoiceState(ctx, 106, [32]byte{'f'}, [32]byte{'r'}, [32]byte{'F'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	st, root, err = prepareForkchoiceState(ctx, 107, [32]byte{'g'}, [32]byte{'f'}, [32]byte{'G'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	roots, err := f.SetOptimisticToInvalid(ctx, [32]byte{'x'}, [32]byte{'d'}, [32]byte{})
	require.NoError(t, err)
	require.Equal(t, 4, len(roots))
	require.Equal(t, true, slicesEqual(roots, [][32]byte{{'b'}, {'c'}, {'d'}, {'e'}}))
}

// Pow       |      Pos
//
//  CA -------- B -- C-----D
//   \           \--------------E
//    \
//     --A -------------------------F -- G
// B is INVALID
//
func TestSetOptimisticToInvalid_ForkAtMerge_bis(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)

	st, root, err := prepareForkchoiceState(ctx, 100, [32]byte{'r'}, [32]byte{}, [32]byte{}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	st, root, err = prepareForkchoiceState(ctx, 101, [32]byte{'a'}, [32]byte{'r'}, [32]byte{}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	st, root, err = prepareForkchoiceState(ctx, 102, [32]byte{'b'}, [32]byte{}, [32]byte{'B'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	st, root, err = prepareForkchoiceState(ctx, 103, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	st, root, err = prepareForkchoiceState(ctx, 104, [32]byte{'d'}, [32]byte{'c'}, [32]byte{'D'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	st, root, err = prepareForkchoiceState(ctx, 105, [32]byte{'e'}, [32]byte{'b'}, [32]byte{'E'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	st, root, err = prepareForkchoiceState(ctx, 106, [32]byte{'f'}, [32]byte{'a'}, [32]byte{'F'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	st, root, err = prepareForkchoiceState(ctx, 107, [32]byte{'g'}, [32]byte{'f'}, [32]byte{'G'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, st, root))

	roots, err := f.SetOptimisticToInvalid(ctx, [32]byte{'d'}, [32]byte{'c'}, [32]byte{})
	require.NoError(t, err)
	require.Equal(t, 1, len(roots))
	require.Equal(t, [32]byte{'d'}, roots[0])
}
