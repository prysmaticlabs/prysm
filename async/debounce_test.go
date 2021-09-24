package async_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/async"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestDebounce_NoEvents(t *testing.T) {
	eventsChan := make(chan interface{}, 100)
	ctx, cancel := context.WithCancel(context.Background())
	interval := time.Second
	timesHandled := 0
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		time.AfterFunc(interval, func() {
			cancel()
		})
	}()
	go func() {
		async.Debounce(ctx, interval, eventsChan, func(event interface{}) {
			timesHandled++
		})
		wg.Done()
	}()
	if util.WaitTimeout(wg, interval*2) {
		t.Fatalf("Test should have exited by now, timed out")
	}
	assert.Equal(t, 0, timesHandled, "Wrong number of handled calls")
}

func TestDebounce_CtxClosing(t *testing.T) {
	eventsChan := make(chan interface{}, 100)
	ctx, cancel := context.WithCancel(context.Background())
	interval := time.Second
	timesHandled := 0
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		ticker := time.NewTicker(time.Millisecond * 100)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				eventsChan <- struct{}{}
			}
		}
	}()
	go func() {
		time.AfterFunc(interval, func() {
			cancel()
		})
	}()
	go func() {
		async.Debounce(ctx, interval, eventsChan, func(event interface{}) {
			timesHandled++
		})
		wg.Done()
	}()
	if util.WaitTimeout(wg, interval*2) {
		t.Fatalf("Test should have exited by now, timed out")
	}
	assert.Equal(t, 0, timesHandled, "Wrong number of handled calls")
}

func TestDebounce_SingleHandlerInvocation(t *testing.T) {
	eventsChan := make(chan interface{}, 100)
	ctx, cancel := context.WithCancel(context.Background())
	interval := time.Second
	timesHandled := 0
	go async.Debounce(ctx, interval, eventsChan, func(event interface{}) {
		timesHandled++
	})
	for i := 0; i < 100; i++ {
		eventsChan <- struct{}{}
	}
	// We should expect 100 rapid fire changes to only have caused
	// 1 handler to trigger after the debouncing period.
	time.Sleep(interval * 2)
	assert.Equal(t, 1, timesHandled, "Wrong number of handled calls")
	cancel()
}

func TestDebounce_MultipleHandlerInvocation(t *testing.T) {
	eventsChan := make(chan interface{}, 100)
	ctx, cancel := context.WithCancel(context.Background())
	interval := time.Second
	timesHandled := 0
	go async.Debounce(ctx, interval, eventsChan, func(event interface{}) {
		timesHandled++
	})
	for i := 0; i < 100; i++ {
		eventsChan <- struct{}{}
	}
	require.Equal(t, 0, timesHandled, "Events must prevent from handler execution")

	// By this time the first event should be triggered.
	time.Sleep(2 * time.Second)
	assert.Equal(t, 1, timesHandled, "Wrong number of handled calls")

	// Second event.
	eventsChan <- struct{}{}
	time.Sleep(2 * time.Second)
	assert.Equal(t, 2, timesHandled, "Wrong number of handled calls")

	cancel()
}
