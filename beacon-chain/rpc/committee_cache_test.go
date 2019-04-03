package rpc

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
)

func TestSlotKeyFn_OK(t *testing.T) {
	cInfo := &committeesInfo{
		slot: 999,
		committees: []*helpers.CrosslinkCommittee{
			{Shard: 1, Committee: []uint64{1, 2, 3}},
			{Shard: 1, Committee: []uint64{4, 5, 6}},
		},
	}

	key, err := slotKeyFn(cInfo)
	if err != nil {
		t.Fatal(err)
	}
	if key != strconv.Itoa(cInfo.slot) {
		t.Errorf("Incorrect hash key: %s, expected %s", key, strconv.Itoa(cInfo.slot))
	}
}

func TestSlotKeyFn_InvalidObj(t *testing.T) {
	_, err := slotKeyFn("bad")
	if err != ErrNotACommitteeInfo {
		t.Errorf("Expected error %v, got %v", ErrNotACommitteeInfo, err)
	}
}

func TestCommitteesCache_CommitteesInfoBySlot(t *testing.T) {
	cache := newCommitteesCache()

	cInfo := &committeesInfo{
		slot:       123,
		committees: []*helpers.CrosslinkCommittee{{Shard: 456}},
	}

	exists, _, err := cache.CommitteesInfoBySlot(cInfo.slot)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("Expected committees info not to exist in empty cache")
	}

	if err := cache.AddCommittees(cInfo); err != nil {
		t.Fatal(err)
	}
	exists, fetchedInfo, err := cache.CommitteesInfoBySlot(cInfo.slot)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Expected committee info to exist")
	}
	if fetchedInfo.slot != cInfo.slot {
		t.Errorf(
			"Expected fetched slot number to be %d, got %d",
			cInfo.slot,
			fetchedInfo.slot,
		)
	}
	if !reflect.DeepEqual(fetchedInfo.committees, cInfo.committees) {
		t.Errorf(
			"Expected fetched info hash to be %v, got %v",
			cInfo.committees,
			fetchedInfo.committees,
		)
	}
}

func TestBlockCache_maxSize(t *testing.T) {
	cache := newCommitteesCache()

	for i := 0; i < maxCacheSize+10; i++ {
		cInfo := &committeesInfo{
			slot: i,
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
