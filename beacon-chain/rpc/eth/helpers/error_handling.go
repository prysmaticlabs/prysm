package helpers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/lookup"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stategen"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PrepareStateFetchGRPCError returns an appropriate gRPC error based on the supplied argument.
// The argument error should be a result of fetching state.
func PrepareStateFetchGRPCError(err error) error {
	if errors.Is(err, stategen.ErrNoDataForSlot) {
		return status.Errorf(codes.NotFound, "lacking historical data needed to fulfill request")
	}
	if stateNotFoundErr, ok := err.(*lookup.StateNotFoundError); ok {
		return status.Errorf(codes.NotFound, "State not found: %v", stateNotFoundErr)
	}
	if parseErr, ok := err.(*lookup.StateIdParseError); ok {
		return status.Errorf(codes.InvalidArgument, "Invalid state ID: %v", parseErr)
	}
	return status.Errorf(codes.Internal, "Invalid state ID: %v", err)
}

// HandleStateFetchError writes the appropriate HTTP error based on the supplied error.
func HandleStateFetchError(w http.ResponseWriter, err error) {
	if errors.Is(err, stategen.ErrNoDataForSlot) {
		http2.HandleError(w, "Lacking historical data needed to fulfill request", http.StatusNotFound)
		return
	}
	if stateNotFoundErr, ok := err.(*lookup.StateNotFoundError); ok {
		http2.HandleError(w, "State not found: "+stateNotFoundErr.Error(), http.StatusNotFound)
		return
	}
	if parseErr, ok := err.(*lookup.StateIdParseError); ok {
		http2.HandleError(w, "Invalid state ID: "+parseErr.Error(), http.StatusBadRequest)
		return
	}
	http2.HandleError(w, "Invalid state ID: "+err.Error(), http.StatusInternalServerError)
}

// IndexedVerificationFailure represents a collection of verification failures.
type IndexedVerificationFailure struct {
	Failures []*SingleIndexedVerificationFailure `json:"failures"`
}

// SingleIndexedVerificationFailure represents an issue when verifying a single indexed object e.g. an item in an array.
type SingleIndexedVerificationFailure struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

// PrepareStateFetchError returns an appropriate error based on the supplied argument.
// The argument error should be a result of fetching state.
func PrepareStateFetchError(err error) error {
	if errors.Is(err, stategen.ErrNoDataForSlot) {
		return errors.New("lacking historical data needed to fulfill request")
	}
	if stateNotFoundErr, ok := err.(*lookup.StateNotFoundError); ok {
		return fmt.Errorf("state not found: %v", stateNotFoundErr)
	}
	return fmt.Errorf("could not fetch state: %v", err)
}
