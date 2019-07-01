package cache

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestSeedKeyFn_OK(t *testing.T) {
	tInfo := &SeedByEpoch{
		Epoch: 44,
		Seed:  []byte{'A'},
	}

	key, err := seedKeyFn(tInfo)
	if err != nil {
		t.Fatal(err)
	}
	if key != strconv.Itoa(int(tInfo.Epoch)) {
		t.Errorf("Incorrect hash key: %s, expected %s", key, strconv.Itoa(int(tInfo.Epoch)))
	}
}

func TestSeedKeyFn_InvalidObj(t *testing.T) {
	_, err := seedKeyFn("bad")
	if err != ErrNotSeedInfo {
		t.Errorf("Expected error %v, got %v", ErrNotSeedInfo, err)
	}
}

func TestSeedCache_SeedByEpoch(t *testing.T) {
	cache := NewSeedCache()

	tInfo := &SeedByEpoch{
		Epoch: 55,
		Seed:  []byte{'B'},
	}
	seed, err := cache.SeedInEpoch(tInfo.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if seed != nil {
		t.Error("Expected seed not to exist in empty cache")
	}

	if err := cache.AddSeed(tInfo); err != nil {
		t.Fatal(err)
	}
	seed, err = cache.SeedInEpoch(tInfo.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(seed, tInfo.Seed) {
		t.Errorf(
			"Expected fetched seed to be %v, got %v",
			tInfo.Seed,
			seed,
		)
	}
}

func TestSeed_MaxSize(t *testing.T) {
	cache := NewSeedCache()

	for i := uint64(0); i < params.BeaconConfig().EpochsPerHistoricalVector+100; i++ {
		tInfo := &SeedByEpoch{
			Epoch: i,
		}
		if err := cache.AddSeed(tInfo); err != nil {
			t.Fatal(err)
		}
	}

	if len(cache.seedCache.ListKeys()) != maxSeedListSize {
		t.Errorf(
			"Expected hash cache key size to be %d, got %d",
			maxSeedListSize,
			len(cache.seedCache.ListKeys()),
		)
	}
}
