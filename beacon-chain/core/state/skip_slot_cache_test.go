package state_test

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestSkipSlotCache_OK(t *testing.T) {
	deps, _, privs := testutil.SetupInitialDeposits(t, params.MinimalSpecConfig().MinGenesisActiveValidatorCount)
	bState, err := state.GenesisBeaconState(deps, uint64(time.Now().Unix()), testutil.GenerateEth1Data(t, deps))
	if err != nil {
		t.Fatalf("Could not generate genesis state: %v", err)
	}

	originalState := proto.Clone(bState).(*pb.BeaconState)

	blkCfg := testutil.DefaultBlockGenConfig()
	blkCfg.MaxAttestations = 1
	blkCfg.MaxDeposits = 0
	blkCfg.MaxVoluntaryExits = 0
	blkCfg.MaxProposerSlashings = 0
	blkCfg.MaxAttesterSlashings = 0

	cfg := featureconfig.Get()
	cfg.EnableSkipSlotsCache = true
	featureconfig.Init(cfg)

	// First transition will be with an empty cache, so the cache becomes populated
	// with the state
	blk := testutil.GenerateFullBlock(t, bState, privs, blkCfg, originalState.Slot+10)
	originalState, err = state.ExecuteStateTransition(context.Background(), originalState, blk)
	if err != nil {
		t.Fatalf("Could not run state transition: %v", err)
	}

	bState, err = state.ExecuteStateTransition(context.Background(), bState, blk)
	if err != nil {
		t.Fatalf("Could not process state transition: %v", err)
	}

	if !ssz.DeepEqual(originalState, bState) {
		t.Fatal("Skipped slots cache leads to different states")
	}

}
