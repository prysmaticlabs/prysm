package doublylinkedtree

import (
	"context"
	"sort"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

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
// And every block in the Fork choice is optimistic.
func TestPruneInvalid(t *testing.T) {
	tests := []struct {
		wantedErr        error
		wantedRoots      [][32]byte
		wantedNodeNumber int
		root             [32]byte
		parentRoot       [32]byte
		payload          [32]byte
	}{
		{ // Bogus LVH, root not in forkchoice
			root:             [32]byte{'x'},
			parentRoot:       [32]byte{'i'},
			payload:          [32]byte{'R'},
			wantedNodeNumber: 13,
			wantedRoots:      [][32]byte{},
			wantedErr:        nil,
		},
		{
			// Bogus LVH
			root:             [32]byte{'i'},
			parentRoot:       [32]byte{'h'},
			payload:          [32]byte{'R'},
			wantedNodeNumber: 12,
			wantedRoots:      [][32]byte{{'i'}},
			wantedErr:        nil,
		},
		{
			root:             [32]byte{'j'},
			parentRoot:       [32]byte{'b'},
			payload:          [32]byte{'B'},
			wantedNodeNumber: 12,
			wantedRoots:      [][32]byte{{'j'}},
			wantedErr:        nil,
		},
		{
			root:             [32]byte{'c'},
			parentRoot:       [32]byte{'b'},
			payload:          [32]byte{'B'},
			wantedNodeNumber: 4,
			wantedRoots: [][32]byte{{'f'}, {'e'}, {'i'}, {'h'}, {'l'},
				{'k'}, {'g'}, {'d'}, {'c'}},
			wantedErr: nil,
		},
		{
			root:             [32]byte{'i'},
			parentRoot:       [32]byte{'h'},
			payload:          [32]byte{'H'},
			wantedNodeNumber: 12,
			wantedRoots:      [][32]byte{{'i'}},
			wantedErr:        nil,
		},
		{
			root:             [32]byte{'h'},
			parentRoot:       [32]byte{'g'},
			payload:          [32]byte{'G'},
			wantedNodeNumber: 11,
			wantedRoots:      [][32]byte{{'i'}, {'h'}},
			wantedErr:        nil,
		},
		{
			root:             [32]byte{'g'},
			parentRoot:       [32]byte{'d'},
			payload:          [32]byte{'D'},
			wantedNodeNumber: 8,
			wantedRoots:      [][32]byte{{'i'}, {'h'}, {'l'}, {'k'}, {'g'}},
			wantedErr:        nil,
		},
		{
			root:             [32]byte{'i'},
			parentRoot:       [32]byte{'h'},
			payload:          [32]byte{'D'},
			wantedNodeNumber: 8,
			wantedRoots:      [][32]byte{{'i'}, {'h'}, {'l'}, {'k'}, {'g'}},
			wantedErr:        nil,
		},
		{
			root:             [32]byte{'f'},
			parentRoot:       [32]byte{'e'},
			payload:          [32]byte{'D'},
			wantedNodeNumber: 11,
			wantedRoots:      [][32]byte{{'f'}, {'e'}},
			wantedErr:        nil,
		},
		{
			root:             [32]byte{'h'},
			parentRoot:       [32]byte{'g'},
			payload:          [32]byte{'C'},
			wantedNodeNumber: 5,
			wantedRoots: [][32]byte{
				{'f'},
				{'e'},
				{'i'},
				{'h'},
				{'l'},
				{'k'},
				{'g'},
				{'d'},
			},
			wantedErr: nil,
		},
		{
			root:             [32]byte{'g'},
			parentRoot:       [32]byte{'d'},
			payload:          [32]byte{'E'},
			wantedNodeNumber: 8,
			wantedRoots:      [][32]byte{{'i'}, {'h'}, {'l'}, {'k'}, {'g'}},
			wantedErr:        nil,
		},
		{
			root:             [32]byte{'z'},
			parentRoot:       [32]byte{'j'},
			payload:          [32]byte{'B'},
			wantedNodeNumber: 12,
			wantedRoots:      [][32]byte{{'j'}},
			wantedErr:        nil,
		},
		{
			root:             [32]byte{'z'},
			parentRoot:       [32]byte{'j'},
			payload:          [32]byte{'J'},
			wantedNodeNumber: 13,
			wantedRoots:      [][32]byte{},
			wantedErr:        nil,
		},
		{
			root:             [32]byte{'j'},
			parentRoot:       [32]byte{'a'},
			payload:          [32]byte{'B'},
			wantedNodeNumber: 0,
			wantedRoots:      [][32]byte{},
			wantedErr:        errInvalidParentRoot,
		},
		{
			root:             [32]byte{'z'},
			parentRoot:       [32]byte{'h'},
			payload:          [32]byte{'D'},
			wantedNodeNumber: 8,
			wantedRoots:      [][32]byte{{'i'}, {'h'}, {'l'}, {'k'}, {'g'}},
			wantedErr:        nil,
		},
		{
			root:             [32]byte{'z'},
			parentRoot:       [32]byte{'h'},
			payload:          [32]byte{'D'},
			wantedNodeNumber: 8,
			wantedRoots:      [][32]byte{{'i'}, {'h'}, {'l'}, {'k'}, {'g'}},
			wantedErr:        nil,
		},
	}
	for _, tc := range tests {
		ctx := context.Background()
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

		roots, err := f.store.setOptimisticToInvalid(context.Background(), tc.root, tc.parentRoot, tc.payload)
		if tc.wantedErr == nil {
			require.NoError(t, err)
			require.DeepEqual(t, tc.wantedRoots, roots)
			require.Equal(t, tc.wantedNodeNumber, f.NodeCount())
		} else {
			require.ErrorIs(t, tc.wantedErr, err)
		}
	}
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

// This is a regression test (10565)
//     ----- C
//   /
//  A <- B
//   \
//     ----------D
// D is invalid

func TestSetOptimisticToInvalid_CorrectChildren(t *testing.T) {
	ctx := context.Background()
	f := setup(1, 1)

	state, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'c'}, [32]byte{'a'}, [32]byte{'C'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 103, [32]byte{'d'}, [32]byte{'a'}, [32]byte{'D'}, 1, 1)
	require.NoError(t, err)
	require.NoError(t, f.InsertNode(ctx, state, blkRoot))

	_, err = f.store.setOptimisticToInvalid(ctx, [32]byte{'d'}, [32]byte{'a'}, [32]byte{'A'})
	require.NoError(t, err)
	require.Equal(t, 2, len(f.store.nodeByRoot[[32]byte{'a'}].children))

}

// Pow       |      Pos
//
//  CA -- A -- B -- C-----D
//   \          \--------------E
//    \
//     ----------------------F -- G
// B is INVALID
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
	sort.Slice(roots, func(i, j int) bool {
		return bytesutil.BytesToUint64BigEndian(roots[i][:]) < bytesutil.BytesToUint64BigEndian(roots[j][:])
	})
	require.DeepEqual(t, roots, [][32]byte{{'b'}, {'c'}, {'d'}, {'e'}})
}

// Pow       |      Pos
//
//  CA -------- B -- C-----D
//   \           \--------------E
//    \
//     --A -------------------------F -- G
// B is INVALID
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

	roots, err := f.SetOptimisticToInvalid(ctx, [32]byte{'x'}, [32]byte{'d'}, [32]byte{})
	require.NoError(t, err)
	require.Equal(t, 4, len(roots))
	sort.Slice(roots, func(i, j int) bool {
		return bytesutil.BytesToUint64BigEndian(roots[i][:]) < bytesutil.BytesToUint64BigEndian(roots[j][:])
	})
	require.DeepEqual(t, roots, [][32]byte{{'b'}, {'c'}, {'d'}, {'e'}})
}

func TestSetOptimisticToValid(t *testing.T) {
	f := setup(1, 1)
	op, err := f.IsOptimistic([32]byte{})
	require.NoError(t, err)
	require.Equal(t, true, op)
	require.NoError(t, f.SetOptimisticToValid(context.Background(), [32]byte{}))
	op, err = f.IsOptimistic([32]byte{})
	require.NoError(t, err)
	require.Equal(t, false, op)
}
