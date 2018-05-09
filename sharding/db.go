package sharding

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type shardKV struct {
	// Shard state storage is a mapping of hashes to RLP encoded values.
	kv map[common.Hash][]byte
}

func makeShardKV() *shardKV {
	return &shardKV{kv: make(map[common.Hash][]byte)}
}

func (sb *shardKV) Get(k common.Hash) ([]byte, error) {
	v, ok := sb.kv[k]
	if !ok {
		return nil, fmt.Errorf("key not found: %v", k)
	}
	return v, nil
}

func (sb *shardKV) Has(k common.Hash) bool {
	v := sb.kv[k]
	return v != nil
}

func (sb *shardKV) Put(k common.Hash, v []byte) {
	sb.kv[k] = v
}

func (sb *shardKV) Delete(k common.Hash) {
	delete(sb.kv, k)
}
