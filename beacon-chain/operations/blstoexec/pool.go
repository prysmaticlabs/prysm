package blstoexec

import (
	"math"
	"sync"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	doublylinkedlist "github.com/prysmaticlabs/prysm/v3/container/doubly-linked-list"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// PoolManager maintains pending and seen BLS-to-execution-change objects.
// This pool is used by proposers to insert BLS-to-execution-change objects into new blocks.
type PoolManager interface {
	PendingBLSToExecChanges() ([]*ethpb.SignedBLSToExecutionChange, error)
	BLSToExecChangesForInclusion() ([]*ethpb.SignedBLSToExecutionChange, error)
	InsertBLSToExecChange(change *ethpb.SignedBLSToExecutionChange)
	MarkIncluded(change *ethpb.SignedBLSToExecutionChange) error
}

// Pool is a concrete implementation of PoolManager.
type Pool struct {
	lock    sync.RWMutex
	pending doublylinkedlist.List[*ethpb.SignedBLSToExecutionChange]
	m       map[types.ValidatorIndex]*doublylinkedlist.Node[*ethpb.SignedBLSToExecutionChange]
}

// NewPool returns an initialized pool.
func NewPool() *Pool {
	return &Pool{
		pending: doublylinkedlist.List[*ethpb.SignedBLSToExecutionChange]{},
		m:       make(map[types.ValidatorIndex]*doublylinkedlist.Node[*ethpb.SignedBLSToExecutionChange]),
	}
}

// PendingBLSToExecChanges returns all objects from the pool.
func (p *Pool) PendingBLSToExecChanges() ([]*ethpb.SignedBLSToExecutionChange, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	result := make([]*ethpb.SignedBLSToExecutionChange, p.pending.Len())
	node := p.pending.First()
	var err error
	for i := 0; node != nil; i++ {
		result[i], err = node.Value()
		if err != nil {
			return nil, err
		}
		node, err = node.Next()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// BLSToExecChangesForInclusion returns objects that are ready for inclusion at the given slot.
// This method will not return more than the block enforced MaxBlsToExecutionChanges.
func (p *Pool) BLSToExecChangesForInclusion() ([]*ethpb.SignedBLSToExecutionChange, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	length := int(math.Min(float64(params.BeaconConfig().MaxBlsToExecutionChanges), float64(p.pending.Len())))
	result := make([]*ethpb.SignedBLSToExecutionChange, length)
	node := p.pending.First()
	var err error
	for i := 0; node != nil && i < length; i++ {
		result[i], err = node.Value()
		if err != nil {
			return nil, err
		}
		node, err = node.Next()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// InsertBLSToExecChange inserts an object into the pool.
func (p *Pool) InsertBLSToExecChange(change *ethpb.SignedBLSToExecutionChange) {
	p.lock.Lock()
	defer p.lock.Unlock()

	_, exists := p.m[change.Message.ValidatorIndex]
	if exists {
		return
	}

	p.pending.Append(doublylinkedlist.NewNode(change))
	p.m[change.Message.ValidatorIndex] = p.pending.Last()
}

// MarkIncluded is used when an object has been included in a beacon block. Every block seen by this
// listNode should call this method to include the object. This will remove the object from the pool.
func (p *Pool) MarkIncluded(change *ethpb.SignedBLSToExecutionChange) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	node := p.m[change.Message.ValidatorIndex]
	if node == nil {
		return nil
	}

	delete(p.m, change.Message.ValidatorIndex)
	p.pending.Remove(node)
	return nil
}
