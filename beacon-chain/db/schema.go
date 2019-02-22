package db

import "github.com/prysmaticlabs/prysm/shared/bytesutil"

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
	attestationBucket     = []byte("attestation-bucket")
	blockOperationsBucket = []byte("block-operations-bucket")
	blockBucket           = []byte("block-bucket")
	mainChainBucket       = []byte("main-chain-bucket")
	chainInfoBucket       = []byte("chain-info")

	mainChainHeightKey = []byte("chain-height")
	stateLookupKey     = []byte("state")

	// DB internal use
	cleanupHistoryBucket    = []byte("cleanup-history-bucket")
	cleanedFinalizedSlotKey = []byte("cleaned-finalized-slot")
)

// encodeSlotNumber encodes a slot number as little-endian uint32.
func encodeSlotNumber(number uint64) []byte {
	return bytesutil.Bytes8(number)
}

// decodeSlotNumber returns a slot number which has been
// encoded as a little-endian uint32 in the byte array.
func decodeToSlotNumber(bytearray []byte) uint64 {
	return bytesutil.FromBytes8(bytearray)
}
