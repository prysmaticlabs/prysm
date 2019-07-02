package cache

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestTotalBalanceKeyFn_OK(t *testing.T) {
	tInfo := &TotalBalanceByEpoch{
		Epoch:        333,
		TotalBalance: 321321323,
	}

	key, err := totalBalanceKeyFn(tInfo)
	if err != nil {
		t.Fatal(err)
	}
	if key != strconv.Itoa(int(tInfo.Epoch)) {
		t.Errorf("Incorrect hash key: %s, expected %s", key, strconv.Itoa(int(tInfo.Epoch)))
	}
}

func TestTotalBalanceKeyFn_InvalidObj(t *testing.T) {
	_, err := totalBalanceKeyFn("bad")
	if err != ErrNotTotalBalanceInfo {
		t.Errorf("Expected error %v, got %v", ErrNotTotalBalanceInfo, err)
	}
}

func TestTotalBalanceCache_TotalBalanceByEpoch(t *testing.T) {
	cache := NewTotalBalanceCache()

	tInfo := &TotalBalanceByEpoch{
		Epoch:        111,
		TotalBalance: 345435435,
	}
	totalBalance, err := cache.TotalBalanceInEpoch(tInfo.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if totalBalance != params.BeaconConfig().FarFutureEpoch {
		t.Error("Expected total balance not to exist in empty cache")
	}

	if err := cache.AddTotalBalance(tInfo); err != nil {
		t.Fatal(err)
	}
	totalBalance, err = cache.TotalBalanceInEpoch(tInfo.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(totalBalance, tInfo.TotalBalance) {
		t.Errorf(
			"Expected fetched total balance to be %v, got %v",
			tInfo.TotalBalance,
			totalBalance,
		)
	}
}

func TestTotalBalance_MaxSize(t *testing.T) {
	cache := NewTotalBalanceCache()

	for i := uint64(0); i < params.BeaconConfig().EpochsPerHistoricalVector+100; i++ {
		tInfo := &TotalBalanceByEpoch{
			Epoch: i,
		}
		if err := cache.AddTotalBalance(tInfo); err != nil {
			t.Fatal(err)
		}
	}

	if len(cache.totalBalanceCache.ListKeys()) != maxTotalBalanceListSize {
		t.Errorf(
			"Expected hash cache key size to be %d, got %d",
			maxTotalBalanceListSize,
			len(cache.totalBalanceCache.ListKeys()),
		)
	}
}
