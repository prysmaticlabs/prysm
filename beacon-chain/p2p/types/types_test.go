package types

import (
	"encoding/hex"
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

func TestSSZUint64(t *testing.T) {
	tests := []struct {
		name            string
		serializedBytes []byte
		actualValue     uint64
		root            []byte
		wantErr         bool
	}{
		{
			name:            "max",
			serializedBytes: hexDecodeOrDie(t, "ffffffffffffffff"),
			actualValue:     18446744073709551615,
			root:            hexDecodeOrDie(t, "ffffffffffffffff000000000000000000000000000000000000000000000000"),
			wantErr:         false,
		},
		{
			name:            "random",
			serializedBytes: hexDecodeOrDie(t, "357c8de9d7204577"),
			actualValue:     8594311575614880821,
			root:            hexDecodeOrDie(t, "357c8de9d7204577000000000000000000000000000000000000000000000000"),
			wantErr:         false,
		},
		{
			name:            "zero",
			serializedBytes: hexDecodeOrDie(t, "0000000000000000"),
			actualValue:     0,
			root:            hexDecodeOrDie(t, "0000000000000000000000000000000000000000000000000000000000000000"),
			wantErr:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s SSZUint64
			if err := s.UnmarshalSSZ(tt.serializedBytes); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalSSZ() error = %v, wantErr %v", err, tt.wantErr)
			}
			require.Equal(t, uint64(s), tt.actualValue)

			serializedBytes, err := s.MarshalSSZ()
			require.NoError(t, err)
			require.DeepEqual(t, tt.serializedBytes, serializedBytes)

			htr, err := s.HashTreeRoot()
			require.NoError(t, err)
			require.DeepEqual(t, tt.root, htr[:])
		})
	}
}

func TestSSZBytes_HashTreeRoot(t *testing.T) {
	tests := []struct {
		name        string
		actualValue []byte
		root        []byte
		wantErr     bool
	}{
		{
			name:        "random1",
			actualValue: hexDecodeOrDie(t, "844e1063e0b396eed17be8eddb7eecd1fe3ea46542a4b72f7466e77325e5aa6d"),
			root:        hexDecodeOrDie(t, "844e1063e0b396eed17be8eddb7eecd1fe3ea46542a4b72f7466e77325e5aa6d"),
			wantErr:     false,
		},
		{
			name:        "random1",
			actualValue: hexDecodeOrDie(t, "7b16162ecd9a28fa80a475080b0e4fff4c27efe19ce5134ce3554b72274d59fd534400ba4c7f699aa1c307cd37c2b103"),
			root:        hexDecodeOrDie(t, "128ed34ee798b9f00716f9ba5c000df5c99443dabc4d3f2e9bb86c77c732e007"),
			wantErr:     false,
		},
		{
			name:        "random2",
			actualValue: []byte{},
			root:        hexDecodeOrDie(t, "0000000000000000000000000000000000000000000000000000000000000000"),
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := SSZBytes(tt.actualValue)
			htr, err := s.HashTreeRoot()
			require.NoError(t, err)
			require.DeepEqual(t, tt.root, htr[:])
		})
	}
}

func hexDecodeOrDie(t *testing.T, str string) []byte {
	decoded, err := hex.DecodeString(str)
	require.NoError(t, err)
	return decoded
}
