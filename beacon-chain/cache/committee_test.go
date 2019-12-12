package cache

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestCommitteeKeyFn_OK(t *testing.T) {
	item := &Committee{
		Epoch:          999,
		CommitteeCount: 1,
		Seed:           [32]byte{'A'},
		Committee:      []uint64{1, 2, 3, 4, 5},
	}

	k, err := committeeKeyFn(item)
	if err != nil {
		t.Fatal(err)
	}
	if k != key(item.Epoch, item.Seed) {
		t.Errorf("Incorrect hash k: %s, expected %s", k, key(item.Epoch, item.Seed))
	}
}

func TestCommitteeKeyFn_InvalidObj(t *testing.T) {
	_, err := committeeKeyFn("bad")
	if err != ErrNotCommittee {
		t.Errorf("Expected error %v, got %v", ErrNotCommittee, err)
	}
}

func TestCommitteeCache_CommitteesByEpoch(t *testing.T) {
	cache := NewCommitteeCache()

	item := &Committee{
		Epoch:          1,
		Committee:      []uint64{1, 2, 3, 4, 5, 6},
		Seed:           [32]byte{'A'},
		CommitteeCount: 3,
	}

	slot := uint64(item.Epoch * params.BeaconConfig().SlotsPerEpoch)
	committeeIndex := uint64(1)
	indices, err := cache.ShuffledIndices(slot, item.Seed, committeeIndex)
	if err != nil {
		t.Fatal(err)
	}
	if indices != nil {
		t.Error("Expected committee not to exist in empty cache")
	}

	if err := cache.AddCommitteeShuffledList(item); err != nil {
		t.Fatal(err)
	}
	wantedIndex := uint64(0)
	indices, err = cache.ShuffledIndices(slot, item.Seed, wantedIndex)
	if err != nil {
		t.Fatal(err)
	}

	start, end := startEndIndices(item, wantedIndex)
	if !reflect.DeepEqual(indices, item.Committee[start:end]) {
		t.Errorf(
			"Expected fetched active indices to be %v, got %v",
			indices,
			item.Committee[start:end],
		)
	}
}

func TestCommitteeCache_ActiveIndices(t *testing.T) {
	cache := NewCommitteeCache()

	item := &Committee{Epoch: 1, Seed: [32]byte{'A'}, Committee: []uint64{1, 2, 3, 4, 5, 6}}
	indices, err := cache.ActiveIndices(1, item.Seed)
	if err != nil {
		t.Fatal(err)
	}
	if indices != nil {
		t.Error("Expected committee count not to exist in empty cache")
	}

	if err := cache.AddCommitteeShuffledList(item); err != nil {
		t.Fatal(err)
	}

	indices, err = cache.ActiveIndices(1, item.Seed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(indices, item.Committee) {
		t.Error("Did not receive correct active indices from cache")
	}
}

func sum(values []uint64) uint64 {
	sum := uint64(0)
	for _, v := range values {
		sum = v + sum
	}
	return sum
}
