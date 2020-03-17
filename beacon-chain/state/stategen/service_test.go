package stategen

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestResume(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)
	root := [32]byte{'A'}
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch - 2)

	service.beaconDB.SaveArchivedPointState(ctx, beaconState, 1)
	service.beaconDB.SaveLastArchivedIndex(ctx, 1)

	resumeState, err := service.Resume(ctx, root)
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(beaconState.InnerStateUnsafe(), resumeState.InnerStateUnsafe()) {
		t.Error("Diff saved state")
	}
	if !service.beaconDB.HasStateSummary(ctx, root) {
		t.Error("Did not save state summary")
	}
	if cachedRoot, _ := service.epochBoundaryRoot(params.BeaconConfig().SlotsPerEpoch); cachedRoot != root {
		t.Error("Did not save boundary root")
	}
}

func TestVerifySlotsPerArchivePoint(t *testing.T) {
	type tc struct {
		input  uint64
		result bool
	}
	tests := []tc{
		{0, false},
		{1, false},
		{params.BeaconConfig().SlotsPerEpoch, true},
		{params.BeaconConfig().SlotsPerEpoch + 1, false},
		{params.BeaconConfig().SlotsPerHistoricalRoot, true},
		{params.BeaconConfig().SlotsPerHistoricalRoot + 1, false},
	}
	for _, tt := range tests {
		if got := verifySlotsPerArchivePoint(tt.input); got != tt.result {
			t.Errorf("verifySlotsPerArchivePoint(%d) = %v, want %v", tt.input, got, tt.result)
		}
	}
}
