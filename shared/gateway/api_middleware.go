package gateway

import (
	"net/http"
	"reflect"

	"github.com/gorilla/mux"
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

// Endpoint is a representation of an API HTTP endpoint that should be proxied by the middleware.
type Endpoint struct {
	Path                  string         // The path of the HTTP endpoint.
	PostRequest           interface{}    // The struct corresponding to the JSON structure used in a POST request.
	GetRequestUrlLiterals []string       // Names of URL parameters that should not be base64-encoded.
	GetRequestQueryParams []QueryParam   // Query parameters of the GET request.
	GetResponse           interface{}    // The struct corresponding to the JSON structure used in a GET response.
	Err                   ErrorJson      // The struct corresponding to the error that should be returned in case of a request failure.
	Hooks                 HookCollection // A collection of functions that can be invoked at various stages of the request/response cycle.
}

// QueryParam represents a single query parameter's metadata.
type QueryParam struct {
	Name string
	Hex  bool
	Enum bool
}

// Hook is a function that can be invoked at various stages of the request/response cycle, leading to custom behaviour for a specific endpoint.
type Hook = func(endpoint Endpoint, writer http.ResponseWriter, request *http.Request) ErrorJson

// CustomHandler is a function that can be invoked at the very beginning of the request,
// essentially replacing the whole default request/response logic with custom logic for a specific endpoint.
type CustomHandler = func(m *ApiProxyMiddleware, endpoint Endpoint, writer http.ResponseWriter, request *http.Request) (handled bool)

// HookCollection contains handlers/hooks that can be used to amend the default request/response cycle with custom logic for a specific endpoint.
type HookCollection struct {
	CustomHandlers                            []CustomHandler
	OnPostStart                               []Hook
	OnPostDeserializeRequestBodyIntoContainer []Hook
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
	m.router.HandleFunc(endpoint.Path, func(writer http.ResponseWriter, request *http.Request) {
		for _, handler := range endpoint.Hooks.CustomHandlers {
			if handler(m, endpoint, writer, request) {
				return
			}
		}

		if request.Method == "POST" {
			for _, hook := range endpoint.Hooks.OnPostStart {
				if errJson := hook(endpoint, writer, request); errJson != nil {
					WriteError(writer, errJson, nil)
					return
				}
			}

			if errJson := DeserializeRequestBodyIntoContainer(request.Body, endpoint.PostRequest); errJson != nil {
				WriteError(writer, errJson, nil)
				return
			}
			for _, hook := range endpoint.Hooks.OnPostDeserializeRequestBodyIntoContainer {
				if errJson := hook(endpoint, writer, request); errJson != nil {
					WriteError(writer, errJson, nil)
					return
				}
			}

			if errJson := ProcessRequestContainerFields(endpoint.PostRequest); errJson != nil {
				WriteError(writer, errJson, nil)
				return
			}
			if errJson := SetRequestBodyToRequestContainer(endpoint.PostRequest, request); errJson != nil {
				WriteError(writer, errJson, nil)
				return
			}
		}

		if errJson := m.PrepareRequestForProxying(endpoint, request); errJson != nil {
			WriteError(writer, errJson, nil)
			return
		}
		grpcResponse, errJson := ProxyRequest(request)
		if errJson != nil {
			WriteError(writer, errJson, nil)
			return
		}
		grpcResponseBody, errJson := ReadGrpcResponseBody(grpcResponse.Body)
		if errJson != nil {
			WriteError(writer, errJson, nil)
			return
		}
		if errJson := DeserializeGrpcResponseBodyIntoErrorJson(endpoint.Err, grpcResponseBody); errJson != nil {
			WriteError(writer, errJson, nil)
			return
		}

		var responseJson []byte
		if endpoint.Err.Msg() != "" {
			HandleGrpcResponseError(endpoint.Err, grpcResponse, writer)
			return
		} else if !GrpcResponseIsStatusCodeOnly(request, endpoint.GetResponse) {
			if errJson := DeserializeGrpcResponseBodyIntoContainer(grpcResponseBody, endpoint.GetResponse); errJson != nil {
				WriteError(writer, errJson, nil)
				return
			}
			if errJson := ProcessMiddlewareResponseFields(endpoint.GetResponse); errJson != nil {
				WriteError(writer, errJson, nil)
				return
			}
			var errJson ErrorJson
			responseJson, errJson = SerializeMiddlewareResponseIntoJson(endpoint.GetResponse)
			if errJson != nil {
				WriteError(writer, errJson, nil)
				return
			}
		}

		if errJson := WriteMiddlewareResponseHeadersAndBody(request, grpcResponse, responseJson, writer); errJson != nil {
			WriteError(writer, errJson, nil)
			return
		}
		if errJson := Cleanup(grpcResponse.Body); errJson != nil {
			WriteError(writer, errJson, nil)
			return
		}
	})
}
