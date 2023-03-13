package cache

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

var ErrNoBlockFound = errors.New("no block found")

type BlockCache struct {
	sync.RWMutex
	cache map[[32]byte]*ethpb.BeaconBlockCapella
}

func NewBlockCache() *BlockCache {
	return &BlockCache{
		cache: make(map[[32]byte]*ethpb.BeaconBlockCapella),
	}
}

func (p *BlockCache) Get(key [32]byte) (*ethpb.BeaconBlockCapella, error) {
	p.RLock()
	defer p.RUnlock()
	if b, ok := p.cache[key]; ok {
		return b, nil
	}

	return nil, ErrNoBlockFound
}

func (p *BlockCache) Set(b *ethpb.BeaconBlockCapella) {
	p.Lock()
	defer p.Unlock()
	h := b.Body.ExecutionPayload.BlockHash
	p.cache[bytesutil.ToBytes32(h)] = b
}

// Delete Blocks older than slot
func (p *BlockCache) Delete(slot uint64) {
	p.Lock()
	defer p.Unlock()

	for _, b := range p.cache {
		if slot >= b.Body.ExecutionPayload.Timestamp/params.BeaconConfig().SecondsPerSlot {
			delete(p.cache, bytesutil.ToBytes32(b.Body.ExecutionPayload.BlockHash))
		}
	}
}
