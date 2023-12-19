package beacon_api

import (
	"context"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api"
)

const eventByteLimit = 512

type EventHandler struct {
	httpClient *http.Client
	host       string
	subs       []chan<- event
}

type event struct {
	eventType string
	data      string
}

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

	httpResp, err := h.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to perform HTTP request")
	}
	go func() {
		for {
			rawData := make([]byte, eventByteLimit)
			_, err = httpResp.Body.Read(rawData)
			if err != nil {
				if err = httpResp.Body.Close(); err != nil {
					log.WithError(err).Error("Failed to close events response body")
				}
				eventErrCh <- err
				return
			}

			e := strings.Split(string(rawData), "\n")
			// we expect: event type, newline, event data, newline, newline
			if len(e) != 4 {
				continue
			}

			for _, ch := range h.subs {
				ch <- event{eventType: e[0], data: e[1]}
			}
		}
	}()

	return nil
}
