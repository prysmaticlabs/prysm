package apimiddleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/grpc"
)

// DeserializeRequestBodyIntoContainer deserializes the request's body into an endpoint-specific struct.
func DeserializeRequestBodyIntoContainer(body io.Reader, requestContainer interface{}) ErrorJson {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&requestContainer); err != nil {
		if strings.Contains(err.Error(), "json: unknown field") {
			e := errors.Wrap(err, "could not decode request body")
			return &DefaultErrorJson{
				Message: e.Error(),
				Code:    http.StatusBadRequest,
			}
		}
		return InternalServerErrorWithMessage(err, "could not decode request body")
	}
	return nil
}

// ProcessRequestContainerFields processes fields of an endpoint-specific container according to field tags.
func ProcessRequestContainerFields(requestContainer interface{}) ErrorJson {
	if err := processField(requestContainer, []fieldProcessor{
		{
			tag: "hex",
			f:   hexToBase64Processor,
		},
		{
			tag: "uint256",
			f:   uint256ToBase64Processor,
		},
	}); err != nil {
		return InternalServerErrorWithMessage(err, "could not process request data")
	}
	return nil
}

// SetRequestBodyToRequestContainer makes the endpoint-specific container the new body of the request.
func SetRequestBodyToRequestContainer(requestContainer interface{}, req *http.Request) ErrorJson {
	// Serialize the struct, which now includes a base64-encoded value, into JSON.
	j, err := json.Marshal(requestContainer)
	if err != nil {
		return InternalServerErrorWithMessage(err, "could not marshal request")
	}
	// Set the body to the new JSON.
	req.Body = io.NopCloser(bytes.NewReader(j))
	req.Header.Set("Content-Length", strconv.Itoa(len(j)))
	req.ContentLength = int64(len(j))
	return nil
}

// PrepareRequestForProxying applies additional logic to the request so that it can be correctly proxied to grpc-gateway.
func (m *ApiProxyMiddleware) PrepareRequestForProxying(endpoint Endpoint, req *http.Request) ErrorJson {
	req.URL.Scheme = "http"
	req.URL.Host = m.GatewayAddress
	req.RequestURI = ""
	if errJson := HandleURLParameters(endpoint.Path, req, endpoint.RequestURLLiterals); errJson != nil {
		return errJson
	}
	if errJson := HandleQueryParameters(req, endpoint.RequestQueryParams); errJson != nil {
		return errJson
	}
	// We have to add the prefix after handling parameters because adding the prefix changes URL segment indexing.
	req.URL.Path = "/internal" + req.URL.Path
	return nil
}

// ProxyRequest proxies the request to grpc-gateway.
func (m *ApiProxyMiddleware) ProxyRequest(req *http.Request) (*http.Response, ErrorJson) {
	// We do not use http.DefaultClient because it does not have any timeout.
	netClient := &http.Client{Timeout: m.Timeout}
	grpcResp, err := netClient.Do(req)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return nil, TimeoutError()
		}
		return nil, InternalServerErrorWithMessage(err, "could not proxy request")
	}
	if grpcResp == nil {
		return nil, &DefaultErrorJson{Message: "nil response from gRPC-gateway", Code: http.StatusInternalServerError}
	}
	return grpcResp, nil
}

// ReadGrpcResponseBody reads the body from the grpc-gateway's response.
func ReadGrpcResponseBody(r io.Reader) ([]byte, ErrorJson) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, InternalServerErrorWithMessage(err, "could not read response body")
	}
	return body, nil
}

// HandleGrpcResponseError acts on an error that resulted from a grpc-gateway's response.
// Whether there was an error is indicated by the bool return value. In case of an error,
// there is no need to write to the response because it's taken care of by the function.
func HandleGrpcResponseError(errJson ErrorJson, resp *http.Response, respBody []byte, w http.ResponseWriter) (bool, ErrorJson) {
	responseHasError := false
	if err := json.Unmarshal(respBody, errJson); err != nil {
		return false, InternalServerErrorWithMessage(err, "could not unmarshal error")
	}
	if errJson.Msg() != "" {
		responseHasError = true
		// Something went wrong, but the request completed, meaning we can write headers and the error message.
		for h, vs := range resp.Header {
			for _, v := range vs {
				w.Header().Set(h, v)
			}
		}
		// Handle gRPC timeout.
		if resp.StatusCode == http.StatusGatewayTimeout {
			WriteError(w, TimeoutError(), resp.Header)
		} else {
			// Set code to HTTP code because unmarshalled body contained gRPC code.
			errJson.SetCode(resp.StatusCode)
			WriteError(w, errJson, resp.Header)
		}
	}
	return responseHasError, nil
}

// GrpcResponseIsEmpty determines whether the grpc-gateway's response body contains no data.
func GrpcResponseIsEmpty(grpcResponseBody []byte) bool {
	return len(grpcResponseBody) == 0 || string(grpcResponseBody) == "{}"
}

// DeserializeGrpcResponseBodyIntoContainer deserializes the grpc-gateway's response body into an endpoint-specific struct.
func DeserializeGrpcResponseBodyIntoContainer(body []byte, responseContainer interface{}) ErrorJson {
	if err := json.Unmarshal(body, &responseContainer); err != nil {
		return InternalServerErrorWithMessage(err, "could not unmarshal response")
	}
	return nil
}

// ProcessMiddlewareResponseFields processes fields of an endpoint-specific container according to field tags.
func ProcessMiddlewareResponseFields(responseContainer interface{}) ErrorJson {
	if err := processField(responseContainer, []fieldProcessor{
		{
			tag: "hex",
			f:   base64ToHexProcessor,
		},
		{
			tag: "address",
			f:   base64ToChecksumAddressProcessor,
		},
		{
			tag: "enum",
			f:   enumToLowercaseProcessor,
		},
		{
			tag: "time",
			f:   timeToUnixProcessor,
		},
		{
			tag: "uint256",
			f:   base64ToUint256Processor,
		},
	}); err != nil {
		return InternalServerErrorWithMessage(err, "could not process response data")
	}
	return nil
}

// SerializeMiddlewareResponseIntoJson serializes the endpoint-specific response struct into a JSON representation.
func SerializeMiddlewareResponseIntoJson(responseContainer interface{}) (jsonResponse []byte, errJson ErrorJson) {
	j, err := json.Marshal(responseContainer)
	if err != nil {
		return nil, InternalServerErrorWithMessage(err, "could not marshal response")
	}
	return j, nil
}

// WriteMiddlewareResponseHeadersAndBody populates headers and the body of the final response.
func WriteMiddlewareResponseHeadersAndBody(grpcResp *http.Response, responseJson []byte, w http.ResponseWriter) ErrorJson {
	var statusCodeHeader string
	for h, vs := range grpcResp.Header {
		// We don't want to expose any gRPC metadata in the HTTP response, so we skip forwarding metadata headers.
		if strings.HasPrefix(h, "Grpc-Metadata") {
			if h == "Grpc-Metadata-"+grpc.HttpCodeMetadataKey {
				statusCodeHeader = vs[0]
			}
		} else {
			for _, v := range vs {
				w.Header().Set(h, v)
			}
		}
	}
	if !GrpcResponseIsEmpty(responseJson) {
		w.Header().Set("Content-Length", strconv.Itoa(len(responseJson)))
		if statusCodeHeader != "" {
			code, err := strconv.Atoi(statusCodeHeader)
			if err != nil {
				return InternalServerErrorWithMessage(err, "could not parse status code")
			}
			w.WriteHeader(code)
		} else {
			w.WriteHeader(grpcResp.StatusCode)
		}
		if _, err := io.Copy(w, io.NopCloser(bytes.NewReader(responseJson))); err != nil {
			return InternalServerErrorWithMessage(err, "could not write response message")
		}
	} else {
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(grpcResp.StatusCode)
	}
	return nil
}

// WriteError writes the error by manipulating headers and the body of the final response.
func WriteError(w http.ResponseWriter, errJson ErrorJson, responseHeader http.Header) {
	// Include custom error in the error JSON.
	hasCustomError := false
	if responseHeader != nil {
		customError, ok := responseHeader["Grpc-Metadata-"+grpc.CustomErrorMetadataKey]
		if ok {
			hasCustomError = true
			// Assume header has only one value and read the 0 index.
			if err := json.Unmarshal([]byte(customError[0]), errJson); err != nil {
				log.WithError(err).Error("Could not unmarshal custom error message")
				return
			}
		}
	}

	var j []byte
	if hasCustomError {
		var err error
		j, err = json.Marshal(errJson)
		if err != nil {
			log.WithError(err).Error("Could not marshal error message")
			return
		}
	} else {
		var err error
		// We marshal the response body into a DefaultErrorJson if the custom error is not present.
		// This is because the ErrorJson argument is the endpoint's error definition, which may contain custom fields.
		// In such a scenario marhaling the endpoint's error would populate the resulting JSON
		// with these fields even if they are not present in the gRPC header.
		d := &DefaultErrorJson{
			Message: errJson.Msg(),
			Code:    errJson.StatusCode(),
		}
		j, err = json.Marshal(d)
		if err != nil {
			log.WithError(err).Error("Could not marshal error message")
			return
		}
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(j)))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(errJson.StatusCode())
	if _, err := io.Copy(w, io.NopCloser(bytes.NewReader(j))); err != nil {
		log.WithError(err).Error("Could not write error message")
	}
}

// Cleanup performs final cleanup on the initial response from grpc-gateway.
func Cleanup(grpcResponseBody io.ReadCloser) ErrorJson {
	if err := grpcResponseBody.Close(); err != nil {
		return InternalServerErrorWithMessage(err, "could not close response body")
	}
	return nil
}
