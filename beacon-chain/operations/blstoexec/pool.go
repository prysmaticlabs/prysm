package blstoexec

import (
	"math"
	"sync"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	doublylinkedlist "github.com/prysmaticlabs/prysm/v3/container/doubly-linked-list"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
)

// We recycle the BLS changes pool to avoid the backing map growing without
// bound. The cycling operation is expensive because it copies all elements, so
// we only do it when the map is smaller than this upper bound.
const blsChangesPoolThreshold = 2000

// PoolManager maintains pending and seen BLS-to-execution-change objects.
// This pool is used by proposers to insert BLS-to-execution-change objects into new blocks.
type PoolManager interface {
	PendingBLSToExecChanges() ([]*ethpb.SignedBLSToExecutionChange, error)
	BLSToExecChangesForInclusion(beaconState state.ReadOnlyBeaconState) ([]*ethpb.SignedBLSToExecutionChange, error)
	InsertBLSToExecChange(change *ethpb.SignedBLSToExecutionChange)
	MarkIncluded(change *ethpb.SignedBLSToExecutionChange)
	ValidatorExists(idx primitives.ValidatorIndex) bool
}

// Pool is a concrete implementation of PoolManager.
type Pool struct {
	lock    sync.RWMutex
	pending doublylinkedlist.List[*ethpb.SignedBLSToExecutionChange]
	m       map[primitives.ValidatorIndex]*doublylinkedlist.Node[*ethpb.SignedBLSToExecutionChange]
}

// NewPool returns an initialized pool.
func NewPool() *Pool {
	return &Pool{
		pending: doublylinkedlist.List[*ethpb.SignedBLSToExecutionChange]{},
		m:       make(map[primitives.ValidatorIndex]*doublylinkedlist.Node[*ethpb.SignedBLSToExecutionChange]),
	}
}

// Copies the internal map and returns a new one.
func (p *Pool) cycleMap() {
	newMap := make(map[primitives.ValidatorIndex]*doublylinkedlist.Node[*ethpb.SignedBLSToExecutionChange])
	for k, v := range p.m {
		newMap[k] = v
	}
	p.m = newMap
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

// BLSToExecChangesForInclusion returns objects that are ready for inclusion.
// This method will not return more than the block enforced MaxBlsToExecutionChanges.
func (p *Pool) BLSToExecChangesForInclusion(st state.ReadOnlyBeaconState) ([]*ethpb.SignedBLSToExecutionChange, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	length := int(math.Min(float64(params.BeaconConfig().MaxBlsToExecutionChanges), float64(p.pending.Len())))
	result := make([]*ethpb.SignedBLSToExecutionChange, 0, length)
	node := p.pending.Last()
	for node != nil && len(result) < length {
		change, err := node.Value()
		if err != nil {
			return nil, err
		}
		_, err = blocks.ValidateBLSToExecutionChange(st, change)
		if err != nil {
			logrus.WithError(err).Warning("removing invalid BLSToExecutionChange from pool")
			// MarkIncluded removes the invalid change from the pool
			p.lock.RUnlock()
			p.MarkIncluded(change)
			p.lock.RLock()
		} else {
			result = append(result, change)
		}
		node, err = node.Prev()
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
// node should call this method to include the object. This will remove the object from the pool.
func (p *Pool) MarkIncluded(change *ethpb.SignedBLSToExecutionChange) {
	p.lock.Lock()
	defer p.lock.Unlock()

	node := p.m[change.Message.ValidatorIndex]
	if node == nil {
		return
	}

	delete(p.m, change.Message.ValidatorIndex)
	p.pending.Remove(node)
	if p.numPending() == blsChangesPoolThreshold {
		p.cycleMap()
	}
}

// ValidatorExists checks if the bls to execution change object exists
// for that particular validator.
func (p *Pool) ValidatorExists(idx primitives.ValidatorIndex) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	node := p.m[idx]

	return node != nil
}

// numPending returns the number of pending bls to execution changes in the pool
func (p *Pool) numPending() int {
	return p.pending.Len()
}
