package cache

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestActiveCountKeyFn_OK(t *testing.T) {
	aInfo := &ActiveCountByEpoch{
		Epoch:       999,
		ActiveCount: 10,
	}

	key, err := activeCountKeyFn(aInfo)
	if err != nil {
		t.Fatal(err)
	}
	if key != strconv.Itoa(int(aInfo.Epoch)) {
		t.Errorf("Incorrect hash key: %s, expected %s", key, strconv.Itoa(int(aInfo.Epoch)))
	}
}

func TestActiveCountKeyFn_InvalidObj(t *testing.T) {
	_, err := activeCountKeyFn("bad")
	if err != ErrNotActiveCountInfo {
		t.Errorf("Expected error %v, got %v", ErrNotActiveCountInfo, err)
	}
}

func TestActiveCountCache_ActiveCountByEpoch(t *testing.T) {
	cache := NewActiveCountCache()

	aInfo := &ActiveCountByEpoch{
		Epoch:       99,
		ActiveCount: 11,
	}
	activeCount, err := cache.ActiveCountInEpoch(aInfo.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if activeCount != params.BeaconConfig().FarFutureEpoch {
		t.Error("Expected active count not to exist in empty cache")
	}

	if err := cache.AddActiveCount(aInfo); err != nil {
		t.Fatal(err)
	}
	activeCount, err = cache.ActiveCountInEpoch(aInfo.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(activeCount, aInfo.ActiveCount) {
		t.Errorf(
			"Expected fetched active count to be %v, got %v",
			aInfo.ActiveCount,
			activeCount,
		)
	}
}

func TestActiveCount_MaxSize(t *testing.T) {
	cache := NewActiveCountCache()

	for i := uint64(0); i < 1001; i++ {
		aInfo := &ActiveCountByEpoch{
			Epoch: i,
		}
		if err := cache.AddActiveCount(aInfo); err != nil {
			t.Fatal(err)
		}
	}

	if len(cache.activeCountCache.ListKeys()) != maxActiveCountListSize {
		t.Errorf(
			"Expected hash cache key size to be %d, got %d",
			maxActiveCountListSize,
			len(cache.activeCountCache.ListKeys()),
		)
	}
}
