package state_native

import (
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

// DepositReceiptsStartIndex is used for returning the deposit receipts start index which is used for eip6110
func (b *BeaconState) DepositReceiptsStartIndex() (uint64, error) {
	if b.version < version.Electra {
		return 0, errNotSupported("DepositReceiptsStartIndex", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.depositReceiptsStartIndex, nil
}
