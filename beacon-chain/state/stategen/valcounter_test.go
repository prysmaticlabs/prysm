package stategen

import (
	"context"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
)

func TestLastFinalizedValidatorCounter(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	sg := New(beaconDB)
	lf := NewLastFinalizedValidatorCounter(0, beaconDB, sg)
	c, err := lf.ActiveValidatorCount(ctx)
	require.Equal(t, uint64(0), c)
	require.ErrorIs(t, err, errUnknownState)

	var genCount uint64 = 64
	gst, err := util.NewBeaconState(valPopulator(genCount))
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveGenesisData(ctx, gst))
	c, err = lf.ActiveValidatorCount(ctx)
	require.NoError(t, err)
	require.Equal(t, genCount, c)

	finCount := params.BeaconConfig().MinGenesisActiveValidatorCount
	var expectedSlot types.Slot = 3200
	st, err := util.NewBeaconState(valPopulator(finCount))
	require.NoError(t, err)
	require.NoError(t, st.SetSlot(expectedSlot))

	// get the genesis block so we can use a valid parent root when for the new final block
	gb, err := beaconDB.GenesisBlock(ctx)
	require.NoError(t, err)
	gbr, err := gb.Block().HashTreeRoot()
	require.NoError(t, err)

	b := util.NewBeaconBlock()
	b.Block.ParentRoot = gbr[:]
	b.Block.Slot = expectedSlot

	br, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := blocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(context.Background(), wsb))
	require.NoError(t, beaconDB.SaveState(ctx, st, br))
	require.NoError(t, beaconDB.SaveFinalizedCheckpoint(context.Background(), &ethpb.Checkpoint{
		Root:  br[:],
		Epoch: types.Epoch(expectedSlot / params.BeaconConfig().SlotsPerEpoch),
	}))
	c, err = lf.ActiveValidatorCount(ctx)
	require.NoError(t, err)

	// genCount will still be cached
	require.Equal(t, genCount, c)

	// reach in and reset the cache to cause it to miss and repopulate
	lf.count = 0
	c, err = lf.ActiveValidatorCount(ctx)
	require.NoError(t, err)

	// updated value should match most recently finalized
	require.Equal(t, finCount, c)
}

func valPopulator(valCount uint64) func(state *ethpb.BeaconState) error {
	return func(state *ethpb.BeaconState) error {
		validators := make([]*ethpb.Validator, valCount)
		for i := 0; i < len(validators); i++ {
			validators[i] = &ethpb.Validator{
				PublicKey:             make([]byte, 48),
				WithdrawalCredentials: make([]byte, 32),
				ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
				Slashed:               false,
			}
		}
		state.Validators = validators
		return nil
	}
}
