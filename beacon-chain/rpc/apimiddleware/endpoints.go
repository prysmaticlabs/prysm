package apimiddleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/gateway"
)

// TODO: Documentation
func RegisterMiddlewareEndpoints() []gateway.Endpoint {
	return []gateway.Endpoint{
		{
			Url:         "/eth/v1/beacon/genesis",
			GetResponse: &genesisResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/beacon/states/{state_id}/root",
			GetResponse: &stateRootResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/beacon/states/{state_id}/fork",
			GetResponse: &stateForkResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/beacon/states/{state_id}/finality_checkpoints",
			GetResponse: &stateFinalityCheckpointResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:                   "/eth/v1/beacon/states/{state_id}/validators",
			GetRequestQueryParams: []gateway.QueryParam{{Name: "id", Hex: true}, {Name: "status", Enum: true}},
			GetResponse:           &stateValidatorsResponseJson{},
			Err:                   &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/beacon/states/{state_id}/validators/{validator_id}",
			GetResponse: &stateValidatorResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/beacon/states/{state_id}/validators/{validator_id}",
			GetResponse: &stateValidatorResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:                   "/eth/v1/beacon/states/{state_id}/validator_balances",
			GetRequestQueryParams: []gateway.QueryParam{{Name: "id", Hex: true}},
			GetResponse:           &validatorBalancesResponseJson{},
			Err:                   &gateway.DefaultErrorJson{},
		},
		{
			Url:                   "/eth/v1/beacon/states/{state_id}/committees",
			GetRequestQueryParams: []gateway.QueryParam{{Name: "epoch"}, {Name: "index"}, {Name: "slot"}},
			GetResponse:           &stateCommitteesResponseJson{},
			Err:                   &gateway.DefaultErrorJson{},
		},
		{
			Url:                   "/eth/v1/beacon/headers",
			GetRequestQueryParams: []gateway.QueryParam{{Name: "slot"}, {Name: "parent_root", Hex: true}},
			GetResponse:           &blockHeadersResponseJson{},
			Err:                   &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/beacon/headers/{block_id}",
			GetResponse: &blockHeaderResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/beacon/blocks",
			PostRequest: &beaconBlockContainerJson{},
			Err:         &gateway.DefaultErrorJson{},
			Hooks: gateway.HookCollection{
				OnPostDeserializedRequestBodyIntoContainer: []gateway.Hook{prepareGraffiti},
			},
		},
		{
			Url:         "/eth/v1/beacon/blocks/{block_id}",
			GetResponse: &blockResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/beacon/blocks/{block_id}/root",
			GetResponse: &blockRootResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/beacon/blocks/{block_id}/attestations",
			GetResponse: &blockAttestationsResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:                   "/eth/v1/beacon/pool/attestations",
			GetRequestQueryParams: []gateway.QueryParam{{Name: "slot"}, {Name: "committee_index"}},
			GetResponse:           &attestationsPoolResponseJson{},
			PostRequest:           &submitAttestationRequestJson{},
			Err:                   &submitAttestationsErrorJson{},
			Hooks: gateway.HookCollection{
				OnPostStart: []gateway.Hook{wrapAttestationsArray},
			},
		},
		{
			Url:         "/eth/v1/beacon/pool/attester_slashings",
			PostRequest: &attesterSlashingJson{},
			GetResponse: &attesterSlashingsPoolResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/beacon/pool/proposer_slashings",
			PostRequest: &proposerSlashingJson{},
			GetResponse: &proposerSlashingsPoolResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/beacon/pool/voluntary_exits",
			PostRequest: &signedVoluntaryExitJson{},
			GetResponse: &voluntaryExitsPoolResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/node/identity",
			GetResponse: &identityResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/node/peers",
			GetResponse: &peersResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:                   "/eth/v1/node/peers/{peer_id}",
			GetRequestUrlLiterals: []string{"peer_id"},
			GetResponse:           &peerResponseJson{},
			Err:                   &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/node/peer_count",
			GetResponse: &peerCountResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/node/version",
			GetResponse: &versionResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/node/syncing",
			GetResponse: &syncingResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url: "/eth/v1/node/health",
			Err: &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/debug/beacon/states/{state_id}",
			GetResponse: &beaconStateResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/debug/beacon/heads",
			GetResponse: &forkChoiceHeadsResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/config/fork_schedule",
			GetResponse: &forkScheduleResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/config/deposit_contract",
			GetResponse: &depositContractResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
		{
			Url:         "/eth/v1/config/spec",
			GetResponse: &specResponseJson{},
			Err:         &gateway.DefaultErrorJson{}},
	}
}

// https://ethereum.github.io/eth2.0-APIs/#/Beacon/submitPoolAttestations expects posting a top-level array.
// We make it more proto-friendly by wrapping it in a struct with a 'data' field.
func wrapAttestationsArray(endpoint gateway.Endpoint, _ http.ResponseWriter, req *http.Request) gateway.ErrorJson {
	if _, ok := endpoint.PostRequest.(*submitAttestationRequestJson); ok {
		atts := make([]*attestationJson, 0)
		if err := json.NewDecoder(req.Body).Decode(&atts); err != nil {
			e := fmt.Errorf("could not decode attestations array: %w", err)
			return &gateway.DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
		}
		j := &submitAttestationRequestJson{Data: atts}
		b, err := json.Marshal(j)
		if err != nil {
			e := fmt.Errorf("could not marshal wrapped attestations array: %w", err)
			return &gateway.DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(b))
	}
	return nil
}

// Posted graffiti needs to have length of 32 bytes, but client is allowed to send data of any length.
func prepareGraffiti(endpoint gateway.Endpoint, _ http.ResponseWriter, _ *http.Request) gateway.ErrorJson {
	if block, ok := endpoint.PostRequest.(*beaconBlockContainerJson); ok {
		b := bytesutil.ToBytes32([]byte(block.Message.Body.Graffiti))
		block.Message.Body.Graffiti = hexutil.Encode(b[:])
	}
	return nil
}
