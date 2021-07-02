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

	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/eth/v1/events"
	"github.com/prysmaticlabs/prysm/shared/gateway"
	"github.com/prysmaticlabs/prysm/shared/grpcutils"
	"github.com/r3labs/sse"
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
		return nil, gateway.InternalServerErrorWithMessage(err, "could not decode response body into base64")
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
			return gateway.InternalServerErrorWithMessage(err, "could not parse status code")
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
		return gateway.InternalServerErrorWithMessage(err, "could not write response message")
	}
	return nil
}

func handleEvents(m *gateway.ApiProxyMiddleware, _ gateway.Endpoint, w http.ResponseWriter, req *http.Request) (handled bool) {
	sseClient := sse.NewClient("http://" + m.GatewayAddress + req.URL.RequestURI())
	eventChan := make(chan *sse.Event)

	// We use grpc-gateway as the server side of events, not the sse library.
	// Because of this subscribing to streams doesn't work as intended, resulting in each event being handled by all subscriptions.
	// To handle events properly, we subscribe just once using a placeholder value ('events') and handle all topics inside this subscription.
	if err := sseClient.SubscribeChan("events", eventChan); err != nil {
		gateway.WriteError(w, gateway.InternalServerError(err), nil)
		sseClient.Unsubscribe(eventChan)
		return
	}

	errJson := receiveEvents(eventChan, w, req)
	if errJson != nil {
		gateway.WriteError(w, errJson, nil)
	}

	sseClient.Unsubscribe(eventChan)
	return true
}

func receiveEvents(eventChan <-chan *sse.Event, w http.ResponseWriter, req *http.Request) gateway.ErrorJson {
	for {
		select {
		case msg := <-eventChan:
			var data interface{}

			switch strings.TrimSpace(string(msg.Event)) {
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
					return gateway.InternalServerError(err)
				}
				attData, err := json.Marshal(eventData.Aggregate)
				if err != nil {
					return gateway.InternalServerError(err)
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
				return &gateway.DefaultErrorJson{
					Message: fmt.Sprintf("Event type '%s' not supported", strings.TrimSpace(string(msg.Event))),
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

func writeEvent(msg *sse.Event, w http.ResponseWriter, data interface{}) gateway.ErrorJson {
	if err := json.Unmarshal(msg.Data, data); err != nil {
		return gateway.InternalServerError(err)
	}
	if errJson := gateway.ProcessMiddlewareResponseFields(data); errJson != nil {
		return errJson
	}
	dataJson, errJson := gateway.SerializeMiddlewareResponseIntoJson(data)
	if errJson != nil {
		return errJson
	}

	w.Header().Set("Content-Type", "text/event-stream")

	if _, err := w.Write([]byte("event: ")); err != nil {
		return gateway.InternalServerError(err)
	}
	if _, err := w.Write(msg.Event); err != nil {
		return gateway.InternalServerError(err)
	}
	if _, err := w.Write([]byte("\ndata: ")); err != nil {
		return gateway.InternalServerError(err)
	}
	if _, err := w.Write(dataJson); err != nil {
		return gateway.InternalServerError(err)
	}
	if _, err := w.Write([]byte("\n\n")); err != nil {
		return gateway.InternalServerError(err)
	}

	return nil
}

func flushEvent(w http.ResponseWriter) gateway.ErrorJson {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return &gateway.DefaultErrorJson{Message: fmt.Sprintf("Flush not supported in %T", w), Code: http.StatusInternalServerError}
	}
	flusher.Flush()
	return nil
}
