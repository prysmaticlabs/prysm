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

func handleGetBeaconStateSSZ(m *gateway.ApiProxyMiddleware, endpoint gateway.Endpoint, writer http.ResponseWriter, request *http.Request) (handled bool) {
	config := sszConfig{
		sszPath:      "/eth/v1/debug/beacon/states/{state_id}/ssz",
		fileName:     "beacon_state.ssz",
		responseJson: &beaconStateSSZResponseJson{},
	}
	return handleGetSSZ(m, endpoint, writer, request, config)
}

func handleGetBeaconBlockSSZ(m *gateway.ApiProxyMiddleware, endpoint gateway.Endpoint, writer http.ResponseWriter, request *http.Request) (handled bool) {
	config := sszConfig{
		sszPath:      "/eth/v1/beacon/blocks/{block_id}/ssz",
		fileName:     "beacon_block.ssz",
		responseJson: &blockSSZResponseJson{},
	}
	return handleGetSSZ(m, endpoint, writer, request, config)
}

func handleGetSSZ(
	m *gateway.ApiProxyMiddleware,
	endpoint gateway.Endpoint,
	writer http.ResponseWriter,
	request *http.Request,
	config sszConfig,
) (handled bool) {
	if !sszRequested(request) {
		return false
	}

	if errJson := prepareSSZRequestForProxying(m, endpoint, request, config.sszPath); errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return true
	}
	grpcResponse, errJson := gateway.ProxyRequest(request)
	if errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return true
	}
	grpcResponseBody, errJson := gateway.ReadGrpcResponseBody(grpcResponse.Body)
	if errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return true
	}
	if errJson := gateway.DeserializeGrpcResponseBodyIntoErrorJson(endpoint.Err, grpcResponseBody); errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return true
	}
	if endpoint.Err.Msg() != "" {
		gateway.HandleGrpcResponseError(endpoint.Err, grpcResponse, writer)
		return true
	}
	if errJson := gateway.DeserializeGrpcResponseBodyIntoContainer(grpcResponseBody, config.responseJson); errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return true
	}
	responseSsz, errJson := serializeMiddlewareResponseIntoSSZ(config.responseJson.SSZData())
	if errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return true
	}
	if errJson := writeSSZResponseHeaderAndBody(grpcResponse, writer, responseSsz, config.fileName); errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return true
	}
	if errJson := gateway.Cleanup(grpcResponse.Body); errJson != nil {
		gateway.WriteError(writer, errJson, nil)
		return true
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

func prepareSSZRequestForProxying(m *gateway.ApiProxyMiddleware, endpoint gateway.Endpoint, request *http.Request, sszPath string) gateway.ErrorJson {
	request.URL.Scheme = "http"
	request.URL.Host = m.GatewayAddress
	request.RequestURI = ""
	request.URL.Path = sszPath
	return gateway.HandleURLParameters(endpoint.Path, request, []string{})
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

func writeSSZResponseHeaderAndBody(grpcResponse *http.Response, writer http.ResponseWriter, responseSsz []byte, fileName string) gateway.ErrorJson {
	var statusCodeHeader string
	for h, vs := range grpcResponse.Header {
		// We don't want to expose any gRPC metadata in the HTTP response, so we skip forwarding metadata headers.
		if strings.HasPrefix(h, "Grpc-Metadata") {
			if h == "Grpc-Metadata-"+grpcutils.HttpCodeMetadataKey {
				statusCodeHeader = vs[0]
			}
		} else {
			for _, v := range vs {
				writer.Header().Set(h, v)
			}
		}
	}
	if statusCodeHeader != "" {
		code, err := strconv.Atoi(statusCodeHeader)
		if err != nil {
			e := errors.Wrapf(err, "could not parse status code")
			return &gateway.DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
		}
		writer.WriteHeader(code)
	} else {
		writer.WriteHeader(grpcResponse.StatusCode)
	}
	writer.Header().Set("Content-Length", strconv.Itoa(len(responseSsz)))
	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	writer.WriteHeader(grpcResponse.StatusCode)
	if _, err := io.Copy(writer, ioutil.NopCloser(bytes.NewReader(responseSsz))); err != nil {
		e := errors.Wrapf(err, "could not write response message")
		return &gateway.DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
	}
	return nil
}
