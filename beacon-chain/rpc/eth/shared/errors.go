package shared

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/lookup"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
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

// IndexedVerificationFailureError wraps a collection of verification failures.
type IndexedVerificationFailureError struct {
	Message  string                        `json:"message"`
	Code     int                           `json:"code"`
	Failures []*IndexedVerificationFailure `json:"failures"`
}

func (e *IndexedVerificationFailureError) StatusCode() int {
	return e.Code
}

// IndexedVerificationFailure represents an issue when verifying a single indexed object e.g. an item in an array.
type IndexedVerificationFailure struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

// WriteBlockFetchError writes an appropriate error based on the supplied argument.
// The argument error should be a result of fetching block.
func WriteBlockFetchError(w http.ResponseWriter, blk interfaces.ReadOnlySignedBeaconBlock, err error) bool {
	if invalidBlockIdErr, ok := err.(*lookup.BlockIdParseError); ok {
		http2.HandleError(w, "Invalid block ID: "+invalidBlockIdErr.Error(), http.StatusBadRequest)
		return false
	}
	if err != nil {
		http2.HandleError(w, "Could not get block from block ID: %s"+err.Error(), http.StatusInternalServerError)
		return false
	}
	if err = blocks.BeaconBlockIsNil(blk); err != nil {
		http2.HandleError(w, "Could not find requested block: %s"+err.Error(), http.StatusNotFound)
		return false
	}
	return true
}
