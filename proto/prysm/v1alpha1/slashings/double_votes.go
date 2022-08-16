package slashings

import "github.com/prysmaticlabs/prysm/v3/config/params"

// SigningRootsDiffer verifies that an incoming vs. existing attestation has a different signing root.
// If the existing signing root is empty, then we consider an attestation as different always.
func SigningRootsDiffer(existingSigningRoot, incomingSigningRoot [32]byte) bool {
	zeroHash := params.BeaconConfig().ZeroHash
	// If the existing signing root is empty, we always consider the incoming
	// attestation as a double vote to be safe.
	if existingSigningRoot == zeroHash {
		return true
	}
	// Otherwise, we consider any sort of inequality to be a double vote.
	return existingSigningRoot != incomingSigningRoot
}
