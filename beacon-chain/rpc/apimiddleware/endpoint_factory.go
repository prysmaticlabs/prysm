package apimiddleware

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api/gateway/apimiddleware"
)

// BeaconEndpointFactory creates endpoints used for running beacon chain API calls through the API Middleware.
type BeaconEndpointFactory struct {
}

func (f *BeaconEndpointFactory) IsNil() bool {
	return f == nil
}

// Paths is a collection of all valid beacon chain API paths.
func (_ *BeaconEndpointFactory) Paths() []string {
	return []string{
		"/eth/v1/beacon/weak_subjectivity",
	}
}

// Create returns a new endpoint for the provided API path.
func (_ *BeaconEndpointFactory) Create(path string) (*apimiddleware.Endpoint, error) {
	endpoint := apimiddleware.DefaultEndpoint()
	switch path {
	case "/eth/v1/beacon/weak_subjectivity":
		endpoint.GetResponse = &WeakSubjectivityResponse{}
	default:
		return nil, errors.New("invalid path")
	}

	endpoint.Path = path
	return &endpoint, nil
}
