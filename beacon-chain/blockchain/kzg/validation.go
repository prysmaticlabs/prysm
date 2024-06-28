package kzg

import (
	GoKZG "github.com/crate-crypto/go-kzg-4844"
	ckzg4844 "github.com/ethereum/c-kzg-4844/bindings/go"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
)

// CellsAndProofs represents the Cells and Proofs corresponding to
// a single blob.
type CellsAndProofs struct {
	Cells  [ckzg4844.CellsPerExtBlob]ckzg4844.Cell
	Proofs [ckzg4844.CellsPerExtBlob]ckzg4844.KZGProof
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
