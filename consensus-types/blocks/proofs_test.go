package blocks

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/container/trie"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestComputeBlockBodyFieldRoots_Phase0(t *testing.T) {
	blockBodyPhase0 := hydrateBeaconBlockBody()
	i, err := NewBeaconBlockBody(blockBodyPhase0)
	require.NoError(t, err)

	b := i.(*BeaconBlockBody)

	fieldRoots, err := ComputeBlockBodyFieldRoots(context.Background(), b)
	require.NoError(t, err)
	trie, err := trie.GenerateTrieFromItems(fieldRoots, 3)
	require.NoError(t, err)
	layers := trie.ToProto().GetLayers()

	hash := layers[len(layers)-1].Layer[0]
	require.NoError(t, err)

	correctHash, err := b.HashTreeRoot()
	require.NoError(t, err)

	require.DeepEqual(t, correctHash[:], hash)
}
