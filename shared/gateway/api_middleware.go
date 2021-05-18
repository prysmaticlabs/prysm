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
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/beaconv1"
	butil "github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/wealdtech/go-bytesutil"
)

// ApiProxyMiddleware is a proxy between an Eth2 API HTTP client and gRPC-gateway.
// The purpose of the proxy is to handle HTTP requests and gRPC responses in such a way that:
//   - Eth2 API requests can be handled by gRPC-gateway correctly
//   - gRPC responses can be returned as spec-compliant Eth2 API responses
type ApiProxyMiddleware struct {
	GatewayAddress string
	ProxyAddress   string
	router         *mux.Router
}

type endpointInfo struct {
	postRequestType reflect.Type
	getResponseType reflect.Type
	errorType       reflect.Type
}

type endpointData struct {
	postRequest interface{}
	getResponse interface{}
	err         ErrorJson
}

type fieldProcessor struct {
	tag string
	f   func(value reflect.Value) error
}

// Run starts the proxy, registering all proxy endpoints on ApiProxyMiddleware.ProxyAddress.
func (m *ApiProxyMiddleware) Run() error {
	m.router = mux.NewRouter()

	m.handleApiEndpoint("/eth/v1/beacon/genesis",
		endpointInfo{
			getResponseType: reflect.TypeOf(GenesisResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/beacon/states/{state_id}/root",
		endpointInfo{
			getResponseType: reflect.TypeOf(StateRootResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/beacon/states/{state_id}/fork",
		endpointInfo{
			getResponseType: reflect.TypeOf(StateForkResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/beacon/states/{state_id}/finality_checkpoints",
		endpointInfo{
			getResponseType: reflect.TypeOf(StateFinalityCheckpointResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/beacon/headers/{block_id}",
		endpointInfo{
			getResponseType: reflect.TypeOf(BlockHeaderResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{})},
	)
	m.handleApiEndpoint("/eth/v1/beacon/blocks",
		endpointInfo{postRequestType: reflect.TypeOf(BeaconBlockContainerJson{}),
			errorType: reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/beacon/blocks/{block_id}",
		endpointInfo{
			getResponseType: reflect.TypeOf(BlockResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/beacon/blocks/{block_id}/root",
		endpointInfo{
			getResponseType: reflect.TypeOf(BlockRootResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/beacon/blocks/{block_id}/attestations",
		endpointInfo{
			getResponseType: reflect.TypeOf(BlockAttestationsResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/beacon/pool/attestations",
		endpointInfo{
			postRequestType: reflect.TypeOf(SubmitAttestationRequestJson{}),
			getResponseType: reflect.TypeOf(BlockAttestationsResponseJson{}),
			errorType:       reflect.TypeOf(SubmitAttestationsErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/beacon/pool/attester_slashings",
		endpointInfo{
			postRequestType: reflect.TypeOf(AttesterSlashingJson{}),
			getResponseType: reflect.TypeOf(AttesterSlashingsPoolResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/beacon/pool/proposer_slashings",
		endpointInfo{
			postRequestType: reflect.TypeOf(ProposerSlashingJson{}),
			getResponseType: reflect.TypeOf(ProposerSlashingsPoolResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/beacon/pool/voluntary_exits",
		endpointInfo{
			postRequestType: reflect.TypeOf(SignedVoluntaryExitJson{}),
			getResponseType: reflect.TypeOf(VoluntaryExitsPoolResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/node/identity",
		endpointInfo{
			getResponseType: reflect.TypeOf(IdentityResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/node/peers",
		endpointInfo{
			getResponseType: reflect.TypeOf(PeersResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/node/peers/{peer_id}",
		endpointInfo{getResponseType: reflect.TypeOf(PeerResponseJson{}),
			errorType: reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/node/peer_count",
		endpointInfo{
			getResponseType: reflect.TypeOf(PeerCountResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/node/version",
		endpointInfo{
			getResponseType: reflect.TypeOf(VersionResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/node/health", endpointInfo{errorType: reflect.TypeOf(DefaultErrorJson{})})
	m.handleApiEndpoint("/eth/v1/debug/beacon/states/{state_id}",
		endpointInfo{
			getResponseType: reflect.TypeOf(BeaconStateResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/debug/beacon/heads",
		endpointInfo{
			getResponseType: reflect.TypeOf(ForkChoiceHeadsResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/config/fork_schedule",
		endpointInfo{
			getResponseType: reflect.TypeOf(ForkScheduleResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/config/deposit_contract",
		endpointInfo{
			getResponseType: reflect.TypeOf(DepositContractResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})
	m.handleApiEndpoint("/eth/v1/config/spec",
		endpointInfo{
			getResponseType: reflect.TypeOf(SpecResponseJson{}),
			errorType:       reflect.TypeOf(DefaultErrorJson{}),
		})

	return http.ListenAndServe(m.ProxyAddress, m.router)
}

func (m *ApiProxyMiddleware) handleApiEndpoint(endpoint string, info endpointInfo) {
	m.router.HandleFunc(endpoint, func(writer http.ResponseWriter, request *http.Request) {
		data := prepareData(info)
		if request.Method == "POST" {
			// https://ethereum.github.io/eth2.0-APIs/#/Beacon/submitPoolAttestations expects posting a top-level array.
			// We make it more proto-friendly by wrapping it in a struct with a 'data' field.
			if err := wrapAttestationsArray(data, request); err != nil {
				e := fmt.Errorf("could not decode request body: %w", err)
				writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
				return
			}

			// Deserialize the body into the 'm' struct, and post it to grpc-gateway.
			if err := json.NewDecoder(request.Body).Decode(&data.postRequest); err != nil {
				e := fmt.Errorf("could not decode request body: %w", err)
				writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
				return
			}
			// Posted graffiti needs to have length of 32 bytes, but client is allowed to send data of any length.
			prepareGraffiti(data)
			// Apply processing functions to fields with specific tags.
			if err := processField(data.postRequest, []fieldProcessor{
				{
					tag: "hex",
					f:   hexToBase64Processor,
				},
			}); err != nil {
				e := fmt.Errorf("could not process request hex data: %w", err)
				writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
				return
			}
			// Serialize the struct, which now includes a base64-encoded value, into JSON.
			j, err := json.Marshal(data.postRequest)
			if err != nil {
				e := fmt.Errorf("could not marshal request: %w", err)
				writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
				return
			}
			// Set the body to the new JSON.
			request.Body = ioutil.NopCloser(bytes.NewReader(j))
			request.Header.Set("Content-Length", strconv.Itoa(len(j)))
			request.ContentLength = int64(len(j))
		}

		request.URL.Scheme = "http"
		request.URL.Host = m.GatewayAddress
		request.RequestURI = ""

		// Handle hex in URL parameters.
		segments := strings.Split(endpoint, "/")
		for i, s := range segments {
			// We only care about segments which are parameterized.
			if len(s) > 0 && s[0] == '{' && s[len(s)-1] == '}' {
				bRouteVar := []byte(mux.Vars(request)[s[1:len(s)-1]])
				var routeVar string
				isHex, err := butil.IsBytes32Hex(bRouteVar)
				if err != nil {
					e := fmt.Errorf("could not process URL parameter: %w", err)
					writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
					return
				}
				if isHex {
					b, err := bytesutil.FromHexString(string(bRouteVar))
					if err != nil {
						e := fmt.Errorf("could not process URL parameter: %w", err)
						writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
						return
					}
					routeVar = base64.URLEncoding.EncodeToString(b)
				} else {
					routeVar = base64.StdEncoding.EncodeToString(bRouteVar)
				}
				splitPath := strings.Split(request.URL.Path, "/")
				splitPath[i] = routeVar
				request.URL.Path = strings.Join(splitPath, "/")
				break
			}
		}

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

		// Deserialize the output of grpc-gateway into the error struct.
		body, err := ioutil.ReadAll(grpcResp.Body)
		if err != nil {
			e := fmt.Errorf("could not read response body: %w", err)
			writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
			return
		}
		if err := json.Unmarshal(body, data.err); err != nil {
			e := fmt.Errorf("could not unmarshal error: %w", err)
			writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
			return
		}
		var j []byte
		if data.err.Msg() != "" {
			// Something went wrong, but the request completed, meaning we can write headers and the error message.
			for h, vs := range grpcResp.Header {
				for _, v := range vs {
					writer.Header().Set(h, v)
				}
			}
			// Set code to HTTP code because unmarshalled body contained gRPC code.
			data.err.SetCode(grpcResp.StatusCode)
			writeError(writer, data.err, grpcResp.Header)
			return
		} else {
			// Don't do anything if the response is only a status code.
			if request.Method == "GET" && data.getResponse != nil {
				// Deserialize the output of grpc-gateway.
				if err := json.Unmarshal(body, &data.getResponse); err != nil {
					e := fmt.Errorf("could not unmarshal response: %w", err)
					writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
					return
				}
				// Apply processing functions to fields with specific tags.
				if err := processField(data.getResponse, []fieldProcessor{
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
					e := fmt.Errorf("could not process response hex data: %w", err)
					writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
					return
				}
				// Serialize the return value into JSON.
				j, err = json.Marshal(data.getResponse)
				if err != nil {
					e := fmt.Errorf("could not marshal response: %w", err)
					writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
					return
				}
			}
		}

		// Write the response (headers + content) and PROFIT!
		for h, vs := range grpcResp.Header {
			for _, v := range vs {
				writer.Header().Set(h, v)
			}
		}
		if request.Method == "GET" {
			writer.Header().Set("Content-Length", strconv.Itoa(len(j)))
			writer.WriteHeader(grpcResp.StatusCode)
			if _, err := io.Copy(writer, ioutil.NopCloser(bytes.NewReader(j))); err != nil {
				e := fmt.Errorf("could not write response message: %w", err)
				writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
				return
			}
		} else if request.Method == "POST" {
			writer.WriteHeader(grpcResp.StatusCode)
		}

		if err := grpcResp.Body.Close(); err != nil {
			e := fmt.Errorf("could not close response body: %w", err)
			writeError(writer, &DefaultErrorJson{Message: e.Error(), Code: http.StatusInternalServerError}, nil)
			return
		}
	})
}

// prepareData constructs an empty struct whose contents are populated from endpoint information.
func prepareData(info endpointInfo) endpointData {
	data := endpointData{}
	if info.postRequestType != nil {
		data.postRequest = reflect.New(info.postRequestType).Interface()
	}
	if info.getResponseType != nil {
		data.getResponse = reflect.New(info.getResponseType).Interface()
	}
	if info.errorType != nil {
		data.err = reflect.New(info.errorType).Interface().(ErrorJson)
	}
	return data
}

func prepareGraffiti(data endpointData) {
	if block, ok := data.postRequest.(*BeaconBlockContainerJson); ok {
		b := bytesutil.ToBytes32([]byte(block.Message.Body.Graffiti))
		block.Message.Body.Graffiti = hexutil.Encode(b[:])
	}
}

func wrapAttestationsArray(data endpointData, req *http.Request) error {
	if _, ok := data.postRequest.(*SubmitAttestationRequestJson); ok {
		atts := make([]*AttestationJson, 0)
		if err := json.NewDecoder(req.Body).Decode(&atts); err != nil {
			return fmt.Errorf("could not decode attestations array: %w", err)
		}
		j := &SubmitAttestationRequestJson{Data: atts}
		b, err := json.Marshal(j)
		if err != nil {
			return fmt.Errorf("could not marshal wrapped attestations array: %w", err)
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(b))
	}
	return nil
}

func writeError(writer http.ResponseWriter, e ErrorJson, responseHeader http.Header) {
	// Include custom error in the error JSON.
	if responseHeader != nil {
		customError, ok := responseHeader["Grpc-Metadata-"+beaconv1.CustomErrorMetadataKey]
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

// processField calls 'processor' on any field that has the 'hex' tag set.
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
						return fmt.Errorf("could not process field: %w", err)
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
								return fmt.Errorf("could not process field: %w", err)
							}
						}
					}
				}

			}
		// Recursively process struct pointers.
		case reflect.Ptr:
			if v.Field(i).Elem().Kind() == reflect.Struct {
				if err := processField(v.Field(i).Interface(), processors); err != nil {
					return fmt.Errorf("could not process field: %w", err)
				}
			}
		default:
			field := t.Field(i)
			for _, proc := range processors {
				if _, hasTag := field.Tag.Lookup(proc.tag); hasTag {
					if err := proc.f(v.Field(i)); err != nil {
						return fmt.Errorf("could not process field: %w", err)
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
	b, err := base64.StdEncoding.DecodeString(v.Interface().(string))
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
