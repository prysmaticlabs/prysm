package kv

// The schema will define how to store and retrieve data from the db.
// we can prefix or suffix certain values such as `block` with attributes
// for prefix-wide scans across the underlying BoltDB buckets
// using that as the key to store blocks.
//
// `block` + hash -> block
var (
	attestationBucket       = []byte("attestation-bucket")
	attestationTargetBucket = []byte("attestation-target-bucket")
	blockOperationsBucket   = []byte("block-operations-bucket")
	blockBucket             = []byte("block-bucket")
	mainChainBucket         = []byte("main-chain-bucket")
	histStateBucket         = []byte("historical-state-bucket")
	chainInfoBucket         = []byte("chain-info")
	validatorBucket         = []byte("validator")

	mainChainHeightKey      = []byte("chain-height")
	canonicalHeadKey        = []byte("canonical-head")
	stateLookupKey          = []byte("state")
	finalizedStateLookupKey = []byte("finalized-state")
	justifiedStateLookupKey = []byte("justified-state")
	finalizedBlockLookupKey = []byte("finalized-block")
	justifiedBlockLookupKey = []byte("justified-block")

	// DB internal use
	cleanupHistoryBucket = []byte("cleanup-history-bucket")
)
