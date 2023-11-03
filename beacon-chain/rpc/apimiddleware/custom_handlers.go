package apimiddleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/prysmaticlabs/prysm/v4/api/gateway/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/events"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/r3labs/sse/v2"
)

func handleEvents(m *apimiddleware.ApiProxyMiddleware, _ apimiddleware.Endpoint, w http.ResponseWriter, req *http.Request) (handled bool) {
	sseClient := sse.NewClient("http://" + m.GatewayAddress + "/internal" + req.URL.RequestURI())
	sseClient.Headers["Grpc-Timeout"] = "0S"
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

type dataSubset struct {
	Version string `json:"version"`
}

func receiveEvents(eventChan <-chan *sse.Event, w http.ResponseWriter, req *http.Request) apimiddleware.ErrorJson {
	for {
		select {
		case msg := <-eventChan:
			var data interface{}

			// The message's event comes to us with trailing whitespace. Remove it here for
			// ease of future processing.
			msg.Event = bytes.TrimSpace(msg.Event)

			switch string(msg.Event) {
			case events.HeadTopic:
				data = &EventHeadJson{}
			case events.BlockTopic:
				data = &ReceivedBlockDataJson{}
			case events.AttestationTopic:
				data = &AttestationJson{}

				// Data received in the aggregated att event does not fit the expected event stream output.
				// We extract the underlying attestation from event data
				// and assign the attestation back to event data for further processing.
				aggEventData := &AggregatedAttReceivedDataJson{}
				if err := json.Unmarshal(msg.Data, aggEventData); err != nil {
					return apimiddleware.InternalServerError(err)
				}
				var attData []byte
				var err error
				// If true, then we have an unaggregated attestation
				if aggEventData.Aggregate == nil {
					unaggEventData := &UnaggregatedAttReceivedDataJson{}
					if err := json.Unmarshal(msg.Data, unaggEventData); err != nil {
						return apimiddleware.InternalServerError(err)
					}
					attData, err = json.Marshal(unaggEventData)
					if err != nil {
						return apimiddleware.InternalServerError(err)
					}
				} else {
					attData, err = json.Marshal(aggEventData.Aggregate)
					if err != nil {
						return apimiddleware.InternalServerError(err)
					}
				}
				msg.Data = attData
			case events.VoluntaryExitTopic:
				data = &SignedVoluntaryExitJson{}
			case events.FinalizedCheckpointTopic:
				data = &EventFinalizedCheckpointJson{}
			case events.ChainReorgTopic:
				data = &EventChainReorgJson{}
			case events.SyncCommitteeContributionTopic:
				data = &SignedContributionAndProofJson{}
			case events.BLSToExecutionChangeTopic:
				data = &SignedBLSToExecutionChangeJson{}
			case events.PayloadAttributesTopic:
				dataSubset := &dataSubset{}
				if err := json.Unmarshal(msg.Data, dataSubset); err != nil {
					return apimiddleware.InternalServerError(err)
				}
				switch dataSubset.Version {
				case version.String(version.Capella):
					data = &EventPayloadAttributeStreamV2Json{}
				case version.String(version.Bellatrix):
					data = &EventPayloadAttributeStreamV1Json{}
				default:
					return apimiddleware.InternalServerError(errors.New("payload version unsupported"))
				}
			case events.BlobSidecarTopic:
				data = &EventBlobSidecarJson{}
			case "error":
				data = &EventErrorJson{}
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
