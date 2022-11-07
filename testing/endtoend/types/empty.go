package types

import (
	"context"
	"sync"
)

// EmptyComponent satisfies the component interface.
// It can be embedded in other types in order to turn them into components.
type EmptyComponent struct {
	sync.Mutex
	startc chan struct{}
}

func (b *EmptyComponent) Start(ctx context.Context) error {
	b.chanInit()
	close(b.startc)
	return nil
}

func (b *EmptyComponent) chanInit() {
	b.Lock()
	defer b.Unlock()
	if b.startc == nil {
		b.startc = make(chan struct{})
	}
}

func (b *EmptyComponent) Started() <-chan struct{} {
	b.chanInit()
	return b.startc
}

func (b *EmptyComponent) Pause() error {
	return nil
}

func (m *EmptyComponent) Resume() error {
	return nil
}

func (m *EmptyComponent) Stop() error {
	return nil
}
