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

// The fields below define the suffix of keys in the db.
var (
	// CanonicalHeadLookupKey tracks the latest canonical head.
	canonicalHeadLookupKey = []byte("latest-canonical-head")

	// ActiveStateLookupKey tracks the current active state.
	activeStateLookupKey = []byte("beacon-active-state")

	// CrystallizedStateLookupKey tracks the current crystallized state.
	crystallizedStateLookupKey = []byte("beacon-crystallized-state")

	// GenesisLookupKey tracks the genesis block.
	genesisLookupKey = []byte("genesis")

	// Data item suffixes.
	// TODO(#514): Change suffixes back to prefixes as originally designed after issue 514 is solved or after BoltDB migration
	blockSuffix             = []byte("-block")             // blockhash + blockPrefix -> block
	canonicalSuffix         = []byte("-canonical")         // num(uint64 big endian) + cannoicalSuffix -> blockhash
	attestationSuffix       = []byte("-attestation")       // attestationHash + attestationSuffix -> attestation
	attestationHashesSuffix = []byte("-attestationHashes") // blockHash + attestationHashesPrefix -> attestationHashes
)

// encodeSlotNumber encodes a slot number as big endian uint64.
func encodeSlotNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// blockKey = blockPrefix + blockHash.
func blockKey(hash [32]byte) []byte {
	return append(hash[:], blockSuffix...)
}

// canonicalBlockKey = canonicalPrefix + num(uint64 big endian)
func canonicalBlockKey(slotnumber uint64) []byte {
	return append(encodeSlotNumber(slotnumber)[:], canonicalSuffix...)
}

// AttestationKey = attestationPrefix + attestationHash.
func AttestationKey(hash [32]byte) []byte {
	return append(hash[:], attestationSuffix...)
}

// AttestationHashListKey = attestationHashesPrefix + blockHash.
func AttestationHashListKey(hash [32]byte) []byte {
	return append(hash[:], attestationHashesSuffix...)
}
