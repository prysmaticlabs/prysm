package cache

import (
	"math"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestCommitteeKeyFn_OK(t *testing.T) {
	item := &Committees{
		CommitteeCount:  1,
		Seed:            [32]byte{'A'},
		ShuffledIndices: []uint64{1, 2, 3, 4, 5},
	}

	k, err := committeeKeyFn(item)
	if err != nil {
		t.Fatal(err)
	}
	if k != key(item.Seed) {
		t.Errorf("Incorrect hash k: %s, expected %s", k, key(item.Seed))
	}
}

func TestCommitteeKeyFn_InvalidObj(t *testing.T) {
	_, err := committeeKeyFn("bad")
	if err != ErrNotCommittee {
		t.Errorf("Expected error %v, got %v", ErrNotCommittee, err)
	}
}

func TestCommitteeCache_CommitteesByEpoch(t *testing.T) {
	cache := NewCommitteesCache()

	item := &Committees{
		ShuffledIndices: []uint64{1, 2, 3, 4, 5, 6},
		Seed:            [32]byte{'A'},
		CommitteeCount:  3,
	}

	slot := params.BeaconConfig().SlotsPerEpoch
	committeeIndex := uint64(1)
	indices, err := cache.Committee(slot, item.Seed, committeeIndex)
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
	indices, err = cache.Committee(slot, item.Seed, wantedIndex)
	if err != nil {
		t.Fatal(err)
	}

	start, end := startEndIndices(item, wantedIndex)
	if !reflect.DeepEqual(indices, item.ShuffledIndices[start:end]) {
		t.Errorf(
			"Expected fetched active indices to be %v, got %v",
			indices,
			item.ShuffledIndices[start:end],
		)
	}
}

func TestCommitteeCache_ActiveIndices(t *testing.T) {
	cache := NewCommitteesCache()

	item := &Committees{Seed: [32]byte{'A'}, SortedIndices: []uint64{1, 2, 3, 4, 5, 6}}
	indices, err := cache.ActiveIndices(item.Seed)
	if err != nil {
		t.Fatal(err)
	}
	if indices != nil {
		t.Error("Expected committee count not to exist in empty cache")
	}

	if err := cache.AddCommitteeShuffledList(item); err != nil {
		t.Fatal(err)
	}

	indices, err = cache.ActiveIndices(item.Seed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(indices, item.SortedIndices) {
		t.Error("Did not receive correct active indices from cache")
	}
}

func TestCommitteeCache_ActiveCount(t *testing.T) {
	cache := NewCommitteesCache()

	item := &Committees{Seed: [32]byte{'A'}, SortedIndices: []uint64{1, 2, 3, 4, 5, 6}}
	count, err := cache.ActiveIndicesCount(item.Seed)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Error("Expected active count not to exist in empty cache")
	}

	if err := cache.AddCommitteeShuffledList(item); err != nil {
		t.Fatal(err)
	}

	count, err = cache.ActiveIndicesCount(item.Seed)
	if err != nil {
		t.Fatal(err)
	}
	if count != len(item.SortedIndices) {
		t.Error("Did not receive correct active acount from cache")
	}
}

func TestCommitteeCache_AddProposerIndicesList(t *testing.T) {
	cache := NewCommitteesCache()

	seed := [32]byte{'A'}
	indices := []uint64{1, 2, 3, 4, 5}
	indices, err := cache.ProposerIndices(seed)
	if err != nil {
		t.Fatal(err)
	}
	if indices != nil {
		t.Error("Expected committee count not to exist in empty cache")
	}
	if err := cache.AddProposerIndicesList(seed, indices); err != nil {
		t.Fatal(err)
	}
	received, err := cache.ProposerIndices(seed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(indices, received) {
		t.Error("Did not receive correct proposer indices from cache")
	}

	item := &Committees{Seed: [32]byte{'B'}, SortedIndices: []uint64{1, 2, 3, 4, 5, 6}}
	if err := cache.AddCommitteeShuffledList(item); err != nil {
		t.Fatal(err)
	}
	indices, err = cache.ProposerIndices(item.Seed)
	if err != nil {
		t.Fatal(err)
	}
	if indices != nil {
		t.Error("Expected committee count not to exist in empty cache")
	}
	if err := cache.AddProposerIndicesList(item.Seed, indices); err != nil {
		t.Fatal(err)
	}
	received, err = cache.ProposerIndices(item.Seed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(indices, received) {
		t.Error("Did not receive correct proposer indices from cache")
	}

}

func TestCommitteeCache_CanRotate(t *testing.T) {
	cache := NewCommitteesCache()

	// Should rotate out all the epochs except 190 through 199.
	for i := 100; i < 200; i++ {
		s := []byte(strconv.Itoa(i))
		item := &Committees{Seed: bytesutil.ToBytes32(s)}
		if err := cache.AddCommitteeShuffledList(item); err != nil {
			t.Fatal(err)
		}
	}

	k := cache.CommitteeCache.ListKeys()
	if uint64(len(k)) != maxCommitteesCacheSize {
		t.Errorf("wanted: %d, got: %d", maxCommitteesCacheSize, len(k))
	}

	sort.Slice(k, func(i, j int) bool {
		return k[i] < k[j]
	})
	s := bytesutil.ToBytes32([]byte(strconv.Itoa(190)))
	if k[0] != key(s) {
		t.Error("incorrect key received for slot 190")
	}
	s = bytesutil.ToBytes32([]byte(strconv.Itoa(199)))
	if k[len(k)-1] != key(s) {
		t.Error("incorrect key received for slot 199")
	}
}

func TestCommitteeCacheOutOfRange(t *testing.T) {
	cache := NewCommitteesCache()
	seed := bytesutil.ToBytes32([]byte("foo"))
	err := cache.CommitteeCache.Add(&Committees{
		CommitteeCount:  1,
		Seed:            seed,
		ShuffledIndices: []uint64{0},
		SortedIndices:   []uint64{},
		ProposerIndices: []uint64{},
	})
	if err != nil {
		t.Error(err)
	}
	_, err = cache.Committee(0, seed, math.MaxUint64) // Overflow!
	if err == nil {
		t.Fatal("Did not fail as expected")
	}
}
