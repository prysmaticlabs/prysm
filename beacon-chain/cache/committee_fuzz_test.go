package cache

import (
	"reflect"
	"testing"

	fuzz "github.com/google/gofuzz"
)

func TestCommitteeKeyFuzz_OK(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	c := &Committees{}

	for i := 0; i < 100000; i++ {
		fuzzer.Fuzz(c)
		k, err := committeeKeyFn(c)
		if err != nil {
			t.Fatal(err)
		}
		if k != key(c.Seed) {
			t.Errorf("Incorrect hash k: %s, expected %s", k, key(c.Seed))
		}
	}
}

func TestCommitteeCache_FuzzCommitteesByEpoch(t *testing.T) {
	cache := NewCommitteesCache()
	fuzzer := fuzz.NewWithSeed(0)
	c := &Committees{}

	for i := 0; i < 100000; i++ {
		fuzzer.Fuzz(c)
		if err := cache.AddCommitteeShuffledList(c); err != nil {
			t.Fatal(err)
		}
		if _, err := cache.Committee(0, c.Seed, 0); err != nil {
			t.Fatal(err)
		}
	}

	if uint64(len(cache.CommitteeCache.ListKeys())) != maxCommitteesCacheSize {
		t.Error("Incorrect key size")
	}
}

func TestCommitteeCache_FuzzActiveIndices(t *testing.T) {
	cache := NewCommitteesCache()
	fuzzer := fuzz.NewWithSeed(0)
	c := &Committees{}

	for i := 0; i < 100000; i++ {
		fuzzer.Fuzz(c)
		if err := cache.AddCommitteeShuffledList(c); err != nil {
			t.Fatal(err)
		}
		indices, err := cache.ActiveIndices(c.Seed)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(indices, c.SortedIndices) {
			t.Error("Saved indices not the same")
		}
	}

	if uint64(len(cache.CommitteeCache.ListKeys())) != maxCommitteesCacheSize {
		t.Error("Incorrect key size")
	}
}
