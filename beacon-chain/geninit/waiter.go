package geninit

import (
	"context"
	"time"
)

type ClockWaiter interface {
	WaitForClock(context.Context) (Clock, error)
}

type ClockSetter interface {
	SetGenesisTime(time.Time)
	SetGenesisClock(Clock)
}

type ClockSync struct {
	ready chan struct{}
	c Clock
}

func (w *ClockSync) SetGenesisTime(g time.Time) {
	w.c = NewClock(g)
	close(w.ready)
	w.ready = nil
}

func (w *ClockSync) SetGenesisClock(c Clock) {
	w.c = c
	close(w.ready)
	w.ready = nil
}

func (w *ClockSync) WaitForClock(ctx context.Context) (Clock, error) {
	if w.ready == nil {
		return w.c, nil
	}
	select {
	case <-w.ready:
		return w.c, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func NewClockSync() *ClockSync {
	return &ClockSync{
		ready: make(chan struct{}),
	}
}
