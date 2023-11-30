package verification

import (
	"context"
	"sync"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/kzg"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

// Database represents the db methods that the verifiers need.
type Database interface {
	Block(ctx context.Context, blockRoot [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error)
}

// Forkchoicer represents the forkchoice methods that the verifiers need.
// Note that forkchoice is used here in a lock-free fashion, assuming that a version of forkchoice
// is given that internally handles the details of locking the underlying store.
type Forkchoicer interface {
	FinalizedCheckpoint() *forkchoicetypes.Checkpoint
	HasNode([32]byte) bool
	IsCanonical(root [32]byte) bool
	Slot([32]byte) (primitives.Slot, error)
}

// StateByRooter describes a stategen-ish type that can produce arbitrary states by their root
type StateByRooter interface {
	StateByRoot(ctx context.Context, blockRoot [32]byte) (state.BeaconState, error)
}

// sharedResources provides access to resources that are required by different verification types.
// for example, sidecar verifcation and block verification share the block signature verification cache.
type sharedResources struct {
	sync.RWMutex
	ready bool
	cw    startup.ClockWaiter
	clock *startup.Clock
	fc    Forkchoicer
	sc    SignatureCache
	pc    ProposerCache
	db    Database
	sr    StateByRooter
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

func NewInitializerWaiter(cw startup.ClockWaiter, fc Forkchoicer, sc SignatureCache, pc ProposerCache, db Database, sr StateByRooter) *InitializerWaiter {
	shared := &sharedResources{
		cw: cw,
		fc: fc,
		sc: sc,
		pc: pc,
		db: db,
		sr: sr,
	}
	return &InitializerWaiter{ini: &Initializer{shared: shared}}
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
	return &BlobVerifier{
		sharedResources:      ini.shared,
		blob:                 b,
		results:              newResults(reqs...),
		verifyBlobCommitment: kzg.VerifyROBlobCommitment,
	}
}
