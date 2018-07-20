package database

import (
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
)

// Verifies that KVStore implements the ethdb interface.
var _ = ethdb.Database(&KVStore{})

func Test_KVStorePut(t *testing.T) {
	kv := NewKVStore()

	if err := kv.Put([]byte("ralph merkle"), []byte{1, 2, 3}); err != nil {
		t.Errorf("could not save value in kv store: %v", err)
	}
}

func Test_KVStoreHas(t *testing.T) {
	kv := NewKVStore()
	key := []byte("ralph merkle")

	if err := kv.Put(key, []byte{1, 2, 3}); err != nil {
		t.Fatalf("could not save value in kv store: %v", err)
	}

	has, err := kv.Has(key)
	if err != nil {
		t.Errorf("could not check if kv store has key: %v", err)
	}
	if !has {
		t.Errorf("kv store should have key: %v", key)
	}

	key2 := []byte{}
	has2, err := kv.Has(key2)
	if err != nil {
		t.Errorf("could not check if kv store has key: %v", err)
	}
	if has2 {
		t.Errorf("kv store should not have non-existent key: %v", key2)
	}
}

func Test_KVStoreGet(t *testing.T) {
	kv := NewKVStore()
	key := []byte("ralph merkle")

	if err := kv.Put(key, []byte{1, 2, 3}); err != nil {
		t.Fatalf("could not save value in kv store: %v", err)
	}

	val, err := kv.Get(key)
	if err != nil {
		t.Errorf("get failed: %v", err)
	}
	if len(val) == 0 {
		t.Errorf("no value stored for key")
	}

	key2 := []byte{}
	val2, err := kv.Get(key2)
	if err == nil {
		t.Error("kv.Get for non-existent key should have returned an error")
	}
	if len(val2) != 0 {
		t.Errorf("non-existent key should not have a value. key=%v, value=%v", key2, val2)
	}
}

func Test_KVStoreDelete(t *testing.T) {
	kv := NewKVStore()
	key := []byte("ralph merkle")

	if err := kv.Put(key, []byte{1, 2, 3}); err != nil {
		t.Fatalf("could not save value in kv store: %v", err)
	}

	if err := kv.Delete(key); err != nil {
		t.Errorf("could not delete key: %v", key)
	}
}
