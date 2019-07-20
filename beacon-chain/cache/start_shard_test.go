package cache

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestStartShardKeyFn_OK(t *testing.T) {
	tInfo := &StartShardByEpoch{
		Epoch:      44,
		StartShard: 3,
	}

	key, err := startShardKeyFn(tInfo)
	if err != nil {
		t.Fatal(err)
	}
	if key != strconv.Itoa(int(tInfo.Epoch)) {
		t.Errorf("Incorrect hash key: %s, expected %s", key, strconv.Itoa(int(tInfo.Epoch)))
	}
}

func TestStartShardKeyFn_InvalidObj(t *testing.T) {
	_, err := startShardKeyFn("bad")
	if err != ErrNotStartShardInfo {
		t.Errorf("Expected error %v, got %v", ErrNotStartShardInfo, err)
	}
}

func TestStartShardCache_StartShardByEpoch(t *testing.T) {
	cache := NewStartShardCache()

	tInfo := &StartShardByEpoch{
		Epoch:      55,
		StartShard: 3,
	}
	startShard, err := cache.StartShardInEpoch(tInfo.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if startShard != params.BeaconConfig().FarFutureEpoch {
		t.Error("Expected start shard not to exist in empty cache")
	}

	if err := cache.AddStartShard(tInfo); err != nil {
		t.Fatal(err)
	}
	startShard, err = cache.StartShardInEpoch(tInfo.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(startShard, tInfo.StartShard) {
		t.Errorf(
			"Expected fetched start shard to be %v, got %v",
			tInfo.StartShard,
			startShard,
		)
	}
}

func TestStartShard_MaxSize(t *testing.T) {
	cache := NewStartShardCache()

	for i := uint64(0); i < params.BeaconConfig().ShardCount+1; i++ {
		tInfo := &StartShardByEpoch{
			Epoch: i,
		}
		if err := cache.AddStartShard(tInfo); err != nil {
			t.Fatal(err)
		}
	}

	if len(cache.startShardCache.ListKeys()) != maxStartShardListSize {
		t.Errorf(
			"Expected hash cache key size to be %d, got %d",
			maxStartShardListSize,
			len(cache.startShardCache.ListKeys()),
		)
	}
}
