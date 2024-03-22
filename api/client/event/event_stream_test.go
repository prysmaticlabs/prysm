package event

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
	log "github.com/sirupsen/logrus"
)

func TestNewEventStream(t *testing.T) {
	validURL := "http://localhost:8080"
	invalidURL := "://invalid"
	topics := []string{"topic1", "topic2"}

	tests := []struct {
		name    string
		host    string
		topics  []string
		wantErr bool
	}{
		{"Valid input", validURL, topics, false},
		{"Invalid URL", invalidURL, topics, true},
		{"No topics", validURL, []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEventStream(context.Background(), &http.Client{}, tt.host, tt.topics)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEventStream() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEventStream(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/eth/v1/events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		require.Equal(t, true, ok)
		for i := 1; i <= 2; i++ {
			_, err := fmt.Fprintf(w, "event: head\ndata: data%d\n\n", i)
			require.NoError(t, err)
			flusher.Flush()                    // Trigger flush to simulate streaming data
			time.Sleep(100 * time.Millisecond) // Simulate delay between events
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	topics := []string{"head"}
	eventsChannel := make(chan *Event, 1)
	stream, err := NewEventStream(context.Background(), http.DefaultClient, server.URL, topics)
	require.NoError(t, err)
	go stream.Subscribe(eventsChannel)

	// Collect events
	var events []*Event

	for len(events) != 2 {
		select {
		case event := <-eventsChannel:
			log.Info(event)
			events = append(events, event)
		}
	}

	// Assertions to verify the events content
	expectedData := []string{"data1", "data2"}
	for i, event := range events {
		if string(event.Data) != expectedData[i] {
			t.Errorf("Expected event data %q, got %q", expectedData[i], string(event.Data))
		}
	}
}
