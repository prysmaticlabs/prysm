package ssz_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestUint64Root(t *testing.T) {
	uintVal := uint64(1234567890)
	expected := [32]byte{210, 2, 150, 73, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	result := ssz.Uint64Root(uintVal)
	assert.Equal(t, expected, result)
}

func TestForkRoot(t *testing.T) {
	testFork := ethpb.Fork{
		PreviousVersion: []byte{123},
		CurrentVersion:  []byte{124},
		Epoch:           1234567890,
	}
	expected := [32]byte{19, 46, 77, 103, 92, 175, 247, 33, 100, 64, 17, 111, 199, 145, 69, 38, 217, 112, 6, 16, 149, 201, 225, 144, 192, 228, 197, 172, 157, 78, 114, 140}

	result, err := ssz.ForkRoot(&testFork)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestCheckPointRoot(t *testing.T) {
	testHasher := hash.CustomSHA256Hasher()
	testCheckpoint := ethpb.Checkpoint{
		Epoch: 1234567890,
		Root:  []byte{222},
	}
	expected := [32]byte{228, 65, 39, 109, 183, 249, 167, 232, 125, 239, 25, 155, 207, 4, 84, 174, 176, 229, 175, 224, 62, 33, 215, 254, 170, 220, 132, 65, 246, 128, 68, 194}

	result, err := ssz.CheckpointRoot(testHasher, &testCheckpoint)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestByteArrayRootWithLimit(t *testing.T) {
	testHistoricalRoots := [][]byte{{123}, {234}}
	expected := [32]byte{70, 204, 150, 196, 89, 138, 190, 205, 65, 207, 120, 166, 179, 247, 147, 20, 29, 133, 117, 116, 151, 234, 129, 32, 22, 15, 79, 178, 98, 73, 132, 152}

	result, err := ssz.ByteArrayRootWithLimit(testHistoricalRoots, 16777216)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestSlashingsRoot(t *testing.T) {
	testSlashingsRoot := []uint64{123, 234}
	expected := [32]byte{123, 0, 0, 0, 0, 0, 0, 0, 234, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	result, err := ssz.SlashingsRoot(testSlashingsRoot)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}
