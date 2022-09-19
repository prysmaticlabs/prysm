package voluntaryexits

import (
	"context"
	"sort"
	"sync"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"go.opencensus.io/trace"
)

// PoolManager maintains pending and seen voluntary exits.
// This pool is used by proposers to insert voluntary exits into new blocks.
type PoolManager interface {
	PendingExits(state state.ReadOnlyBeaconState, slot types.Slot, noLimit bool) []*ethpb.SignedVoluntaryExit
	InsertVoluntaryExit(ctx context.Context, state state.ReadOnlyBeaconState, exit *ethpb.SignedVoluntaryExit)
	MarkIncluded(exit *ethpb.SignedVoluntaryExit)
}

// Pool is a concrete implementation of PoolManager.
type Pool struct {
	lock    sync.RWMutex
	pending []*ethpb.SignedVoluntaryExit
}

// NewPool accepts a head fetcher (for reading the validator set) and returns an initialized
// voluntary exit pool.
func NewPool() *Pool {
	return &Pool{
		pending: make([]*ethpb.SignedVoluntaryExit, 0),
	}
}

// PendingExits returns exits that are ready for inclusion at the given slot. This method will not
// return more than the block enforced MaxVoluntaryExits.
func (p *Pool) PendingExits(state state.ReadOnlyBeaconState, slot types.Slot, noLimit bool) []*ethpb.SignedVoluntaryExit {
	p.lock.RLock()
	defer p.lock.RUnlock()

	// Allocate pending slice with a capacity of min(len(p.pending), maxVoluntaryExits) since the
	// array cannot exceed the max and is typically less than the max value.
	maxExits := params.BeaconConfig().MaxVoluntaryExits
	if noLimit {
		maxExits = uint64(len(p.pending))
	}
	pending := make([]*ethpb.SignedVoluntaryExit, 0, maxExits)
	for _, e := range p.pending {
		if e.Exit.Epoch > slots.ToEpoch(slot) {
			continue
		}
		if v, err := state.ValidatorAtIndexReadOnly(e.Exit.ValidatorIndex); err == nil &&
			v.ExitEpoch() == params.BeaconConfig().FarFutureEpoch {
			pending = append(pending, e)
			if uint64(len(pending)) == maxExits {
				break
			}
		}
	}
	return pending
}

// InsertVoluntaryExit into the pool. This method is a no-op if the pending exit already exists,
// or the validator is already exited.
func (p *Pool) InsertVoluntaryExit(ctx context.Context, state state.ReadOnlyBeaconState, exit *ethpb.SignedVoluntaryExit) {
	ctx, span := trace.StartSpan(ctx, "exitPool.InsertVoluntaryExit")
	defer span.End()
	p.lock.Lock()
	defer p.lock.Unlock()

	// Prevent malformed messages from being inserted.
	if exit == nil || exit.Exit == nil {
		return
	}

	existsInPending, index := existsInList(p.pending, exit.Exit.ValidatorIndex)
	// If the item exists in the pending list and includes a more favorable, earlier
	// exit epoch, we replace it in the pending list. If it exists but the prior condition is false,
	// we simply return.
	if existsInPending {
		if exit.Exit.Epoch < p.pending[index].Exit.Epoch {
			p.pending[index] = exit
		}
		return
	}

	// Has the validator been exited already?
	if v, err := state.ValidatorAtIndexReadOnly(exit.Exit.ValidatorIndex); err != nil ||
		v.ExitEpoch() != params.BeaconConfig().FarFutureEpoch {
		return
	}

	// Insert into pending list and sort.
	p.pending = append(p.pending, exit)
	sort.Slice(p.pending, func(i, j int) bool {
		return p.pending[i].Exit.ValidatorIndex < p.pending[j].Exit.ValidatorIndex
	})
}

// MarkIncluded is used when an exit has been included in a beacon block. Every block seen by this
// node should call this method to include the exit. This will remove the exit from
// the pending exits slice.
func (p *Pool) MarkIncluded(exit *ethpb.SignedVoluntaryExit) {
	p.lock.Lock()
	defer p.lock.Unlock()
	exists, index := existsInList(p.pending, exit.Exit.ValidatorIndex)
	if exists {
		// Exit we want is present at p.pending[index], so we remove it.
		p.pending = append(p.pending[:index], p.pending[index+1:]...)
	}
}

// Binary search to check if the index exists in the list of pending exits.
func existsInList(pending []*ethpb.SignedVoluntaryExit, searchingFor types.ValidatorIndex) (bool, int) {
	i := sort.Search(len(pending), func(j int) bool {
		return pending[j].Exit.ValidatorIndex >= searchingFor
	})
	if i < len(pending) && pending[i].Exit.ValidatorIndex == searchingFor {
		return true, i
	}
	return false, -1
}
