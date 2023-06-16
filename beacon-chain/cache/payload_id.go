package cache

import (
	"bytes"
	"sync"

	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
)

const keyLength = 40
const vIdLength = 8
const pIdLength = 8
const vpIdsLength = vIdLength + pIdLength

// ProposerPayloadIDsCache is a cache of proposer payload IDs.
// The key is the concatenation of the slot and the block root.
// The value is the concatenation of the proposer and payload IDs, 8 bytes each.
type ProposerPayloadIDsCache struct {
	slotToProposerAndPayloadIDs map[[keyLength]byte][vpIdsLength]byte
	sync.RWMutex
}

// NewProposerPayloadIDsCache creates a new proposer payload IDs cache.
func NewProposerPayloadIDsCache() *ProposerPayloadIDsCache {
	return &ProposerPayloadIDsCache{
		slotToProposerAndPayloadIDs: make(map[[keyLength]byte][vpIdsLength]byte),
	}
}

// GetProposerPayloadIDs returns the proposer and payload IDs for the given slot and head root to build the block.
func (f *ProposerPayloadIDsCache) GetProposerPayloadIDs(
	slot primitives.Slot,
	r [fieldparams.RootLength]byte,
) (primitives.ValidatorIndex, [pIdLength]byte, bool) {
	f.RLock()
	defer f.RUnlock()
	ids, ok := f.slotToProposerAndPayloadIDs[idKey(slot, r)]
	if !ok {
		return 0, [pIdLength]byte{}, false
	}
	vId := ids[:vIdLength]

	b := ids[vIdLength:]
	var pId [pIdLength]byte
	copy(pId[:], b)

	return primitives.ValidatorIndex(bytesutil.BytesToUint64BigEndian(vId)), pId, true
}

// SetProposerAndPayloadIDs sets the proposer and payload IDs for the given slot and head root to build block.
func (f *ProposerPayloadIDsCache) SetProposerAndPayloadIDs(
	slot primitives.Slot,
	vId primitives.ValidatorIndex,
	pId [pIdLength]byte,
	r [fieldparams.RootLength]byte,
) {
	f.Lock()
	defer f.Unlock()
	var vIdBytes [vIdLength]byte
	copy(vIdBytes[:], bytesutil.Uint64ToBytesBigEndian(uint64(vId)))

	var bs [vpIdsLength]byte
	copy(bs[:], append(vIdBytes[:], pId[:]...))

	k := idKey(slot, r)
	ids, ok := f.slotToProposerAndPayloadIDs[k]
	// Ok to overwrite if the slot is already set but the cached payload ID is not set.
	// This combats the re-org case where payload assignment could change at the start of the epoch.
	var byte8 [vIdLength]byte
	if !ok || (ok && bytes.Equal(ids[vIdLength:], byte8[:])) {
		f.slotToProposerAndPayloadIDs[k] = bs
	}
}

// PrunePayloadIDs removes the payload ID entries older than input slot.
func (f *ProposerPayloadIDsCache) PrunePayloadIDs(slot primitives.Slot) {
	f.Lock()
	defer f.Unlock()

	for k := range f.slotToProposerAndPayloadIDs {
		s := primitives.Slot(bytesutil.BytesToUint64BigEndian(k[:8]))
		if slot > s {
			delete(f.slotToProposerAndPayloadIDs, k)
		}
	}
}

func idKey(slot primitives.Slot, r [fieldparams.RootLength]byte) [keyLength]byte {
	var k [keyLength]byte
	copy(k[:], append(bytesutil.Uint64ToBytesBigEndian(uint64(slot)), r[:]...))
	return k
}
