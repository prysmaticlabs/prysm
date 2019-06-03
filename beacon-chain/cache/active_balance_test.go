package cache

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestActiveBalanceKeyFn_OK(t *testing.T) {
	tInfo := &ActiveBalanceByEpoch{
		Epoch:         45,
		ActiveBalance: 7456,
	}

	key, err := activeBalanceKeyFn(tInfo)
	if err != nil {
		t.Fatal(err)
	}
	if key != strconv.Itoa(int(tInfo.Epoch)) {
		t.Errorf("Incorrect hash key: %s, expected %s", key, strconv.Itoa(int(tInfo.Epoch)))
	}
}

func TestActiveBalanceKeyFn_InvalidObj(t *testing.T) {
	_, err := activeBalanceKeyFn("bad")
	if err != ErrNotActiveBalanceInfo {
		t.Errorf("Expected error %v, got %v", ErrNotActiveBalanceInfo, err)
	}
}

func TestActiveBalanceCache_ActiveBalanceByEpoch(t *testing.T) {
	cache := NewActiveBalanceCache()

	tInfo := &ActiveBalanceByEpoch{
		Epoch:         16511,
		ActiveBalance: 4456547,
	}
	activeBalance, err := cache.ActiveBalanceInEpoch(tInfo.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if activeBalance != params.BeaconConfig().FarFutureEpoch {
		t.Error("Expected active balance not to exist in empty cache")
	}

	if err := cache.AddActiveBalance(tInfo); err != nil {
		t.Fatal(err)
	}
	activeBalance, err = cache.ActiveBalanceInEpoch(tInfo.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(activeBalance, tInfo.ActiveBalance) {
		t.Errorf(
			"Expected fetched active balance to be %v, got %v",
			tInfo.ActiveBalance,
			activeBalance,
		)
	}
}

func TestActiveBalance_MaxSize(t *testing.T) {
	cache := NewActiveBalanceCache()

	for i := uint64(0); i < 1001; i++ {
		tInfo := &ActiveBalanceByEpoch{
			Epoch: i,
		}
		if err := cache.AddActiveBalance(tInfo); err != nil {
			t.Fatal(err)
		}
	}

	if len(cache.activeBalanceCache.ListKeys()) != maxActiveBalanceListSize {
		t.Errorf(
			"Expected hash cache key size to be %d, got %d",
			maxActiveBalanceListSize,
			len(cache.activeBalanceCache.ListKeys()),
		)
	}
}
