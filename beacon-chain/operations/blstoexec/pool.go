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
	PendingBLSToExecChanges(returnAll bool) []*ethpb.SignedBLSToExecutionChange
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

// PendingBLSToExecChanges returns objects that are ready for inclusion at the given slot.
// Without returnAll, this method will not return more than the block enforced MaxBlsToExecutionChanges.
func (p *Pool) PendingBLSToExecChanges(returnAll bool) []*ethpb.SignedBLSToExecutionChange {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if returnAll {
		result := make([]*ethpb.SignedBLSToExecutionChange, p.pending.len)
		node := p.pending.first
		for i := 0; node != nil; i++ {
			result[i] = node.value
			node = node.next
		}
		return result
	}

	length := int(math.Min(float64(params.BeaconConfig().MaxBlsToExecutionChanges), float64(p.pending.len)))
	result := make([]*ethpb.SignedBLSToExecutionChange, length)
	node := p.pending.first
	for i := 0; i < length; i++ {
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
	delete(p.m, node.value.Message.ValidatorIndex)
	node = nil
	p.pending.len--
}
