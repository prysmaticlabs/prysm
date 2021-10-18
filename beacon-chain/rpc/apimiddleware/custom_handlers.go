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
	"strings"

	"github.com/prysmaticlabs/prysm/api/gateway/apimiddleware"
	"github.com/prysmaticlabs/prysm/api/grpc"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/eth/events"
	"github.com/r3labs/sse"
)

type sszConfig struct {
	sszPath      string
	fileName     string
	responseJson sszResponseJson
}

func handleGetBeaconStateSSZ(m *apimiddleware.ApiProxyMiddleware, endpoint apimiddleware.Endpoint, w http.ResponseWriter, req *http.Request) (handled bool) {
	config := sszConfig{
		sszPath:      "/eth/v1/debug/beacon/states/{state_id}/ssz",
		fileName:     "beacon_state.ssz",
		responseJson: &beaconStateSSZResponseJson{},
	}
	return handleGetSSZ(m, endpoint, w, req, config)
}

func handleGetBeaconBlockSSZ(m *apimiddleware.ApiProxyMiddleware, endpoint apimiddleware.Endpoint, w http.ResponseWriter, req *http.Request) (handled bool) {
	config := sszConfig{
		sszPath:      "/eth/v1/beacon/blocks/{block_id}/ssz",
		fileName:     "beacon_block.ssz",
		responseJson: &blockSSZResponseJson{},
	}
	return handleGetSSZ(m, endpoint, w, req, config)
}

func handleGetBeaconStateSSZV2(m *apimiddleware.ApiProxyMiddleware, endpoint apimiddleware.Endpoint, w http.ResponseWriter, req *http.Request) (handled bool) {
	config := sszConfig{
		sszPath:      "/eth/v2/debug/beacon/states/{state_id}/ssz",
		fileName:     "beacon_state.ssz",
		responseJson: &beaconStateSSZResponseV2Json{},
	}
	return handleGetSSZ(m, endpoint, w, req, config)
}

func handleGetBeaconBlockSSZV2(m *apimiddleware.ApiProxyMiddleware, endpoint apimiddleware.Endpoint, w http.ResponseWriter, req *http.Request) (handled bool) {
	config := sszConfig{
		sszPath:      "/eth/v2/beacon/blocks/{block_id}/ssz",
		fileName:     "beacon_block.ssz",
		responseJson: &blockSSZResponseV2Json{},
	}
	return handleGetSSZ(m, endpoint, w, req, config)
}

func handleGetSSZ(
	m *apimiddleware.ApiProxyMiddleware,
	endpoint apimiddleware.Endpoint,
	w http.ResponseWriter,
	req *http.Request,
	config sszConfig,
) (handled bool) {
	if !sszRequested(req) {
		return false
	}

	if errJson := prepareSSZRequestForProxying(m, endpoint, req, config.sszPath); errJson != nil {
		apimiddleware.WriteError(w, errJson, nil)
		return true
	}
	grpcResponse, errJson := apimiddleware.ProxyRequest(req)
	if errJson != nil {
		apimiddleware.WriteError(w, errJson, nil)
		return true
	}
	grpcResponseBody, errJson := apimiddleware.ReadGrpcResponseBody(grpcResponse.Body)
	if errJson != nil {
		apimiddleware.WriteError(w, errJson, nil)
		return true
	}
	respHasError, errJson := apimiddleware.HandleGrpcResponseError(endpoint.Err, grpcResponse, grpcResponseBody, w)
	if errJson != nil {
		apimiddleware.WriteError(w, errJson, nil)
		return
	}
	if respHasError {
		return
	}
	if errJson := apimiddleware.DeserializeGrpcResponseBodyIntoContainer(grpcResponseBody, config.responseJson); errJson != nil {
		apimiddleware.WriteError(w, errJson, nil)
		return true
	}
	respVersion, responseSsz, errJson := serializeMiddlewareResponseIntoSSZ(config.responseJson)
	if errJson != nil {
		apimiddleware.WriteError(w, errJson, nil)
		return true
	}
	if errJson := writeSSZResponseHeaderAndBody(grpcResponse, w, responseSsz, respVersion, config.fileName); errJson != nil {
		apimiddleware.WriteError(w, errJson, nil)
		return true
	}
	if errJson := apimiddleware.Cleanup(grpcResponse.Body); errJson != nil {
		apimiddleware.WriteError(w, errJson, nil)
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

func prepareSSZRequestForProxying(
	m *apimiddleware.ApiProxyMiddleware,
	endpoint apimiddleware.Endpoint,
	req *http.Request, sszPath string,
) apimiddleware.ErrorJson {
	req.URL.Scheme = "http"
	req.URL.Host = m.GatewayAddress
	req.RequestURI = ""
	req.URL.Path = sszPath
	if errJson := apimiddleware.HandleURLParameters(endpoint.Path, req, []string{}); errJson != nil {
		return errJson
	}
	// We have to add the prefix after handling parameters because adding the prefix changes URL segment indexing.
	req.URL.Path = "/internal" + req.URL.Path
	return nil
}

func serializeMiddlewareResponseIntoSSZ(respJson sszResponseJson) (version string, ssz []byte, errJson apimiddleware.ErrorJson) {
	// Serialize the SSZ part of the deserialized value.
	data, err := base64.StdEncoding.DecodeString(respJson.SSZData())
	if err != nil {
		return "", nil, apimiddleware.InternalServerErrorWithMessage(err, "could not decode response body into base64")
	}
	return strings.ToLower(respJson.SSZVersion()), data, nil
}

func writeSSZResponseHeaderAndBody(grpcResp *http.Response, w http.ResponseWriter, respSsz []byte, respVersion, fileName string) apimiddleware.ErrorJson {
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
	w.Header().Set("Content-Length", strconv.Itoa(len(respSsz)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Eth-Consensus-Version", respVersion)
	if statusCodeHeader != "" {
		code, err := strconv.Atoi(statusCodeHeader)
		if err != nil {
			return apimiddleware.InternalServerErrorWithMessage(err, "could not parse status code")
		}
		w.WriteHeader(code)
	} else {
		w.WriteHeader(grpcResp.StatusCode)
	}
	if _, err := io.Copy(w, ioutil.NopCloser(bytes.NewReader(respSsz))); err != nil {
		return apimiddleware.InternalServerErrorWithMessage(err, "could not write response message")
	}
	return nil
}

func handleEvents(m *apimiddleware.ApiProxyMiddleware, _ apimiddleware.Endpoint, w http.ResponseWriter, req *http.Request) (handled bool) {
	sseClient := sse.NewClient("http://" + m.GatewayAddress + "/internal" + req.URL.RequestURI())
	eventChan := make(chan *sse.Event)

	// We use grpc-gateway as the server side of events, not the sse library.
	// Because of this subscribing to streams doesn't work as intended, resulting in each event being handled by all subscriptions.
	// To handle events properly, we subscribe just once using a placeholder value ('events') and handle all topics inside this subscription.
	if err := sseClient.SubscribeChan("events", eventChan); err != nil {
		apimiddleware.WriteError(w, apimiddleware.InternalServerError(err), nil)
		sseClient.Unsubscribe(eventChan)
		return
	}

	errJson := receiveEvents(eventChan, w, req)
	if errJson != nil {
		apimiddleware.WriteError(w, errJson, nil)
	}

	sseClient.Unsubscribe(eventChan)
	return true
}

func receiveEvents(eventChan <-chan *sse.Event, w http.ResponseWriter, req *http.Request) apimiddleware.ErrorJson {
	for {
		select {
		case msg := <-eventChan:
			var data interface{}

			// The message's event comes to us with trailing whitespace.  Remove it here for
			// ease of future procesing.
			msg.Event = bytes.TrimSpace(msg.Event)

			switch string(msg.Event) {
			case events.HeadTopic:
				data = &eventHeadJson{}
			case events.BlockTopic:
				data = &receivedBlockDataJson{}
			case events.AttestationTopic:
				data = &attestationJson{}

				// Data received in the event does not fit the expected event stream output.
				// We extract the underlying attestation from event data
				// and assign the attestation back to event data for further processing.
				eventData := &aggregatedAttReceivedDataJson{}
				if err := json.Unmarshal(msg.Data, eventData); err != nil {
					return apimiddleware.InternalServerError(err)
				}
				attData, err := json.Marshal(eventData.Aggregate)
				if err != nil {
					return apimiddleware.InternalServerError(err)
				}
				msg.Data = attData
			case events.VoluntaryExitTopic:
				data = &signedVoluntaryExitJson{}
			case events.FinalizedCheckpointTopic:
				data = &eventFinalizedCheckpointJson{}
			case events.ChainReorgTopic:
				data = &eventChainReorgJson{}
			case "error":
				data = &eventErrorJson{}
			default:
				return &apimiddleware.DefaultErrorJson{
					Message: fmt.Sprintf("Event type '%s' not supported", string(msg.Event)),
					Code:    http.StatusInternalServerError,
				}
			}

			if errJson := writeEvent(msg, w, data); errJson != nil {
				return errJson
			}
			if errJson := flushEvent(w); errJson != nil {
				return errJson
			}
		case <-req.Context().Done():
			return nil
		}
	}
}

func writeEvent(msg *sse.Event, w http.ResponseWriter, data interface{}) apimiddleware.ErrorJson {
	if err := json.Unmarshal(msg.Data, data); err != nil {
		return apimiddleware.InternalServerError(err)
	}
	if errJson := apimiddleware.ProcessMiddlewareResponseFields(data); errJson != nil {
		return errJson
	}
	dataJson, errJson := apimiddleware.SerializeMiddlewareResponseIntoJson(data)
	if errJson != nil {
		return errJson
	}

	w.Header().Set("Content-Type", "text/event-stream")

	if _, err := w.Write([]byte("event: ")); err != nil {
		return apimiddleware.InternalServerError(err)
	}
	if _, err := w.Write(msg.Event); err != nil {
		return apimiddleware.InternalServerError(err)
	}
	if _, err := w.Write([]byte("\ndata: ")); err != nil {
		return apimiddleware.InternalServerError(err)
	}
	if _, err := w.Write(dataJson); err != nil {
		return apimiddleware.InternalServerError(err)
	}
	if _, err := w.Write([]byte("\n\n")); err != nil {
		return apimiddleware.InternalServerError(err)
	}

	return nil
}

func flushEvent(w http.ResponseWriter) apimiddleware.ErrorJson {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return &apimiddleware.DefaultErrorJson{Message: fmt.Sprintf("Flush not supported in %T", w), Code: http.StatusInternalServerError}
	}
	flusher.Flush()
	return nil
}
