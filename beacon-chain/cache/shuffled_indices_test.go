package cache

import (
	"reflect"
	"strconv"
	"testing"
)

func TestShuffleKeyFn_OK(t *testing.T) {
	sInfo := &IndicesByIndexSeed{
		Index:           999,
		Seed:            []byte{'A'},
		ShuffledIndices: []uint64{1, 2, 3, 4, 5},
	}

	key, err := shuffleKeyFn(sInfo)
	if err != nil {
		t.Fatal(err)
	}
	if key != string(sInfo.Seed)+strconv.Itoa(int(sInfo.Index)) {
		t.Errorf("Incorrect hash key: %s, expected %s", key, string(sInfo.Seed)+strconv.Itoa(int(sInfo.Index)))
	}
}

func TestShuffleKeyFn_InvalidObj(t *testing.T) {
	_, err := shuffleKeyFn("bad")
	if err != ErrNotValidatorListInfo {
		t.Errorf("Expected error %v, got %v", ErrNotValidatorListInfo, err)
	}
}

func TestShuffledIndicesCache_ShuffledIndicesBySeed2(t *testing.T) {
	cache := NewShuffledIndicesCache()

	sInfo := &IndicesByIndexSeed{
		Index:           99,
		Seed:            []byte{'A'},
		ShuffledIndices: []uint64{1, 2, 3, 4},
	}

	shuffledIndices, err := cache.IndicesByIndexSeed(sInfo.Index, sInfo.Seed)
	if err != nil {
		t.Fatal(err)
	}
	if shuffledIndices != nil {
		t.Error("Expected shuffled indices not to exist in empty cache")
	}

	if err := cache.AddShuffledValidatorList(sInfo); err != nil {
		t.Fatal(err)
	}
	shuffledIndices, err = cache.IndicesByIndexSeed(sInfo.Index, sInfo.Seed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(shuffledIndices, sInfo.ShuffledIndices) {
		t.Errorf(
			"Expected fetched info committee to be %v, got %v",
			sInfo.ShuffledIndices,
			shuffledIndices,
		)
	}
}

func TestShuffledIndices_MaxSize(t *testing.T) {
	cache := NewShuffledIndicesCache()

	for i := uint64(0); i < 1001; i++ {
		sInfo := &IndicesByIndexSeed{
			Index: i,
			Seed:  []byte{byte(i)},
		}
		if err := cache.AddShuffledValidatorList(sInfo); err != nil {
			t.Fatal(err)
		}
	}

	if len(cache.shuffledIndicesCache.ListKeys()) != maxShuffledListSize {
		t.Errorf(
			"Expected hash cache key size to be %d, got %d",
			maxShuffledListSize,
			len(cache.shuffledIndicesCache.ListKeys()),
		)
	}
}
