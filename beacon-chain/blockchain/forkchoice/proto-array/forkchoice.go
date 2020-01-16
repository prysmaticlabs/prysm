package protoarray

import "github.com/prysmaticlabs/prysm/shared/params"

// New initializes a new fork choice store.
func (f *ForkChoice) New(justifiedEpoch uint64, finalizedEpoch uint64, finalizedSlot uint64, finalizedRoot [32]byte) {
	f.store = &Store{
		justifiedEpoch: justifiedEpoch,
		finalizedEpoch: finalizedEpoch,
		finalizedRoot:  finalizedRoot,
		nodes:          make([]Node, 0),
		nodeIndices:    make(map[[32]byte]uint64),
	}

	f.store.Insert(finalizedSlot, finalizedRoot, params.BeaconConfig().ZeroHash, justifiedEpoch, finalizedEpoch)

	f.balances = make([]uint64, 0)
	f.votes = make([]Vote, 0)
}

// ProcessAttestation processes attestation for vote accounting to be used for fork choice.
func (f *ForkChoice) ProcessAttestation(validatorIndex uint64, blockRoot [32]byte, targetEpoch uint64) {
	if targetEpoch > f.votes[validatorIndex].nextEpoch {
		f.votes[validatorIndex].nextEpoch = targetEpoch
		f.votes[validatorIndex].nextRoot = blockRoot
	}
}

// ProcessBlock processes block by inserting it to the fork choice store.
func (f *ForkChoice) ProcessBlock(slot uint64, blockRoot [32]byte, parentRoot [32]byte, finalizedEpoch uint64, justifiedEpoch uint64) {
	f.store.Insert(slot, blockRoot, parentRoot, justifiedEpoch, finalizedEpoch)
}
