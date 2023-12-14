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
	subs       []chan<- []byte
}

func NewEventHandler(httpClient *http.Client, host string) *EventHandler {
	return &EventHandler{
		httpClient: httpClient,
		host:       host,
		subs:       make([]chan<- []byte, 0),
	}
}

func (h *EventHandler) subscribe(ch chan<- []byte) {
	h.subs = append(h.subs, ch)
}

func (h *EventHandler) get(ctx context.Context, topics []string) error {
	if len(topics) == 0 {
		return errors.New("no topics provided")
	}

	url := h.host + "/eth/v1/events?topics=" + strings.Join(topics, ",")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept", api.EventStreamMediaType)
	req.Header.Set("Connection", "keep-alive")

	httpResp, err := h.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to perform HTTP request")
	}
	go func() {
		for {
			data := make([]byte, eventByteLimit)
			_, err = httpResp.Body.Read(data)
			if err != nil {
				// TODO error channel? cancel context?
				log.WithError(err).Error("Failed to read response body")
				if err = httpResp.Body.Close(); err != nil {
					log.WithError(err).Error("Failed to close events response body")
				}
				return
			}

			// TODO: remove
			log.Info(string(data))

			for _, ch := range h.subs {
				ch <- data
			}
		}
	}()

	return nil
}
