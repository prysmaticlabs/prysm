package blockchain

import (
	"encoding/binary"
)

// The fields below define the prefixing of keys in the db.
var (

	// CanonicalHeadLookupKey tracks the latest canonical head.
	CanonicalHeadLookupKey = []byte("latest-canonical-head")

	// ActiveStateLookupKey tracks the current active state.
	ActiveStateLookupKey = []byte("beacon-active-state")

	// CrystallizedStateLookupKey tracks the current crystallized state.
	CrystallizedStateLookupKey = []byte("beacon-crystallized-state")

	// GenesisLookupKey tracks the genesis block.
	GenesisLookupKey = []byte("genesis")

	// BlockSlotRegistryPrefix tracks the registry of stored blockhashes.
	blockSlotRegistryPrefix = []byte("block-registry")

	// Data item prefixes.
	blockPrefix = []byte("b") // blockPrefix + num(uint64 big endian) + blockhash -> block

)

// encodeSlotNumber encodes a slot number as big endian uint64.
func encodeSlotNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// blockKey = blockPrefix + hash.
func blockKey(hash [32]byte) []byte {
	return append(blockPrefix, hash[:]...)
}
