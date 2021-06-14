package apimiddleware

import (
	"bytes"
	"encoding/base64"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/gateway"
	"github.com/prysmaticlabs/prysm/shared/grpcutils"
)

type sszConfig struct {
	sszPath      string
	fileName     string
	responseJson sszResponseJson
}

func handleGetBeaconStateSSZ(m *gateway.ApiProxyMiddleware, endpoint gateway.Endpoint, w http.ResponseWriter, req *http.Request) (handled bool) {
	config := sszConfig{
		sszPath:      "/eth/v1/debug/beacon/states/{state_id}/ssz",
		fileName:     "beacon_state.ssz",
		responseJson: &beaconStateSSZResponseJson{},
	}
	return handleGetSSZ(m, endpoint, w, req, config)
}

func handleGetBeaconBlockSSZ(m *gateway.ApiProxyMiddleware, endpoint gateway.Endpoint, w http.ResponseWriter, req *http.Request) (handled bool) {
	config := sszConfig{
		sszPath:      "/eth/v1/beacon/blocks/{block_id}/ssz",
		fileName:     "beacon_block.ssz",
		responseJson: &blockSSZResponseJson{},
	}
	return handleGetSSZ(m, endpoint, w, req, config)
}

func handleGetSSZ(
	m *gateway.ApiProxyMiddleware,
	endpoint gateway.Endpoint,
	w http.ResponseWriter,
	req *http.Request,
	config sszConfig,
) (handled bool) {
	if !sszRequested(req) {
		return false
	}

	if errJson := prepareSSZRequestForProxying(m, endpoint, req, config.sszPath); errJson != nil {
		gateway.WriteError(w, errJson, nil)
		return true
	}
	grpcResponse, errJson := gateway.ProxyRequest(req)
	if errJson != nil {
		gateway.WriteError(w, errJson, nil)
		return true
	}
	grpcResponseBody, errJson := gateway.ReadGrpcResponseBody(grpcResponse.Body)
	if errJson != nil {
		gateway.WriteError(w, errJson, nil)
		return true
	}
	if errJson := gateway.DeserializeGrpcResponseBodyIntoErrorJson(endpoint.Err, grpcResponseBody); errJson != nil {
		gateway.WriteError(w, errJson, nil)
		return true
	}
	if endpoint.Err.Msg() != "" {
		gateway.HandleGrpcResponseError(endpoint.Err, grpcResponse, w)
		return true
	}
	if errJson := gateway.DeserializeGrpcResponseBodyIntoContainer(grpcResponseBody, config.responseJson); errJson != nil {
		gateway.WriteError(w, errJson, nil)
		return true
	}
	responseSsz, errJson := serializeMiddlewareResponseIntoSSZ(config.responseJson.SSZData())
	if errJson != nil {
		gateway.WriteError(w, errJson, nil)
		return true
	}
	if errJson := writeSSZResponseHeaderAndBody(grpcResponse, w, responseSsz, config.fileName); errJson != nil {
		gateway.WriteError(w, errJson, nil)
		return true
	}
	if errJson := gateway.Cleanup(grpcResponse.Body); errJson != nil {
		gateway.WriteError(w, errJson, nil)
		return true
	}

	return true
}

func sszRequested(req *http.Request) bool {
	accept, ok := req.Header["Accept"]
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

func prepareSSZRequestForProxying(m *gateway.ApiProxyMiddleware, endpoint gateway.Endpoint, req *http.Request, sszPath string) gateway.ErrorJson {
	req.URL.Scheme = "http"
	req.URL.Host = m.GatewayAddress
	req.RequestURI = ""
	req.URL.Path = sszPath
	return gateway.HandleURLParameters(endpoint.Path, req, []string{})
}

func serializeMiddlewareResponseIntoSSZ(data string) (sszResponse []byte, errJson gateway.ErrorJson) {
	// Serialize the SSZ part of the deserialized value.
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		e := errors.Wrapf(err, "could not decode response body into base64")
		return nil, &gateway.DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
	}
	return b, nil
}

func writeSSZResponseHeaderAndBody(grpcResp *http.Response, w http.ResponseWriter, responseSsz []byte, fileName string) gateway.ErrorJson {
	var statusCodeHeader string
	for h, vs := range grpcResp.Header {
		// We don't want to expose any gRPC metadata in the HTTP response, so we skip forwarding metadata headers.
		if strings.HasPrefix(h, "Grpc-Metadata") {
			if h == "Grpc-Metadata-"+grpcutils.HttpCodeMetadataKey {
				statusCodeHeader = vs[0]
			}
		} else {
			for _, v := range vs {
				w.Header().Set(h, v)
			}
		}
	}
	if statusCodeHeader != "" {
		code, err := strconv.Atoi(statusCodeHeader)
		if err != nil {
			e := errors.Wrapf(err, "could not parse status code")
			return &gateway.DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
		}
		w.WriteHeader(code)
	} else {
		w.WriteHeader(grpcResp.StatusCode)
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(responseSsz)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.WriteHeader(grpcResp.StatusCode)
	if _, err := io.Copy(w, ioutil.NopCloser(bytes.NewReader(responseSsz))); err != nil {
		e := errors.Wrapf(err, "could not write response message")
		return &gateway.DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
	}
	return nil
}
