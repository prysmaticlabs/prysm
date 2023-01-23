package blstoexec

import (
	"math"
	"sync"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	doublylinkedlist "github.com/prysmaticlabs/prysm/v3/container/doubly-linked-list"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls/blst"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
)

// PoolManager maintains pending and seen BLS-to-execution-change objects.
// This pool is used by proposers to insert BLS-to-execution-change objects into new blocks.
type PoolManager interface {
	PendingBLSToExecChanges() ([]*ethpb.SignedBLSToExecutionChange, error)
	BLSToExecChangesForInclusion(beaconState state.ReadOnlyBeaconState) ([]*ethpb.SignedBLSToExecutionChange, error)
	InsertBLSToExecChange(change *ethpb.SignedBLSToExecutionChange)
	MarkIncluded(change *ethpb.SignedBLSToExecutionChange)
	ValidatorExists(idx types.ValidatorIndex) bool
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

// BLSToExecChangesForInclusion returns objects that are ready for inclusion.
// This method will not return more than the block enforced MaxBlsToExecutionChanges.
func (p *Pool) BLSToExecChangesForInclusion(st state.ReadOnlyBeaconState) ([]*ethpb.SignedBLSToExecutionChange, error) {
	p.lock.RLock()
	length := int(math.Min(float64(params.BeaconConfig().MaxBlsToExecutionChanges), float64(p.pending.Len())))
	result := make([]*ethpb.SignedBLSToExecutionChange, 0, length)
	node := p.pending.Last()
	for node != nil && len(result) < length {
		change, err := node.Value()
		if err != nil {
			p.lock.RUnlock()
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
			p.lock.RUnlock()
			return nil, err
		}
	}
	p.lock.RUnlock()
	if len(result) == 0 {
		return result, nil
	}
	// We now verify the signatures in batches
	cSet, err := blocks.BLSChangesSignatureBatch(st, result)
	if err != nil {
		logrus.WithError(err).Warning("could not get BLSToExecutionChanges signatures")
	} else {
		ok, err := cSet.Verify()
		if err != nil {
			logrus.WithError(err).Warning("could not batch verify BLSToExecutionChanges signatures")
		} else if ok {
			return result, nil
		}
	}
	// Batch signature failed, check signatures individually
	verified := make([]*ethpb.SignedBLSToExecutionChange, 0, length)
	for i, sig := range cSet.Signatures {
		signature, err := blst.SignatureFromBytes(sig)
		if err != nil {
			logrus.WithError(err).Warning("could not get signature from bytes")
			continue
		}
		if !signature.Verify(cSet.PublicKeys[i], cSet.Messages[i][:]) {
			logrus.Warning("removing BLSToExecutionChange with invalid signature from pool")
			p.MarkIncluded(result[i])
		} else {
			verified = append(verified, result[i])
		}
	}
	return verified, nil
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
}

// ValidatorExists checks if the bls to execution change object exists
// for that particular validator.
func (p *Pool) ValidatorExists(idx types.ValidatorIndex) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	node := p.m[idx]

	return node != nil
}
