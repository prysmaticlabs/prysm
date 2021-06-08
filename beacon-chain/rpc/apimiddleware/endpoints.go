package apimiddleware

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/gateway"
)

// MiddlewareEndpointsRegistry is a registry of all endpoints that should be proxied by the API Middleware between an HTTP client and the grpc-gateway.
//
// All endpoints from the official Eth2 API specification must run through the middleware to maintain full compatibility with the specification.
func MiddlewareEndpointsRegistry() []gateway.Endpoint {
	return []gateway.Endpoint{
		{
			Url:         "/eth/v1/beacon/genesis",
			GetResponse: &genesisResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/beacon/states/{state_id}/root",
			GetResponse: &stateRootResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/beacon/states/{state_id}/fork",
			GetResponse: &stateForkResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/beacon/states/{state_id}/finality_checkpoints",
			GetResponse: &stateFinalityCheckpointResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:                   "/eth/v1/beacon/states/{state_id}/validators",
			GetRequestQueryParams: []gateway.QueryParam{{Name: "id", Hex: true}, {Name: "status", Enum: true}},
			GetResponse:           &stateValidatorsResponseJson{},
			Err:                   &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/beacon/states/{state_id}/validators/{validator_id}",
			GetResponse: &stateValidatorResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
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
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/beacon/blocks/{block_id}/root",
			GetResponse: &blockRootResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/beacon/blocks/{block_id}/attestations",
			GetResponse: &blockAttestationsResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
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
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/beacon/pool/proposer_slashings",
			PostRequest: &proposerSlashingJson{},
			GetResponse: &proposerSlashingsPoolResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/beacon/pool/voluntary_exits",
			PostRequest: &signedVoluntaryExitJson{},
			GetResponse: &voluntaryExitsPoolResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/node/identity",
			GetResponse: &identityResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/node/peers",
			GetResponse: &peersResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:                   "/eth/v1/node/peers/{peer_id}",
			GetRequestUrlLiterals: []string{"peer_id"},
			GetResponse:           &peerResponseJson{},
			Err:                   &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/node/peer_count",
			GetResponse: &peerCountResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/node/version",
			GetResponse: &versionResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/node/syncing",
			GetResponse: &syncingResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url: "/eth/v1/node/health",
			Err: &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/debug/beacon/states/{state_id}",
			GetResponse: &beaconStateResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
			Hooks: gateway.HookCollection{
				CustomHandlers: []gateway.CustomHandler{handleGetBeaconStateSsz},
			},
		},
		{
			Url:         "/eth/v1/debug/beacon/heads",
			GetResponse: &forkChoiceHeadsResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/config/fork_schedule",
			GetResponse: &forkScheduleResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/config/deposit_contract",
			GetResponse: &depositContractResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
		{
			Url:         "/eth/v1/config/spec",
			GetResponse: &specResponseJson{},
			Err:         &gateway.DefaultErrorJson{},
		},
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

func handleGetBeaconStateSsz(m *gateway.ApiProxyMiddleware, endpoint gateway.Endpoint, writer http.ResponseWriter, request *http.Request) (handled bool) {
	if !sszRequested(request) {
		return false
	}

	if errJson := prepareSszRequestForProxying(m, endpoint, request, "/eth/v1/debug/beacon/states/{state_id}/ssz"); errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return
	}
	grpcResponse, errJson := gateway.ProxyRequest(request)
	if errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return
	}
	grpcResponseBody, errJson := gateway.ReadGrpcResponseBody(grpcResponse.Body)
	if errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return
	}
	if errJson := gateway.DeserializeGrpcResponseBodyIntoErrorJson(endpoint.Err, grpcResponseBody); errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return
	}
	if endpoint.Err.Msg() != "" {
		gateway.HandleGrpcResponseError(endpoint.Err, grpcResponse, writer)
		return
	}
	responseJson := &beaconStateSszResponseJson{}
	if errJson := gateway.DeserializeGrpcResponseBodyIntoContainer(grpcResponseBody, responseJson); errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return
	}
	responseSsz, errJson := serializeMiddlewareResponseIntoSsz(responseJson.Data)
	if errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return
	}
	if errJson := writeSszResponseHeaderAndBody(grpcResponse, writer, responseSsz, "beacon_state.ssz"); errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return
	}
	if errJson := gateway.Cleanup(grpcResponse.Body); errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return
	}

	return true
}

func sszRequested(request *http.Request) bool {
	accept, ok := request.Header["Accept"]
	if !ok {
		return false
	}
	for _, v := range accept {
		if v == "application/octet-stream" {
			return true
		}
	}
	return false
}

func prepareSszRequestForProxying(m *gateway.ApiProxyMiddleware, endpoint gateway.Endpoint, request *http.Request, sszPath string) gateway.ErrorJson {
	request.URL.Scheme = "http"
	request.URL.Host = m.GatewayAddress
	request.RequestURI = ""
	request.URL.Path = sszPath
	return gateway.HandleUrlParameters(endpoint.Url, request, []string{})
}

func serializeMiddlewareResponseIntoSsz(data string) (sszResponse []byte, errJson gateway.ErrorJson) {
	// Serialize the SSZ part of the deserialized value.
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		e := fmt.Errorf("could not decode response body into base64: %w", err)
		return nil, &gateway.DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
	}
	return b, nil
}

func writeSszResponseHeaderAndBody(grpcResponse *http.Response, writer http.ResponseWriter, responseSsz []byte, fileName string) gateway.ErrorJson {
	for h, vs := range grpcResponse.Header {
		for _, v := range vs {
			writer.Header().Set(h, v)
		}
	}
	writer.Header().Set("Content-Length", strconv.Itoa(len(responseSsz)))
	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	writer.WriteHeader(grpcResponse.StatusCode)
	if _, err := io.Copy(writer, ioutil.NopCloser(bytes.NewReader(responseSsz))); err != nil {
		e := fmt.Errorf("could not write response message: %w", err)
		return &gateway.DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
	}
	return nil
}
