package kzg

import (
	"fmt"

	GoKZG "github.com/crate-crypto/go-kzg-4844"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

// IsDataAvailable checks that
// - all blobs in the block are available
// - Expected KZG commitments match the number of blobs in the block
// - That the number of proofs match the number of blobs
// - That the proofs are verified against the KZG commitments
func IsDataAvailable(commitments [][]byte, sidecars []*ethpb.DeprecatedBlobSidecar) error {
	if len(commitments) != len(sidecars) {
		return fmt.Errorf("could not check data availability, expected %d commitments, obtained %d",
			len(commitments), len(sidecars))
	}
	if len(commitments) == 0 {
		return nil
	}
	blobs := make([]GoKZG.Blob, len(commitments))
	proofs := make([]GoKZG.KZGProof, len(commitments))
	cmts := make([]GoKZG.KZGCommitment, len(commitments))
	for i, sidecar := range sidecars {
		blobs[i] = bytesToBlob(sidecar.Blob)
		proofs[i] = bytesToKZGProof(sidecar.KzgProof)
		cmts[i] = bytesToCommitment(commitments[i])
	}
	return kzgContext.VerifyBlobKZGProofBatch(blobs, cmts, proofs)
}

// Verify performs single or batch verification of commitments depending on the number of given BlobSidecars.
func Verify(sidecars ...blocks.ROBlob) error {
	if len(sidecars) == 0 {
		return nil
	}
	if len(sidecars) == 1 {
		return kzgContext.VerifyBlobKZGProof(
			bytesToBlob(sidecars[0].Blob),
			bytesToCommitment(sidecars[0].KzgCommitment),
			bytesToKZGProof(sidecars[0].KzgProof))
	}
	blobs := make([]GoKZG.Blob, len(sidecars))
	cmts := make([]GoKZG.KZGCommitment, len(sidecars))
	proofs := make([]GoKZG.KZGProof, len(sidecars))
	for i, sidecar := range sidecars {
		blobs[i] = bytesToBlob(sidecar.Blob)
		cmts[i] = bytesToCommitment(sidecar.KzgCommitment)
		proofs[i] = bytesToKZGProof(sidecar.KzgProof)
	}
	return kzgContext.VerifyBlobKZGProofBatch(blobs, cmts, proofs)
}

func bytesToBlob(blob []byte) (ret GoKZG.Blob) {
	copy(ret[:], blob)
	return
}

func bytesToCommitment(commitment []byte) (ret GoKZG.KZGCommitment) {
	copy(ret[:], commitment)
	return
}

func bytesToKZGProof(proof []byte) (ret GoKZG.KZGProof) {
	copy(ret[:], proof)
	return
}
