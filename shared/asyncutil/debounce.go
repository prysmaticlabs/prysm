package asyncutil

import (
	"context"
	"time"
)

// Debounce events fired over a channel by a specified duration, ensuring no events
// are handled until a certain interval of time has passed.
func Debounce(ctx context.Context, interval time.Duration, eventsChan <-chan interface{}, handler func(interface{})) {
	for event := range eventsChan {
	loop:
		for {
			// If an event is received, wait the specified interval before calling the
			// handler.
			// If another event is received before the interval has passed, store
			// it and reset the timer.
			select {
			// Do nothing until we can handle the events after the debounce interval.
			case event = <-eventsChan:
			case <-time.After(interval):
				handler(event)
				break loop
			case <-ctx.Done():
				return
			}
		}
	}
}
