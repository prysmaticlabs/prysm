package state_native

import (
	nativetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// SetEth1Data for the beacon state.
func (b *BeaconState) SetEth1Data(val *ethpb.Eth1Data) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.eth1Data = val
	b.markFieldAsDirty(nativetypes.Eth1Data)
	return nil
}

// SetEth1DataVotes for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetEth1DataVotes(val []*ethpb.Eth1Data) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[nativetypes.Eth1DataVotes].MinusRef()
	b.sharedFieldReferences[nativetypes.Eth1DataVotes] = stateutil.NewRef(1)

	b.eth1DataVotes = val
	b.markFieldAsDirty(nativetypes.Eth1DataVotes)
	b.rebuildTrie[nativetypes.Eth1DataVotes] = true
	return nil
}

// SetEth1DepositIndex for the beacon state.
func (b *BeaconState) SetEth1DepositIndex(val uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.eth1DepositIndex = val
	b.markFieldAsDirty(nativetypes.Eth1DepositIndex)
	return nil
}

// AppendEth1DataVotes for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendEth1DataVotes(val *ethpb.Eth1Data) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	votes := b.eth1DataVotes
	if b.sharedFieldReferences[nativetypes.Eth1DataVotes].Refs() > 1 {
		// Copy elements in underlying array by reference.
		votes = make([]*ethpb.Eth1Data, len(b.eth1DataVotes))
		copy(votes, b.eth1DataVotes)
		b.sharedFieldReferences[nativetypes.Eth1DataVotes].MinusRef()
		b.sharedFieldReferences[nativetypes.Eth1DataVotes] = stateutil.NewRef(1)
	}

	b.eth1DataVotes = append(votes, val)
	b.markFieldAsDirty(nativetypes.Eth1DataVotes)
	b.addDirtyIndices(nativetypes.Eth1DataVotes, []uint64{uint64(len(b.eth1DataVotes) - 1)})
	return nil
}
