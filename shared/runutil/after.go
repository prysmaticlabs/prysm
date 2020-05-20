package runutil

import (
	"context"
	"time"
)

// RunAfter will execute the callback function after a designated period of time, if the context is
// not cancelled.
func RunAfter(ctx context.Context, period time.Duration, f func()) {
	go func () {
		select {
		case <-time.After(period):
			f()
		case <-ctx.Done():
			return
		}
	}()
}
