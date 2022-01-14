package v3

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// SetEth1Data for the beacon state.
func (b *BeaconState) SetEth1Data(val *ethpb.Eth1Data) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.Eth1Data = val
	b.markFieldAsDirty(eth1Data)
	return nil
}

// SetEth1DataVotes for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetEth1DataVotes(val []*ethpb.Eth1Data) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[eth1DataVotes].MinusRef()
	b.sharedFieldReferences[eth1DataVotes] = stateutil.NewRef(1)

	b.state.Eth1DataVotes = val
	b.markFieldAsDirty(eth1DataVotes)
	b.rebuildTrie[eth1DataVotes] = true
	return nil
}

// SetEth1DepositIndex for the beacon state.
func (b *BeaconState) SetEth1DepositIndex(val uint64) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.Eth1DepositIndex = val
	b.markFieldAsDirty(eth1DepositIndex)
	return nil
}

// AppendEth1DataVotes for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendEth1DataVotes(val *ethpb.Eth1Data) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	votes := b.state.Eth1DataVotes
	if b.sharedFieldReferences[eth1DataVotes].Refs() > 1 {
		// Copy elements in underlying array by reference.
		votes = make([]*ethpb.Eth1Data, len(b.state.Eth1DataVotes))
		copy(votes, b.state.Eth1DataVotes)
		b.sharedFieldReferences[eth1DataVotes].MinusRef()
		b.sharedFieldReferences[eth1DataVotes] = stateutil.NewRef(1)
	}

	b.state.Eth1DataVotes = append(votes, val)
	b.markFieldAsDirty(eth1DataVotes)
	b.addDirtyIndices(eth1DataVotes, []uint64{uint64(len(b.state.Eth1DataVotes) - 1)})
	return nil
}
