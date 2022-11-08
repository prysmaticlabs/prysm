package blstoexec

import (
	"math"
	"sync"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/container/queue"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// PoolManager maintains pending and seen BLS-to-execution-change objects.
// This pool is used by proposers to insert BLS-to-execution-change objects into new blocks.
type PoolManager interface {
	PendingBLSToExecChanges(returnAll bool) []*ethpb.SignedBLSToExecutionChange
	InsertBLSToExecChange(change *ethpb.SignedBLSToExecutionChange)
	MarkIncluded(change *ethpb.SignedBLSToExecutionChange)
}

// Pool is a concrete implementation of PoolManager.
type Pool struct {
	lock    sync.RWMutex
	pending queue.Queue[*ethpb.SignedBLSToExecutionChange]
}

// NewPool returns an initialized pool.
func NewPool() *Pool {
	return &Pool{
		pending: queue.Queue[*ethpb.SignedBLSToExecutionChange]{},
	}
}

// PendingBLSToExecChanges returns objects that are ready for inclusion at the given slot.
// Without returnAll, this method will not return more than the block enforced MaxBlsToExecutionChanges.
func (p *Pool) PendingBLSToExecChanges(returnAll bool) []*ethpb.SignedBLSToExecutionChange {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if returnAll {
		return p.pending.Peek(p.pending.Len())
	}
	length := int(math.Min(float64(params.BeaconConfig().MaxBlsToExecutionChanges), float64(p.pending.Len())))
	pending := make([]*ethpb.SignedBLSToExecutionChange, length)
	copy(pending, p.pending.Peek(length))
	return pending
}

// InsertBLSToExecChange inserts an object into the pool.
func (p *Pool) InsertBLSToExecChange(change *ethpb.SignedBLSToExecutionChange) {
	p.lock.Lock()
	defer p.lock.Unlock()
	
	p.pending.Push(change)
}

// MarkIncluded is used when an object has been included in a beacon block. Every block seen by this
// node should call this method to include the object. This will remove the object from the pool.
func (p *Pool) MarkIncluded(change *ethpb.SignedBLSToExecutionChange) {
	p.lock.Lock()
	defer p.lock.Unlock()
}
