package gateway

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/api/grpc"
	"github.com/wealdtech/go-bytesutil"
)

// DeserializeRequestBodyIntoContainer deserializes the request's body into an endpoint-specific struct.
func DeserializeRequestBodyIntoContainer(body io.Reader, requestContainer interface{}) ErrorJson {
	if err := json.NewDecoder(body).Decode(&requestContainer); err != nil {
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
	req.Body = ioutil.NopCloser(bytes.NewReader(j))
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
	return HandleQueryParameters(req, endpoint.RequestQueryParams)
}

// ProxyRequest proxies the request to grpc-gateway.
func ProxyRequest(req *http.Request) (*http.Response, ErrorJson) {
	// We do not use http.DefaultClient because it does not have any timeout.
	netClient := &http.Client{Timeout: time.Minute * 2}
	grpcResp, err := netClient.Do(req)
	if err != nil {
		return nil, InternalServerErrorWithMessage(err, "could not proxy request")
	}
	if grpcResp == nil {
		return nil, &DefaultErrorJson{Message: "nil response from gRPC-gateway", Code: http.StatusInternalServerError}
	}
	return grpcResp, nil
}

// ReadGrpcResponseBody reads the body from the grpc-gateway's response.
func ReadGrpcResponseBody(r io.Reader) ([]byte, ErrorJson) {
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, InternalServerErrorWithMessage(err, "could not read response body")
	}
	return body, nil
}

// DeserializeGrpcResponseBodyIntoErrorJson deserializes the body from the grpc-gateway's response into an error struct.
// The struct can be later examined to check if the request resulted in an error.
func DeserializeGrpcResponseBodyIntoErrorJson(errJson ErrorJson, body []byte) ErrorJson {
	if err := json.Unmarshal(body, errJson); err != nil {
		return InternalServerErrorWithMessage(err, "could not unmarshal error")
	}
	return nil
}

// HandleGrpcResponseError acts on an error that resulted from a grpc-gateway's response.
func HandleGrpcResponseError(errJson ErrorJson, resp *http.Response, w http.ResponseWriter) {
	// Something went wrong, but the request completed, meaning we can write headers and the error message.
	for h, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Set(h, v)
		}
	}
	// Set code to HTTP code because unmarshalled body contained gRPC code.
	errJson.SetCode(resp.StatusCode)
	WriteError(w, errJson, resp.Header)
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
			tag: "enum",
			f:   enumToLowercaseProcessor,
		},
		{
			tag: "time",
			f:   timeToUnixProcessor,
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
		if _, err := io.Copy(w, ioutil.NopCloser(bytes.NewReader(responseJson))); err != nil {
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
	if responseHeader != nil {
		customError, ok := responseHeader["Grpc-Metadata-"+grpc.CustomErrorMetadataKey]
		if ok {
			// Assume header has only one value and read the 0 index.
			if err := json.Unmarshal([]byte(customError[0]), errJson); err != nil {
				log.WithError(err).Error("Could not unmarshal custom error message")
				return
			}
		}
	}

	j, err := json.Marshal(errJson)
	if err != nil {
		log.WithError(err).Error("Could not marshal error message")
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(j)))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(errJson.StatusCode())
	if _, err := io.Copy(w, ioutil.NopCloser(bytes.NewReader(j))); err != nil {
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

// processField calls each processor function on any field that has the matching tag set.
// It is a recursive function.
func processField(s interface{}, processors []fieldProcessor) error {
	kind := reflect.TypeOf(s).Kind()
	if kind != reflect.Ptr && kind != reflect.Slice && kind != reflect.Array {
		return fmt.Errorf("processing fields of kind '%v' is unsupported", kind)
	}

	t := reflect.TypeOf(s).Elem()
	v := reflect.Indirect(reflect.ValueOf(s))

	for i := 0; i < t.NumField(); i++ {
		switch v.Field(i).Kind() {
		case reflect.Slice:
			sliceElem := t.Field(i).Type.Elem()
			kind := sliceElem.Kind()
			// Recursively process slices to struct pointers.
			if kind == reflect.Ptr && sliceElem.Elem().Kind() == reflect.Struct {
				for j := 0; j < v.Field(i).Len(); j++ {
					if err := processField(v.Field(i).Index(j).Interface(), processors); err != nil {
						return errors.Wrapf(err, "could not process field '%s'", t.Field(i).Name)
					}
				}
			}
			// Process each string in string slices.
			if kind == reflect.String {
				for _, proc := range processors {
					_, hasTag := t.Field(i).Tag.Lookup(proc.tag)
					if hasTag {
						for j := 0; j < v.Field(i).Len(); j++ {
							if err := proc.f(v.Field(i).Index(j)); err != nil {
								return errors.Wrapf(err, "could not process field '%s'", t.Field(i).Name)
							}
						}
					}
				}

			}
		// Recursively process struct pointers.
		case reflect.Ptr:
			if v.Field(i).Elem().Kind() == reflect.Struct {
				if err := processField(v.Field(i).Interface(), processors); err != nil {
					return errors.Wrapf(err, "could not process field '%s'", t.Field(i).Name)
				}
			}
		default:
			field := t.Field(i)
			for _, proc := range processors {
				if _, hasTag := field.Tag.Lookup(proc.tag); hasTag {
					if err := proc.f(v.Field(i)); err != nil {
						return errors.Wrapf(err, "could not process field '%s'", t.Field(i).Name)
					}
				}
			}
		}
	}
	return nil
}

func hexToBase64Processor(v reflect.Value) error {
	b, err := bytesutil.FromHexString(v.String())
	if err != nil {
		return err
	}
	v.SetString(base64.StdEncoding.EncodeToString(b))
	return nil
}

func base64ToHexProcessor(v reflect.Value) error {
	if v.String() == "" {
		return nil
	}
	b, err := base64.StdEncoding.DecodeString(v.String())
	if err != nil {
		return err
	}
	v.SetString(hexutil.Encode(b))
	return nil
}

func enumToLowercaseProcessor(v reflect.Value) error {
	v.SetString(strings.ToLower(v.String()))
	return nil
}

func timeToUnixProcessor(v reflect.Value) error {
	t, err := time.Parse(time.RFC3339, v.String())
	if err != nil {
		return err
	}
	v.SetString(strconv.FormatUint(uint64(t.Unix()), 10))
	return nil
}
