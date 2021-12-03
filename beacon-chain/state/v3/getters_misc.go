package v3

import (
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
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

	return b.genesisTimeInternal()
}

// genesisTimeInternal of the beacon state as a uint64.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) genesisTimeInternal() uint64 {
	if !b.hasInnerState() {
		return 0
	}

	return b.genesisTime
}

// GenesisValidatorRoot of the beacon state.
func (b *BeaconState) GenesisValidatorRoot() [32]byte {
	if !b.hasInnerState() {
		return params.BeaconConfig().ZeroHash
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.genesisValidatorRootInternal()
}

// genesisValidatorRootInternal of the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) genesisValidatorRootInternal() [32]byte {
	if !b.hasInnerState() {
		return params.BeaconConfig().ZeroHash
	}

	return b.genesisValidatorsRoot
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

	return time.Unix(int64(b.genesisTime), 0)
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

	if b.latestBlockHeader == nil {
		return [32]byte{}
	}

	parentRoot := [32]byte{}
	copy(parentRoot[:], b.latestBlockHeader.ParentRoot)
	return parentRoot
}

// Version of the beacon state. This method
// is strictly meant to be used without a lock
// internally.
func (b *BeaconState) Version() int {
	return version.Merge
}

// Slot of the current beacon chain state.
func (b *BeaconState) Slot() types.Slot {
	if !b.hasInnerState() {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.slotInternal()
}

// slotInternal of the current beacon chain state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) slotInternal() types.Slot {
	if !b.hasInnerState() {
		return 0
	}

	return b.slot
}

// Fork version of the beacon chain.
func (b *BeaconState) Fork() *ethpb.Fork {
	if !b.hasInnerState() {
		return nil
	}
	if b.fork == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.forkInternal()
}

// forkInternal version of the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) forkInternal() *ethpb.Fork {
	if !b.hasInnerState() {
		return nil
	}
	if b.fork == nil {
		return nil
	}

	prevVersion := make([]byte, len(b.fork.PreviousVersion))
	copy(prevVersion, b.fork.PreviousVersion)
	currVersion := make([]byte, len(b.fork.CurrentVersion))
	copy(currVersion, b.fork.CurrentVersion)
	return &ethpb.Fork{
		PreviousVersion: prevVersion,
		CurrentVersion:  currVersion,
		Epoch:           b.fork.Epoch,
	}
}

// HistoricalRoots based on epochs stored in the beacon state.
func (b *BeaconState) HistoricalRoots() [][32]byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.historicalRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.historicalRootsInternal()
}

// historicalRootsInternal based on epochs stored in the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) historicalRootsInternal() customtypes.HistoricalRoots {
	if !b.hasInnerState() {
		return nil
	}
	return bytesutil.SafeCopy2d32Bytes(b.historicalRoots)
}

// balancesLength returns the length of the balances slice.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) balancesLength() int {
	if !b.hasInnerState() {
		return 0
	}
	if b.balances == nil {
		return 0
	}

	return len(b.balances)
}
