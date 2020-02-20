package cache_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

func TestSkipSlotCache_RoundTrip(t *testing.T) {
	ctx := context.Background()
	c := cache.NewSkipSlotCache()
	fc := featureconfig.Get()
	fc.EnableSkipSlotsCache = true
	featureconfig.Init(fc)

	state, err := c.Get(ctx, 5)
	if err != nil {
		t.Error(err)
	}

	if state != nil {
		t.Errorf("Empty cache returned an object: %v", state)
	}

	if err := c.MarkInProgress(5); err != nil {
		t.Error(err)
	}

	state, err = stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot: 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = c.Put(ctx, 5, state); err != nil {
		t.Error(err)
	}

	if err := c.MarkNotInProgress(5); err != nil {
		t.Error(err)
	}

	res, err := c.Get(ctx, 5)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(state.CloneInnerState(), res.CloneInnerState()) {
		t.Error("Expected equal protos to return from cache")
	}
}
