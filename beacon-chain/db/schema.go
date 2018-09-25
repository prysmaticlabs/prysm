package db

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

	// aStateLookupKey tracks the current active state.
	aStateLookupKey = []byte("beacon-active-state")

	// cStateLookupKey tracks the current crystallized state.
	cStateLookupKey = []byte("beacon-crystallized-state")

	// mainChainHeightKey tracks the height of the current chain.
	mainChainHeightKey = []byte("chain-height")

	lastSimulatedBlockKey = []byte("last-simulated-block")

	// blockBucket contains blocks by hash.
	blockBucket = []byte("block-bucket")

	// attestationBucket contains attestations by hash.
	attestationBucket = []byte("attestation-bucket")

	// mainChainBucket contains hashes of blocks in the main chain by slot.
	mainChainBucket = []byte("main-chain-bucket")

	// chainInfoBucket contains metadata regarding the chain such as the current height and state
	chainInfoBucket = []byte("chain-info")
)

// encodeSlotNumber encodes a slot number as big endian uint64.
func encodeSlotNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

func decodeSlotNumber(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}
