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
		V1           uint64
		V2           []byte
		dontIgnoreMe string
	}
	store1 := Store{uint64(1234), nil, "hi there"}
	store2 := Store{uint64(1234), []byte{}, "hi there"}
	store3 := Store{uint64(4321), []byte{}, "wow"}
	store4 := Store{uint64(4321), []byte{}, "bow wow"}
	assert.Equal(t, true, sszutil.DeepEqual(store1, store2))
	assert.Equal(t, false, sszutil.DeepEqual(store1, store3))
	assert.Equal(t, false, sszutil.DeepEqual(store3, store4))
}

func TestDeepEqualProto(t *testing.T) {
	var fork1, fork2 pb.Fork
	assert.Equal(t, true, sszutil.DeepEqual(&fork1, &fork2))

	fork1 = pb.Fork{
		PreviousVersion: []byte{123},
		CurrentVersion:  []byte{124},
		Epoch:           1234567890,
	}
	fork2 = pb.Fork{
		PreviousVersion: []byte{123},
		CurrentVersion:  []byte{125},
		Epoch:           1234567890,
	}
	assert.Equal(t, true, sszutil.DeepEqual(&fork1, &fork1))
	assert.Equal(t, false, sszutil.DeepEqual(&fork1, &fork2))

	checkpoint1 := ethpb.Checkpoint{
		Epoch: 1234567890,
		Root:  []byte{},
	}
	checkpoint2 := ethpb.Checkpoint{
		Epoch: 1234567890,
		Root:  nil,
	}
	assert.Equal(t, true, sszutil.DeepEqual(&checkpoint1, &checkpoint2))
}

func TestImplementsProtoMessage(t *testing.T) {
	type C struct{}

	tests := []struct {
		name  string
		input interface{}
		want  bool
	}{
		{
			name:  "Pointer to struct which implements proto.Message returns true",
			input: &ethpb.Attestation{},
			want:  true,
		},
		{
			name:  "Pointer to struct which does not implement proto.Message returns false",
			input: &C{},
			want:  false,
		},
		{
			name:  "Struct which implements proto.Message returns true",
			input: ethpb.Attestation{},
			want:  true,
		},
		{
			name:  "Struct which does not implement proto.Message returns false",
			input: C{},
			want:  false,
		},
		{
			name:  "Slice of slices of struct values which implement proto.Message returns true",
			input: [][]ethpb.Attestation{{{}}},
			want:  true,
		},
		{
			name:  "Slice of struct values which implement proto.Message returns true",
			input: []ethpb.Attestation{{}},
			want:  true,
		},
		{
			name:  "Empty slice of struct values which implement proto.Message returns true",
			input: []ethpb.Attestation{},
			want:  true,
		},
		{
			name:  "Slice of struct values which do not implement proto.Message returns false",
			input: []C{{}},
			want:  false,
		},
		{
			name:  "Empty slice of struct values which do not implement proto.Message returns false",
			input: []C{},
			want:  false,
		},
		{
			name:  "Map of value with struct type implementing proto.Message returns true",
			input: make(map[int]ethpb.Attestation),
			want:  true,
		},
		{
			name:  "Map of value with struct type not implementing proto.Message returns false",
			input: make(map[int]C),
			want:  false,
		},
		{
			name:  "Map of value with pointer type implementing proto.Message returns true",
			input: make(map[int]*ethpb.Attestation),
			want:  true,
		},
		{
			name:  "Map of value with slice of pointer type implementing proto.Message returns true",
			input: make(map[int][]*ethpb.Attestation),
			want:  true,
		},
		{
			name:  "Map of value with slice of value type implementing proto.Message returns true",
			input: make(map[int][]ethpb.Attestation),
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sszutil.ImplementsProtoMessageReflect(tt.input); got != tt.want {
				t.Errorf("ImplementsProtoMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}
