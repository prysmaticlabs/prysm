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
	sub := make(chan event, 1)
	handler.subscribe(sub)

	require.NoError(t, handler.get(context.Background(), []string{"head"}, make(chan error)))

	e := <-sub
	assert.Equal(t, "head", e.eventType)
	assert.Equal(t, "data", e.data)
}
