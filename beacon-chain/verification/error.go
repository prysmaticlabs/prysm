package verification

import "github.com/pkg/errors"

// ErrMissingVerification indicates that the given verification function was never performed on the value.
var ErrMissingVerification = errors.New("verification was not performed for requirement")

// VerificationMultiError is a custom error that can be used to access individual verification failures.
type VerificationMultiError struct {
	r   *results
	err error
}

func (ve VerificationMultiError) Unwrap() error {
	return ve.err
}

func (ve VerificationMultiError) Error() string {
	return ve.err.Error()
}

func (ve VerificationMultiError) Failures() map[Requirement]error {
	return ve.r.failures()
}

func newVerificationMultiError(r *results, err error) VerificationMultiError {
	return VerificationMultiError{r: r, err: err}
}
