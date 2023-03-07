package util

import (
	"context"
	"sync"
	"testing"
	"time"
)

// WaitTimeout will wait for a WaitGroup to resolve within a timeout interval.
// Returns true if the waitgroup exceeded the timeout.
func WaitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		wg.Wait()
	}()
	select {
	case <-ch:
		return false
	case <-time.After(timeout):
		return true
	}
}

type Waiter struct {
	c chan struct{}
}

func NewWaiter() *Waiter {
	return &Waiter{
		c: make(chan struct{}),
	}
}

func (w *Waiter) Done() {
	close(w.c)
}

func (w *Waiter) RequireDoneAfter(t *testing.T, timeout time.Duration) {
	select {
	case <-w.c:
		return
	case <-time.After(timeout):
		t.Fatalf("timeout after %s", timeout)
	}
}

func (w *Waiter) RequireDoneBeforeCancel(t *testing.T, ctx context.Context) {
	select {
	case <-w.c:
		return
	case <-ctx.Done():
		t.Fatalf("context canceled before Done with error=%s", ctx.Err())
	}
}
