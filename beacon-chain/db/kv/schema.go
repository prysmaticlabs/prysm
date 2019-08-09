package kv

// The schema will define how to store and retrieve data from the db.
// we can prefix or suffix certain values such as `block` with attributes
// for prefix-wide scans across the underlying BoltDB buckets
// using that as the key to store blocks.
//
// `block` + hash -> block
var (
	attestationsBucket = []byte("attestations")
	blocksBucket       = []byte("blocks")
	validatorsBucket   = []byte("validators")
	stateBucket        = []byte("state")
)
