package das

import (
	"fmt"
	"strings"

	errors "github.com/pkg/errors"
)

var (
	errDAIncomplete  = errors.New("some BlobSidecars are not available at this time")
	errDAEquivocated = errors.New("cache contains BlobSidecars that do not match block commitments")
	errMixedRoots    = errors.New("BlobSidecars must all be for the same block")
)

// The following errors are exported so that gossip verification can use errors.Is to determine the correct pubsub.ValidationResult.
var (
	// ErrInvalidInclusionProof is returned when the inclusion proof check on the BlobSidecar fails.
	ErrInvalidInclusionProof = errors.New("BlobSidecar inclusion proof is invalid")
	// ErrInvalidBlockSignature is returned when the BlobSidecar.SignedBeaconBlockHeader signature cannot be verified against the block root.
	ErrInvalidBlockSignature = errors.New("SignedBeaconBlockHeader signature could not verified")
	// ErrInvalidCommitment is returned when the kzg_commitment cannot be verified against the kzg_proof and blob.
	ErrInvalidCommitment = errors.New("BlobSidecar.kzg_commitment verification failed")
)

// NewMissingIndicesError creates a MissingIndicesError, internally storing the list of
// missing indices so that they can be accessed with the .Missing method.
func NewMissingIndicesError(missing []uint64) MissingIndicesError {
	return MissingIndicesError{indices: missing}
}

// MissingIndicesError occurs when a da check is performend and not all indices are in the cache.
type MissingIndicesError struct {
	indices []uint64
}

var _ error = MissingIndicesError{}

// Error satisfies the stdlib error interface.
func (m MissingIndicesError) Error() string {
	is := make([]string, 0, len(m.indices))
	for i := range m.indices {
		is = append(is, fmt.Sprintf("%d", m.indices[i]))
	}
	return fmt.Sprintf("%s at indices %s", errDAIncomplete.Error(), strings.Join(is, ","))
}

// Missing allows the recipient of an error to assert it to the MissingIndicesError type
// and obtain the list of exactly which indices are missing.
func (m MissingIndicesError) Missing() []uint64 {
	return m.indices
}

// Unwrap is defined to support unwrapping the error, ie for errors.Is.
func (MissingIndicesError) Unwrap() error {
	return errDAIncomplete
}

// NewCommitmentMismatchError creates a CommitmentMismatchError, internally storing the
// list of mismatched indices for introspection by the caller.
func NewCommitmentMismatchError(mismatch []uint64) CommitmentMismatchError {
	return CommitmentMismatchError{mismatch: mismatch}
}

// CommitmentMismatchError occurs when an AvailabilityStore caches a sidecar for an index that
// does not match the commitments seen in the block during the da check.
type CommitmentMismatchError struct {
	mismatch []uint64
}

var _ error = CommitmentMismatchError{}

// Error satisfies the stdlib error interface.
func (m CommitmentMismatchError) Error() string {
	is := make([]string, 0, len(m.mismatch))
	for i := range m.mismatch {
		is = append(is, fmt.Sprintf("%d", m.mismatch[i]))
	}
	return fmt.Sprintf("%s at indices %s", errDAEquivocated.Error(), strings.Join(is, ","))
}

// Mismatch allows the recipient of an error to assert it to the CommitmentMismatchError type
// and obtain the list of exactly which indices did not match.
func (m CommitmentMismatchError) Mismatch() []uint64 {
	return m.mismatch
}

// Unwrap is defined to support unwrapping the error, ie for errors.Is.
func (CommitmentMismatchError) Unwrap() error {
	return errDAEquivocated
}
