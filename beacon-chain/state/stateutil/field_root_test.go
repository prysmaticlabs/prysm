package stateutil

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/assert"
)

func TestArraysTreeRoot_OnlyPowerOf2(t *testing.T) {
	_, err := arraysRoot([][]byte{}, 1)
	assert.NoError(t, err)
	_, err = arraysRoot([][]byte{}, 4)
	assert.NoError(t, err)
	_, err = arraysRoot([][]byte{}, 8)
	assert.NoError(t, err)
	_, err = arraysRoot([][]byte{}, 10)
	assert.ErrorContains(t, "hash layer is a non power of 2", err)
}

func TestArraysTreeRoot_ZeroLength(t *testing.T) {
	_, err := arraysRoot([][]byte{}, 0)
	assert.ErrorContains(t, "zero leaves provided", err)
}
