package debug

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestServer_GetBeaconState(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	db := dbTest.SetupDB(t)
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
	gen := stategen.New(db, cache.NewStateSummaryCache())
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

func TestServer_GetIndividualVotes_RequestFutureSlot(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()
	ds := &Server{GenesisTimeFetcher: &mock.ChainService{}}
	req := &pbrpc.IndividualVotesRequest{
		Epoch: helpers.SlotToEpoch(ds.GenesisTimeFetcher.CurrentSlot()) + 1,
	}
	wanted := "Cannot retrieve information about an epoch in the future"
	if _, err := ds.GetIndividualVotes(context.Background(), req); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_GetIndividualVotes_ValidatorsDontExist(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	params.UseMinimalConfig()
	defer params.UseMainnetConfig()
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	validators := uint64(64)
	stateWithValidators, _ := testutil.DeterministicGenesisState(t, validators)
	beaconState := testutil.NewBeaconState()
	if err := beaconState.SetValidators(stateWithValidators.Validators()); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch); err != nil {
		t.Fatal(err)
	}

	b := testutil.NewBeaconBlock()
	b.Block.Slot = params.BeaconConfig().SlotsPerEpoch
	if err := db.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	gRoot, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	gen := stategen.New(db, cache.NewStateSummaryCache())
	if err := gen.SaveState(ctx, gRoot, beaconState); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, gRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, gRoot); err != nil {
		t.Fatal(err)
	}
	bs := &Server{
		StateGen:           gen,
		GenesisTimeFetcher: &mock.ChainService{},
	}

	// Test non exist public key.
	res, err := bs.GetIndividualVotes(ctx, &pbrpc.IndividualVotesRequest{
		PublicKeys: [][]byte{{'a'}},
		Epoch:      0,
	})
	if err != nil {
		t.Fatal(err)
	}
	wanted := &pbrpc.IndividualVotesRespond{
		IndividualVotes: []*pbrpc.IndividualVotesRespond_IndividualVote{
			{PublicKey: []byte{'a'}, ValidatorIndex: ^uint64(0)},
		},
	}
	if !reflect.DeepEqual(res, wanted) {
		t.Error("Did not get wanted respond")
	}

	// Test non-existent validator index.
	res, err = bs.GetIndividualVotes(ctx, &pbrpc.IndividualVotesRequest{
		Indices: []uint64{100},
		Epoch:   0,
	})
	if err != nil {
		t.Fatal(err)
	}
	wanted = &pbrpc.IndividualVotesRespond{
		IndividualVotes: []*pbrpc.IndividualVotesRespond_IndividualVote{
			{ValidatorIndex: 100},
		},
	}
	if !reflect.DeepEqual(res, wanted) {
		t.Log(res, wanted)
		t.Error("Did not get wanted respond")
	}

	// Test both.
	res, err = bs.GetIndividualVotes(ctx, &pbrpc.IndividualVotesRequest{
		PublicKeys: [][]byte{{'a'}, {'b'}},
		Indices:    []uint64{100, 101},
		Epoch:      0,
	})
	if err != nil {
		t.Fatal(err)
	}
	wanted = &pbrpc.IndividualVotesRespond{
		IndividualVotes: []*pbrpc.IndividualVotesRespond_IndividualVote{
			{PublicKey: []byte{'a'}, ValidatorIndex: ^uint64(0)},
			{PublicKey: []byte{'b'}, ValidatorIndex: ^uint64(0)},
			{ValidatorIndex: 100},
			{ValidatorIndex: 101},
		},
	}
	if !reflect.DeepEqual(res, wanted) {
		t.Log(res, wanted)
		t.Error("Did not get wanted respond")
	}
}

func TestServer_GetIndividualVotes_Working(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	params.UseMinimalConfig()
	defer params.UseMainnetConfig()
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	validators := uint64(64)
	stateWithValidators, _ := testutil.DeterministicGenesisState(t, validators)
	beaconState := testutil.NewBeaconState()
	if err := beaconState.SetValidators(stateWithValidators.Validators()); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch); err != nil {
		t.Fatal(err)
	}

	bf := []byte{0xff}
	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 0}},
		AggregationBits: bf}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 0}},
		AggregationBits: bf}
	rt := [32]byte{'A'}
	att1.Data.Target.Root = rt[:]
	att1.Data.BeaconBlockRoot = rt[:]
	br := beaconState.BlockRoots()
	newRt := [32]byte{'B'}
	br[0] = newRt[:]
	if err := beaconState.SetBlockRoots(br); err != nil {
		t.Fatal(err)
	}
	att2.Data.Target.Root = rt[:]
	att2.Data.BeaconBlockRoot = newRt[:]
	err := beaconState.SetPreviousEpochAttestations([]*pb.PendingAttestation{{Data: att1.Data, AggregationBits: bf}})
	if err != nil {
		t.Fatal(err)
	}
	err = beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{{Data: att2.Data, AggregationBits: bf}})
	if err != nil {
		t.Fatal(err)
	}

	b := testutil.NewBeaconBlock()
	b.Block.Slot = params.BeaconConfig().SlotsPerEpoch
	if err := db.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	gRoot, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	gen := stategen.New(db, cache.NewStateSummaryCache())
	if err := gen.SaveState(ctx, gRoot, beaconState); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, gRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, gRoot); err != nil {
		t.Fatal(err)
	}
	bs := &Server{
		StateGen:           gen,
		GenesisTimeFetcher: &mock.ChainService{},
	}

	res, err := bs.GetIndividualVotes(ctx, &pbrpc.IndividualVotesRequest{
		Indices: []uint64{0, 1},
		Epoch:   0,
	})
	if err != nil {
		t.Fatal(err)
	}
	wanted := &pbrpc.IndividualVotesRespond{
		IndividualVotes: []*pbrpc.IndividualVotesRespond_IndividualVote{
			{ValidatorIndex: 0, PublicKey: beaconState.Validators()[0].PublicKey, IsActiveInCurrentEpoch: true, IsActiveInPreviousEpoch: true,
				CurrentEpochEffectiveBalanceGwei: params.BeaconConfig().MaxEffectiveBalance, InclusionSlot: params.BeaconConfig().FarFutureEpoch, InclusionDistance: params.BeaconConfig().FarFutureEpoch},
			{ValidatorIndex: 1, PublicKey: beaconState.Validators()[1].PublicKey, IsActiveInCurrentEpoch: true, IsActiveInPreviousEpoch: true,
				CurrentEpochEffectiveBalanceGwei: params.BeaconConfig().MaxEffectiveBalance, InclusionSlot: params.BeaconConfig().FarFutureEpoch, InclusionDistance: params.BeaconConfig().FarFutureEpoch},
		},
	}
	if !reflect.DeepEqual(res, wanted) {
		t.Log(res, wanted)
		t.Error("Did not get wanted respond")
	}
}
