package cache

import (
	"sync"

	types "github.com/prysmaticlabs/eth2-types"
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
func (f *ProposerPayloadIDsCache) GetProposerPayloadIDs(slot types.Slot) (types.ValidatorIndex, uint64, bool) {
	f.RLock()
	defer f.RUnlock()
	ids, ok := f.slotToProposerAndPayloadIDs[slot]
	if !ok {
		return 0, 0, false
	}
	vId := ids[:vIdLength]
	pId := ids[vIdLength:]
	return types.ValidatorIndex(bytesutil.BytesToUint64BigEndian(vId)), bytesutil.BytesToUint64BigEndian(pId), true
}

// SetProposerAndPayloadIDs sets the proposer and payload IDs for the given slot.
func (f *ProposerPayloadIDsCache) SetProposerAndPayloadIDs(slot types.Slot, vId types.ValidatorIndex, pId uint64) {
	f.Lock()
	defer f.Unlock()
	var vIdBytes [vIdLength]byte
	copy(vIdBytes[:], bytesutil.Uint64ToBytesBigEndian(uint64(vId)))
	var pIdBytes [pIdLength]byte
	copy(pIdBytes[:], bytesutil.Uint64ToBytesBigEndian(pId))

	var bytes [vpIdsLength]byte
	copy(bytes[:], append(vIdBytes[:], pIdBytes[:]...))

	_, ok := f.slotToProposerAndPayloadIDs[slot]
	// Ok to overwrite if the slot is already set but the payload ID is not set.
	// This combats the re-org case where payload assignment could change the epoch of.
	if !ok || (ok && pId != 0) {
		f.slotToProposerAndPayloadIDs[slot] = bytes
	}
}
