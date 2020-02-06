package cache

import (
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func TestCheckpointStateCacheKeyFn_OK(t *testing.T) {
	cp := &ethpb.Checkpoint{Epoch: 1, Root: []byte{'A'}}
	st, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot: 64,
	})
	if err != nil {
		t.Fatal(err)
	}
	info := &CheckpointState{
		Checkpoint: cp,
		State:      st,
	}
	key, err := checkpointState(info)
	if err != nil {
		t.Fatal(err)
	}
	wantedKey, err := hashutil.HashProto(cp)
	if err != nil {
		t.Fatal(err)
	}
	if key != string(wantedKey[:]) {
		t.Errorf("Incorrect hash key: %s, expected %s", key, string(wantedKey[:]))
	}
}

func TestCheckpointStateCacheKeyFn_InvalidObj(t *testing.T) {
	_, err := checkpointState("bad")
	if err != ErrNotCheckpointState {
		t.Errorf("Expected error %v, got %v", ErrNotCheckpointState, err)
	}
}

func TestCheckpointStateCache_StateByCheckpoint(t *testing.T) {
	cache := NewCheckpointStateCache()

	cp1 := &ethpb.Checkpoint{Epoch: 1, Root: []byte{'A'}}
	st, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot: 64,
	})
	if err != nil {
		t.Fatal(err)
	}
	info1 := &CheckpointState{
		Checkpoint: cp1,
		State:      st,
	}
	state, err := cache.StateByCheckpoint(cp1)
	if err != nil {
		t.Fatal(err)
	}
	if state != nil {
		t.Error("Expected state not to exist in empty cache")
	}

	if err := cache.AddCheckpointState(info1); err != nil {
		t.Fatal(err)
	}
	state, err = cache.StateByCheckpoint(cp1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state.InnerStateUnsafe(), info1.State.InnerStateUnsafe()) {
		t.Error("incorrectly cached state")
	}

	cp2 := &ethpb.Checkpoint{Epoch: 2, Root: []byte{'B'}}
	st2, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot: 128,
	})
	if err != nil {
		t.Fatal(err)
	}
	info2 := &CheckpointState{
		Checkpoint: cp2,
		State:      st2,
	}
	if err := cache.AddCheckpointState(info2); err != nil {
		t.Fatal(err)
	}
	state, err = cache.StateByCheckpoint(cp2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state.CloneInnerState(), info2.State.CloneInnerState()) {
		t.Error("incorrectly cached state")
	}

	state, err = cache.StateByCheckpoint(cp1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state.CloneInnerState(), info1.State.CloneInnerState()) {
		t.Error("incorrectly cached state")
	}
}

func TestCheckpointStateCache_MaxSize(t *testing.T) {
	c := NewCheckpointStateCache()
	st, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxCheckpointStateSize+100; i++ {
		if err := st.SetSlot(uint64(i)); err != nil {
			t.Fatal(err)
		}
		info := &CheckpointState{
			Checkpoint: &ethpb.Checkpoint{Epoch: uint64(i)},
			State:      st,
		}
		if err := c.AddCheckpointState(info); err != nil {
			t.Fatal(err)
		}
	}

	if len(c.cache.ListKeys()) != maxCheckpointStateSize {
		t.Errorf(
			"Expected hash cache key size to be %d, got %d",
			maxCheckpointStateSize,
			len(c.cache.ListKeys()),
		)
	}
}
