package voluntaryexits

import (
	"context"
	"sort"
	"sync"

	"go.opencensus.io/trace"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Pool implements a struct to maintain pending and recently included voluntary exits. This pool
// is used by proposers to insert into new blocks.
type Pool struct {
	lock     sync.RWMutex
	pending  []*ethpb.SignedVoluntaryExit
	included map[uint64]bool
}

// NewPool accepts a head fetcher (for reading the validator set) and returns an initialized
// voluntary exit pool.
func NewPool() *Pool {
	return &Pool{
		pending:  make([]*ethpb.SignedVoluntaryExit, 0),
		included: make(map[uint64]bool),
	}
}

// PendingExits returns exits that are ready for inclusion at the given slot. This method will not
// return more than the block enforced MaxVoluntaryExits.
func (p *Pool) PendingExits(state *beaconstate.BeaconState, slot uint64) []*ethpb.SignedVoluntaryExit {
	p.lock.RLock()
	defer p.lock.RUnlock()
	pending := make([]*ethpb.SignedVoluntaryExit, 0)
	for _, e := range p.pending {
		if e.Exit.Epoch > helpers.SlotToEpoch(slot) {
			continue
		}
		if v, err := state.ValidatorAtIndexReadOnly(e.Exit.ValidatorIndex); err == nil && v.ExitEpoch() == params.BeaconConfig().FarFutureEpoch {
			pending = append(pending, e)
		}
	}
	if uint64(len(pending)) > params.BeaconConfig().MaxVoluntaryExits {
		pending = pending[:params.BeaconConfig().MaxVoluntaryExits]
	}
	return pending
}

// InsertVoluntaryExit into the pool. This method is a no-op if the pending exit already exists,
// has been included recently, or the validator is already exited.
func (p *Pool) InsertVoluntaryExit(ctx context.Context, state *beaconstate.BeaconState, exit *ethpb.SignedVoluntaryExit) {
	ctx, span := trace.StartSpan(ctx, "exitPool.InsertVoluntaryExit")
	defer span.End()

	p.lock.Lock()
	defer p.lock.Unlock()

	// Has this validator index been included recently?
	if p.included[exit.Exit.ValidatorIndex] {
		return
	}

	// Has the validator been exited already?
	if v, err := state.ValidatorAtIndexReadOnly(exit.Exit.ValidatorIndex); err != nil || v.ExitEpoch() != params.BeaconConfig().FarFutureEpoch {
		return
	}

	// Does this validator exist in the list already? Use binary search to find the answer.
	if found := sort.Search(len(p.pending), func(i int) bool {
		e := p.pending[i].Exit
		return e.ValidatorIndex == exit.Exit.ValidatorIndex
	}); found != len(p.pending) {
		// If an exit exists with this validator index, prefer one with an earlier exit epoch.
		if p.pending[found].Exit.Epoch > exit.Exit.Epoch {
			p.pending[found] = exit
		}
		return
	}

	// Insert into pending list and sort again.
	p.pending = append(p.pending, exit)
	sort.Slice(p.pending, func(i, j int) bool {
		return p.pending[i].Exit.ValidatorIndex < p.pending[j].Exit.ValidatorIndex
	})
}

// MarkIncluded is used when an exit has been included in a beacon block. Every block seen by this
// node should call this method to include the exit.
func (p *Pool) MarkIncluded(exit *ethpb.SignedVoluntaryExit) {
	p.lock.Lock()
	defer p.lock.Unlock()
	i := sort.Search(len(p.pending), func(i int) bool {
		return p.pending[i].Exit.ValidatorIndex == exit.Exit.ValidatorIndex
	})
	if i != len(p.pending) {
		p.pending = append(p.pending[:i], p.pending[i+1:]...)
	}
	p.included[exit.Exit.ValidatorIndex] = true
}

// HasBeenIncluded returns true if the pool has recorded that a validator index has been recorded.
func (p *Pool) HasBeenIncluded(bIdx uint64) bool {
	return p.included[bIdx]
}
