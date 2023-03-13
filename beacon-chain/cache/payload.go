package cache

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
)

var ErrNoPayloadFound = errors.New("no payload found")

type PayloadCache struct {
	sync.RWMutex
	cache map[[32]byte]interfaces.ExecutionData
}

func NewPayloadCache() *PayloadCache {
	return &PayloadCache{
		cache: make(map[[32]byte]interfaces.ExecutionData),
	}
}

func (p *PayloadCache) Get(key [32]byte) (interfaces.ExecutionData, error) {
	p.RLock()
	defer p.RUnlock()
	if payload, ok := p.cache[key]; ok {
		return payload, nil
	}

	return nil, ErrNoPayloadFound
}

func (p *PayloadCache) Set(payload interfaces.ExecutionData) {
	p.Lock()
	defer p.Unlock()
	h := payload.BlockHash()
	p.cache[bytesutil.ToBytes32(h)] = payload
}

// Delete payloads older than slot
func (p *PayloadCache) Delete(slot uint64) {
	p.Lock()
	defer p.Unlock()

	for _, data := range p.cache {
		if slot >= data.Timestamp()/params.BeaconConfig().SecondsPerSlot {
			delete(p.cache, bytesutil.ToBytes32(data.BlockHash()))
		}
	}
}
