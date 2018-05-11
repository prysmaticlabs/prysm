package database

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func Test_ShardKVGet(t *testing.T) {
	kv := MakeShardKV()
	hash := common.StringToHash("ralph merkle")
	kv.Put(hash, []byte{1, 2, 3})

	val, err := kv.Get(hash)
	if err != nil {
		t.Errorf("get failed: %v", err)
	}
	if val == nil {
		t.Errorf("no value stored for key")
	}
}
