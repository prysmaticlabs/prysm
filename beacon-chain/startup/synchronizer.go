package startup

import (
	"context"

	"github.com/pkg/errors"
)

var ErrGenesisSet = errors.New("refusing to change genesis after it is set")

type GenesisSynchronizer struct {
	ready chan struct{}
	g     *Genesis
}

type GenesisWaiter interface {
	WaitForGenesis(context.Context) (*Genesis, error)
}

type GenesisSetter interface {
	SetGenesis(g *Genesis) error
}

func (w *GenesisSynchronizer) SetGenesis(g *Genesis) error {
	if w.g != nil {
		return errors.Wrapf(ErrGenesisSet, "when SetGenesis called, Genesis already set to time=%d", w.g.Time().Unix())
	}
	w.g = g
	close(w.ready)
	return nil
}

func (w *GenesisSynchronizer) WaitForGenesis(ctx context.Context) (*Genesis, error) {
	select {
	case <-w.ready:
		return w.g, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func NewGenesisSynchronizer() *GenesisSynchronizer {
	return &GenesisSynchronizer{
		ready: make(chan struct{}),
	}
}
