package voluntaryexits

import (
	"math"
	"sync"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	doublylinkedlist "github.com/prysmaticlabs/prysm/v3/container/doubly-linked-list"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls/blst"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
)

// PoolManager maintains pending and seen voluntary exits.
// This pool is used by proposers to insert voluntary exits into new blocks.
type PoolManager interface {
	PendingExits() []*ethpb.SignedVoluntaryExit
	ExitsForInclusion(state state.ReadOnlyBeaconState) ([]*ethpb.SignedVoluntaryExit, error)
	InsertVoluntaryExit(exit *ethpb.SignedVoluntaryExit)
	MarkIncluded(exit *ethpb.SignedVoluntaryExit)
}

// Pool is a concrete implementation of PoolManager.
type Pool struct {
	lock    sync.RWMutex
	pending doublylinkedlist.List[*ethpb.SignedVoluntaryExit]
	m       map[types.ValidatorIndex]*doublylinkedlist.Node[*ethpb.SignedVoluntaryExit]
}

// NewPool returns an initialized pool.
func NewPool() *Pool {
	return &Pool{
		pending: doublylinkedlist.List[*ethpb.SignedVoluntaryExit]{},
		m:       make(map[types.ValidatorIndex]*doublylinkedlist.Node[*ethpb.SignedVoluntaryExit]),
	}
}

// PendingExits returns all objects from the pool.
func (p *Pool) PendingExits() ([]*ethpb.SignedVoluntaryExit, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	result := make([]*ethpb.SignedVoluntaryExit, p.pending.Len())
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

// ExitsForInclusion returns objects that are ready for inclusion at the given slot. This method will not
// return more than the block enforced MaxVoluntaryExits.
func (p *Pool) ExitsForInclusion(state state.ReadOnlyBeaconState) ([]*ethpb.SignedVoluntaryExit, error) {
	p.lock.RLock()
	length := int(math.Min(float64(params.BeaconConfig().MaxVoluntaryExits), float64(p.pending.Len())))
	result := make([]*ethpb.SignedVoluntaryExit, 0, length)
	node := p.pending.First()
	for node != nil && len(result) < length {
		exit, err := node.Value()
		if err != nil {
			p.lock.RUnlock()
			return nil, err
		}
		validator, err := state.ValidatorAtIndexReadOnly(exit.Exit.ValidatorIndex)
		if err != nil {
			return nil, p.handleInvalidExit(err, exit)
		}
		if err = blocks.VerifyExitConditions(validator, state.Slot(), exit.Exit); err != nil {
			return nil, p.handleInvalidExit(err, exit)
		}
		result = append(result, exit)
		node, err = node.Next()
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
	cSet, err := blocks.ExitSignatureBatch(state, result)
	if err != nil {
		logrus.WithError(err).Warning("could not get exit signatures")
	} else {
		ok, err := cSet.Verify()
		if err != nil {
			logrus.WithError(err).Warning("could not batch verify exit signatures")
		} else if ok {
			return result, nil
		}
	}
	// Batch signature failed, check signatures individually
	verified := make([]*ethpb.SignedVoluntaryExit, 0, length)
	for i, sig := range cSet.Signatures {
		signature, err := blst.SignatureFromBytes(sig)
		if err != nil {
			logrus.WithError(err).Warning("could not get signature from bytes")
			continue
		}
		if !signature.Verify(cSet.PublicKeys[i], cSet.Messages[i][:]) {
			logrus.Warning("removing exit with invalid signature from pool")
			if err := p.MarkIncluded(result[i]); err != nil {
				return nil, errors.Wrap(err, "could not mark exit as included")
			}
		} else {
			verified = append(verified, result[i])
		}
	}
	return verified, nil
}

// InsertVoluntaryExit into the pool.
func (p *Pool) InsertVoluntaryExit(exit *ethpb.SignedVoluntaryExit) {
	p.lock.Lock()
	defer p.lock.Unlock()

	_, exists := p.m[exit.Exit.ValidatorIndex]
	if exists {
		return
	}

	p.pending.Append(doublylinkedlist.NewNode(exit))
	p.m[exit.Exit.ValidatorIndex] = p.pending.Last()
}

// MarkIncluded is used when an exit has been included in a beacon block. Every block seen by this
// node should call this method to include the exit. This will remove the exit from the pool.
func (p *Pool) MarkIncluded(exit *ethpb.SignedVoluntaryExit) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	node := p.m[exit.Exit.ValidatorIndex]
	if node == nil {
		return nil
	}

	delete(p.m, exit.Exit.ValidatorIndex)
	p.pending.Remove(node)
	return nil
}

func (p *Pool) handleInvalidExit(err error, exit *ethpb.SignedVoluntaryExit) error {
	logrus.WithError(err).Warning("removing invalid exit from pool")
	// MarkIncluded removes the invalid exit from the pool
	p.lock.RUnlock()
	if err := p.MarkIncluded(exit); err != nil {
		return errors.Wrap(err, "could not mark exit as included")
	}
	p.lock.RLock()
	return err
}
