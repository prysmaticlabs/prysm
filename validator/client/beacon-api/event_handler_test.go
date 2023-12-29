package beacon_api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestEventHandler(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/eth/v1/events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		require.Equal(t, true, ok)
		_, err := fmt.Fprint(w, "head\ndata\n\n")
		require.NoError(t, err)
		flusher.Flush()
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	handler := NewEventHandler(http.DefaultClient, server.URL)
	ch1 := make(chan event, 1)
	sub1 := eventSub{ch: ch1}
	ch2 := make(chan event, 1)
	sub2 := eventSub{ch: ch2}
	sub3 := eventSub{name: "sub3"}
	handler.subscribe(sub1)
	handler.subscribe(sub2)
	handler.subscribe(sub3)

	require.NoError(t, handler.get(context.Background(), []string{"head"}, make(chan error)))

	e := <-ch1
	assert.Equal(t, "head", e.eventType)
	assert.Equal(t, "data", e.data)
	e = <-ch2
	assert.Equal(t, "head", e.eventType)
	assert.Equal(t, "data", e.data)
}
