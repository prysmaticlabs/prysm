package sharding

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type shardKV struct {
	kv map[common.Hash][]byte
}

func (sb *shardKV) Get(k common.Hash) ([]byte, error) {
	v := sb.kv[k]
	if v == nil {
		return nil, fmt.Errorf("Key Not Found")
	}
	return v, nil
}

func (sb *shardKV) Put(k common.Hash, v []byte) {
	sb.kv[k] = v
	return
}

func (sb *shardKV) Delete(k common.Hash) {
	delete(sb.kv, k)
	return
}
