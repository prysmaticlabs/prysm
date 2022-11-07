package blstoexec

import (
	"sort"
	"sync"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// PoolManager maintains pending and seen BLS-to-execution-change objects.
// This pool is used by proposers to insert BLS-to-execution-change objects into new blocks.
type PoolManager interface {
	PendingBLSToExecChanges(noLimit bool) []*ethpb.SignedBLSToExecutionChange
	InsertBLSToExecChange(state state.ReadOnlyBeaconState, change *ethpb.SignedBLSToExecutionChange)
	MarkIncluded(change *ethpb.SignedBLSToExecutionChange)
}

// Pool is a concrete implementation of PoolManager.
type Pool struct {
	lock    sync.RWMutex
	pending []*ethpb.SignedBLSToExecutionChange
}

// NewPool returns an initialized pool.
func NewPool() *Pool {
	return &Pool{
		pending: make([]*ethpb.SignedBLSToExecutionChange, 0),
	}
}

// PendingBLSToExecChanges returns objects that are ready for inclusion at the given slot.
// Without returnAll, this method will not return more than the block enforced MaxBlsToExecutionChanges.
func (p *Pool) PendingBLSToExecChanges(returnAll bool) []*ethpb.SignedBLSToExecutionChange {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if returnAll {
		return p.pending
	}
	maxChanges := params.BeaconConfig().MaxBlsToExecutionChanges
	pending := make([]*ethpb.SignedBLSToExecutionChange, maxChanges)
	for i, ch := range p.pending {
		pending[i] = ch
		if uint64(len(pending)) == maxChanges {
			break
		}
	}
	return pending
}

// InsertBLSToExecChange inserts an object into the pool.
// This method is a no-op if the pending exit already exists or the validator is already exited.
func (p *Pool) InsertBLSToExecChange(state state.ReadOnlyBeaconState, change *ethpb.SignedBLSToExecutionChange) {
	p.lock.Lock()
	defer p.lock.Unlock()

	// Prevent malformed messages from being inserted.
	if change == nil ||
		change.Message == nil ||
		len(change.Signature) != fieldparams.BLSSignatureLength ||
		len(change.Message.FromBlsPubkey) != fieldparams.BLSPubkeyLength ||
		len(change.Message.ToExecutionAddress) != fieldparams.ExecutionAddressLength {
		return
	}

	// Do we already have a pending object for the validator?
	existsInPending, _ := existsInList(p.pending, change.Message.ValidatorIndex)
	if existsInPending {
		return
	}

	// Does the validator already have ETH1 withdrawal credentials?
	v, err := state.ValidatorAtIndexReadOnly(change.Message.ValidatorIndex)
	if err != nil || v.HasETH1WithdrawalCredential() {
		return
	}

	// Insert into pending list and sort.
	p.pending = append(p.pending, change)
	sort.Slice(p.pending, func(i, j int) bool {
		return p.pending[i].Message.ValidatorIndex < p.pending[j].Message.ValidatorIndex
	})
}

// MarkIncluded is used when an object has been included in a beacon block. Every block seen by this
// node should call this method to include the object. This will remove the object from
// the pending objects slice.
func (p *Pool) MarkIncluded(change *ethpb.SignedBLSToExecutionChange) {
	p.lock.Lock()
	defer p.lock.Unlock()

	exists, index := existsInList(p.pending, change.Message.ValidatorIndex)
	if exists {
		// Object we want is present at p.pending[index], so we remove it.
		p.pending = append(p.pending[:index], p.pending[index+1:]...)
	}
}

// Binary search to check if the index exists in the list of pending objects.
func existsInList(pending []*ethpb.SignedBLSToExecutionChange, searchingFor types.ValidatorIndex) (bool, int) {
	i := sort.Search(len(pending), func(j int) bool {
		return pending[j].Message.ValidatorIndex >= searchingFor
	})
	if i < len(pending) && pending[i].Message.ValidatorIndex == searchingFor {
		return true, i
	}
	return false, -1
}
