package operations

import (
	"sync"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/params"
)

type recentAttestationMultiMap struct {
	lock           sync.RWMutex
	slotRootMap    map[uint64][32]byte
	rootBitlistMap map[[32]byte]bitfield.Bitlist
}

func newRecentAttestationMultiMap() *recentAttestationMultiMap {
	return &recentAttestationMultiMap{
		slotRootMap:    make(map[uint64][32]byte),
		rootBitlistMap: make(map[[32]byte]bitfield.Bitlist),
	}
}

// Prune removes expired attestation references from the map.
func (r *recentAttestationMultiMap) Prune(currentSlot uint64) {
	r.lock.Lock()
	defer r.lock.Unlock()
	for slot, root := range r.slotRootMap {
		// Block expiration period is slots_per_epoch, we'll keep references to attestations within
		// twice that range to act as a short circuit for incoming attestations that may have been
		// delayed in the network.
		if slot+(2*params.BeaconConfig().SlotsPerEpoch)+1 < currentSlot {
			delete(r.slotRootMap, slot)
			delete(r.rootBitlistMap, root)
		}
	}
}

func (r *recentAttestationMultiMap) Insert(slot uint64, root [32]byte, bitlist bitfield.Bitlist) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.slotRootMap[slot] = root
	if b, exists := r.rootBitlistMap[root]; exists {
		r.rootBitlistMap[root] = b.Or(bitlist)
	} else {
		r.rootBitlistMap[root] = bitlist
	}
}

func (r *recentAttestationMultiMap) Contains(root [32]byte, b bitfield.Bitlist) bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	a, ok := r.rootBitlistMap[root]
	if !ok {
		return false
	}
	return a.Contains(b)
}
