package state_native

import (
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// GenesisTime of the beacon state as a uint64.
func (b *State) GenesisTime() uint64 {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.genesisTime
}

// GenesisValidatorsRoot of the beacon state.
func (b *State) GenesisValidatorsRoot() []byte {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.genesisValidatorsRoot[:]
}

// Version of the beacon state. This method
// is strictly meant to be used without a lock
// internally.
func (b *State) Version() int {
	return b.version
}

// Slot of the current beacon chain state.
func (b *State) Slot() primitives.Slot {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.slot
}

// Fork version of the beacon chain.
func (b *State) Fork() *ethpb.Fork {
	if b.fork == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.forkVal()
}

// forkVal version of the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *State) forkVal() *ethpb.Fork {
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
func (b *State) HistoricalRoots() ([][]byte, error) {
	if b.historicalRoots == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.historicalRoots.Slice(), nil
}

// balancesLength returns the length of the balances slice.
// This assumes that a lock is already held on BeaconState.
func (b *State) balancesLength() int {
	if b.balances == nil {
		return 0
	}

	return len(b.balances)
}

// HistoricalSummaries of the beacon state.
func (b *State) HistoricalSummaries() ([]*ethpb.HistoricalSummary, error) {
	if b.version < version.Capella {
		return nil, errNotSupported("HistoricalSummaries", b.version)
	}

	if b.historicalSummaries == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.historicalSummariesVal(), nil
}

// historicalSummariesVal of the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *State) historicalSummariesVal() []*ethpb.HistoricalSummary {
	return ethpb.CopyHistoricalSummaries(b.historicalSummaries)
}
