package state_native

import (
	"testing"

	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/custom-types"
	"github.com/prysmaticlabs/prysm/testing/assert"
)

func TestBlockRootsToSlice(t *testing.T) {
	a, b, c := [32]byte{'a'}, [32]byte{'b'}, [32]byte{'c'}
	roots := customtypes.BlockRoots{}
	roots[1] = a
	roots[10] = b
	roots[100] = c
	slice := BlockRootsToSlice(&roots)
	assert.DeepEqual(t, a[:], slice[1])
	assert.DeepEqual(t, b[:], slice[10])
	assert.DeepEqual(t, c[:], slice[100])
}

func TestStateRootsToSlice(t *testing.T) {
	a, b, c := [32]byte{'a'}, [32]byte{'b'}, [32]byte{'c'}
	roots := customtypes.StateRoots{}
	roots[1] = a
	roots[10] = b
	roots[100] = c
	slice := StateRootsToSlice(&roots)
	assert.DeepEqual(t, a[:], slice[1])
	assert.DeepEqual(t, b[:], slice[10])
	assert.DeepEqual(t, c[:], slice[100])
}

func TestHistoricalRootsToSlice(t *testing.T) {
	a, b, c := [32]byte{'a'}, [32]byte{'b'}, [32]byte{'c'}
	roots := customtypes.HistoricalRoots{a, b, c}
	slice := HistoricalRootsToSlice(roots)
	assert.DeepEqual(t, a[:], slice[0])
	assert.DeepEqual(t, b[:], slice[1])
	assert.DeepEqual(t, c[:], slice[2])
}

func TestRandaoMixesToSlice(t *testing.T) {
	a, b, c := [32]byte{'a'}, [32]byte{'b'}, [32]byte{'c'}
	roots := customtypes.RandaoMixes{}
	roots[1] = a
	roots[10] = b
	roots[100] = c
	slice := RandaoMixesToSlice(&roots)
	assert.DeepEqual(t, a[:], slice[1])
	assert.DeepEqual(t, b[:], slice[10])
	assert.DeepEqual(t, c[:], slice[100])
}
