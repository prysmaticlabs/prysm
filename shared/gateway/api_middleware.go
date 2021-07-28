package gateway

import (
	"net/http"
	"reflect"

	"github.com/gorilla/mux"
)

// ApiProxyMiddleware is a proxy between an Ethereum consensus API HTTP client and grpc-gateway.
// The purpose of the proxy is to handle HTTP requests and gRPC responses in such a way that:
//   - Ethereum consensus API requests can be handled by grpc-gateway correctly
//   - gRPC responses can be returned as spec-compliant Ethereum consensus API responses
type ApiProxyMiddleware struct {
	GatewayAddress  string
	ProxyAddress    string
	EndpointCreator EndpointFactory
	router          *mux.Router
}

// EndpointFactory is responsible for creating new instances of Endpoint values.
type EndpointFactory interface {
	Create(path string) (*Endpoint, error)
	Paths() []string
	IsNil() bool
}

// Endpoint is a representation of an API HTTP endpoint that should be proxied by the middleware.
type Endpoint struct {
	Path               string         // The path of the HTTP endpoint.
	PostRequest        interface{}    // The struct corresponding to the JSON structure used in a POST request.
	PostResponse       interface{}    // The struct corresponding to the JSON structure used in a POST response.
	RequestURLLiterals []string       // Names of URL parameters that should not be base64-encoded.
	RequestQueryParams []QueryParam   // Query parameters of the request.
	GetResponse        interface{}    // The struct corresponding to the JSON structure used in a GET response.
	Err                ErrorJson      // The struct corresponding to the error that should be returned in case of a request failure.
	Hooks              HookCollection // A collection of functions that can be invoked at various stages of the request/response cycle.
}

// QueryParam represents a single query parameter's metadata.
type QueryParam struct {
	Name string
	Hex  bool
	Enum bool
}

// Hook is a function that can be invoked at various stages of the request/response cycle, leading to custom behaviour for a specific endpoint.
type Hook = func(endpoint Endpoint, w http.ResponseWriter, req *http.Request) ErrorJson

// CustomHandler is a function that can be invoked at the very beginning of the request,
// essentially replacing the whole default request/response logic with custom logic for a specific endpoint.
type CustomHandler = func(m *ApiProxyMiddleware, endpoint Endpoint, w http.ResponseWriter, req *http.Request) (handled bool)

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

	for _, path := range m.EndpointCreator.Paths() {
		m.handleApiPath(path, m.EndpointCreator)
	}

	return http.ListenAndServe(m.ProxyAddress, m.router)
}

func (m *ApiProxyMiddleware) handleApiPath(path string, endpointFactory EndpointFactory) {
	m.router.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		endpoint, err := endpointFactory.Create(path)
		if err != nil {
			errJson := InternalServerErrorWithMessage(err, "could not create endpoint")
			WriteError(w, errJson, nil)
		}

		for _, handler := range endpoint.Hooks.CustomHandlers {
			if handler(m, *endpoint, w, req) {
				return
			}
		}

		if req.Method == "POST" {
			for _, hook := range endpoint.Hooks.OnPostStart {
				if errJson := hook(*endpoint, w, req); errJson != nil {
					WriteError(w, errJson, nil)
					return
				}
			}

			if errJson := DeserializeRequestBodyIntoContainer(req.Body, endpoint.PostRequest); errJson != nil {
				WriteError(w, errJson, nil)
				return
			}
			for _, hook := range endpoint.Hooks.OnPostDeserializeRequestBodyIntoContainer {
				if errJson := hook(*endpoint, w, req); errJson != nil {
					WriteError(w, errJson, nil)
					return
				}
			}

			if errJson := ProcessRequestContainerFields(endpoint.PostRequest); errJson != nil {
				WriteError(w, errJson, nil)
				return
			}
			if errJson := SetRequestBodyToRequestContainer(endpoint.PostRequest, req); errJson != nil {
				WriteError(w, errJson, nil)
				return
			}
		}

		if errJson := m.PrepareRequestForProxying(*endpoint, req); errJson != nil {
			WriteError(w, errJson, nil)
			return
		}
		grpcResponse, errJson := ProxyRequest(req)
		if errJson != nil {
			WriteError(w, errJson, nil)
			return
		}
		grpcResponseBody, errJson := ReadGrpcResponseBody(grpcResponse.Body)
		if errJson != nil {
			WriteError(w, errJson, nil)
			return
		}
		if errJson := DeserializeGrpcResponseBodyIntoErrorJson(endpoint.Err, grpcResponseBody); errJson != nil {
			WriteError(w, errJson, nil)
			return
		}

		var responseJson []byte
		if endpoint.Err.Msg() != "" {
			HandleGrpcResponseError(endpoint.Err, grpcResponse, w)
			return
		} else if !GrpcResponseIsStatusCodeOnly(endpoint.GetResponse) {
			var response interface{}
			if req.Method == "GET" {
				response = endpoint.GetResponse
			} else {
				response = endpoint.PostResponse
			}
			if errJson := DeserializeGrpcResponseBodyIntoContainer(grpcResponseBody, response); errJson != nil {
				WriteError(w, errJson, nil)
				return
			}
			if errJson := ProcessMiddlewareResponseFields(response); errJson != nil {
				WriteError(w, errJson, nil)
				return
			}
			var errJson ErrorJson
			responseJson, errJson = SerializeMiddlewareResponseIntoJson(response)
			if errJson != nil {
				WriteError(w, errJson, nil)
				return
			}
		}

		if errJson := WriteMiddlewareResponseHeadersAndBody(req, grpcResponse, responseJson, w); errJson != nil {
			WriteError(w, errJson, nil)
			return
		}
		if errJson := Cleanup(grpcResponse.Body); errJson != nil {
			WriteError(w, errJson, nil)
			return
		}
	})
}
