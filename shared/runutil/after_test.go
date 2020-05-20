package runutil_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/runutil"
)

func TestRunAfter_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var called bool
	runutil.RunAfter(ctx, 10*time.Second, func() {
		called = true
	})
	cancel()
	if called {
		t.Error("Expected callback not to have been called.")
	}
}

func TestRunAfter_TimeExceeded(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	var called bool
	runutil.RunAfter(ctx, 50*time.Millisecond, func() {
		called = true
		wg.Done()
	})
	wg.Wait()
	if !called {
		t.Error("Expected callback to have been called.")
	}
}