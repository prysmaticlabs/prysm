package doublylinkedtree

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/require"
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
//
func TestPruneInvalid(t *testing.T) {
	tests := []struct {
		root             [32]byte // the root of the new INVALID block
		parentRoot       [32]byte // the root of the parent block
		payload          [32]byte // the last valid hash
		wantedNodeNumber int
		wantedRoots      [][32]byte
		wantedErr        error
	}{
		{
			[32]byte{'j'},
			[32]byte{'b'},
			[32]byte{'B'},
			12,
			[][32]byte{[32]byte{'j'}},
			nil,
		},
		{
			[32]byte{'c'},
			[32]byte{'b'},
			[32]byte{'B'},
			4,
			[][32]byte{[32]byte{'f'}, [32]byte{'e'}, [32]byte{'i'}, [32]byte{'h'}, [32]byte{'l'},
				[32]byte{'k'}, [32]byte{'g'}, [32]byte{'d'}, [32]byte{'c'}},
			nil,
		},
		{
			[32]byte{'i'},
			[32]byte{'h'},
			[32]byte{'H'},
			12,
			[][32]byte{[32]byte{'i'}},
			nil,
		},
		{
			[32]byte{'h'},
			[32]byte{'g'},
			[32]byte{'G'},
			11,
			[][32]byte{[32]byte{'i'}, [32]byte{'h'}},
			nil,
		},
		{
			[32]byte{'g'},
			[32]byte{'d'},
			[32]byte{'D'},
			8,
			[][32]byte{[32]byte{'i'}, [32]byte{'h'}, [32]byte{'l'}, [32]byte{'k'}, [32]byte{'g'}},
			nil,
		},
		{
			[32]byte{'i'},
			[32]byte{'h'},
			[32]byte{'D'},
			8,
			[][32]byte{[32]byte{'i'}, [32]byte{'h'}, [32]byte{'l'}, [32]byte{'k'}, [32]byte{'g'}},
			nil,
		},
		{
			[32]byte{'f'},
			[32]byte{'e'},
			[32]byte{'D'},
			11,
			[][32]byte{[32]byte{'f'}, [32]byte{'e'}},
			nil,
		},
		{
			[32]byte{'h'},
			[32]byte{'g'},
			[32]byte{'C'},
			5,
			[][32]byte{
				[32]byte{'f'},
				[32]byte{'e'},
				[32]byte{'i'},
				[32]byte{'h'},
				[32]byte{'l'},
				[32]byte{'k'},
				[32]byte{'g'},
				[32]byte{'d'},
			},
			nil,
		},
		{
			[32]byte{'g'},
			[32]byte{'d'},
			[32]byte{'E'},
			8,
			[][32]byte{[32]byte{'i'}, [32]byte{'h'}, [32]byte{'l'}, [32]byte{'k'}, [32]byte{'g'}},
			nil,
		},
		{
			[32]byte{'z'},
			[32]byte{'j'},
			[32]byte{'B'},
			12,
			[][32]byte{[32]byte{'j'}},
			nil,
		},
		{
			[32]byte{'z'},
			[32]byte{'j'},
			[32]byte{'J'},
			13,
			[][32]byte{},
			nil,
		},
		{
			[32]byte{'j'},
			[32]byte{'a'},
			[32]byte{'B'},
			0,
			[][32]byte{},
			errInvalidParentRoot,
		},
		{
			[32]byte{'z'},
			[32]byte{'h'},
			[32]byte{'D'},
			8,
			[][32]byte{[32]byte{'i'}, [32]byte{'h'}, [32]byte{'l'}, [32]byte{'k'}, [32]byte{'g'}},
			nil,
		},
		{
			[32]byte{'z'},
			[32]byte{'h'},
			[32]byte{'D'},
			8,
			[][32]byte{[32]byte{'i'}, [32]byte{'h'}, [32]byte{'l'}, [32]byte{'k'}, [32]byte{'g'}},
			nil,
		},
	}
	for _, tc := range tests {
		ctx := context.Background()
		f := setup(1, 1)

		require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'j'}, [32]byte{'b'}, [32]byte{'J'}, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 103, [32]byte{'d'}, [32]byte{'c'}, [32]byte{'D'}, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 104, [32]byte{'e'}, [32]byte{'d'}, [32]byte{'E'}, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 104, [32]byte{'g'}, [32]byte{'d'}, [32]byte{'G'}, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'f'}, [32]byte{'e'}, [32]byte{'F'}, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'h'}, [32]byte{'g'}, [32]byte{'H'}, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 105, [32]byte{'k'}, [32]byte{'g'}, [32]byte{'K'}, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 106, [32]byte{'i'}, [32]byte{'h'}, [32]byte{'I'}, 1, 1))
		require.NoError(t, f.InsertOptimisticBlock(ctx, 106, [32]byte{'l'}, [32]byte{'k'}, [32]byte{'L'}, 1, 1))

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

	require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 101, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'C'}, 1, 1))
	f.store.proposerBoostLock.Lock()
	f.store.proposerBoostRoot = [32]byte{'c'}
	f.store.previousProposerBoostScore = 10
	f.store.previousProposerBoostRoot = [32]byte{'b'}
	f.store.proposerBoostLock.Unlock()

	_, err := f.SetOptimisticToInvalid(ctx, [32]byte{'c'}, [32]byte{'b'}, [32]byte{'A'})
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

	require.NoError(t, f.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, params.BeaconConfig().ZeroHash, [32]byte{'A'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'B'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 102, [32]byte{'c'}, [32]byte{'a'}, [32]byte{'C'}, 1, 1))
	require.NoError(t, f.InsertOptimisticBlock(ctx, 103, [32]byte{'d'}, [32]byte{'a'}, [32]byte{'D'}, 1, 1))

	_, err := f.store.setOptimisticToInvalid(ctx, [32]byte{'d'}, [32]byte{'a'}, [32]byte{'A'})
	require.NoError(t, err)
	require.Equal(t, 2, len(f.store.nodeByRoot[[32]byte{'a'}].children))

}
