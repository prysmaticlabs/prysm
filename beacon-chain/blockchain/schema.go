package blockchain

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
	CanonicalHeadLookupKey = []byte("latest-canonical-head")

	// ActiveStateLookupKey tracks the current active state.
	ActiveStateLookupKey = []byte("beacon-active-state")

	// CrystallizedStateLookupKey tracks the current crystallized state.
	CrystallizedStateLookupKey = []byte("beacon-crystallized-state")

	// GenesisLookupKey tracks the genesis block.
	GenesisLookupKey = []byte("genesis")

	// Data item prefixes.
	blockPrefix = []byte("block-") // blockPrefix + blockhash -> block

)

// blockKey = blockPrefix + hash.
func blockKey(hash [32]byte) []byte {
	return append(blockPrefix, hash[:]...)
}
