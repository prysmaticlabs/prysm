package types

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSSZUint64_Limit(t *testing.T) {
	sszType := SSZUint64(0)
	serializedObj := [7]byte{}
	require.ErrorContains(t, "expected buffer with length", sszType.UnmarshalSSZ(serializedObj[:]))
}

func TestBeaconBlockByRootsReq_Limit(t *testing.T) {
	fixedRoots := make([][32]byte, 0)
	for i := uint64(0); i < params.BeaconNetworkConfig().MaxRequestBlocks+100; i++ {
		fixedRoots = append(fixedRoots, [32]byte{byte(i)})
	}
	req := BeaconBlockByRootsReq(fixedRoots)

	_, err := req.MarshalSSZ()
	require.ErrorContains(t, "beacon block by roots request exceeds max size", err)

	buf := make([]byte, 0)
	for _, rt := range fixedRoots {
		buf = append(buf, rt[:]...)
	}
	req2 := BeaconBlockByRootsReq(nil)
	require.ErrorContains(t, "expected buffer with length of upto", req2.UnmarshalSSZ(buf))
}

func TestErrorResponse_Limit(t *testing.T) {
	errorMessage := make([]byte, 0)
	// Provide a message of size 6400 bytes.
	for i := uint64(0); i < 200; i++ {
		byteArr := [32]byte{byte(i)}
		errorMessage = append(errorMessage, byteArr[:]...)
	}
	errMsg := ErrorMessage{}
	require.ErrorContains(t, "expected buffer with length of upto", errMsg.UnmarshalSSZ(errorMessage))
}

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
	fixedRoots := make([][32]byte, 0)
	for i := 0; i < 200; i++ {
		fixedRoots = append(fixedRoots, [32]byte{byte(i)})
	}
	req := BeaconBlockByRootsReq(fixedRoots)

	marshalledObj, err := req.MarshalSSZ()
	require.NoError(t, err)
	newVal := BeaconBlockByRootsReq(nil)

	require.NoError(t, newVal.UnmarshalSSZ(marshalledObj))
	assert.DeepEqual(t, [][32]byte(newVal), fixedRoots)
}

func roundTripTestErrorMessage(t *testing.T) {
	errMsg := []byte{'e', 'r', 'r', 'o', 'r'}
	sszErr := make(ErrorMessage, len(errMsg))
	copy(sszErr, errMsg)

	marshalledObj, err := sszErr.MarshalSSZ()
	require.NoError(t, err)
	newVal := ErrorMessage(nil)

	require.NoError(t, newVal.UnmarshalSSZ(marshalledObj))
	assert.DeepEqual(t, []byte(newVal), errMsg)
}
