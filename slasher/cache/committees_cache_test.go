package cache_test

import (
	"context"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/cache"
)

func TestCommitteesCache_RoundTrip(t *testing.T) {
	ctx := context.Background()
	c := cache.NewCommitteesCache()
	numValidators := 64
	wanted := make(map[uint64]*ethpb.BeaconCommittees_CommitteesList)
	committeeItems := make([]*ethpb.BeaconCommittees_CommitteeItem, 1)
	committeeItems[0] = &ethpb.BeaconCommittees_CommitteeItem{ValidatorIndices: []uint64{1, 2, 3}}
	wanted[0] = &ethpb.BeaconCommittees_CommitteesList{Committees: committeeItems}
	wantedRes := &ethpb.BeaconCommittees{
		Epoch:                5,
		Committees:           wanted,
		ActiveValidatorCount: uint64(numValidators),
	}

	committees, err := c.Get(ctx, 5)
	if err != nil {
		t.Error(err)
	}

	if committees != nil {
		t.Errorf("Empty cache returned an object: %v", committees)
	}

	if err = c.Put(ctx, 5, wantedRes); err != nil {
		t.Error(err)
	}

	res, err := c.Get(ctx, 5)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(wantedRes, res) {
		t.Error("Expected equal protos to return from cache")
	}
}
