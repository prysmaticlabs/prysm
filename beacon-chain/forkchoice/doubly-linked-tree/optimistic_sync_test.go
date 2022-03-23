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
		wantedNodeNumber int
	}{
		{
			[32]byte{'j'},
			12,
		},
		{
			[32]byte{'c'},
			4,
		},
		{
			[32]byte{'i'},
			12,
		},
		{
			[32]byte{'h'},
			11,
		},
		{
			[32]byte{'g'},
			8,
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

		require.NoError(t, f.store.removeNode(context.Background(), tc.root))
		require.Equal(t, tc.wantedNodeNumber, f.NodeCount())
	}
}
