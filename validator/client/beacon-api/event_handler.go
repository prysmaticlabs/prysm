package beacon_api

import (
	"context"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api"
)

// Currently set to the first power of 2 bigger than the size of the `head` event
// which is 446 bytes
const eventByteLimit = 512

// EventHandler is responsible for subscribing to the Beacon API events endpoint
// and dispatching received events to subscribers.
type EventHandler struct {
	httpClient *http.Client
	host       string
	subs       []chan<- event
}

type event struct {
	eventType string
	data      string
}

// NewEventHandler returns a new handler.
func NewEventHandler(httpClient *http.Client, host string) *EventHandler {
	return &EventHandler{
		httpClient: httpClient,
		host:       host,
		subs:       make([]chan<- event, 0),
	}
}

func (h *EventHandler) subscribe(ch chan<- event) {
	h.subs = append(h.subs, ch)
}

func (h *EventHandler) get(ctx context.Context, topics []string, eventErrCh chan<- error) error {
	if len(topics) == 0 {
		return errors.New("no topics provided")
	}

	allTopics := strings.Join(topics, ",")
	log.Info("Starting listening to Beacon API events on topics " + allTopics)
	url := h.host + "/eth/v1/events?topics=" + allTopics
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create HTTP request")
	}

	req.Header.Set("Accept", api.EventStreamMediaType)
	req.Header.Set("Connection", "keep-alive")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to perform HTTP request")
	}
	go func() {
		// We signal an EOF error in a special way. When we get this error while reading the response body,
		// there might still be an event received in the body that we should handle.
		eof := false
		for {
			if ctx.Err() != nil {
				eventErrCh <- ctx.Err()
				return
			}

			rawData := make([]byte, eventByteLimit)
			_, err = resp.Body.Read(rawData)
			if err != nil {
				if strings.Contains(err.Error(), "EOF") {
					log.Error("Received EOF while reading events response body")
					eof = true
				} else {
					if closeErr := resp.Body.Close(); closeErr != nil {
						log.WithError(closeErr).Error("Failed to close events response body")
					}
					eventErrCh <- err
					return
				}
			}

			e := strings.Split(string(rawData), "\n")
			// We expect that the event format will contain event type and data separated with a newline
			if len(e) < 2 {
				// We reached EOF and there is no event to send
				if eof {
					if closeErr := resp.Body.Close(); closeErr != nil {
						log.WithError(closeErr).Error("Failed to close events response body")
					}
					return
				}
				continue
			}

			for _, ch := range h.subs {
				ch <- event{eventType: e[0], data: e[1]}
			}
			// We reached EOF and sent the last event
			if eof {
				if closeErr := resp.Body.Close(); closeErr != nil {
					log.WithError(closeErr).Error("Failed to close events response body")
				}
				return
			}
		}
	}()

	return nil
}
