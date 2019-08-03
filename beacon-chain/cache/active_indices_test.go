package cache

import (
	"reflect"
	"strconv"
	"testing"
)

func TestActiveIndicesKeyFn_OK(t *testing.T) {
	aInfo := &ActiveIndicesByEpoch{
		Epoch:         999,
		ActiveIndices: []uint64{1, 2, 3, 4, 5},
	}

	key, err := activeIndicesKeyFn(aInfo)
	if err != nil {
		t.Fatal(err)
	}
	if key != strconv.Itoa(int(aInfo.Epoch)) {
		t.Errorf("Incorrect hash key: %s, expected %s", key, strconv.Itoa(int(aInfo.Epoch)))
	}
}

func TestActiveIndicesKeyFn_InvalidObj(t *testing.T) {
	_, err := activeIndicesKeyFn("bad")
	if err != ErrNotActiveIndicesInfo {
		t.Errorf("Expected error %v, got %v", ErrNotActiveIndicesInfo, err)
	}
}

func TestActiveIndicesCache_ActiveIndicesByEpoch(t *testing.T) {
	cache := NewActiveIndicesCache()

	aInfo := &ActiveIndicesByEpoch{
		Epoch:         99,
		ActiveIndices: []uint64{1, 2, 3, 4},
	}

	activeIndices, err := cache.ActiveIndicesInEpoch(aInfo.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if activeIndices != nil {
		t.Error("Expected active indices not to exist in empty cache")
	}

	if err := cache.AddActiveIndicesList(aInfo); err != nil {
		t.Fatal(err)
	}
	activeIndices, err = cache.ActiveIndicesInEpoch(aInfo.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(activeIndices, aInfo.ActiveIndices) {
		t.Errorf(
			"Expected fetched active indices to be %v, got %v",
			aInfo.ActiveIndices,
			activeIndices,
		)
	}
}

func TestActiveIndices_MaxSize(t *testing.T) {
	cache := NewActiveIndicesCache()

	for i := uint64(0); i < 100; i++ {
		aInfo := &ActiveIndicesByEpoch{
			Epoch: i,
		}
		if err := cache.AddActiveIndicesList(aInfo); err != nil {
			t.Fatal(err)
		}
	}

	if len(cache.activeIndicesCache.ListKeys()) != maxActiveIndicesListSize {
		t.Errorf(
			"Expected hash cache key size to be %d, got %d",
			maxActiveIndicesListSize,
			len(cache.activeIndicesCache.ListKeys()),
		)
	}
}
