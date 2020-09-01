package asyncutil

import (
	"context"
	"testing"
	"time"
)

func TestDebounce(t *testing.T) {
	eventsChan := make(chan interface{}, 100)
	ctx, cancel := context.WithCancel(context.Background())
	interval := time.Second
	timesHandled := 0
	go Debounce(ctx, interval, eventsChan, func(event interface{}) {
		timesHandled++
	})
	for i := 0; i < 100; i++ {
		eventsChan <- struct{}{}
	}
	time.Sleep(interval)
	cancel()
	// We should expect 100 rapid fire changes to only have caused
	// 1 handler to trigger after the debouncing period.
	if timesHandled != 1 {
		t.Errorf("Expected 1 handler call, received %d", timesHandled)
	}
}
