package cache_test

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestHotStateCache_RoundTrip(t *testing.T) {
	c := cache.NewHotStateCache()
	root := [32]byte{'A'}
	state := c.Get(root)
	if state != nil {
		t.Errorf("Empty cache returned an object: %v", state)
	}
	if c.Has(root) {
		t.Error("Empty cache has an object")
	}

	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	c.Put(root, state)

	if !c.Has(root) {
		t.Error("Empty cache does not have an object")
	}
	res := c.Get(root)
	if state == nil {
		t.Errorf("Empty cache returned an object: %v", state)
	}
	if !reflect.DeepEqual(state.CloneInnerState(), res.CloneInnerState()) {
		t.Error("Expected equal protos to return from cache")
	}

	c.Delete(root)
	if c.Has(root) {
		t.Error("Cache not suppose to have the object")
	}
}
