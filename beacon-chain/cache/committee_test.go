package cache

import (
	"reflect"
	"strconv"
	"testing"
)

func TestSlotKeyFn_OK(t *testing.T) {
	cInfo := &CommitteesInSlot{
		Slot: 999,
		Committees: []*CommitteeInfo{
			{Shard: 1, Committee: []uint64{1, 2, 3}},
			{Shard: 1, Committee: []uint64{4, 5, 6}},
		},
	}

	key, err := slotKeyFn(cInfo)
	if err != nil {
		t.Fatal(err)
	}
	strSlot := strconv.Itoa(int(cInfo.Slot))
	if key != strSlot {
		t.Errorf("Incorrect hash key: %s, expected %s", key, strSlot)
	}
}

func TestSlotKeyFn_InvalidObj(t *testing.T) {
	_, err := slotKeyFn("bad")
	if err != ErrNotACommitteeInfo {
		t.Errorf("Expected error %v, got %v", ErrNotACommitteeInfo, err)
	}
}

func TestCommitteesCache_CommitteesInfoBySlot(t *testing.T) {
	cache := NewCommitteesCache()

	cInfo := &CommitteesInSlot{
		Slot:       123,
		Committees: []*CommitteeInfo{{Shard: 456}},
	}

	fetchedInfo, err := cache.CommitteesInfoBySlot(cInfo.Slot)
	if err != nil {
		t.Fatal(err)
	}
	if fetchedInfo != nil {
		t.Error("Expected committees info not to exist in empty cache")
	}

	if err := cache.AddCommittees(cInfo); err != nil {
		t.Fatal(err)
	}
	fetchedInfo, err = cache.CommitteesInfoBySlot(cInfo.Slot)
	if err != nil {
		t.Fatal(err)
	}
	if fetchedInfo == nil {
		t.Error("Expected committee info to exist")
	}
	if fetchedInfo.Slot != cInfo.Slot {
		t.Errorf(
			"Expected fetched slot number to be %d, got %d",
			cInfo.Slot,
			fetchedInfo.Slot,
		)
	}
	if !reflect.DeepEqual(fetchedInfo.Committees, cInfo.Committees) {
		t.Errorf(
			"Expected fetched info committee to be %v, got %v",
			cInfo.Committees,
			fetchedInfo.Committees,
		)
	}
}

func TestBlockCache_maxSize(t *testing.T) {
	cache := NewCommitteesCache()

	for i := 0; i < maxCacheSize+10; i++ {
		cInfo := &CommitteesInSlot{
			Slot: uint64(i),
		}
		if err := cache.AddCommittees(cInfo); err != nil {
			t.Fatal(err)
		}
	}

	if len(cache.committeesCache.ListKeys()) != maxCacheSize {
		t.Errorf(
			"Expected hash cache key size to be %d, got %d",
			maxCacheSize,
			len(cache.committeesCache.ListKeys()),
		)
	}
}
