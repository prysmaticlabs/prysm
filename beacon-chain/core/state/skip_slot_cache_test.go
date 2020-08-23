package state_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSkipSlotCache_OK(t *testing.T) {
	state.SkipSlotCache.Enable()
	defer state.SkipSlotCache.Disable()
	bState, privs := testutil.DeterministicGenesisState(t, params.MinimalSpecConfig().MinGenesisActiveValidatorCount)
	originalState, err := beaconstate.InitializeFromProto(bState.CloneInnerState())
	require.NoError(t, err)

	blkCfg := testutil.DefaultBlockGenConfig()
	blkCfg.NumAttestations = 1

	// First transition will be with an empty cache, so the cache becomes populated
	// with the state
	blk, err := testutil.GenerateFullBlock(bState, privs, blkCfg, originalState.Slot()+10)
	require.NoError(t, err)
	originalState, err = state.ExecuteStateTransition(context.Background(), originalState, blk)
	require.NoError(t, err, "Could not run state transition")

	bState, err = state.ExecuteStateTransition(context.Background(), bState, blk)
	require.NoError(t, err, "Could not process state transition")

	if !ssz.DeepEqual(originalState.CloneInnerState(), bState.CloneInnerState()) {
		t.Fatal("Skipped slots cache leads to different states")
	}
}
