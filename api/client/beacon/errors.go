package beacon

import "github.com/pkg/errors"

// ErrNotOK is used to indicate when an HTTP request to the Beacon Node API failed with any non-2xx response code.
// More specific errors may be returned, but an error in reaction to a non-2xx response will always wrap ErrNotOK.
var ErrNotOK = errors.New("did not receive 2xx response from API")

// ErrNotFound specifically means that a '404 - NOT FOUND' response was received from the API.
var ErrNotFound = errors.Wrap(ErrNotOK, "recv 404 NotFound response from API")

// ErrInvalidNodeVersion indicates that the /eth/v1/node/version api response format was not recognized.
var ErrInvalidNodeVersion = errors.New("invalid node version response")
