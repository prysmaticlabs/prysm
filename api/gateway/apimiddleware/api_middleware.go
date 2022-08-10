package apimiddleware

import (
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/mux"
)

// ApiProxyMiddleware is a proxy between an Ethereum consensus API HTTP client and grpc-gateway.
// The purpose of the proxy is to handle HTTP requests and gRPC responses in such a way that:
//   - Ethereum consensus API requests can be handled by grpc-gateway correctly
//   - gRPC responses can be returned as spec-compliant Ethereum consensus API responses
type ApiProxyMiddleware struct {
	GatewayAddress  string
	EndpointCreator EndpointFactory
	Timeout         time.Duration
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
	Path               string          // The path of the HTTP endpoint.
	GetResponse        interface{}     // The struct corresponding to the JSON structure used in a GET response.
	PostRequest        interface{}     // The struct corresponding to the JSON structure used in a POST request.
	PostResponse       interface{}     // The struct corresponding to the JSON structure used in a POST response.
	DeleteRequest      interface{}     // The struct corresponding to the JSON structure used in a DELETE request.
	DeleteResponse     interface{}     // The struct corresponding to the JSON structure used in a DELETE response.
	RequestURLLiterals []string        // Names of URL parameters that should not be base64-encoded.
	RequestQueryParams []QueryParam    // Query parameters of the request.
	Err                ErrorJson       // The struct corresponding to the error that should be returned in case of a request failure.
	Hooks              HookCollection  // A collection of functions that can be invoked at various stages of the request/response cycle.
	CustomHandlers     []CustomHandler // Functions that will be executed instead of the default request/response behaviour.
}

// RunDefault expresses whether the default processing logic should be carried out after running a pre hook.
type RunDefault bool

// DefaultEndpoint returns an Endpoint with default configuration, e.g. DefaultErrorJson for error handling.
func DefaultEndpoint() Endpoint {
	return Endpoint{
		Err: &DefaultErrorJson{},
	}
}

// QueryParam represents a single query parameter's metadata.
type QueryParam struct {
	Name string
	Hex  bool
	Enum bool
}

// CustomHandler is a function that can be invoked at the very beginning of the request,
// essentially replacing the whole default request/response logic with custom logic for a specific endpoint.
type CustomHandler = func(m *ApiProxyMiddleware, endpoint Endpoint, w http.ResponseWriter, req *http.Request) (handled bool)

// HookCollection contains hooks that can be used to amend the default request/response cycle with custom logic for a specific endpoint.
type HookCollection struct {
	OnPreDeserializeRequestBodyIntoContainer      func(endpoint *Endpoint, w http.ResponseWriter, req *http.Request) (RunDefault, ErrorJson)
	OnPostDeserializeRequestBodyIntoContainer     func(endpoint *Endpoint, w http.ResponseWriter, req *http.Request) ErrorJson
	OnPreDeserializeGrpcResponseBodyIntoContainer func([]byte, interface{}) (RunDefault, ErrorJson)
	OnPreSerializeMiddlewareResponseIntoJson      func(interface{}) (RunDefault, []byte, ErrorJson)
}

// fieldProcessor applies the processing function f to a value when the tag is present on the field.
type fieldProcessor struct {
	tag string
	f   func(value reflect.Value) error
}

// Run starts the proxy, registering all proxy endpoints.
func (m *ApiProxyMiddleware) Run(gatewayRouter *mux.Router) {
	for _, path := range m.EndpointCreator.Paths() {
		gatewayRouter.HandleFunc(path, m.WithMiddleware(path))
	}
	m.router = gatewayRouter
}

// ServeHTTP for the proxy middleware.
func (m *ApiProxyMiddleware) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	m.router.ServeHTTP(w, req)
}

// WithMiddleware wraps the given endpoint handler with the middleware logic.
func (m *ApiProxyMiddleware) WithMiddleware(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		endpoint, err := m.EndpointCreator.Create(path)
		if err != nil {
			log.WithError(err).Errorf("Could not create endpoint for path: %s", path)
			return
		}

		for _, handler := range endpoint.CustomHandlers {
			if handler(m, *endpoint, w, req) {
				return
			}
		}

		if req.Method == "POST" {
			if errJson := handlePostRequestForEndpoint(endpoint, w, req); errJson != nil {
				WriteError(w, errJson, nil)
				return
			}
		}

		if req.Method == "DELETE" && req.Body != http.NoBody {
			if errJson := handleDeleteRequestForEndpoint(endpoint, req); errJson != nil {
				WriteError(w, errJson, nil)
				return
			}
		}

		if errJson := m.PrepareRequestForProxying(*endpoint, req); errJson != nil {
			WriteError(w, errJson, nil)
			return
		}
		grpcResp, errJson := m.ProxyRequest(req)
		if errJson != nil {
			WriteError(w, errJson, nil)
			return
		}
		grpcRespBody, errJson := ReadGrpcResponseBody(grpcResp.Body)
		if errJson != nil {
			WriteError(w, errJson, nil)
			return
		}

		var respJson []byte
		if !GrpcResponseIsEmpty(grpcRespBody) {
			respHasError, errJson := HandleGrpcResponseError(endpoint.Err, grpcResp, grpcRespBody, w)
			if errJson != nil {
				WriteError(w, errJson, nil)
				return
			}
			if respHasError {
				return
			}

			var resp interface{}
			if req.Method == "GET" {
				resp = endpoint.GetResponse
			} else if req.Method == "DELETE" {
				resp = endpoint.DeleteResponse
			} else {
				resp = endpoint.PostResponse
			}
			if errJson := deserializeGrpcResponseBodyIntoContainerWrapped(endpoint, grpcRespBody, resp); errJson != nil {
				WriteError(w, errJson, nil)
				return
			}
			if errJson := ProcessMiddlewareResponseFields(resp); errJson != nil {
				WriteError(w, errJson, nil)
				return
			}

			respJson, errJson = serializeMiddlewareResponseIntoJsonWrapped(endpoint, respJson, resp)
			if errJson != nil {
				WriteError(w, errJson, nil)
				return
			}
		}

		if errJson := WriteMiddlewareResponseHeadersAndBody(grpcResp, respJson, w); errJson != nil {
			WriteError(w, errJson, nil)
			return
		}
		if errJson := Cleanup(grpcResp.Body); errJson != nil {
			WriteError(w, errJson, nil)
			return
		}
	}
}

func handlePostRequestForEndpoint(endpoint *Endpoint, w http.ResponseWriter, req *http.Request) ErrorJson {
	if errJson := deserializeRequestBodyIntoContainerWrapped(endpoint, req, w); errJson != nil {
		return errJson
	}
	if errJson := ProcessRequestContainerFields(endpoint.PostRequest); errJson != nil {
		return errJson
	}
	return SetRequestBodyToRequestContainer(endpoint.PostRequest, req)
}

func handleDeleteRequestForEndpoint(endpoint *Endpoint, req *http.Request) ErrorJson {
	if errJson := DeserializeRequestBodyIntoContainer(req.Body, endpoint.DeleteRequest); errJson != nil {
		return errJson
	}
	if errJson := ProcessRequestContainerFields(endpoint.DeleteRequest); errJson != nil {
		return errJson
	}
	return SetRequestBodyToRequestContainer(endpoint.DeleteRequest, req)
}

func deserializeRequestBodyIntoContainerWrapped(endpoint *Endpoint, req *http.Request, w http.ResponseWriter) ErrorJson {
	runDefault := true
	if endpoint.Hooks.OnPreDeserializeRequestBodyIntoContainer != nil {
		run, errJson := endpoint.Hooks.OnPreDeserializeRequestBodyIntoContainer(endpoint, w, req)
		if errJson != nil {
			return errJson
		}
		if !run {
			runDefault = false
		}
	}
	if runDefault {
		if errJson := DeserializeRequestBodyIntoContainer(req.Body, endpoint.PostRequest); errJson != nil {
			return errJson
		}
	}
	if endpoint.Hooks.OnPostDeserializeRequestBodyIntoContainer != nil {
		if errJson := endpoint.Hooks.OnPostDeserializeRequestBodyIntoContainer(endpoint, w, req); errJson != nil {
			return errJson
		}
	}
	return nil
}

func deserializeGrpcResponseBodyIntoContainerWrapped(endpoint *Endpoint, grpcResponseBody []byte, resp interface{}) ErrorJson {
	runDefault := true
	if endpoint.Hooks.OnPreDeserializeGrpcResponseBodyIntoContainer != nil {
		run, errJson := endpoint.Hooks.OnPreDeserializeGrpcResponseBodyIntoContainer(grpcResponseBody, resp)
		if errJson != nil {
			return errJson
		}
		if !run {
			runDefault = false
		}
	}
	if runDefault {
		if errJson := DeserializeGrpcResponseBodyIntoContainer(grpcResponseBody, resp); errJson != nil {
			return errJson
		}
	}
	return nil
}

func serializeMiddlewareResponseIntoJsonWrapped(endpoint *Endpoint, respJson []byte, resp interface{}) ([]byte, ErrorJson) {
	runDefault := true
	var errJson ErrorJson
	if endpoint.Hooks.OnPreSerializeMiddlewareResponseIntoJson != nil {
		var run RunDefault
		run, respJson, errJson = endpoint.Hooks.OnPreSerializeMiddlewareResponseIntoJson(resp)
		if errJson != nil {
			return nil, errJson
		}
		if !run {
			runDefault = false
		}
	}
	if runDefault {
		respJson, errJson = SerializeMiddlewareResponseIntoJson(resp)
		if errJson != nil {
			return nil, errJson
		}
	}
	return respJson, nil
}
