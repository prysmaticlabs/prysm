package apimiddleware

import (
	"github.com/prysmaticlabs/prysm/shared/gateway"
)

type sszConfig struct {
	sszPath      string
	fileName     string
	responseJson sszResponseJson
}

// MiddlewareEndpointRegistry is a registry of all endpoints that should be proxied by the API Middleware between an HTTP client and the grpc-gateway.
//
// All endpoints from the official Eth2 API specification must run through the middleware to maintain full compatibility with the specification.
func MiddlewareEndpointRegistry() []gateway.Endpoint {
	return []gateway.Endpoint{
		{
			Path:        "/eth/v1/beacon/genesis",
			GetResponse: &genesisResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/beacon/states/{state_id}/root",
			GetResponse: &stateRootResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/beacon/states/{state_id}/fork",
			GetResponse: &stateForkResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/beacon/states/{state_id}/finality_checkpoints",
			GetResponse: &stateFinalityCheckpointResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:                  "/eth/v1/beacon/states/{state_id}/validators",
			GetRequestQueryParams: []gateway.QueryParam{{Name: "id", Hex: true}, {Name: "status", Enum: true}},
			GetResponse:           &stateValidatorsResponseJson{},
			Err:                   &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/beacon/states/{state_id}/validators/{validator_id}",
			GetResponse: &stateValidatorResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/beacon/states/{state_id}/validators/{validator_id}",
			GetResponse: &stateValidatorResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:                  "/eth/v1/beacon/states/{state_id}/validator_balances",
			GetRequestQueryParams: []gateway.QueryParam{{Name: "id", Hex: true}},
			GetResponse:           &validatorBalancesResponseJson{},
			Err:                   &gateway.DefaultErrorJson{},
		},
		{
			Path:                  "/eth/v1/beacon/states/{state_id}/committees",
			GetRequestQueryParams: []gateway.QueryParam{{Name: "epoch"}, {Name: "index"}, {Name: "slot"}},
			GetResponse:           &stateCommitteesResponseJson{},
			Err:                   &gateway.DefaultErrorJson{},
		},
		{
			Path:                  "/eth/v1/beacon/headers",
			GetRequestQueryParams: []gateway.QueryParam{{Name: "slot"}, {Name: "parent_root", Hex: true}},
			GetResponse:           &blockHeadersResponseJson{},
			Err:                   &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/beacon/headers/{block_id}",
			GetResponse: &blockHeaderResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Path:        "/eth/v1/beacon/blocks",
			PostRequest: &beaconBlockContainerJson{},
			Err:         &gateway.DefaultErrorJson{},
			Hooks: gateway.HookCollection{
				OnPostDeserializeRequestBodyIntoContainer: []gateway.Hook{prepareGraffiti},
			},
		},
		{
			Path:        "/eth/v1/beacon/blocks/{block_id}",
			GetResponse: &blockResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
			Hooks: gateway.HookCollection{
				CustomHandlers: []gateway.CustomHandler{handleGetBlockSsz},
			},
		},
		{
			Path:        "/eth/v1/beacon/blocks/{block_id}/root",
			GetResponse: &blockRootResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/beacon/blocks/{block_id}/attestations",
			GetResponse: &blockAttestationsResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:                  "/eth/v1/beacon/pool/attestations",
			GetRequestQueryParams: []gateway.QueryParam{{Name: "slot"}, {Name: "committee_index"}},
			GetResponse:           &attestationsPoolResponseJson{},
			PostRequest:           &submitAttestationRequestJson{},
			Err:                   &submitAttestationsErrorJson{},
			Hooks: gateway.HookCollection{
				OnPostStart: []gateway.Hook{wrapAttestationsArray},
			},
		},
		{
			Path:        "/eth/v1/beacon/pool/attester_slashings",
			PostRequest: &attesterSlashingJson{},
			GetResponse: &attesterSlashingsPoolResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/beacon/pool/proposer_slashings",
			PostRequest: &proposerSlashingJson{},
			GetResponse: &proposerSlashingsPoolResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/beacon/pool/voluntary_exits",
			PostRequest: &signedVoluntaryExitJson{},
			GetResponse: &voluntaryExitsPoolResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/node/identity",
			GetResponse: &identityResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/node/peers",
			GetResponse: &peersResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:                  "/eth/v1/node/peers/{peer_id}",
			GetRequestUrlLiterals: []string{"peer_id"},
			GetResponse:           &peerResponseJson{},
			Err:                   &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/node/peer_count",
			GetResponse: &peerCountResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/node/version",
			GetResponse: &versionResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/node/syncing",
			GetResponse: &syncingResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path: "/eth/v1/node/health",
			Err:  &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/debug/beacon/states/{state_id}",
			GetResponse: &beaconStateResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
			Hooks: gateway.HookCollection{
				CustomHandlers: []gateway.CustomHandler{handleGetBeaconStateSsz},
			},
		},
		{
			Path:        "/eth/v1/debug/beacon/heads",
			GetResponse: &forkChoiceHeadsResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/config/fork_schedule",
			GetResponse: &forkScheduleResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/config/deposit_contract",
			GetResponse: &depositContractResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Path:        "/eth/v1/config/spec",
			GetResponse: &specResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
	}
}
