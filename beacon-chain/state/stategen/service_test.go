package stategen

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestResume(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	root := [32]byte{'A'}
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch))
	require.NoError(t, service.beaconDB.SaveState(ctx, beaconState, root))
	require.NoError(t, service.beaconDB.SaveArchivedPointRoot(ctx, root, 1))
	require.NoError(t, service.beaconDB.SaveLastArchivedIndex(ctx, 1))

	resumeState, err := service.Resume(ctx)
	require.NoError(t, err)

	if !proto.Equal(beaconState.InnerStateUnsafe(), resumeState.InnerStateUnsafe()) {
		t.Error("Diff saved state")
	}
	assert.Equal(t, params.BeaconConfig().SlotsPerEpoch, service.finalizedInfo.slot, "Did not get watned slot")
	assert.Equal(t, service.finalizedInfo.root, root, "Did not get wanted root")
	assert.NotNil(t, service.finalizedState(), "Wanted a non nil finalized state")
}
