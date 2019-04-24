package cache

import (
	"reflect"
	"strconv"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestHeightHeightFn_OK(t *testing.T) {
	height := uint64(999)
	hash := []byte{'A'}
	aInfo := &AncestorInfo{
		Height: height,
		Hash:   hash,
		Target: &pb.AttestationTarget{
			Slot:      height,
			BlockRoot: hash,
		},
	}

	key, err := heightKeyFn(aInfo)
	if err != nil {
		t.Fatal(err)
	}

	strHeightKey := string(aInfo.Target.BlockRoot) + strconv.Itoa(int(aInfo.Target.Slot))
	if key != strHeightKey {
		t.Errorf("Incorrect hash key: %s, expected %s", key, strHeightKey)
	}
}

func TestHeightKeyFn_InvalidObj(t *testing.T) {
	_, err := heightKeyFn("bad")
	if err != ErrNotAncestorCacheObj {
		t.Errorf("Expected error %v, got %v", ErrNotAncestorCacheObj, err)
	}
}

func TestAncestorCache_AncestorInfoByHeight(t *testing.T) {
	cache := NewBlockAncestorCache()

	height := uint64(123)
	hash := []byte{'B'}
	aInfo := &AncestorInfo{
		Height: height,
		Hash:   hash,
		Target: &pb.AttestationTarget{
			Slot:      height,
			BlockRoot: hash,
		},
	}

	fetchedInfo, err := cache.AncestorBySlot(hash, height)
	if err != nil {
		t.Fatal(err)
	}
	if fetchedInfo != nil {
		t.Error("Expected ancestor info not to exist in empty cache")
	}

	if err := cache.AddBlockAncestor(aInfo); err != nil {
		t.Fatal(err)
	}
	fetchedInfo, err = cache.AncestorBySlot(hash, height)
	if err != nil {
		t.Fatal(err)
	}
	if fetchedInfo == nil {
		t.Error("Expected ancestor info to exist")
	}
	if fetchedInfo.Height != height {
		t.Errorf(
			"Expected fetched slot number to be %d, got %d",
			aInfo.Target.Slot,
			fetchedInfo.Target.Slot,
		)
	}
	if !reflect.DeepEqual(fetchedInfo.Target, aInfo.Target) {
		t.Errorf(
			"Expected fetched info committee to be %v, got %v",
			aInfo.Target,
			fetchedInfo.Target,
		)
	}
}

func TestBlockAncestor_maxSize(t *testing.T) {
	cache := NewBlockAncestorCache()

	for i := 0; i < maxCacheSize+10; i++ {
		aInfo := &AncestorInfo{
			Height: uint64(i),
			Target: &pb.AttestationTarget{
				Slot: uint64(i),
			},
		}
		if err := cache.AddBlockAncestor(aInfo); err != nil {
			t.Fatal(err)
		}
	}

	if len(cache.ancestorBlockCache.ListKeys()) != maxCacheSize {
		t.Errorf(
			"Expected hash cache key size to be %d, got %d",
			maxCacheSize,
			len(cache.ancestorBlockCache.ListKeys()),
		)
	}
}
