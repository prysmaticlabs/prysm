package beacon

import (
	"context"
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestServer_ListAssignments_CannotRequestFutureEpoch(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()
	bs := &Server{
		BeaconDB:           db,
		GenesisTimeFetcher: &mock.ChainService{},
	}

	wanted := "Cannot retrieve information about an epoch in the future"
	if _, err := bs.ListValidatorAssignments(
		ctx,
		&ethpb.ListValidatorAssignmentsRequest{
			QueryFilter: &ethpb.ListValidatorAssignmentsRequest_Epoch{
				Epoch: helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot()) + 1,
			},
		},
	); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_ListAssignments_NoResults(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	db, sc := dbTest.SetupDB(t)
	ctx := context.Background()
	st := testutil.NewBeaconState()

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := db.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	gRoot, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, gRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, st, gRoot); err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		BeaconDB:           db,
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db, sc),
	}
	wanted := &ethpb.ValidatorAssignments{
		Assignments:   make([]*ethpb.ValidatorAssignments_CommitteeAssignment, 0),
		TotalSize:     int32(0),
		NextPageToken: strconv.Itoa(0),
	}
	res, err := bs.ListValidatorAssignments(
		ctx,
		&ethpb.ListValidatorAssignmentsRequest{
			QueryFilter: &ethpb.ListValidatorAssignmentsRequest_Genesis{
				Genesis: true,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
}

func TestServer_ListAssignments_Pagination_InputOutOfRange(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	db, sc := dbTest.SetupDB(t)
	ctx := context.Background()
	setupValidators(t, db, 1)
	headState, err := db.HeadState(ctx)
	if err != nil {
		t.Fatal(err)
	}

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := db.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	gRoot, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, gRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, headState, gRoot); err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		BeaconDB:           db,
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db, sc),
	}

	wanted := fmt.Sprintf("page start %d >= list %d", 0, 0)
	if _, err := bs.ListValidatorAssignments(
		context.Background(),
		&ethpb.ListValidatorAssignmentsRequest{
			QueryFilter: &ethpb.ListValidatorAssignmentsRequest_Genesis{Genesis: true},
		},
	); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_ListAssignments_Pagination_ExceedsMaxPageSize(t *testing.T) {
	bs := &Server{}
	exceedsMax := int32(cmd.Get().MaxRPCPageSize + 1)

	wanted := fmt.Sprintf("Requested page size %d can not be greater than max size %d", exceedsMax, cmd.Get().MaxRPCPageSize)
	req := &ethpb.ListValidatorAssignmentsRequest{
		PageToken: strconv.Itoa(0),
		PageSize:  exceedsMax,
	}
	if _, err := bs.ListValidatorAssignments(context.Background(), req); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_ListAssignments_Pagination_DefaultPageSize_NoArchive(t *testing.T) {
	helpers.ClearCache()
	db, sc := dbTest.SetupDB(t)
	ctx := context.Background()
	count := 500
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.LittleEndian.PutUint64(pubKey, uint64(i))
		// Mark the validators with index divisible by 3 inactive.
		if i%3 == 0 {
			validators = append(validators, &ethpb.Validator{
				PublicKey:        pubKey,
				ExitEpoch:        0,
				ActivationEpoch:  0,
				EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
			})
		} else {
			validators = append(validators, &ethpb.Validator{
				PublicKey:        pubKey,
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
				EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
				ActivationEpoch:  0,
			})
		}
	}

	blk := &ethpb.BeaconBlock{
		Slot: 0,
	}
	blockRoot, err := ssz.HashTreeRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

	s := testutil.NewBeaconState()
	if err := s.SetValidators(validators); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, s, blockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, blockRoot); err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: s,
		},
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db, sc),
	}

	res, err := bs.ListValidatorAssignments(context.Background(), &ethpb.ListValidatorAssignmentsRequest{
		QueryFilter: &ethpb.ListValidatorAssignmentsRequest_Genesis{Genesis: true},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Construct the wanted assignments.
	var wanted []*ethpb.ValidatorAssignments_CommitteeAssignment

	activeIndices, err := helpers.ActiveValidatorIndices(s, 0)
	if err != nil {
		t.Fatal(err)
	}
	committeeAssignments, proposerIndexToSlots, err := helpers.CommitteeAssignments(s, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range activeIndices[0:params.BeaconConfig().DefaultPageSize] {
		val, err := s.ValidatorAtIndex(index)
		if err != nil {
			t.Fatal(err)
		}
		wanted = append(wanted, &ethpb.ValidatorAssignments_CommitteeAssignment{
			BeaconCommittees: committeeAssignments[index].Committee,
			CommitteeIndex:   committeeAssignments[index].CommitteeIndex,
			AttesterSlot:     committeeAssignments[index].AttesterSlot,
			ProposerSlots:    proposerIndexToSlots[index],
			PublicKey:        val.PublicKey,
			ValidatorIndex:   index,
		})
	}
	if !reflect.DeepEqual(res.Assignments, wanted) {
		t.Error("Did not receive wanted assignments")
	}
}

func TestServer_ListAssignments_FilterPubkeysIndices_NoPagination(t *testing.T) {
	helpers.ClearCache()
	db, sc := dbTest.SetupDB(t)
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	ctx := context.Background()
	count := 100
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.LittleEndian.PutUint64(pubKey, uint64(i))
		validators = append(validators, &ethpb.Validator{PublicKey: pubKey, ExitEpoch: params.BeaconConfig().FarFutureEpoch})
	}

	blk := &ethpb.BeaconBlock{
		Slot: 0,
	}
	blockRoot, err := ssz.HashTreeRoot(blk)
	if err != nil {
		t.Fatal(err)
	}
	s := testutil.NewBeaconState()
	if err := s.SetValidators(validators); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, s, blockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, blockRoot); err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		BeaconDB: db,
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db, sc),
	}

	pubKey1 := make([]byte, params.BeaconConfig().BLSPubkeyLength)
	binary.LittleEndian.PutUint64(pubKey1, 1)
	pubKey2 := make([]byte, params.BeaconConfig().BLSPubkeyLength)
	binary.LittleEndian.PutUint64(pubKey2, 2)
	req := &ethpb.ListValidatorAssignmentsRequest{PublicKeys: [][]byte{pubKey1, pubKey2}, Indices: []uint64{2, 3}}
	res, err := bs.ListValidatorAssignments(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Construct the wanted assignments.
	var wanted []*ethpb.ValidatorAssignments_CommitteeAssignment

	activeIndices, err := helpers.ActiveValidatorIndices(s, 0)
	if err != nil {
		t.Fatal(err)
	}
	committeeAssignments, proposerIndexToSlots, err := helpers.CommitteeAssignments(s, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range activeIndices[1:4] {
		val, err := s.ValidatorAtIndex(index)
		if err != nil {
			t.Fatal(err)
		}
		wanted = append(wanted, &ethpb.ValidatorAssignments_CommitteeAssignment{
			BeaconCommittees: committeeAssignments[index].Committee,
			CommitteeIndex:   committeeAssignments[index].CommitteeIndex,
			AttesterSlot:     committeeAssignments[index].AttesterSlot,
			ProposerSlots:    proposerIndexToSlots[index],
			PublicKey:        val.PublicKey,
			ValidatorIndex:   index,
		})
	}

	if !reflect.DeepEqual(res.Assignments, wanted) {
		t.Error("Did not receive wanted assignments")
	}
}

func TestServer_ListAssignments_CanFilterPubkeysIndices_WithPagination(t *testing.T) {
	helpers.ClearCache()
	db, sc := dbTest.SetupDB(t)
	ctx := context.Background()
	count := 100
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.LittleEndian.PutUint64(pubKey, uint64(i))
		validators = append(validators, &ethpb.Validator{PublicKey: pubKey, ExitEpoch: params.BeaconConfig().FarFutureEpoch})
	}

	blk := &ethpb.BeaconBlock{
		Slot: 0,
	}
	blockRoot, err := ssz.HashTreeRoot(blk)
	if err != nil {
		t.Fatal(err)
	}
	s := testutil.NewBeaconState()
	if err := s.SetValidators(validators); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, s, blockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, blockRoot); err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		BeaconDB: db,
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db, sc),
	}

	req := &ethpb.ListValidatorAssignmentsRequest{Indices: []uint64{1, 2, 3, 4, 5, 6}, PageSize: 2, PageToken: "1"}
	res, err := bs.ListValidatorAssignments(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Construct the wanted assignments.
	var assignments []*ethpb.ValidatorAssignments_CommitteeAssignment

	activeIndices, err := helpers.ActiveValidatorIndices(s, 0)
	if err != nil {
		t.Fatal(err)
	}
	committeeAssignments, proposerIndexToSlots, err := helpers.CommitteeAssignments(s, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range activeIndices[3:5] {
		val, err := s.ValidatorAtIndex(index)
		if err != nil {
			t.Fatal(err)
		}
		assignments = append(assignments, &ethpb.ValidatorAssignments_CommitteeAssignment{
			BeaconCommittees: committeeAssignments[index].Committee,
			CommitteeIndex:   committeeAssignments[index].CommitteeIndex,
			AttesterSlot:     committeeAssignments[index].AttesterSlot,
			ProposerSlots:    proposerIndexToSlots[index],
			PublicKey:        val.PublicKey,
			ValidatorIndex:   index,
		})
	}

	wantedRes := &ethpb.ValidatorAssignments{
		Assignments:   assignments,
		TotalSize:     int32(len(req.Indices)),
		NextPageToken: "2",
	}

	if !reflect.DeepEqual(res, wantedRes) {
		t.Error("Did not get wanted assignments")
	}

	// Test the wrap around scenario.
	assignments = nil
	req = &ethpb.ListValidatorAssignmentsRequest{Indices: []uint64{1, 2, 3, 4, 5, 6}, PageSize: 5, PageToken: "1"}
	res, err = bs.ListValidatorAssignments(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	cAssignments, proposerIndexToSlots, err := helpers.CommitteeAssignments(s, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range activeIndices[6:7] {
		val, err := s.ValidatorAtIndex(index)
		if err != nil {
			t.Fatal(err)
		}
		assignments = append(assignments, &ethpb.ValidatorAssignments_CommitteeAssignment{
			BeaconCommittees: cAssignments[index].Committee,
			CommitteeIndex:   cAssignments[index].CommitteeIndex,
			AttesterSlot:     cAssignments[index].AttesterSlot,
			ProposerSlots:    proposerIndexToSlots[index],
			PublicKey:        val.PublicKey,
			ValidatorIndex:   index,
		})
	}

	wantedRes = &ethpb.ValidatorAssignments{
		Assignments:   assignments,
		TotalSize:     int32(len(req.Indices)),
		NextPageToken: "",
	}

	if !reflect.DeepEqual(res, wantedRes) {
		t.Error("Did not receive wanted assignments")
	}
}
