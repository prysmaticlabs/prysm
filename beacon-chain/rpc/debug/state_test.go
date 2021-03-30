package debug

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestServer_GetBeaconState(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()
	st, err := testutil.NewBeaconState()
	require.NoError(t, err)
	slot := types.Slot(100)
	require.NoError(t, st.SetSlot(slot))
	b := testutil.NewBeaconBlock()
	b.Block.Slot = slot
	require.NoError(t, db.SaveBlock(ctx, b))
	gRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	gen := stategen.New(db)
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
	wanted, err := st.MarshalSSZ()
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
