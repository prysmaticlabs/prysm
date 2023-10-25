package kzg

import (
	"fmt"
	"strings"

	GoKZG "github.com/crate-crypto/go-kzg-4844"
	"github.com/pkg/errors"
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

var ErrKzgProofFailed = errors.New("failed to prove commitment to BlobSidecar Blob data")

type KzgProofError struct {
	failed [][48]byte
}

func NewKzgProofError(failed [][48]byte) *KzgProofError {
	return &KzgProofError{failed: failed}
}

func (e *KzgProofError) Error() string {
	cmts := make([]string, len(e.failed))
	for i := range e.failed {
		cmts[i] = fmt.Sprintf("%#x", e.failed[i])
	}
	return fmt.Sprintf("%s: bad commitments=%s", ErrKzgProofFailed.Error(), strings.Join(cmts, ","))
}

func (e *KzgProofError) Failed() [][48]byte {
	return e.failed
}

func (e *KzgProofError) Unwrap() error {
	return ErrKzgProofFailed
}

// BisectBlobSidecarKzgProofs tries to batch prove the given sidecars against their own specified commitment.
// The caller is responsible for ensuring that the commitments match those specified by the block.
// If the batch fails, it will then try to verify the proofs one-by-one.
// If an error is returned, it will be a custom error of type KzgProofError that provides access
// to the list of commitments that failed.
func BisectBlobSidecarKzgProofs(sidecars []*ethpb.BlobSidecar) error {
	if len(sidecars) == 0 {
		return nil
	}
	blobs := make([]GoKZG.Blob, len(sidecars))
	cmts := make([]GoKZG.KZGCommitment, len(sidecars))
	proofs := make([]GoKZG.KZGProof, len(sidecars))
	for i, sidecar := range sidecars {
		blobs[i] = bytesToBlob(sidecar.Blob)
		cmts[i] = bytesToCommitment(sidecar.KzgCommitment)
		proofs[i] = bytesToKZGProof(sidecar.KzgProof)
	}
	if err := kzgContext.VerifyBlobKZGProofBatch(blobs, cmts, proofs); err == nil {
		return nil
	}
	failed := make([][48]byte, 0, len(blobs))
	for i := range blobs {
		if err := kzgContext.VerifyBlobKZGProof(blobs[i], cmts[i], proofs[i]); err != nil {
			failed = append(failed, cmts[i])
		}
	}
	return NewKzgProofError(failed)
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
