package types

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestRoundTripSerialization(t *testing.T) {
	roundTripTestSSZUint64(t)
	roundTripTestBlocksByRootReq(t)
	roundTripTestErrorMessage(t)
}

func roundTripTestSSZUint64(t *testing.T) {
	fixedVal := uint64(8)
	sszVal := SSZUint64(fixedVal)

	marshalledObj, err := sszVal.MarshalSSZ()
	require.NoError(t, err)
	newVal := SSZUint64(0)

	err = newVal.UnmarshalSSZ(marshalledObj)
	require.NoError(t, err)
	assert.DeepEqual(t, fixedVal, uint64(newVal))
}

func roundTripTestBlocksByRootReq(t *testing.T) {
	fixedRoots := make([][32]byte, 0, 0)
	for i := 0; i < 200; i++ {
		fixedRoots = append(fixedRoots, [32]byte{byte(i)})
	}
	req := BeaconBlockByRootsReq(fixedRoots)

	marshalledObj, err := req.MarshalSSZ()
	require.NoError(t, err)
	newVal := BeaconBlockByRootsReq(nil)

	err = newVal.UnmarshalSSZ(marshalledObj)
	require.NoError(t, err)
	assert.DeepEqual(t, [][32]byte(newVal), fixedRoots)
}

func roundTripTestErrorMessage(t *testing.T) {
	errMsg := []byte{'e', 'r', 'r', 'o', 'r'}

	sszErr := make(ErrorMessage, len(errMsg))
	copy(sszErr, errMsg)

	marshalledObj, err := sszErr.MarshalSSZ()
	require.NoError(t, err)
	newVal := ErrorMessage(nil)

	err = newVal.UnmarshalSSZ(marshalledObj)
	require.NoError(t, err)
	assert.DeepEqual(t, []byte(newVal), errMsg)
}
