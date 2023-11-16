package verification

import "github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"

// sharedResources provides access to resources that are required by different verification types.
// for example, sidecar verifcation and block verification share the block signature verification cache.
type sharedResources struct{}

// Initializer is used to create different Verifiers.
// Verifiers require access to stateful data structures, like caches,
// and it is Initializer's job to provides access to those.
type Initializer struct {
	shared *sharedResources
}

// NewBlobVerifier creates a BlobVerifier for a single blob, with the given set of requirements.
func (ini *Initializer) NewBlobVerifier(b blocks.ROBlob, reqs ...Requirement) *BlobVerifier {
	return &BlobVerifier{shared: ini.shared, results: newResults(reqs...)}
}
