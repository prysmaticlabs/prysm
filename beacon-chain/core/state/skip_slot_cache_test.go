package state_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestSkipSlotCache_OK(t *testing.T) {
	bState, privs := testutil.DeterministicGenesisState(t, params.MinimalSpecConfig().MinGenesisActiveValidatorCount)
	originalState, _ := beaconstate.InitializeFromProto(bState.CloneInnerState())

	blkCfg := testutil.DefaultBlockGenConfig()
	blkCfg.NumAttestations = 1

	cfg := featureconfig.Get()
	cfg.EnableSkipSlotsCache = true
	featureconfig.Init(cfg)
	defer func() {
		cfg.EnableSkipSlotsCache = false
		featureconfig.Init(cfg)
	}()

	// First transition will be with an empty cache, so the cache becomes populated
	// with the state
	blk, err := testutil.GenerateFullBlock(bState, privs, blkCfg, originalState.Slot()+10)
	if err != nil {
		t.Fatal(err)
	}
	originalState, err = state.ExecuteStateTransition(context.Background(), originalState, blk)
	if err != nil {
		t.Fatalf("Could not run state transition: %v", err)
	}

	bState, err = state.ExecuteStateTransition(context.Background(), bState, blk)
	if err != nil {
		t.Fatalf("Could not process state transition: %v", err)
	}

	if !ssz.DeepEqual(originalState.CloneInnerState(), bState.CloneInnerState()) {
		t.Fatal("Skipped slots cache leads to different states")
	}
}
