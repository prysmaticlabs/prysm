package state_native

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

// SetDepositReceiptsStartIndex for the beacon state. Updates the DepositReceiptsStartIndex
func (b *BeaconState) SetDepositReceiptsStartIndex(index uint64) error {
	if b.version < version.Electra {
		return errNotSupported("SetDepositReceiptsStartIndex", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.depositReceiptsStartIndex = index
	b.markFieldAsDirty(types.DepositReceiptsStartIndex)
	b.rebuildTrie[types.DepositReceiptsStartIndex] = true
	return nil
}
