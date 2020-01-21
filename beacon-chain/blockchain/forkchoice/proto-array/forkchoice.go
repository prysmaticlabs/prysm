package protoarray

import "github.com/pkg/errors"

// New initializes a new fork choice store.
func New(justifiedEpoch uint64, finalizedEpoch uint64, finalizedRoot [32]byte) *ForkChoice {
	s := &Store{
		justifiedEpoch: justifiedEpoch,
		finalizedEpoch: finalizedEpoch,
		finalizedRoot:  finalizedRoot,
		nodes:          make([]*Node, 0),
		nodeIndices:    make(map[[32]byte]uint64),
	}

	b := make([]uint64, 0)
	v := make([]Vote, 0)

	s.nodeIndices[finalizedRoot] = 0
	s.nodes = append(s.nodes,& Node{
		slot:           0,
		root:           finalizedRoot,
		parent:         nonExistentNode,
		justifiedEpoch: justifiedEpoch,
		finalizedEpoch: finalizedEpoch,
		bestChild:      nonExistentNode,
		bestDescendant: nonExistentNode,
		weight:         0,
	})

	return &ForkChoice{store: s, balances: b, votes: v}
}

// Head returns the head root from fork choice store.
func (f *ForkChoice) Head(justifiedEpoch uint64, finalizedEpoch uint64, justifiedRoot [32]byte, justifiedStateBalances []uint64) ([32]byte, error) {
	newBalances := justifiedStateBalances

	deltas, newVotes, err := computeDeltas(f.store.nodeIndices, f.votes, f.balances, newBalances)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "Could not compute deltas")
	}
	f.votes = newVotes

	if err := f.store.applyScoreChanges(justifiedEpoch, finalizedEpoch, deltas); err != nil {
		return [32]byte{}, errors.Wrap(err, "Could not apply score changes")
	}
	f.balances = newBalances

	return f.store.head(justifiedRoot)
}

// ProcessAttestation processes attestation for vote accounting to be used for fork choice.
func (f *ForkChoice) ProcessAttestation(validatorIndices []uint64, blockRoot [32]byte, targetEpoch uint64) {
	for _, index := range validatorIndices {
		// Validator indices will grow the votes cache on demand.
		newIndex := false
		for index >= uint64(len(f.votes)) {
			f.votes = append(f.votes, Vote{})
			newIndex = true
		}

		if targetEpoch > f.votes[index].nextEpoch || newIndex {
			f.votes[index].nextEpoch = targetEpoch
			f.votes[index].nextRoot = blockRoot
		}
	}
}

// ProcessBlock processes block by inserting it to the fork choice store.
func (f *ForkChoice) ProcessBlock(slot uint64, blockRoot [32]byte, parentRoot [32]byte, justifiedEpoch uint64, finalizedEpoch uint64) error {
	return f.store.insert(slot, blockRoot, parentRoot, justifiedEpoch, finalizedEpoch)
}
