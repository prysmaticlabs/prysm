package debug

import (
	"bytes"
	"context"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestServer_GetBeaconState(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	db, sc := dbTest.SetupDB(t)
	ctx := context.Background()
	st := testutil.NewBeaconState()
	slot := uint64(100)
	if err := st.SetSlot(slot); err != nil {
		t.Fatal(err)
	}
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{
		Slot: slot,
	}}
	if err := db.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	gRoot, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	gen := stategen.New(db, sc)
	if err := gen.SaveState(ctx, gRoot, st); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, st, gRoot); err != nil {
		t.Fatal(err)
	}
	bs := &Server{
		StateGen:           gen,
		GenesisTimeFetcher: &mock.ChainService{},
	}
	if _, err := bs.GetBeaconState(ctx, &pbrpc.BeaconStateRequest{}); err == nil {
		t.Errorf("Expected error without a query filter, received nil")
	}
	req := &pbrpc.BeaconStateRequest{
		QueryFilter: &pbrpc.BeaconStateRequest_BlockRoot{
			BlockRoot: gRoot[:],
		},
	}
	res, err := bs.GetBeaconState(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	wanted, err := st.CloneInnerState().MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(wanted, res.Encoded) {
		t.Errorf("Wanted %v, received %v", wanted, res.Encoded)
	}
	req = &pbrpc.BeaconStateRequest{
		QueryFilter: &pbrpc.BeaconStateRequest_Slot{
			Slot: slot,
		},
	}
	res, err = bs.GetBeaconState(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(wanted, res.Encoded) {
		t.Errorf("Wanted %v, received %v", wanted, res.Encoded)
	}
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
	if _, err := ds.GetBeaconState(context.Background(), req); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}
