package verification

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
)

// ErrMissingVerification indicates that the given verification function was never performed on the value.
var ErrMissingVerification = errors.New("verification was not performed for requirement")

// VerificationMultiError is a custom error that can be used to access individual verification failures.
type VerificationMultiError struct {
	r   *results
	err error
}

// Unwrap is used by errors.Is to unwrap errors.
func (ve VerificationMultiError) Unwrap() error {
	if ve.err == nil {
		return nil
	}
	return ve.err
}

// Error satisfies the standard error interface.
func (ve VerificationMultiError) Error() string {
	if ve.err == nil {
		return ""
	}
	return ve.err.Error()
}

// Failures provides access to map of Requirements->error messages
// so that calling code can introspect on what went wrong.
func (ve VerificationMultiError) Failures() map[Requirement]error {
	return ve.r.failures()
}

func newVerificationMultiError(r *results, err error) VerificationMultiError {
	return VerificationMultiError{r: r, err: err}
}

// VerifiedROBlobError can be used by methods that have a VerifiedROBlob return type but do not have permission to
// create a value of that type in order to generate an error return value.
func VerifiedROBlobError(err error) (blocks.VerifiedROBlob, error) {
	if err == nil {
		panic("VerifiedROBlobError used to create a VerifiedROBlob without a checkable error.")
	}
	return blocks.VerifiedROBlob{}, err
}
