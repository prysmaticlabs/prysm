package cache

import (
	"bytes"
	"sync"

	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
)

const vIdLength = 8
const pIdLength = 8
const vpIdsLength = vIdLength + pIdLength

// ProposerPayloadIDsCache is a cache of proposer payload IDs.
// The key is the slot. The value is the concatenation of the proposer and payload IDs. 8 bytes each.
type ProposerPayloadIDsCache struct {
	slotToProposerAndPayloadIDs map[types.Slot][vpIdsLength]byte
	sync.RWMutex
}

// NewProposerPayloadIDsCache creates a new proposer payload IDs cache.
func NewProposerPayloadIDsCache() *ProposerPayloadIDsCache {
	return &ProposerPayloadIDsCache{
		slotToProposerAndPayloadIDs: make(map[types.Slot][vpIdsLength]byte),
	}
}

// GetProposerPayloadIDs returns the proposer and  payload IDs for the given slot.
func (f *ProposerPayloadIDsCache) GetProposerPayloadIDs(slot types.Slot) (types.ValidatorIndex, [8]byte, bool) {
	f.RLock()
	defer f.RUnlock()
	ids, ok := f.slotToProposerAndPayloadIDs[slot]
	if !ok {
		return 0, [8]byte{}, false
	}
	vId := ids[:vIdLength]

	b := ids[vIdLength:]
	var pId [pIdLength]byte
	copy(pId[:], b)

	return types.ValidatorIndex(bytesutil.BytesToUint64BigEndian(vId)), pId, true
}

// SetProposerAndPayloadIDs sets the proposer and payload IDs for the given slot.
func (f *ProposerPayloadIDsCache) SetProposerAndPayloadIDs(slot types.Slot, vId types.ValidatorIndex, pId [8]byte) {
	f.Lock()
	defer f.Unlock()
	var vIdBytes [vIdLength]byte
	copy(vIdBytes[:], bytesutil.Uint64ToBytesBigEndian(uint64(vId)))

	var bs [vpIdsLength]byte
	copy(bs[:], append(vIdBytes[:], pId[:]...))

	ids, ok := f.slotToProposerAndPayloadIDs[slot]
	// Ok to overwrite if the slot is already set but the cached payload ID is not set.
	// This combats the re-org case where payload assignment could change at the start of the epoch.
	byte8 := [vIdLength]byte{}
	if !ok || (ok && bytes.Equal(ids[vIdLength:], byte8[:])) {
		f.slotToProposerAndPayloadIDs[slot] = bs
	}
}

// PrunePayloadIDs removes the payload id entries that's current than input slot.
func (f *ProposerPayloadIDsCache) PrunePayloadIDs(slot types.Slot) {
	f.Lock()
	defer f.Unlock()

	for s := range f.slotToProposerAndPayloadIDs {
		if slot > s {
			delete(f.slotToProposerAndPayloadIDs, s)
		}
	}
}
