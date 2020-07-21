package debug

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestServer_GetBeaconState(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	db, sc := dbTest.SetupDB(t)
	ctx := context.Background()
	st := testutil.NewBeaconState()
	slot := uint64(100)
	require.NoError(t, st.SetSlot(slot))
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{
		Slot: slot,
	}}
	require.NoError(t, db.SaveBlock(ctx, b))
	gRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	gen := stategen.New(db, sc)
	require.NoError(t, gen.SaveState(ctx, gRoot, st))
	require.NoError(t, db.SaveState(ctx, st, gRoot))
	bs := &Server{
		StateGen:           gen,
		GenesisTimeFetcher: &mock.ChainService{},
	}
	_, err = bs.GetBeaconState(ctx, &pbrpc.BeaconStateRequest{})
	assert.ErrorContains(t, "Need to specify either a block root or slot to request state", err)
	req := &pbrpc.BeaconStateRequest{
		QueryFilter: &pbrpc.BeaconStateRequest_BlockRoot{
			BlockRoot: gRoot[:],
		},
	}
	res, err := bs.GetBeaconState(ctx, req)
	require.NoError(t, err)
	wanted, err := st.CloneInnerState().MarshalSSZ()
	require.NoError(t, err)
	assert.DeepEqual(t, wanted, res.Encoded)
	req = &pbrpc.BeaconStateRequest{
		QueryFilter: &pbrpc.BeaconStateRequest_Slot{
			Slot: slot,
		},
	}
	res, err = bs.GetBeaconState(ctx, req)
	require.NoError(t, err)
	assert.DeepEqual(t, wanted, res.Encoded)
}

func TestServer_GetBeaconState_RequestFutureSlot(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	ds := &Server{GenesisTimeFetcher: &mock.ChainService{}}
	req := &pbrpc.BeaconStateRequest{
		QueryFilter: &pbrpc.BeaconStateRequest_Slot{
			Slot: ds.GenesisTimeFetcher.CurrentSlot() + 1,
		},
	}
	wanted := "Cannot retrieve information about a slot in the future"
	_, err := ds.GetBeaconState(context.Background(), req)
	assert.ErrorContains(t, wanted, err)
}
