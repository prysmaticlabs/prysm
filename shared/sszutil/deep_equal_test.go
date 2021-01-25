package sszutil_test

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/sszutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestDeepEqualBasicTypes(t *testing.T) {
	assert.Equal(t, true, sszutil.DeepEqual(true, true))
	assert.Equal(t, false, sszutil.DeepEqual(true, false))

	assert.Equal(t, true, sszutil.DeepEqual(byte(222), byte(222)))
	assert.Equal(t, false, sszutil.DeepEqual(byte(222), byte(111)))

	assert.Equal(t, true, sszutil.DeepEqual(uint64(1234567890), uint64(1234567890)))
	assert.Equal(t, false, sszutil.DeepEqual(uint64(1234567890), uint64(987653210)))

	assert.Equal(t, true, sszutil.DeepEqual("hello", "hello"))
	assert.Equal(t, false, sszutil.DeepEqual("hello", "world"))

	assert.Equal(t, true, sszutil.DeepEqual([3]byte{1, 2, 3}, [3]byte{1, 2, 3}))
	assert.Equal(t, false, sszutil.DeepEqual([3]byte{1, 2, 3}, [3]byte{1, 2, 4}))

	var nilSlice1, nilSlice2 []byte
	assert.Equal(t, true, sszutil.DeepEqual(nilSlice1, nilSlice2))
	assert.Equal(t, true, sszutil.DeepEqual(nilSlice1, []byte{}))
	assert.Equal(t, true, sszutil.DeepEqual([]byte{1, 2, 3}, []byte{1, 2, 3}))
	assert.Equal(t, false, sszutil.DeepEqual([]byte{1, 2, 3}, []byte{1, 2, 4}))
}

func TestDeepEqualStructs(t *testing.T) {
	type Store struct {
		V1 uint64
		V2 []byte
	}
	store1 := Store{uint64(1234), nil}
	store2 := Store{uint64(1234), []byte{}}
	store3 := Store{uint64(4321), []byte{}}
	assert.Equal(t, true, sszutil.DeepEqual(store1, store2))
	assert.Equal(t, false, sszutil.DeepEqual(store1, store3))
}

func TestDeepEqualStructs_Unexported(t *testing.T) {
	type Store struct {
		V1       uint64
		V2       []byte
		ignoreMe string
	}
	store1 := Store{uint64(1234), nil, "hi there"}
	store2 := Store{uint64(1234), []byte{}, "oh hey"}
	store3 := Store{uint64(4321), []byte{}, "wow"}
	assert.Equal(t, true, sszutil.DeepEqual(store1, store2))
	assert.Equal(t, false, sszutil.DeepEqual(store1, store3))
}

func TestDeepEqualProto(t *testing.T) {
	var fork1, fork2 pb.Fork
	assert.Equal(t, true, sszutil.DeepEqual(fork1, fork2))

	fork1 = pb.Fork{
		PreviousVersion: []byte{123},
		CurrentVersion:  []byte{124},
		Epoch:           uint64(1234567890),
	}
	fork2 = pb.Fork{
		PreviousVersion: []byte{123},
		CurrentVersion:  []byte{125},
		Epoch:           uint64(1234567890),
	}
	assert.Equal(t, true, sszutil.DeepEqual(fork1, fork1))
	assert.Equal(t, false, sszutil.DeepEqual(fork1, fork2))

	checkpoint1 := ethpb.Checkpoint{
		Epoch: uint64(1234567890),
		Root:  []byte{},
	}
	checkpoint2 := ethpb.Checkpoint{
		Epoch: uint64(1234567890),
		Root:  nil,
	}
	assert.Equal(t, true, sszutil.DeepEqual(checkpoint1, checkpoint2))
}
