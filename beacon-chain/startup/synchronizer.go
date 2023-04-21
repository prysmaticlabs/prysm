package startup

import (
	"context"

	"github.com/pkg/errors"
)

var ErrClockSet = errors.New("refusing to change clock after it is set")

type ClockSynchronizer struct {
	ready chan struct{}
	g     *Clock
}

type ClockWaiter interface {
	WaitForClock(context.Context) (*Clock, error)
}

type ClockSetter interface {
	SetClock(g *Clock) error
}

func (w *ClockSynchronizer) SetClock(g *Clock) error {
	if w.g != nil {
		return errors.Wrapf(ErrClockSet, "when SetClock called, Clock already set to time=%d", w.g.GenesisTime().Unix())
	}
	w.g = g
	close(w.ready)
	return nil
}

func (w *ClockSynchronizer) WaitForClock(ctx context.Context) (*Clock, error) {
	select {
	case <-w.ready:
		return w.g, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func NewClockSynchronizer() *ClockSynchronizer {
	return &ClockSynchronizer{
		ready: make(chan struct{}),
	}
}
