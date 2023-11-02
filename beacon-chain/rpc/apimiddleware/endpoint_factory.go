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
		"/eth/v1/beacon/pool/attester_slashings",
		"/eth/v1/beacon/pool/proposer_slashings",
		"/eth/v1/beacon/weak_subjectivity",
		"/eth/v1/config/fork_schedule",
		"/eth/v1/config/spec",
		"/eth/v1/events",
		"/eth/v1/validator/blocks/{slot}",
		"/eth/v2/validator/blocks/{slot}",
		"/eth/v1/validator/blinded_blocks/{slot}",
	}
}

// Create returns a new endpoint for the provided API path.
func (_ *BeaconEndpointFactory) Create(path string) (*apimiddleware.Endpoint, error) {
	endpoint := apimiddleware.DefaultEndpoint()
	switch path {
	case "/eth/v1/beacon/pool/attester_slashings":
		endpoint.PostRequest = &AttesterSlashingJson{}
		endpoint.GetResponse = &AttesterSlashingsPoolResponseJson{}
	case "/eth/v1/beacon/pool/proposer_slashings":
		endpoint.PostRequest = &ProposerSlashingJson{}
		endpoint.GetResponse = &ProposerSlashingsPoolResponseJson{}
	case "/eth/v1/beacon/weak_subjectivity":
		endpoint.GetResponse = &WeakSubjectivityResponse{}
	case "/eth/v1/config/fork_schedule":
		endpoint.GetResponse = &ForkScheduleResponseJson{}
	case "/eth/v1/config/spec":
		endpoint.GetResponse = &SpecResponseJson{}
	case "/eth/v1/events":
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleEvents}
	case "/eth/v1/validator/blocks/{slot}":
		endpoint.GetResponse = &ProduceBlockResponseJson{}
		endpoint.RequestURLLiterals = []string{"slot"}
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "randao_reveal", Hex: true}, {Name: "graffiti", Hex: true}}
	case "/eth/v2/validator/blocks/{slot}":
		endpoint.GetResponse = &ProduceBlockResponseV2Json{}
		endpoint.RequestURLLiterals = []string{"slot"}
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "randao_reveal", Hex: true}, {Name: "graffiti", Hex: true}}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreSerializeMiddlewareResponseIntoJson: serializeProducedV2Block,
		}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleProduceBlockSSZ}
	case "/eth/v1/validator/blinded_blocks/{slot}":
		endpoint.GetResponse = &ProduceBlindedBlockResponseJson{}
		endpoint.RequestURLLiterals = []string{"slot"}
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "randao_reveal", Hex: true}, {Name: "graffiti", Hex: true}}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreSerializeMiddlewareResponseIntoJson: serializeProducedBlindedBlock,
		}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleProduceBlindedBlockSSZ}
	default:
		return nil, errors.New("invalid path")
	}

	endpoint.Path = path
	return &endpoint, nil
}
