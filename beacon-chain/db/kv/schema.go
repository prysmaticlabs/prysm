package kv

// The schema will define how to store and retrieve data from the db.
// we can prefix or suffix certain values such as `block` with attributes
// for prefix-wide scans across the underlying BoltDB buckets when filtering data.
// For example, we might store attestations as shard + attestation_root -> attestation, making
// it easy to scan for keys that have a certain shard number as a prefix and return those
// corresponding attestations.
var (
	attestationsBucket       = []byte("attestations")
	attestationIndicesBucket = []byte("attestation-indices")
	blocksBucket             = []byte("blocks")
	blockIndicesBucket       = []byte("block-indices")
	validatorsBucket         = []byte("validators")
	stateBucket              = []byte("state")

	// Key indices buckets.
	parentRootIndicesBucket = []byte("parent-root-indices")
	blockSlotIndicesBucket  = []byte("block-slot-indices")
	shardIndicesBucket      = []byte("shard-indices")
	slotIndicesBucket       = []byte("attestation-slot-indices")
	startEpochIndicesBucket = []byte("start-epoch-indices")
	endEpochIndicesBucket   = []byte("end-epoch-indices")

	// Block keys.
	headBlockRootKey = []byte("head-root")
)
