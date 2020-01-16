package protoarray

// New initializes a new fork choice store.
func New(justifiedEpoch uint64, finalizedEpoch uint64, finalizedRoot [32]byte) *ForkChoice {
	s := &Store{
		justifiedEpoch: justifiedEpoch,
		finalizedEpoch: finalizedEpoch,
		finalizedRoot:  finalizedRoot,
		nodes:          make([]Node, 0),
		nodeIndices:    make(map[[32]byte]uint64),
	}

	//if err := f.store.insert(finalizedSlot, finalizedRoot, params.BeaconConfig().ZeroHash, justifiedEpoch, finalizedEpoch); err != nil {
	//	return err
	//}

	b := make([]uint64, 0)
	v := make([]Vote, 0)

	return &ForkChoice{store: s, balances: b, votes: v}
}

// Head returns the head root from fork choice store.
func (f *ForkChoice) Head(justifiedEpoch uint64, finalizedEpoch uint64, justifiedRoot [32]byte, justifiedStateBalances []uint64) ([32]byte, error) {
	newBalances := justifiedStateBalances
	deltas, err := computeDeltas(f.store.nodeIndices, f.votes, f.balances, newBalances)
	if err != nil {
		return [32]byte{}, err
	}

	if err := f.store.applyScoreChanges(justifiedEpoch, finalizedEpoch, deltas); err != nil {
		return [32]byte{}, err
	}

	return f.store.head(justifiedRoot)
}

// ProcessAttestation processes attestation for vote accounting to be used for fork choice.
func (f *ForkChoice) ProcessAttestation(validatorIndex uint64, blockRoot [32]byte, targetEpoch uint64) {
	if targetEpoch > f.votes[validatorIndex].nextEpoch {
		f.votes[validatorIndex].nextEpoch = targetEpoch
		f.votes[validatorIndex].nextRoot = blockRoot
	}
}

// ProcessBlock processes block by inserting it to the fork choice store.
func (f *ForkChoice) ProcessBlock(slot uint64, blockRoot [32]byte, parentRoot [32]byte, finalizedEpoch uint64, justifiedEpoch uint64) error {
	return f.store.insert(slot, blockRoot, parentRoot, justifiedEpoch, finalizedEpoch)
}
