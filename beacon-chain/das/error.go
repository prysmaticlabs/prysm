package das

import (
	"fmt"
	"strings"

	errors "github.com/pkg/errors"
)

var (
	errDAIncomplete  = errors.New("some BlobSidecars are not available at this time")
	errDAEquivocated = errors.New("cache contains BlobSidecars that do not match block commitments")
)

func NewMissingIndicesError(missing []uint64) *MissingIndicesError {
	return &MissingIndicesError{indices: missing}
}

type MissingIndicesError struct {
	indices []uint64
}

func (m *MissingIndicesError) Error() string {
	is := make([]string, 0, len(m.indices))
	for i := range m.indices {
		is = append(is, fmt.Sprintf("%d", m.indices[i]))
	}
	return fmt.Sprintf("%s at indices %s", errDAIncomplete.Error(), strings.Join(is, ","))
}

func (m *MissingIndicesError) Missing() []uint64 {
	return m.indices
}

func (m *MissingIndicesError) Unwrap() error {
	return errDAIncomplete
}

var _ error = &MissingIndicesError{}

func NewCommitmentMismatchError(mismatch []uint64) *CommitmentMismatchError {
	return &CommitmentMismatchError{mismatch: mismatch}
}

type CommitmentMismatchError struct {
	mismatch []uint64
}

func (m *CommitmentMismatchError) Error() string {
	is := make([]string, 0, len(m.mismatch))
	for i := range m.mismatch {
		is = append(is, fmt.Sprintf("%d", m.mismatch[i]))
	}
	return fmt.Sprintf("%s at indices %s", errDAEquivocated.Error(), strings.Join(is, ","))
}

func (m *CommitmentMismatchError) Mismatch() []uint64 {
	return m.mismatch
}

func (m *CommitmentMismatchError) Unwrap() error {
	return errDAEquivocated
}
