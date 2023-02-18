package doublylinkedtree

import (
	"context"
	"fmt"

	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
)

// resetBoostedProposerRoot sets the value of the proposer boosted root to zeros.
func (f *ForkChoice) resetBoostedProposerRoot(_ context.Context) error {
	f.store.proposerBoostLock.Lock()
	f.store.proposerBoostRoot = [32]byte{}
	f.store.proposerBoostLock.Unlock()
	return nil
}

// applyProposerBoostScore applies the current proposer boost scores to the
// relevant nodes. This function requires a lock in Store.nodesLock.
func (f *ForkChoice) applyProposerBoostScore() error {
	s := f.store
	s.proposerBoostLock.Lock()
	defer s.proposerBoostLock.Unlock()

	// acquire checkpoints lock for the justified balances
	s.checkpointsLock.RLock()
	defer s.checkpointsLock.RUnlock()

	proposerScore := uint64(0)
	if s.previousProposerBoostRoot != params.BeaconConfig().ZeroHash {
		previousNode, ok := s.nodeByRoot[s.previousProposerBoostRoot]
		if !ok || previousNode == nil {
			log.WithError(errInvalidProposerBoostRoot).Errorf(fmt.Sprintf("invalid prev root %#x", s.previousProposerBoostRoot))
		} else {
			previousNode.balance -= s.previousProposerBoostScore
		}
	}

	if s.proposerBoostRoot != params.BeaconConfig().ZeroHash {
		currentNode, ok := s.nodeByRoot[s.proposerBoostRoot]
		if !ok || currentNode == nil {
			log.WithError(errInvalidProposerBoostRoot).Errorf(fmt.Sprintf("invalid current root %#x", s.proposerBoostRoot))
		} else {
			proposerScore = (s.committeeWeight * params.BeaconConfig().ProposerScoreBoost) / 100
			currentNode.balance += proposerScore
		}
	}
	s.previousProposerBoostRoot = s.proposerBoostRoot
	s.previousProposerBoostScore = proposerScore
	return nil
}

// ProposerBoost of fork choice store.
func (s *Store) proposerBoost() [fieldparams.RootLength]byte {
	s.proposerBoostLock.RLock()
	defer s.proposerBoostLock.RUnlock()
	return s.proposerBoostRoot
}
