package stategen

import (
	"context"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/shared/params"
	testing2 "github.com/prysmaticlabs/prysm/testing"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestResume(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)
	b := testing2.NewBeaconBlock()
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b)))
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	beaconState, _ := testing2.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch))
	require.NoError(t, service.beaconDB.SaveState(ctx, beaconState, root))
	require.NoError(t, service.beaconDB.SaveGenesisBlockRoot(ctx, root))
	require.NoError(t, service.beaconDB.SaveFinalizedCheckpoint(ctx, &ethpb.Checkpoint{Root: root[:]}))

	resumeState, err := service.Resume(ctx)
	require.NoError(t, err)
	require.DeepSSZEqual(t, beaconState.InnerStateUnsafe(), resumeState.InnerStateUnsafe())
	assert.Equal(t, params.BeaconConfig().SlotsPerEpoch, service.finalizedInfo.slot, "Did not get watned slot")
	assert.Equal(t, service.finalizedInfo.root, root, "Did not get wanted root")
	assert.NotNil(t, service.finalizedState(), "Wanted a non nil finalized state")
}
