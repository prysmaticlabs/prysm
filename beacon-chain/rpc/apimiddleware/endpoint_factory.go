package apimiddleware

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/gateway"
)

// BeaconEndpointFactory creates endpoints used for running beacon chain API calls through the API Middleware.
type BeaconEndpointFactory struct {
}

func (f *BeaconEndpointFactory) IsNil() bool {
	return f == nil
}

// Paths is a collection of all valid beacon chain API paths.
func (f *BeaconEndpointFactory) Paths() []string {
	return []string{
		"/eth/v1/beacon/genesis",
		"/eth/v1/beacon/states/{state_id}/root",
		"/eth/v1/beacon/states/{state_id}/fork",
		"/eth/v1/beacon/states/{state_id}/finality_checkpoints",
		"/eth/v1/beacon/states/{state_id}/validators",
		"/eth/v1/beacon/states/{state_id}/validators/{validator_id}",
		"/eth/v1/beacon/states/{state_id}/validator_balances",
		"/eth/v1/beacon/states/{state_id}/committees",
		"/eth/v1/beacon/headers",
		"/eth/v1/beacon/headers/{block_id}",
		"/eth/v1/beacon/blocks",
		"/eth/v1/beacon/blocks/{block_id}",
		"/eth/v1/beacon/blocks/{block_id}/root",
		"/eth/v1/beacon/blocks/{block_id}/attestations",
		"/eth/v1/beacon/pool/attestations",
		"/eth/v1/beacon/pool/attester_slashings",
		"/eth/v1/beacon/pool/proposer_slashings",
		"/eth/v1/beacon/pool/voluntary_exits",
		"/eth/v1/node/identity",
		"/eth/v1/node/peers",
		"/eth/v1/node/peers/{peer_id}",
		"/eth/v1/node/peer_count",
		"/eth/v1/node/version",
		"/eth/v1/node/syncing",
		"/eth/v1/node/health",
		"/eth/v1/debug/beacon/states/{state_id}",
		"/eth/v1/debug/beacon/heads",
		"/eth/v1/config/fork_schedule",
		"/eth/v1/config/deposit_contract",
		"/eth/v1/config/spec",
		"/eth/v1/events",
		"/eth/v1/validator/duties/attester/{epoch}",
	}
}

// Create returns a new endpoint for the provided API path.
func (f *BeaconEndpointFactory) Create(path string) (*gateway.Endpoint, error) {
	var endpoint gateway.Endpoint
	switch path {
	case "/eth/v1/beacon/genesis":
		endpoint = gateway.Endpoint{
			GetResponse: &genesisResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/beacon/states/{state_id}/root":
		endpoint = gateway.Endpoint{
			GetResponse: &stateRootResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/beacon/states/{state_id}/fork":
		endpoint = gateway.Endpoint{
			GetResponse: &stateForkResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/beacon/states/{state_id}/finality_checkpoints":
		endpoint = gateway.Endpoint{
			GetResponse: &stateFinalityCheckpointResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/beacon/states/{state_id}/validators":
		endpoint = gateway.Endpoint{
			RequestQueryParams: []gateway.QueryParam{{Name: "id", Hex: true}, {Name: "status", Enum: true}},
			GetResponse:        &stateValidatorsResponseJson{},
			Err:                &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/beacon/states/{state_id}/validators/{validator_id}":
		endpoint = gateway.Endpoint{
			GetResponse: &stateValidatorResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/beacon/states/{state_id}/validator_balances":
		endpoint = gateway.Endpoint{
			RequestQueryParams: []gateway.QueryParam{{Name: "id", Hex: true}},
			GetResponse:        &validatorBalancesResponseJson{},
			Err:                &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/beacon/states/{state_id}/committees":
		endpoint = gateway.Endpoint{
			RequestQueryParams: []gateway.QueryParam{{Name: "epoch"}, {Name: "index"}, {Name: "slot"}},
			GetResponse:        &stateCommitteesResponseJson{},
			Err:                &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/beacon/headers":
		endpoint = gateway.Endpoint{
			RequestQueryParams: []gateway.QueryParam{{Name: "slot"}, {Name: "parent_root", Hex: true}},
			GetResponse:        &blockHeadersResponseJson{},
			Err:                &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/beacon/headers/{block_id}":
		endpoint = gateway.Endpoint{
			GetResponse: &blockHeaderResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/beacon/blocks":
		endpoint = gateway.Endpoint{
			PostRequest: &beaconBlockContainerJson{},
			Err:         &gateway.DefaultErrorJson{},
			Hooks: gateway.HookCollection{
				OnPostDeserializeRequestBodyIntoContainer: []gateway.Hook{prepareGraffiti},
			},
		}
	case "/eth/v1/beacon/blocks/{block_id}":
		endpoint = gateway.Endpoint{
			GetResponse: &blockResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
			Hooks: gateway.HookCollection{
				CustomHandlers: []gateway.CustomHandler{handleGetBeaconBlockSSZ},
			},
		}
	case "/eth/v1/beacon/blocks/{block_id}/root":
		endpoint = gateway.Endpoint{
			GetResponse: &blockRootResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/beacon/blocks/{block_id}/attestations":
		endpoint = gateway.Endpoint{
			GetResponse: &blockAttestationsResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/beacon/pool/attestations":
		endpoint = gateway.Endpoint{
			RequestQueryParams: []gateway.QueryParam{{Name: "slot"}, {Name: "committee_index"}},
			GetResponse:        &attestationsPoolResponseJson{},
			PostRequest:        &submitAttestationRequestJson{},
			Err:                &submitAttestationsErrorJson{},
			Hooks: gateway.HookCollection{
				OnPostStart: []gateway.Hook{wrapAttestationsArray},
			},
		}
	case "/eth/v1/beacon/pool/attester_slashings":
		endpoint = gateway.Endpoint{
			PostRequest: &attesterSlashingJson{},
			GetResponse: &attesterSlashingsPoolResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/beacon/pool/proposer_slashings":
		endpoint = gateway.Endpoint{
			PostRequest: &proposerSlashingJson{},
			GetResponse: &proposerSlashingsPoolResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/beacon/pool/voluntary_exits":
		endpoint = gateway.Endpoint{
			PostRequest: &signedVoluntaryExitJson{},
			GetResponse: &voluntaryExitsPoolResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/node/identity":
		endpoint = gateway.Endpoint{
			GetResponse: &identityResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/node/peers":
		endpoint = gateway.Endpoint{
			RequestQueryParams: []gateway.QueryParam{{Name: "state", Enum: true}, {Name: "direction", Enum: true}},
			GetResponse:        &peersResponseJson{},
			Err:                &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/node/peers/{peer_id}":
		endpoint = gateway.Endpoint{
			RequestURLLiterals: []string{"peer_id"},
			GetResponse:        &peerResponseJson{},
			Err:                &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/node/peer_count":
		endpoint = gateway.Endpoint{
			GetResponse: &peerCountResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/node/version":
		endpoint = gateway.Endpoint{
			GetResponse: &versionResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/node/syncing":
		endpoint = gateway.Endpoint{
			GetResponse: &syncingResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/node/health":
		endpoint = gateway.Endpoint{
			Err: &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/debug/beacon/states/{state_id}":
		endpoint = gateway.Endpoint{
			GetResponse: &beaconStateResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
			Hooks: gateway.HookCollection{
				CustomHandlers: []gateway.CustomHandler{handleGetBeaconStateSSZ},
			},
		}
	case "/eth/v1/debug/beacon/heads":
		endpoint = gateway.Endpoint{
			GetResponse: &forkChoiceHeadsResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/config/fork_schedule":
		endpoint = gateway.Endpoint{
			GetResponse: &forkScheduleResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/config/deposit_contract":
		endpoint = gateway.Endpoint{
			GetResponse: &depositContractResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/config/spec":
		endpoint = gateway.Endpoint{
			GetResponse: &specResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		}
	case "/eth/v1/events":
		endpoint = gateway.Endpoint{
			Err: &gateway.DefaultErrorJson{},
			Hooks: gateway.HookCollection{
				CustomHandlers: []gateway.CustomHandler{handleEvents},
			},
		}
	case "/eth/v1/validator/duties/attester/{epoch}":
		endpoint = gateway.Endpoint{
			PostRequest:        &attesterDutiesRequestJson{},
			PostResponse:       &attesterDutiesResponseJson{},
			RequestURLLiterals: []string{"epoch"},
			Err:                &gateway.DefaultErrorJson{},
			Hooks: gateway.HookCollection{
				OnPostStart: []gateway.Hook{wrapValidatorIndicesArray},
			},
		}
	default:
		return nil, errors.New("invalid path")
	}

	endpoint.Path = path
	return &endpoint, nil
}
