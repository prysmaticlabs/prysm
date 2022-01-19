package v1

import (
	types "github.com/prysmaticlabs/eth2-types"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/custom-types"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

// GenesisTime of the beacon state as a uint64.
func (b *BeaconState) GenesisTime() uint64 {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.genesisTime
}

// GenesisValidatorRoot of the beacon state.
func (b *BeaconState) GenesisValidatorRoot() []byte {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.genesisValidatorsRoot[:]
}

// Version of the beacon state. This method
// is strictly meant to be used without a lock
// internally.
func (_ *BeaconState) Version() int {
	return version.Phase0
}

// Slot of the current beacon chain state.
func (b *BeaconState) Slot() types.Slot {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.slot
}

// Fork version of the beacon chain.
func (b *BeaconState) Fork() *ethpb.Fork {
	if b.fork == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.forkVal()
}

// forkVal version of the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) forkVal() *ethpb.Fork {
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
func (b *BeaconState) HistoricalRoots() [][]byte {
	if b.historicalRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	rootsArr := b.historicalRootsVal()
	roots := make([][]byte, len(rootsArr))
	for i, r := range rootsArr {
		tmp := r
		roots[i] = tmp[:]
	}

	return roots
}

// historicalRootsVal based on epochs stored in the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) historicalRootsVal() customtypes.HistoricalRoots {
	return bytesutil.SafeCopy2d32Bytes(b.historicalRoots)
}

// balancesLength returns the length of the balances slice.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) balancesLength() int {
	if b.balances == nil {
		return 0
	}

	return len(b.balances)
}
