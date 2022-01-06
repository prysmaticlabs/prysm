package v3

import (
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

// GenesisTime of the beacon state as a uint64.
func (b *BeaconState) GenesisTime() uint64 {
	if !b.hasInnerState() {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.genesisTime()
}

// genesisTime of the beacon state as a uint64.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) genesisTime() uint64 {
	if !b.hasInnerState() {
		return 0
	}

	return b.state.GenesisTime
}

// GenesisValidatorRoot of the beacon state.
func (b *BeaconState) GenesisValidatorRoot() []byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.GenesisValidatorsRoot == nil {
		return params.BeaconConfig().ZeroHash[:]
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.genesisValidatorRoot()
}

// genesisValidatorRoot of the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) genesisValidatorRoot() []byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.GenesisValidatorsRoot == nil {
		return params.BeaconConfig().ZeroHash[:]
	}

	root := make([]byte, 32)
	copy(root, b.state.GenesisValidatorsRoot)
	return root
}

// GenesisUnixTime returns the genesis time as time.Time.
func (b *BeaconState) GenesisUnixTime() time.Time {
	if !b.hasInnerState() {
		return time.Unix(0, 0)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.genesisUnixTime()
}

// genesisUnixTime returns the genesis time as time.Time.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) genesisUnixTime() time.Time {
	if !b.hasInnerState() {
		return time.Unix(0, 0)
	}

	return time.Unix(int64(b.state.GenesisTime), 0)
}

// ParentRoot is a convenience method to access state.LatestBlockRoot.ParentRoot.
func (b *BeaconState) ParentRoot() [32]byte {
	if !b.hasInnerState() {
		return [32]byte{}
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.parentRoot()
}

// parentRoot is a convenience method to access state.LatestBlockRoot.ParentRoot.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) parentRoot() [32]byte {
	if !b.hasInnerState() {
		return [32]byte{}
	}

	parentRoot := [32]byte{}
	copy(parentRoot[:], b.state.LatestBlockHeader.ParentRoot)
	return parentRoot
}

// Version of the beacon state. This method
// is strictly meant to be used without a lock
// internally.
func (_ *BeaconState) Version() int {
	return version.Merge
}

// Slot of the current beacon chain state.
func (b *BeaconState) Slot() types.Slot {
	if !b.hasInnerState() {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.slot()
}

// slot of the current beacon chain state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) slot() types.Slot {
	if !b.hasInnerState() {
		return 0
	}

	return b.state.Slot
}

// Fork version of the beacon chain.
func (b *BeaconState) Fork() *ethpb.Fork {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Fork == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.fork()
}

// fork version of the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) fork() *ethpb.Fork {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Fork == nil {
		return nil
	}

	prevVersion := make([]byte, len(b.state.Fork.PreviousVersion))
	copy(prevVersion, b.state.Fork.PreviousVersion)
	currVersion := make([]byte, len(b.state.Fork.CurrentVersion))
	copy(currVersion, b.state.Fork.CurrentVersion)
	return &ethpb.Fork{
		PreviousVersion: prevVersion,
		CurrentVersion:  currVersion,
		Epoch:           b.state.Fork.Epoch,
	}
}

// HistoricalRoots based on epochs stored in the beacon state.
func (b *BeaconState) HistoricalRoots() [][]byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.HistoricalRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.historicalRoots()
}

// historicalRoots based on epochs stored in the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) historicalRoots() [][]byte {
	if !b.hasInnerState() {
		return nil
	}
	return bytesutil.SafeCopy2dBytes(b.state.HistoricalRoots)
}

// balancesLength returns the length of the balances slice.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) balancesLength() int {
	if !b.hasInnerState() {
		return 0
	}
	if b.state.Balances == nil {
		return 0
	}

	return len(b.state.Balances)
}
