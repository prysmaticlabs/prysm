package util

import (
	"sync"
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
