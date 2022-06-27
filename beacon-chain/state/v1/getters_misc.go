package v1

import (
	"bytes"
	"errors"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/attestation"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/time/slots"
	"golang.org/x/net/context"
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

// GenesisValidatorsRoot of the beacon state.
func (b *BeaconState) GenesisValidatorsRoot() []byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.GenesisValidatorsRoot == nil {
		return params.BeaconConfig().ZeroHash[:]
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.genesisValidatorsRoot()
}

// genesisValidatorsRoot of the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) genesisValidatorsRoot() []byte {
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

// Version of the beacon state. This method
// is strictly meant to be used without a lock
// internally.
func (_ *BeaconState) Version() int {
	return version.Phase0
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

// UnrealizedCheckpointBalances returns the total balances: active, target attested in
// previous epoch and target attested in current epoch. This function is used to
// compute the "unrealized justification" that a synced Beacon Block will have.
// This function is less efficient than the corresponding function for Altair
// and Bellatrix types as it will not be used except in syncing from genesis and
// spectests.
func (b *BeaconState) UnrealizedCheckpointBalances(ctx context.Context) (uint64, uint64, uint64, error) {
	if !b.hasInnerState() {
		return 0, 0, 0, ErrNilInnerState
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	targetIdx := params.BeaconConfig().TimelyTargetFlagIndex

	currentEpoch := time.CurrentEpoch(b)
	var currentRoot []byte
	var err error
	if slots.SinceEpochStarts(b.Slot()) > 0 {
		currentRoot, err = helpers.BlockRoot(b, currentEpoch)
		if err != nil {
			return 0, 0, 0, err
		}
	}
	cp := make([]byte, len(b.state.Validators))
	pp := make([]byte, len(b.state.Validators))

	currAtt := b.state.CurrentEpochAttestations

	prevEpoch := time.PrevEpoch(b)
	prevRoot, err := helpers.BlockRoot(b, prevEpoch)
	if err != nil {
		return 0, 0, 0, err
	}
	prevAtt := b.state.PreviousEpochAttestations

	for _, a := range append(prevAtt, currAtt...) {
		if a.InclusionDelay == 0 {
			return 0, 0, 0, errors.New("attestation with inclusion delay of 0")
		}
		currTarget := a.Data.Target.Epoch == currentEpoch && bytes.Equal(a.Data.Target.Root, currentRoot)
		prevTarget := a.Data.Target.Epoch == prevEpoch && bytes.Equal(a.Data.Target.Root, prevRoot)
		if currTarget || prevTarget {
			committee, err := helpers.BeaconCommitteeFromState(ctx, b, a.Data.Slot, a.Data.CommitteeIndex)
			if err != nil {
				return 0, 0, 0, err
			}
			indices, err := attestation.AttestingIndices(a.AggregationBits, committee)
			if err != nil {
				return 0, 0, 0, err
			}
			for _, i := range indices {
				if currTarget {
					cp[i] = (1 << targetIdx)
				}
				if prevTarget {
					pp[i] = (1 << targetIdx)
				}
			}
		}
	}

	return stateutil.UnrealizedCheckpointBalances(cp, pp, b.state.Validators, currentEpoch)
}
