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
	"github.com/gorilla/mux"
	butil "github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/grpcutils"
	"github.com/wealdtech/go-bytesutil"
)

// ApiProxyMiddleware is a proxy between an Eth2 API HTTP client and grpc-gateway.
// The purpose of the proxy is to handle HTTP requests and gRPC responses in such a way that:
//   - Eth2 API requests can be handled by grpc-gateway correctly
//   - gRPC responses can be returned as spec-compliant Eth2 API responses
type ApiProxyMiddleware struct {
	GatewayAddress string
	ProxyAddress   string
	Endpoints      []Endpoint
	router         *mux.Router
}

// TODO: documentation
type Endpoint struct {
	Url                   string
	PostRequest           interface{}
	GetRequestUrlLiterals []string // Names of URL parameters that should not be base64-encoded.
	GetRequestQueryParams []QueryParam
	GetResponse           interface{}
	Err                   ErrorJson
	Hooks                 HookCollection
}

// TODO: Documentation
type QueryParam struct {
	Name string
	Hex  bool
	Enum bool
}

// TODO: Documentation
type Hook = func(endpoint Endpoint, writer http.ResponseWriter, request *http.Request) ErrorJson

// TODO: Documentation
type HookCollection struct {
	OnPostStart                                []Hook
	OnPostDeserializedRequestBodyIntoContainer []Hook
}

// fieldProcessor applies the processing function f to a value when the tag is present on the field.
type fieldProcessor struct {
	tag string
	f   func(value reflect.Value) error
}

// Run starts the proxy, registering all proxy endpoints on ApiProxyMiddleware.ProxyAddress.
func (m *ApiProxyMiddleware) Run() error {
	m.router = mux.NewRouter()

	for _, endpoint := range m.Endpoints {
		m.handleApiEndpoint(endpoint)
	}

	return http.ListenAndServe(m.ProxyAddress, m.router)
}

func (m *ApiProxyMiddleware) handleApiEndpoint(endpoint Endpoint) {
	m.router.HandleFunc(endpoint.Url, func(writer http.ResponseWriter, request *http.Request) {
		/*if beaconStateSszRequested(endpoint.Url, request) {
			m.handleGetBeaconStateSsz(endpoint, writer, request)
			return
		}*/

		if request.Method == "POST" {
			for _, hook := range endpoint.Hooks.OnPostStart {
				if err := hook(endpoint, writer, request); err != nil {
					writeError(writer, err, nil)
					return
				}
			}

			if err := deserializeRequestBodyIntoContainer(request.Body, endpoint.PostRequest); err != nil {
				writeError(writer, err, nil)
				return
			}

			for _, hook := range endpoint.Hooks.OnPostDeserializedRequestBodyIntoContainer {
				if err := hook(endpoint, writer, request); err != nil {
					writeError(writer, err, nil)
					return
				}
			}

			if err := processRequestContainerFields(endpoint.PostRequest); err != nil {
				writeError(writer, err, nil)
				return
			}
			if err := setRequestBodyToRequestContainer(endpoint.PostRequest, request); err != nil {
				writeError(writer, err, nil)
				return
			}
		}

		if err := m.prepareRequestForProxying(endpoint, request); err != nil {
			writeError(writer, err, nil)
			return
		}
		grpcResponse, err := proxyRequest(request)
		if err != nil {
			writeError(writer, err, nil)
			return
		}
		grpcResponseBody, err := readGrpcResponseBody(grpcResponse.Body)
		if err != nil {
			writeError(writer, err, nil)
			return
		}
		if err := deserializeGrpcResponseBodyIntoErrorJson(endpoint.Err, grpcResponseBody); err != nil {
			writeError(writer, err, nil)
			return
		}

		var responseJson []byte
		if endpoint.Err.Msg() != "" {
			handleGrpcResponseError(endpoint.Err, grpcResponse, writer)
			return
		} else if !grpcResponseIsStatusCodeOnly(request, endpoint.GetResponse) {
			if err := deserializeGrpcResponseBodyIntoContainer(grpcResponseBody, endpoint.GetResponse); err != nil {
				writeError(writer, err, nil)
				return
			}
			if err := processMiddlewareResponseFields(endpoint.GetResponse); err != nil {
				writeError(writer, err, nil)
				return
			}
			var errJson ErrorJson
			responseJson, errJson = serializeMiddlewareResponseIntoJson(endpoint.GetResponse)
			if errJson != nil {
				writeError(writer, errJson, nil)
				return
			}
		}

		if err := writeMiddlewareResponseHeadersAndBody(request, grpcResponse, responseJson, writer); err != nil {
			writeError(writer, err, nil)
			return
		}

		cleanup(grpcResponse.Body)
	})
}

/*func (m *ApiProxyMiddleware) handleGetBeaconStateSsz(endpoint Endpoint, writer http.ResponseWriter, request *http.Request) {
	// Prepare request values for proxying.
	request.URL.Scheme = "http"
	request.URL.Host = m.GatewayAddress
	request.RequestURI = ""
	request.URL.Path = "/eth/v1/debug/beacon/states/{state_id}/ssz"
	handleUrlParameters(endpoint.Url, request, writer, []string{})

	// Proxy the request to grpc-gateway.
	grpcResp, err := http.DefaultClient.Do(request)
	if err != nil {
		e := fmt.Errorf("could not proxy request: %w", err)
		writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
		return
	}
	if grpcResp == nil {
		writeError(writer, &DefaultErrorJson{Message: "nil response from gRPC-gateway", Code: http.StatusInternalServerError}, nil)
		return
	}

	// Read the body.
	body, err := ioutil.ReadAll(grpcResp.Body)
	if err != nil {
		e := fmt.Errorf("could not read response body: %w", err)
		writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
		return
	}

	// Deserialize the output of grpc-gateway into the error struct.
	errorJson := &DefaultErrorJson{}
	if err := json.Unmarshal(body, errorJson); err != nil {
		e := fmt.Errorf("could not unmarshal error: %w", err)
		writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
		return
	}
	if errorJson.Msg() != "" {
		// Something went wrong, but the request completed, meaning we can write headers and the error message.
		for h, vs := range grpcResp.Header {
			for _, v := range vs {
				writer.Header().Set(h, v)
			}
		}
		// Set code to HTTP code because unmarshalled body contained gRPC code.
		errorJson.SetCode(grpcResp.StatusCode)
		writeError(writer, errorJson, grpcResp.Header)
		return
		// Don't do anything if the response is only a status code.
	}

	// Deserialize the output of grpc-gateway.
	responseJson := &BeaconStateSszResponseJson{}
	if err := json.Unmarshal(body, responseJson); err != nil {
		e := fmt.Errorf("could not unmarshal response: %w", err)
		writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
		return
	}
	// Serialize the SSZ part of the deserialized value.
	b, err := base64.StdEncoding.DecodeString(responseJson.Data)
	if err != nil {
		e := fmt.Errorf("could not decode response body into base64: %w", err)
		writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
		return
	}

	// Write the response (headers + body) and PROFIT!
	for h, vs := range grpcResp.Header {
		for _, v := range vs {
			writer.Header().Set(h, v)
		}
	}
	writer.Header().Set("Content-Length", strconv.Itoa(len(b)))
	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.Header().Set("Content-Disposition", "attachment; filename=beacon_state_ssz")
	writer.WriteHeader(grpcResp.StatusCode)
	if _, err := io.Copy(writer, ioutil.NopCloser(bytes.NewReader(b))); err != nil {
		e := fmt.Errorf("could not write response message: %w", err)
		writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
		return
	}

	// Final cleanup.
	if err := grpcResp.Body.Close(); err != nil {
		e := fmt.Errorf("could not close response body: %w", err)
		writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
		return
	}
}

func beaconStateSszRequested(url string, request *http.Request) bool {
	return url == "/eth/v1/debug/beacon/states/{state_id}" && request.Header["Accept"][0] == "application/octet-stream"
}*/

// TODO: Wszystkie funkcje powinny kończyć cały request, jeżeli wystąpił błąd. W tej chwili return tylko wychodzi z danej funkcji.
func deserializeRequestBodyIntoContainer(body io.ReadCloser, requestContainer interface{}) ErrorJson {
	if err := json.NewDecoder(body).Decode(&requestContainer); err != nil {
		e := fmt.Errorf("could not decode request body: %w", err)
		return &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
	}
	return nil
}

func processRequestContainerFields(requestContainer interface{}) ErrorJson {
	if err := processField(requestContainer, []fieldProcessor{
		{
			tag: "hex",
			f:   hexToBase64Processor,
		},
	}); err != nil {
		e := fmt.Errorf("could not process request data: %w", err)
		return &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
	}
	return nil
}

func setRequestBodyToRequestContainer(requestContainer interface{}, request *http.Request) ErrorJson {
	// Serialize the struct, which now includes a base64-encoded value, into JSON.
	j, err := json.Marshal(requestContainer)
	if err != nil {
		e := fmt.Errorf("could not marshal request: %w", err)
		return &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
	}
	// Set the body to the new JSON.
	request.Body = ioutil.NopCloser(bytes.NewReader(j))
	request.Header.Set("Content-Length", strconv.Itoa(len(j)))
	request.ContentLength = int64(len(j))
	return nil
}

func (m *ApiProxyMiddleware) prepareRequestForProxying(endpoint Endpoint, request *http.Request) ErrorJson {
	request.URL.Scheme = "http"
	request.URL.Host = m.GatewayAddress
	request.RequestURI = ""
	if err := handleUrlParameters(endpoint.Url, request, endpoint.GetRequestUrlLiterals); err != nil {
		return err
	}
	if err := handleQueryParameters(request, endpoint.GetRequestQueryParams); err != nil {
		return err
	}
	return nil
}

// handleUrlParameters processes URL parameters, allowing parameterized URLs to be safely and correctly proxied to grpc-gateway.
func handleUrlParameters(url string, request *http.Request, literals []string) ErrorJson {
	segments := strings.Split(url, "/")

segmentsLoop:
	for i, s := range segments {
		// We only care about segments which are parameterized.
		if isRequestParam(s) {
			// Don't do anything with parameters which should be forwarded literally to gRPC.
			for _, l := range literals {
				if s == "{"+l+"}" {
					continue segmentsLoop
				}
			}

			routeVar := mux.Vars(request)[s[1:len(s)-1]]
			bRouteVar := []byte(routeVar)
			isHex, err := butil.IsHex(bRouteVar)
			if err != nil {
				e := fmt.Errorf("could not process URL parameter: %w", err)
				return &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
			}
			if isHex {
				bRouteVar, err = bytesutil.FromHexString(string(bRouteVar))
				if err != nil {
					e := fmt.Errorf("could not process URL parameter: %w", err)
					return &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
				}
			}
			// Converting hex to base64 may result in a value which malforms the URL.
			// We use URLEncoding to safely escape such values.
			base64RouteVar := base64.URLEncoding.EncodeToString(bRouteVar)

			// Merge segments back into the full URL.
			splitPath := strings.Split(request.URL.Path, "/")
			splitPath[i] = base64RouteVar
			request.URL.Path = strings.Join(splitPath, "/")
		}
	}
	return nil
}

// handleQueryParameters processes query parameters, allowing them to be safely and correctly proxied to grpc-gateway.
func handleQueryParameters(request *http.Request, params []QueryParam) ErrorJson {
	queryParams := request.URL.Query()

	for key, vals := range queryParams {
		for _, p := range params {
			if key == p.Name {
				if p.Hex {
					queryParams.Del(key)
					for _, v := range vals {
						b := []byte(v)
						isHex, err := butil.IsHex(b)
						if err != nil {
							e := fmt.Errorf("could not process query parameter: %w", err)
							return &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
						}
						if isHex {
							b, err = bytesutil.FromHexString(v)
							if err != nil {
								e := fmt.Errorf("could not process query parameter: %w", err)
								return &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
							}
						}
						queryParams.Add(key, base64.URLEncoding.EncodeToString(b))
					}
				}
				if p.Enum {
					queryParams.Del(key)
					for _, v := range vals {
						// gRPC expectes uppercase enum values.
						queryParams.Add(key, strings.ToUpper(v))
					}
				}
			}
		}
	}
	request.URL.RawQuery = queryParams.Encode()
	return nil
}

func proxyRequest(request *http.Request) (*http.Response, ErrorJson) {
	grpcResp, err := http.DefaultClient.Do(request)
	if err != nil {
		e := fmt.Errorf("could not proxy request: %w", err)
		return nil, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
	}
	if grpcResp == nil {
		return nil, &DefaultErrorJson{Message: "nil response from gRPC-gateway", Code: http.StatusInternalServerError}
	}
	return grpcResp, nil
}

func readGrpcResponseBody(reader io.Reader) ([]byte, ErrorJson) {
	body, err := ioutil.ReadAll(reader)
	if err != nil {
		e := fmt.Errorf("could not read response body: %w", err)
		return nil, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
	}
	return body, nil
}

func deserializeGrpcResponseBodyIntoErrorJson(errJson ErrorJson, body []byte) ErrorJson {
	if err := json.Unmarshal(body, errJson); err != nil {
		e := fmt.Errorf("could not unmarshal error: %w", err)
		return &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
	}
	return nil
}

func handleGrpcResponseError(errJson ErrorJson, response *http.Response, writer http.ResponseWriter) {
	// Something went wrong, but the request completed, meaning we can write headers and the error message.
	for h, vs := range response.Header {
		for _, v := range vs {
			writer.Header().Set(h, v)
		}
	}
	// Set code to HTTP code because unmarshalled body contained gRPC code.
	errJson.SetCode(response.StatusCode)
	writeError(writer, errJson, response.Header)
}

func grpcResponseIsStatusCodeOnly(request *http.Request, responseContainer interface{}) bool {
	return request.Method == "GET" && responseContainer == nil
}

func deserializeGrpcResponseBodyIntoContainer(body []byte, responseContainer interface{}) ErrorJson {
	if err := json.Unmarshal(body, &responseContainer); err != nil {
		e := fmt.Errorf("could not unmarshal response: %w", err)
		return &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
	}
	return nil
}

func processMiddlewareResponseFields(responseContainer interface{}) ErrorJson {
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
		e := fmt.Errorf("could not process response data: %w", err)
		return &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
	}
	return nil
}

func serializeMiddlewareResponseIntoJson(responseContainer interface{}) (jsonResponse []byte, errJson ErrorJson) {
	j, err := json.Marshal(responseContainer)
	if err != nil {
		e := fmt.Errorf("could not marshal response: %w", err)
		return nil, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
	}
	return j, nil
}

func writeMiddlewareResponseHeadersAndBody(request *http.Request, grpcResponse *http.Response, responseJson []byte, writer http.ResponseWriter) ErrorJson {
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
	if request.Method == "GET" {
		writer.Header().Set("Content-Length", strconv.Itoa(len(responseJson)))
		if statusCodeHeader != "" {
			code, err := strconv.Atoi(statusCodeHeader)
			if err != nil {
				e := fmt.Errorf("could not parse status code: %w", err)
				return &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
			}
			writer.WriteHeader(code)
		} else {
			writer.WriteHeader(grpcResponse.StatusCode)
		}
		if _, err := io.Copy(writer, ioutil.NopCloser(bytes.NewReader(responseJson))); err != nil {
			e := fmt.Errorf("could not write response message: %w", err)
			return &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
		}
	} else if request.Method == "POST" {
		writer.WriteHeader(grpcResponse.StatusCode)
	}
	return nil
}

func writeError(writer http.ResponseWriter, e ErrorJson, responseHeader http.Header) {
	// Include custom error in the error JSON.
	if responseHeader != nil {
		customError, ok := responseHeader["Grpc-Metadata-"+grpcutils.CustomErrorMetadataKey]
		if ok {
			// Assume header has only one value and read the 0 index.
			if err := json.Unmarshal([]byte(customError[0]), e); err != nil {
				log.WithError(err).Error("Could not unmarshal custom error message")
				return
			}
		}
	}

	j, err := json.Marshal(e)
	if err != nil {
		log.WithError(err).Error("Could not marshal error message")
		return
	}

	writer.Header().Set("Content-Length", strconv.Itoa(len(j)))
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(e.StatusCode())
	if _, err := io.Copy(writer, ioutil.NopCloser(bytes.NewReader(j))); err != nil {
		log.WithError(err).Error("Could not write error message")
	}
}

// processField calls each processor function on any field that has the matching tag set.
// It is a recursive function.
func processField(s interface{}, processors []fieldProcessor) error {
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
						return fmt.Errorf("could not process field '%s': %w", t.Field(i).Name, err)
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
								return fmt.Errorf("could not process field '%s': %w", t.Field(i).Name, err)
							}
						}
					}
				}

			}
		// Recursively process struct pointers.
		case reflect.Ptr:
			if v.Field(i).Elem().Kind() == reflect.Struct {
				if err := processField(v.Field(i).Interface(), processors); err != nil {
					return fmt.Errorf("could not process field '%s': %w", t.Field(i).Name, err)
				}
			}
		default:
			field := t.Field(i)
			for _, proc := range processors {
				if _, hasTag := field.Tag.Lookup(proc.tag); hasTag {
					if err := proc.f(v.Field(i)); err != nil {
						return fmt.Errorf("could not process field '%s': %w", t.Field(i).Name, err)
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

// isRequestParam verifies whether the passed string is a request parameter.
// Request parameters are enclosed in { and }.
func isRequestParam(s string) bool {
	return len(s) > 0 && s[0] == '{' && s[len(s)-1] == '}'
}

func cleanup(grpcResponseBody io.ReadCloser) ErrorJson {
	if err := grpcResponseBody.Close(); err != nil {
		e := fmt.Errorf("could not close response body: %w", err)
		return &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}
	}
	return nil
}
