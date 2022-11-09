package blstoexec

import (
	"math"
	"sync"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

type listNode struct {
	value *ethpb.SignedBLSToExecutionChange
	prev  *listNode
	next  *listNode
}

type changesList struct {
	first *listNode
	last  *listNode
	len   int
}

// PoolManager maintains pending and seen BLS-to-execution-change objects.
// This pool is used by proposers to insert BLS-to-execution-change objects into new blocks.
type PoolManager interface {
	PendingBLSToExecChanges() []*ethpb.SignedBLSToExecutionChange
	BLSToExecChangesForInclusion() []*ethpb.SignedBLSToExecutionChange
	InsertBLSToExecChange(change *ethpb.SignedBLSToExecutionChange)
	MarkIncluded(change *ethpb.SignedBLSToExecutionChange)
}

// Pool is a concrete implementation of PoolManager.
type Pool struct {
	lock    sync.RWMutex
	pending changesList
	m       map[types.ValidatorIndex]*listNode
}

// NewPool returns an initialized pool.
func NewPool() *Pool {
	return &Pool{
		pending: changesList{},
		m:       make(map[types.ValidatorIndex]*listNode),
	}
}

// PendingBLSToExecChanges returns all objects from the pool.
func (p *Pool) PendingBLSToExecChanges() []*ethpb.SignedBLSToExecutionChange {
	p.lock.RLock()
	defer p.lock.RUnlock()

	result := make([]*ethpb.SignedBLSToExecutionChange, p.pending.len)
	node := p.pending.first
	for i := 0; node != nil; i++ {
		result[i] = node.value
		node = node.next
	}
	return result
}

// BLSToExecChangesForInclusion returns objects that are ready for inclusion at the given slot.
// This method will not return more than the block enforced MaxBlsToExecutionChanges.
func (p *Pool) BLSToExecChangesForInclusion() []*ethpb.SignedBLSToExecutionChange {
	p.lock.RLock()
	defer p.lock.RUnlock()

	length := int(math.Min(float64(params.BeaconConfig().MaxBlsToExecutionChanges), float64(p.pending.len)))
	result := make([]*ethpb.SignedBLSToExecutionChange, length)
	node := p.pending.first
	for i := 0; node != nil && i < length; i++ {
		result[i] = node.value
		node = node.next
	}
	return result
}

// InsertBLSToExecChange inserts an object into the pool.
func (p *Pool) InsertBLSToExecChange(change *ethpb.SignedBLSToExecutionChange) {
	p.lock.Lock()
	defer p.lock.Unlock()

	_, exists := p.m[change.Message.ValidatorIndex]
	if exists {
		return
	}

	var node *listNode
	if p.pending.first == nil {
		node = &listNode{value: change}
		p.pending.first = node
	} else {
		node = &listNode{
			value: change,
			prev:  p.pending.last,
		}
		p.pending.last.next = node
	}
	p.pending.last = node
	p.m[change.Message.ValidatorIndex] = node
	p.pending.len++
}

// MarkIncluded is used when an object has been included in a beacon block. Every block seen by this
// listNode should call this method to include the object. This will remove the object from the pool.
func (p *Pool) MarkIncluded(change *ethpb.SignedBLSToExecutionChange) {
	p.lock.Lock()
	defer p.lock.Unlock()

	node := p.m[change.Message.ValidatorIndex]
	if node == nil {
		return
	}

	if node == p.pending.first {
		if node == p.pending.last {
			p.pending.first = nil
			p.pending.last = nil
		} else {
			node.next.prev = nil
			p.pending.first = node.next
		}
	} else {
		if node == p.pending.last {
			node.prev.next = nil
			p.pending.last = node.prev
		} else {
			node.prev.next = node.next
			node.next.prev = node.prev
		}
	}
	p.m[node.value.Message.ValidatorIndex] = nil
	node = nil
	p.pending.len--
}
