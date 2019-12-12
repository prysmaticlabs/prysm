package cache

import (
	"reflect"
	"sort"
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

func TestCommitteeCache_CanRotate(t *testing.T) {
	cache := NewCommitteeCache()
	seed := [32]byte{'A'}

	// Should rotate out all the epochs except 190 to 199
	for i := 100; i < 200; i++ {
		item := &Committee{Epoch: uint64(i), Seed: seed}
		if err := cache.AddCommitteeShuffledList(item); err != nil {
			t.Fatal(err)
		}
	}

	k := cache.CommitteeCache.ListKeys()
	if len(k) != maxCommitteeSize {
		t.Errorf("wanted: %d, got: %d", maxCommitteeSize, len(k))
	}

	sort.Slice(k, func(i, j int) bool {
		return k[i] < k[j]
	})

	if k[0] != key(190, seed) {
		t.Error("incorrect key received for slot 190")
	}
	if k[len(k)-1] != key(199, seed) {
		t.Error("incorrect key received for slot 199")
	}
}
