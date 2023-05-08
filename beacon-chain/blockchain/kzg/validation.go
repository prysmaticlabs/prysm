package kzg

import (
	"fmt"

	GoKZG "github.com/crate-crypto/go-kzg-4844"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

// IsDataAvailable checks that
// - all blobs in the block are available
// - Expected KZG commitments match the number of blobs in the block
// - That the number of proofs match the number of blobs
// - That the proofs are verified against the KZG commitments
func IsDataAvailable(commitments [][]byte, sidecars []*ethpb.BlobSidecar) error {
	if len(commitments) != len(sidecars) {
		return fmt.Errorf("could not check data availability, expected %d commitments, obtained %d",
			len(commitments), len(sidecars))
	}
	blobs := make([]GoKZG.Blob, len(commitments))
	proofs := make([]GoKZG.KZGProof, len(commitments))
	cmts := make([]GoKZG.KZGCommitment, len(commitments))
	for i, sidecar := range sidecars {
		blobs[i] = GoKZG.Blob(sidecar.Blob)
		proofs[i] = GoKZG.KZGProof(sidecar.KzgProof)
		cmts[i] = GoKZG.KZGCommitment(commitments[i])
	}
	return kzgContext.VerifyBlobKZGProofBatch(blobs, cmts, proofs)
}
