package blockchain

import (
	"encoding/binary"
)

// The Schema will define how to store and retrieve data from the db.
// Currently we store blocks by prefixing `block` to their hash and
// using that as the key to store blocks.
// `block` + hash -> block
//
// We store the crystallized state using the crystallized state lookup key, and
// also the genesis block using the genesis lookup key.
// The canonical head is stored using the canonical head lookup key.

// The fields below define the prefixing of keys in the db.
var (

	// CanonicalHeadLookupKey tracks the latest canonical head.
	canonicalHeadLookupKey = []byte("latest-canonical-head")

	// ActiveStateLookupKey tracks the current active state.
	activeStateLookupKey = []byte("beacon-active-state")

	// CrystallizedStateLookupKey tracks the current crystallized state.
	crystallizedStateLookupKey = []byte("beacon-crystallized-state")

	// GenesisLookupKey tracks the genesis block.
	genesisLookupKey = []byte("genesis")

	// Data item prefixes.
	blockPrefix = []byte("block-") // blockPrefix + blockhash -> block

	canonicalPrefix = []byte("canonical-") // canonicalPrefix + num(uint64 big endian) -> blockhash

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

// canonicalBlockKey = canonicalPrefix + num(uint64 big endian)
func canonicalBlockKey(slotnumber uint64) []byte {
	return append(canonicalPrefix, encodeSlotNumber(slotnumber)...)
}
