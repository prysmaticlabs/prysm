package doublylinkedtree

import (
	"context"
	"fmt"

	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
)

// resetBoostedProposerRoot sets the value of the proposer boosted root to zeros.
func (f *ForkChoice) resetBoostedProposerRoot(_ context.Context) error {
	f.store.proposerBoostRoot = [32]byte{}
	return nil
}

// applyProposerBoostScore applies the current proposer boost scores to the
// relevant nodes.
func (f *ForkChoice) applyProposerBoostScore() error {
	s := f.store
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
	return s.proposerBoostRoot
}
