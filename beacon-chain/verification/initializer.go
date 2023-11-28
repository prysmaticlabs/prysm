package verification

import (
	"context"
	"sync"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
)

// Database represents the db methods that the verifiers need.
type Database interface {
	Block(ctx context.Context, blockRoot [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error)
}

// sharedResources provides access to resources that are required by different verification types.
// for example, sidecar verifcation and block verification share the block signature verification cache.
type sharedResources struct {
	sync.RWMutex
	ready bool
	cw    startup.ClockWaiter
	clock *startup.Clock
	fc    forkchoice.Getter
	cache *Cache
	db    Database
	sg    *stategen.State
}

func (r *sharedResources) isReady() bool {
	r.RLock()
	defer r.RUnlock()
	return r.ready
}

func (r *sharedResources) waitForReady(ctx context.Context) error {
	if r.isReady() {
		return nil
	}
	clock, err := r.cw.WaitForClock(ctx)
	if err != nil {
		return err
	}
	r.Lock()
	defer r.Unlock()
	if !r.ready {
		r.clock = clock
		r.ready = true
	}
	return nil
}

// Initializer is used to create different Verifiers.
// Verifiers require access to stateful data structures, like caches,
// and it is Initializer's job to provides access to those.
type Initializer struct {
	shared *sharedResources
}

// InitializerWaiter provides an Initializer once all dependent resources are ready
// via the WaitForInitializer method.
type InitializerWaiter struct {
	ini *Initializer
}

func NewInitializerWaiter(cw startup.ClockWaiter) *InitializerWaiter {
	return &InitializerWaiter{ini: &Initializer{shared: &sharedResources{cw: cw}}}
}

// WaitForInitializer ensures that asyncronous initialization of the shared resources the initializer
// depends on has completed beofe the underlying Initializer is accessible by client code.
func (w *InitializerWaiter) WaitForInitializer(ctx context.Context) (*Initializer, error) {
	if err := w.ini.shared.waitForReady(ctx); err != nil {
		return nil, err
	}
	return w.ini, nil
}

// NewBlobVerifier creates a BlobVerifier for a single blob, with the given set of requirements.
func (ini *Initializer) NewBlobVerifier(ctx context.Context, b blocks.ROBlob, reqs ...Requirement) *BlobVerifier {
	return &BlobVerifier{sharedResources: ini.shared, ctx: ctx, blob: b, results: newResults(reqs...)}
}
