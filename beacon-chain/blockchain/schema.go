package blockchain

import (
	"encoding/binary"
)

// The fields below define the prefixing of keys in the db.
var (
	// headBlockKey tracks the latest know full block's hash.
	headBlockKey = []byte("LastBlock")

	// canonicalHeadLookupKey tracks the latest canonical head.
	canonicalHeadLookupKey = []byte("latest-canonical-head")

	// activeStateLookupKey tracks the current active state.
	activeStateLookupKey = []byte("beacon-active-state")

	// crystallizedStateLookupKey tracks the current crystallized state.
	crystallizedStateLookupKey = []byte("beacon-crystallized-state")

	// genesisLookupKey tracks the genesis block.
	genesisLookupKey = []byte("genesis")

	//Data item prefixes.
	blockPrefix = []byte("b") // blockPrefix + num(uint64 big endian) + blockhash -> block

)

// BlockLookupEntry is metadata that allows blocks to be retrieved using their hash and slotnumber.
type BlockLookupEntry struct {
	BlockHash  [32]byte
	SlotNumber uint64
}

// encodeSlotNumber encodes a slot number as big endian uint64.
func encodeSlotNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// blockKey = blockPrefix + num (uint64 big endian) + hash.
func blockKey(slotnumber uint64, hash [32]byte) []byte {
	return append(append(blockPrefix, encodeSlotNumber(slotnumber)...), hash[:]...)
}

// activeStateKey = activeStateLookupKey + hash.
func activeStateKey(hash [32]byte) []byte {
	return append(activeStateLookupKey, hash[:]...)
}

// crystallizedStateKey = crytsallizedStateLookupKey + hash.
func crystallizedStateKey(hash [32]byte) []byte {
	return append(crystallizedStateLookupKey, hash[:]...)
}

// CanonicalHeadKey = canonicalHeadLookupKey.
func CanonicalHeadKey() []byte {
	return canonicalHeadLookupKey
}

// CurrentActiveStateKey = activeStateLookupKey.
func CurrentActiveStateKey() []byte {
	return activeStateLookupKey
}

// CurrentCrystallizedStateKey = crystallizedStateLookupKey.
func CurrentCrystallizedStateKey() []byte {
	return crystallizedStateLookupKey
}

// GenesisKey = genesisLookUpKey.
func GenesisKey() []byte {
	return genesisLookupKey
}
