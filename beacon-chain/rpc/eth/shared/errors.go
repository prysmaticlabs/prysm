package shared

import (
	"fmt"
	"strings"
)

// DecodeError represents an error resulting from trying to decode an HTTP request.
// It tracks the full field name for which decoding failed.
type DecodeError struct {
	path []string
	err  error
}

// NewDecodeError wraps an error (either the initial decoding error or another DecodeError).
// The current field that failed decoding must be passed in.
func NewDecodeError(err error, field string) *DecodeError {
	de, ok := err.(*DecodeError)
	if ok {
		return &DecodeError{path: append([]string{field}, de.path...), err: de.err}
	}
	return &DecodeError{path: []string{field}, err: err}
}

// Error returns the formatted error message which contains the full field name and the actual decoding error.
func (e *DecodeError) Error() string {
	return fmt.Sprintf("could not decode %s: %s", strings.Join(e.path, "."), e.err.Error())
}
