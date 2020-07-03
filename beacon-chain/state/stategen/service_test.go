package stategen

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestResume(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	root := [32]byte{'A'}
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(ctx, beaconState, root); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveArchivedPointRoot(ctx, root, 1); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveLastArchivedIndex(ctx, 1); err != nil {
		t.Fatal(err)
	}

	resumeState, err := service.Resume(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(beaconState.InnerStateUnsafe(), resumeState.InnerStateUnsafe()) {
		t.Error("Diff saved state")
	}
	if service.finalizedInfo.slot != params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Did not get watned slot")
	}
	if root != service.finalizedInfo.root {
		t.Errorf("Did not get wanted root")
	}
}
