package cache

import (
	"reflect"
	"strconv"
	"testing"
)

func TestCommitteeKeyFn_OK(t *testing.T) {
	item := &Committee{
		Epoch:          999,
		CommitteeCount: 1,
		Committee:      []uint64{1, 2, 3, 4, 5},
	}

	key, err := committeeKeyFn(item)
	if err != nil {
		t.Fatal(err)
	}
	if key != strconv.Itoa(int(item.Epoch)) {
		t.Errorf("Incorrect hash key: %s, expected %s", key, strconv.Itoa(int(item.Epoch)))
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
		CommitteeCount: 3,
		StartShard:     1,
	}

	epoch := uint64(1)
	startShard := uint64(1)
	indices, err := cache.ShuffledIndices(epoch, startShard)
	if err != nil {
		t.Fatal(err)
	}
	if indices != nil {
		t.Error("Expected committee not to exist in empty cache")
	}

	if err := cache.AddCommitteeShuffledList(item); err != nil {
		t.Fatal(err)
	}
	wantedShard := uint64(2)
	indices, err = cache.ShuffledIndices(epoch, wantedShard)
	if err != nil {
		t.Fatal(err)
	}
	start, end := startEndIndices(item, wantedShard)
	if !reflect.DeepEqual(indices, item.Committee[start:end]) {
		t.Errorf(
			"Expected fetched active indices to be %v, got %v",
			indices,
			item.Committee[start:end],
		)
	}
}

func TestCommitteeCache_CanRotate(t *testing.T) {
	cache := NewCommitteeCache()
	item1 := &Committee{Epoch: 1}
	if err := cache.AddCommitteeShuffledList(item1); err != nil {
		t.Fatal(err)
	}
	item2 := &Committee{Epoch: 2}
	if err := cache.AddCommitteeShuffledList(item2); err != nil {
		t.Fatal(err)
	}
	epochs, err := cache.Epochs()
	if err != nil {
		t.Fatal(err)
	}
	wanted := item1.Epoch + item2.Epoch
	if sum(epochs) != wanted {
		t.Errorf("Wanted: %v, got: %v", wanted, sum(epochs))
	}

	item3 := &Committee{Epoch: 4}
	if err := cache.AddCommitteeShuffledList(item3); err != nil {
		t.Fatal(err)
	}
	epochs, err = cache.Epochs()
	if err != nil {
		t.Fatal(err)
	}
	wanted = item1.Epoch + item2.Epoch + item3.Epoch
	if sum(epochs) != wanted {
		t.Errorf("Wanted: %v, got: %v", wanted, sum(epochs))
	}

	item4 := &Committee{Epoch: 6}
	if err := cache.AddCommitteeShuffledList(item4); err != nil {
		t.Fatal(err)
	}
	epochs, err = cache.Epochs()
	if err != nil {
		t.Fatal(err)
	}
	wanted = item2.Epoch + item3.Epoch + item4.Epoch
	if sum(epochs) != wanted {
		t.Errorf("Wanted: %v, got: %v", wanted, sum(epochs))
	}
}

func TestCommitteeCache_EpochInCache(t *testing.T) {
	cache := NewCommitteeCache()
	if err := cache.AddCommitteeShuffledList(&Committee{Epoch: 1}); err != nil {
		t.Fatal(err)
	}
	if err := cache.AddCommitteeShuffledList(&Committee{Epoch: 2}); err != nil {
		t.Fatal(err)
	}
	if err := cache.AddCommitteeShuffledList(&Committee{Epoch: 99}); err != nil {
		t.Fatal(err)
	}
	if err := cache.AddCommitteeShuffledList(&Committee{Epoch: 100}); err != nil {
		t.Fatal(err)
	}
	inCache, err := cache.EpochInCache(1)
	if err != nil {
		t.Fatal(err)
	}
	if inCache {
		t.Error("Epoch shouldn't be in cache")
	}
	inCache, err = cache.EpochInCache(100)
	if err != nil {
		t.Fatal(err)
	}
	if !inCache {
		t.Error("Epoch should be in cache")
	}
}

func TestCommitteeCache_CommitteesCount(t *testing.T) {
	cache := NewCommitteeCache()

	committeeCount := uint64(3)
	epoch := uint64(10)
	item := &Committee{Epoch: epoch, CommitteeCount: committeeCount}

	_, exists, err := cache.CommitteeCount(1)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("Expected committee count not to exist in empty cache")
	}

	if err := cache.AddCommitteeShuffledList(item); err != nil {
		t.Fatal(err)
	}

	count, exists, err := cache.CommitteeCount(epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Expected committee count to be in cache")
	}
	if count != committeeCount {
		t.Errorf("wanted: %d, got: %d", committeeCount, count)
	}
}

func TestCommitteeCache_ShardCount(t *testing.T) {
	cache := NewCommitteeCache()

	startShard := uint64(7)
	epoch := uint64(3)
	item := &Committee{Epoch: epoch, StartShard: startShard}

	_, exists, err := cache.StartShard(1)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("Expected start shard not to exist in empty cache")
	}

	if err := cache.AddCommitteeShuffledList(item); err != nil {
		t.Fatal(err)
	}

	shard, exists, err := cache.StartShard(epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Expected start shard to be in cache")
	}
	if shard != startShard {
		t.Errorf("wanted: %d, got: %d", startShard, shard)
	}
}

func sum(values []uint64) uint64 {
	sum := uint64(0)
	for _, v := range values {
		sum = v + sum
	}
	return sum
}
