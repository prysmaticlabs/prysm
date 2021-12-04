package stateutil

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/assert"
)

func TestArraysTreeRoot_OnlyPowerOf2(t *testing.T) {
	_, err := NocachedHasher.arraysRoot([][]byte{}, 1, "testing")
	assert.NoError(t, err)
	_, err = NocachedHasher.arraysRoot([][]byte{}, 4, "testing")
	assert.NoError(t, err)
	_, err = NocachedHasher.arraysRoot([][]byte{}, 8, "testing")
	assert.NoError(t, err)
	_, err = NocachedHasher.arraysRoot([][]byte{}, 10, "testing")
	assert.ErrorContains(t, "hash layer is a non power of 2", err)
}

func TestArraysTreeRoot_ZeroLength(t *testing.T) {
	_, err := NocachedHasher.arraysRoot([][]byte{}, 0, "testing")
	assert.ErrorContains(t, "zero leaves provided", err)
}
