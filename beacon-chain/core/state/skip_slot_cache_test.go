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

	blkCfg := testutil.DefaultBlockGenConfig()
	blkCfg.MaxAttestations = 1
	blkCfg.MaxDeposits = 0
	blkCfg.MaxVoluntaryExits = 0
	blkCfg.MaxProposerSlashings = 0
	blkCfg.MaxAttesterSlashings = 0

	for i := 0; i < 5; i++ {
		blk := testutil.GenerateFullBlock(t, bState, privs, blkCfg)
		bState, err = state.ExecuteStateTransition(context.Background(), bState, blk)
		if err != nil {
			t.Fatalf("Could not run state transition: %v", err)
		}
	}
	originalState := proto.Clone(bState).(*pb.BeaconState)
	testState := proto.Clone(bState).(*pb.BeaconState)

	originalState, err = state.ProcessSlots(context.Background(), originalState, 20)
	if err != nil {
		t.Fatalf("Could not process slots: %v", err)
	}
	cfg := featureconfig.Get()
	cfg.EnableSkipSlotsCache = true
	featureconfig.Init(cfg)

	_, err = state.ProcessSlots(context.Background(), testState, 20)
	if err != nil {
		t.Fatalf("Could not process slots: %v", err)
	}

	bState, err = state.ProcessSlots(context.Background(), bState, 20)
	if err != nil {
		t.Fatalf("Could not process slots: %v", err)
	}

	if !ssz.DeepEqual(originalState, bState) {
		t.Fatal("Skipped slots cache leads to different states")
	}

}
