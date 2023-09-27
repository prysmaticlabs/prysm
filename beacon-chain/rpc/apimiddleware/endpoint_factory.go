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
		"/eth/v1/beacon/states/{state_id}/root",
		"/eth/v1/beacon/states/{state_id}/sync_committees",
		"/eth/v1/beacon/states/{state_id}/randao",
		"/eth/v1/beacon/blinded_blocks",
		"/eth/v1/beacon/blocks/{block_id}",
		"/eth/v2/beacon/blocks/{block_id}",
		"/eth/v1/beacon/blocks/{block_id}/attestations",
		"/eth/v1/beacon/blinded_blocks/{block_id}",
		"/eth/v1/beacon/pool/attester_slashings",
		"/eth/v1/beacon/pool/proposer_slashings",
		"/eth/v1/beacon/pool/bls_to_execution_changes",
		"/eth/v1/beacon/pool/bls_to_execution_changes",
		"/eth/v1/beacon/weak_subjectivity",
		"/eth/v1/node/identity",
		"/eth/v1/node/peers",
		"/eth/v1/node/peers/{peer_id}",
		"/eth/v1/node/peer_count",
		"/eth/v1/node/version",
		"/eth/v1/node/health",
		"/eth/v1/debug/beacon/states/{state_id}",
		"/eth/v2/debug/beacon/states/{state_id}",
		"/eth/v1/debug/beacon/heads",
		"/eth/v2/debug/beacon/heads",
		"/eth/v1/debug/fork_choice",
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
	case "/eth/v1/beacon/states/{state_id}/root":
		endpoint.GetResponse = &StateRootResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/sync_committees":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "epoch"}}
		endpoint.GetResponse = &SyncCommitteesResponseJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeGrpcResponseBodyIntoContainer: prepareValidatorAggregates,
		}
	case "/eth/v1/beacon/states/{state_id}/randao":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "epoch"}}
		endpoint.GetResponse = &RandaoResponseJson{}
	case "/eth/v1/beacon/blocks/{block_id}":
		endpoint.GetResponse = &BlockResponseJson{}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleGetBeaconBlockSSZ}
	case "/eth/v2/beacon/blocks/{block_id}":
		endpoint.GetResponse = &BlockV2ResponseJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreSerializeMiddlewareResponseIntoJson: serializeV2Block,
		}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleGetBeaconBlockSSZV2}
	case "/eth/v1/beacon/blocks/{block_id}/attestations":
		endpoint.GetResponse = &BlockAttestationsResponseJson{}
	case "/eth/v1/beacon/blinded_blocks/{block_id}":
		endpoint.GetResponse = &BlindedBlockResponseJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreSerializeMiddlewareResponseIntoJson: serializeBlindedBlock,
		}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleGetBlindedBeaconBlockSSZ}
	case "/eth/v1/beacon/pool/attester_slashings":
		endpoint.PostRequest = &AttesterSlashingJson{}
		endpoint.GetResponse = &AttesterSlashingsPoolResponseJson{}
	case "/eth/v1/beacon/pool/proposer_slashings":
		endpoint.PostRequest = &ProposerSlashingJson{}
		endpoint.GetResponse = &ProposerSlashingsPoolResponseJson{}
	case "/eth/v1/beacon/pool/bls_to_execution_changes":
		endpoint.PostRequest = &SubmitBLSToExecutionChangesRequest{}
		endpoint.GetResponse = &BLSToExecutionChangesPoolResponseJson{}
		endpoint.Err = &IndexedVerificationFailureErrorJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapBLSChangesArray,
		}
	case "/eth/v1/beacon/weak_subjectivity":
		endpoint.GetResponse = &WeakSubjectivityResponse{}
	case "/eth/v1/node/identity":
		endpoint.GetResponse = &IdentityResponseJson{}
	case "/eth/v1/node/peers":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "state", Enum: true}, {Name: "direction", Enum: true}}
		endpoint.GetResponse = &PeersResponseJson{}
	case "/eth/v1/node/peers/{peer_id}":
		endpoint.RequestURLLiterals = []string{"peer_id"}
		endpoint.GetResponse = &PeerResponseJson{}
	case "/eth/v1/node/peer_count":
		endpoint.GetResponse = &PeerCountResponseJson{}
	case "/eth/v1/node/version":
		endpoint.GetResponse = &VersionResponseJson{}
	case "/eth/v1/node/health":
		// Use default endpoint
	case "/eth/v1/debug/beacon/states/{state_id}":
		endpoint.GetResponse = &BeaconStateResponseJson{}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleGetBeaconStateSSZ}
	case "/eth/v2/debug/beacon/states/{state_id}":
		endpoint.GetResponse = &BeaconStateV2ResponseJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreSerializeMiddlewareResponseIntoJson: serializeV2State,
		}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleGetBeaconStateSSZV2}
	case "/eth/v1/debug/beacon/heads":
		endpoint.GetResponse = &ForkChoiceHeadsResponseJson{}
	case "/eth/v2/debug/beacon/heads":
		endpoint.GetResponse = &V2ForkChoiceHeadsResponseJson{}
	case "/eth/v1/debug/fork_choice":
		endpoint.GetResponse = &ForkChoiceDumpJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreSerializeMiddlewareResponseIntoJson: prepareForkChoiceResponse,
		}
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
