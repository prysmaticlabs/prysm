package database

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/sharding"
)

// Verifies that ShardKV implements the ShardBackend interface.
var _ = sharding.ShardBackend(&ShardKV{})

func Test_ShardKVPut(t *testing.T) {
	kv := NewShardKV()
	hash := common.StringToHash("ralph merkle")

	if err := kv.Put(hash, []byte{1, 2, 3}); err != nil {
		t.Errorf("could not save value in kv store: %v", err)
	}
}

func Test_ShardKVHas(t *testing.T) {
	kv := NewShardKV()
	hash := common.StringToHash("ralph merkle")

	if err := kv.Put(hash, []byte{1, 2, 3}); err != nil {
		t.Fatalf("could not save value in kv store: %v", err)
	}

	if !kv.Has(hash) {
		t.Errorf("kv store does not have hash: %v", hash)
	}

	hash2 := common.StringToHash("")
	if kv.Has(hash2) {
		t.Errorf("kv store should not contain unset key: %v", hash2)
	}
}

func Test_ShardKVGet(t *testing.T) {
	kv := NewShardKV()
	hash := common.StringToHash("ralph merkle")

	if err := kv.Put(hash, []byte{1, 2, 3}); err != nil {
		t.Fatalf("could not save value in kv store: %v", err)
	}

	val, err := kv.Get(hash)
	if err != nil {
		t.Errorf("get failed: %v", err)
	}
	if val == nil {
		t.Errorf("no value stored for key")
	}

	hash2 := common.StringToHash("")
	val2, err := kv.Get(hash2)
	if val2 != nil {
		t.Errorf("non-existent key should not have a value. key=%v, value=%v", hash2, val2)
	}
}

func Test_ShardKVDelete(t *testing.T) {
	kv := NewShardKV()
	hash := common.StringToHash("ralph merkle")

	if err := kv.Put(hash, []byte{1, 2, 3}); err != nil {
		t.Fatalf("could not save value in kv store: %v", err)
	}

	if err := kv.Delete(hash); err != nil {
		t.Errorf("could not delete key: %v", hash)
	}
}
