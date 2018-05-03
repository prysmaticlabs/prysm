package sharding

import "github.com/ethereum/go-ethereum/common"

type shardBackend struct {
	kv map[*common.Hash][]byte
}

func (sb *shardBackend) Get(k *common.Hash) []byte {
	return sb.kv[k]
}

func (sb *shardBackend) Put(k *common.Hash, v []byte) {
	sb.kv[k] = v
	return
}

func (sb *shardBackend) Delete(k *common.Hash) {
	delete(sb.kv, k)
	return
}
