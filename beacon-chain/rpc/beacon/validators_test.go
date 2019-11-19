package beacon

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	// Use minimal config to reduce test setup time.
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
}

func TestServer_ListValidatorBalances_CannotRequestFutureEpoch(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: &pbp2p.BeaconState{Slot: 0},
		},
	}

	wanted := "Cannot retrieve information about an epoch in the future"
	if _, err := bs.ListValidatorBalances(
		ctx,
		&ethpb.ListValidatorBalancesRequest{
			QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{
				Epoch: 1,
			},
		},
	); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_ListValidatorBalances_NoResults(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: &pbp2p.BeaconState{Slot: 0},
		},
	}
	wanted := &ethpb.ValidatorBalances{
		Balances:      make([]*ethpb.ValidatorBalances_Balance, 0),
		TotalSize:     int32(0),
		NextPageToken: strconv.Itoa(0),
	}
	res, err := bs.ListValidatorBalances(
		ctx,
		&ethpb.ListValidatorBalancesRequest{
			QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{
				Epoch: 0,
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

func TestServer_ListValidatorBalances_PaginationOutOfRange(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	setupValidators(t, db, 3)

	headState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}

	req := &ethpb.ListValidatorBalancesRequest{PageToken: strconv.Itoa(1), PageSize: 100}
	wanted := fmt.Sprintf("page start %d >= list %d", req.PageSize, len(headState.Balances))
	if _, err := bs.ListValidatorBalances(context.Background(), req); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_ListValidatorBalances_ExceedsMaxPageSize(t *testing.T) {
	bs := &Server{}
	exceedsMax := int32(params.BeaconConfig().MaxPageSize + 1)

	wanted := fmt.Sprintf(
		"Requested page size %d can not be greater than max size %d",
		exceedsMax,
		params.BeaconConfig().MaxPageSize,
	)
	req := &ethpb.ListValidatorBalancesRequest{PageToken: strconv.Itoa(0), PageSize: exceedsMax}
	if _, err := bs.ListValidatorBalances(context.Background(), req); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_ListValidatorBalances_Pagination_Default(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	setupValidators(t, db, 100)

	headState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mock.ChainService{State: headState},
	}

	tests := []struct {
		req *ethpb.ListValidatorBalancesRequest
		res *ethpb.ValidatorBalances
	}{
		{req: &ethpb.ListValidatorBalancesRequest{PublicKeys: [][]byte{{99}}},
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{Index: 99, PublicKey: []byte{99}, Balance: 99},
				},
				NextPageToken: strconv.Itoa(1),
				TotalSize:     1,
			},
		},
		{req: &ethpb.ListValidatorBalancesRequest{Indices: []uint64{1, 2, 3}},
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{Index: 1, PublicKey: []byte{1}, Balance: 1},
					{Index: 2, PublicKey: []byte{2}, Balance: 2},
					{Index: 3, PublicKey: []byte{3}, Balance: 3},
				},
				NextPageToken: strconv.Itoa(1),
				TotalSize:     3,
			},
		},
		{req: &ethpb.ListValidatorBalancesRequest{PublicKeys: [][]byte{{10}, {11}, {12}}},
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{Index: 10, PublicKey: []byte{10}, Balance: 10},
					{Index: 11, PublicKey: []byte{11}, Balance: 11},
					{Index: 12, PublicKey: []byte{12}, Balance: 12},
				},
				NextPageToken: strconv.Itoa(1),
				TotalSize:     3,
			}},
		{req: &ethpb.ListValidatorBalancesRequest{PublicKeys: [][]byte{{2}, {3}}, Indices: []uint64{3, 4}}, // Duplication
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{Index: 2, PublicKey: []byte{2}, Balance: 2},
					{Index: 3, PublicKey: []byte{3}, Balance: 3},
					{Index: 4, PublicKey: []byte{4}, Balance: 4},
				},
				NextPageToken: strconv.Itoa(1),
				TotalSize:     3,
			}},
		{req: &ethpb.ListValidatorBalancesRequest{PublicKeys: [][]byte{{}}, Indices: []uint64{3, 4}}, // Public key has a blank value
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{Index: 3, PublicKey: []byte{3}, Balance: 3},
					{Index: 4, PublicKey: []byte{4}, Balance: 4},
				},
				NextPageToken: strconv.Itoa(1),
				TotalSize:     2,
			}},
	}
	for _, test := range tests {
		res, err := bs.ListValidatorBalances(context.Background(), test.req)
		if err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(res, test.res) {
			t.Errorf("Expected %v, received %v", test.res, res)
		}
	}
}

func TestServer_ListValidatorBalances_Pagination_CustomPageSizes(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	count := 1000
	setupValidators(t, db, count)

	headState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}

	tests := []struct {
		req *ethpb.ListValidatorBalancesRequest
		res *ethpb.ValidatorBalances
	}{
		{req: &ethpb.ListValidatorBalancesRequest{PageToken: strconv.Itoa(1), PageSize: 3},
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{PublicKey: []byte{3}, Index: 3, Balance: uint64(3)},
					{PublicKey: []byte{4}, Index: 4, Balance: uint64(4)},
					{PublicKey: []byte{5}, Index: 5, Balance: uint64(5)}},
				NextPageToken: strconv.Itoa(2),
				TotalSize:     int32(count)}},
		{req: &ethpb.ListValidatorBalancesRequest{PageToken: strconv.Itoa(10), PageSize: 5},
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{PublicKey: []byte{50}, Index: 50, Balance: uint64(50)},
					{PublicKey: []byte{51}, Index: 51, Balance: uint64(51)},
					{PublicKey: []byte{52}, Index: 52, Balance: uint64(52)},
					{PublicKey: []byte{53}, Index: 53, Balance: uint64(53)},
					{PublicKey: []byte{54}, Index: 54, Balance: uint64(54)}},
				NextPageToken: strconv.Itoa(11),
				TotalSize:     int32(count)}},
		{req: &ethpb.ListValidatorBalancesRequest{PageToken: strconv.Itoa(33), PageSize: 3},
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{PublicKey: []byte{99}, Index: 99, Balance: uint64(99)},
					{PublicKey: []byte{100}, Index: 100, Balance: uint64(100)},
					{PublicKey: []byte{101}, Index: 101, Balance: uint64(101)},
				},
				NextPageToken: strconv.Itoa(34),
				TotalSize:     int32(count)}},
		{req: &ethpb.ListValidatorBalancesRequest{PageSize: 2},
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{PublicKey: []byte{0}, Index: 0, Balance: uint64(0)},
					{PublicKey: []byte{1}, Index: 1, Balance: uint64(1)}},
				NextPageToken: strconv.Itoa(1),
				TotalSize:     int32(count)}},
	}
	for _, test := range tests {
		res, err := bs.ListValidatorBalances(context.Background(), test.req)
		if err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(res, test.res) {
			t.Errorf("Expected %v, received %v", test.res, res)
		}
	}
}

func TestServer_ListValidatorBalances_OutOfRange(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	setupValidators(t, db, 1)

	headState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mock.ChainService{State: headState},
	}

	req := &ethpb.ListValidatorBalancesRequest{Indices: []uint64{uint64(1)}}
	wanted := "does not exist"
	if _, err := bs.ListValidatorBalances(context.Background(), req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_ListValidatorBalances_FromArchive(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()
	epoch := uint64(0)
	validators, balances := setupValidators(t, db, 100)

	if err := db.SaveArchivedBalances(ctx, epoch, balances); err != nil {
		t.Fatal(err)
	}

	newerBalances := make([]uint64, len(balances))
	for i := 0; i < len(newerBalances); i++ {
		newerBalances[i] = balances[i] * 2
	}
	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: &pbp2p.BeaconState{
				Slot:       params.BeaconConfig().SlotsPerEpoch * 3,
				Validators: validators,
				Balances:   newerBalances,
			},
		},
	}

	req := &ethpb.ListValidatorBalancesRequest{
		QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{Epoch: 0},
		Indices:     []uint64{uint64(1)},
	}
	res, err := bs.ListValidatorBalances(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// We should expect a response containing the old balance from epoch 0,
	// not the new balance from the current state.
	want := []*ethpb.ValidatorBalances_Balance{
		{
			PublicKey: validators[1].PublicKey,
			Index:     1,
			Balance:   balances[1],
		},
	}
	if !reflect.DeepEqual(want, res.Balances) {
		t.Errorf("Wanted %v, received %v", want, res.Balances)
	}
}

func TestServer_ListValidatorBalances_FromArchive_NewValidatorNotFound(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()
	epoch := uint64(0)
	_, balances := setupValidators(t, db, 100)

	if err := db.SaveArchivedBalances(ctx, epoch, balances); err != nil {
		t.Fatal(err)
	}

	newValidators, newBalances := setupValidators(t, db, 200)
	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: &pbp2p.BeaconState{
				Slot:       params.BeaconConfig().SlotsPerEpoch * 3,
				Validators: newValidators,
				Balances:   newBalances,
			},
		},
	}

	req := &ethpb.ListValidatorBalancesRequest{
		QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{Epoch: 0},
		Indices:     []uint64{1, 150, 161},
	}
	if _, err := bs.ListValidatorBalances(context.Background(), req); !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Wanted out of range error for including newer validators in the arguments, received %v", err)
	}
}

func TestServer_ListValidators_CannotRequestFutureEpoch(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: &pbp2p.BeaconState{Slot: 0},
		},
	}

	wanted := "Cannot retrieve information about an epoch in the future"
	if _, err := bs.ListValidators(
		ctx,
		&ethpb.ListValidatorsRequest{
			QueryFilter: &ethpb.ListValidatorsRequest_Epoch{
				Epoch: 1,
			},
		},
	); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_ListValidators_NoResults(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: &pbp2p.BeaconState{Slot: 0},
		},
	}
	wanted := &ethpb.Validators{
		Validators:    make([]*ethpb.Validator, 0),
		TotalSize:     int32(0),
		NextPageToken: strconv.Itoa(0),
	}
	res, err := bs.ListValidators(
		ctx,
		&ethpb.ListValidatorsRequest{
			QueryFilter: &ethpb.ListValidatorsRequest_Epoch{
				Epoch: 0,
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

func TestServer_ListValidators_OnlyActiveValidators(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	count := 100
	balances := make([]uint64, count)
	validators := make([]*ethpb.Validator, count)
	activeValidators := make([]*ethpb.Validator, 0)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex(ctx, [48]byte{byte(i)}, uint64(i)); err != nil {
			t.Fatal(err)
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance

		// We mark even validators as active, and odd validators as inactive.
		if i%2 == 0 {
			val := &ethpb.Validator{
				PublicKey:       []byte{byte(i)},
				ActivationEpoch: 0,
				ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			}
			validators[i] = val
			activeValidators = append(activeValidators, val)
		} else {
			validators[i] = &ethpb.Validator{
				PublicKey:       []byte{byte(i)},
				ActivationEpoch: 0,
				ExitEpoch:       0,
			}
		}
	}
	headState := &pbp2p.BeaconState{Validators: validators, Balances: balances}

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}

	received, err := bs.ListValidators(context.Background(), &ethpb.ListValidatorsRequest{
		Active: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(activeValidators, received.Validators) {
		t.Errorf("Wanted %v, received %v", activeValidators, received.Validators)
	}
}

func TestServer_ListValidators_NoPagination(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	validators, _ := setupValidators(t, db, 100)
	headState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
	}

	received, err := bs.ListValidators(context.Background(), &ethpb.ListValidatorsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(validators, received.Validators) {
		t.Fatal("Incorrect respond of validators")
	}
}

func TestServer_ListValidators_Pagination(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	count := 100
	setupValidators(t, db, count)

	headState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
	}

	tests := []struct {
		req *ethpb.ListValidatorsRequest
		res *ethpb.Validators
	}{
		{req: &ethpb.ListValidatorsRequest{PageToken: strconv.Itoa(1), PageSize: 3},
			res: &ethpb.Validators{
				Validators: []*ethpb.Validator{
					{PublicKey: []byte{3}},
					{PublicKey: []byte{4}},
					{PublicKey: []byte{5}}},
				NextPageToken: strconv.Itoa(2),
				TotalSize:     int32(count)}},
		{req: &ethpb.ListValidatorsRequest{PageToken: strconv.Itoa(10), PageSize: 5},
			res: &ethpb.Validators{
				Validators: []*ethpb.Validator{
					{PublicKey: []byte{50}},
					{PublicKey: []byte{51}},
					{PublicKey: []byte{52}},
					{PublicKey: []byte{53}},
					{PublicKey: []byte{54}}},
				NextPageToken: strconv.Itoa(11),
				TotalSize:     int32(count)}},
		{req: &ethpb.ListValidatorsRequest{PageToken: strconv.Itoa(33), PageSize: 3},
			res: &ethpb.Validators{
				Validators: []*ethpb.Validator{
					{PublicKey: []byte{99}}},
				NextPageToken: strconv.Itoa(34),
				TotalSize:     int32(count)}},
		{req: &ethpb.ListValidatorsRequest{PageSize: 2},
			res: &ethpb.Validators{
				Validators: []*ethpb.Validator{
					{PublicKey: []byte{0}},
					{PublicKey: []byte{1}}},
				NextPageToken: strconv.Itoa(1),
				TotalSize:     int32(count)}},
	}
	for _, test := range tests {
		res, err := bs.ListValidators(context.Background(), test.req)
		if err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(res, test.res) {
			t.Error("Incorrect respond of validators")
		}
	}
}

func TestServer_ListValidators_PaginationOutOfRange(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	count := 1
	validators, _ := setupValidators(t, db, count)
	headState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
	}

	req := &ethpb.ListValidatorsRequest{PageToken: strconv.Itoa(1), PageSize: 100}
	wanted := fmt.Sprintf("page start %d >= list %d", req.PageSize, len(validators))
	if _, err := bs.ListValidators(context.Background(), req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_ListValidators_ExceedsMaxPageSize(t *testing.T) {
	bs := &Server{}
	exceedsMax := int32(params.BeaconConfig().MaxPageSize + 1)

	wanted := fmt.Sprintf("Requested page size %d can not be greater than max size %d", exceedsMax, params.BeaconConfig().MaxPageSize)
	req := &ethpb.ListValidatorsRequest{PageToken: strconv.Itoa(0), PageSize: exceedsMax}
	if _, err := bs.ListValidators(context.Background(), req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_ListValidators_DefaultPageSize(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	validators, _ := setupValidators(t, db, 1000)
	headState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
	}

	req := &ethpb.ListValidatorsRequest{}
	res, err := bs.ListValidators(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	j := params.BeaconConfig().DefaultPageSize
	if !reflect.DeepEqual(res.Validators, validators[i:j]) {
		t.Error("Incorrect respond of validators")
	}
}

func TestServer_ListValidators_FromOldEpoch(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	numEpochs := 30
	validators := make([]*ethpb.Validator, numEpochs)
	for i := 0; i < numEpochs; i++ {
		validators[i] = &ethpb.Validator{
			ActivationEpoch: uint64(i),
		}
	}

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: &pbp2p.BeaconState{
				Slot:       helpers.StartSlot(30),
				Validators: validators,
			},
		},
	}

	req := &ethpb.ListValidatorsRequest{
		QueryFilter: &ethpb.ListValidatorsRequest_Genesis{
			Genesis: true,
		},
	}
	res, err := bs.ListValidators(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Validators) != 1 {
		t.Errorf("Wanted 1 validator at genesis, received %d", len(res.Validators))
	}

	req = &ethpb.ListValidatorsRequest{
		QueryFilter: &ethpb.ListValidatorsRequest_Epoch{
			Epoch: 20,
		},
	}
	res, err = bs.ListValidators(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.Validators, validators[:21]) {
		t.Errorf("Incorrect number of validators, wanted %d received %d", len(validators[:21]), len(res.Validators))
	}
}

func TestServer_GetValidatorActiveSetChanges_CannotRequestFutureEpoch(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: &pbp2p.BeaconState{Slot: 0},
		},
	}

	wanted := "Cannot retrieve information about an epoch in the future"
	if _, err := bs.GetValidatorActiveSetChanges(
		ctx,
		&ethpb.GetValidatorActiveSetChangesRequest{
			QueryFilter: &ethpb.GetValidatorActiveSetChangesRequest_Epoch{
				Epoch: 1,
			},
		},
	); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_GetValidatorActiveSetChanges(t *testing.T) {
	ctx := context.Background()
	validators := make([]*ethpb.Validator, 6)
	headState := &pbp2p.BeaconState{
		Slot:       0,
		Validators: validators,
	}
	for i := 0; i < len(validators); i++ {
		activationEpoch := params.BeaconConfig().FarFutureEpoch
		withdrawableEpoch := params.BeaconConfig().FarFutureEpoch
		exitEpoch := params.BeaconConfig().FarFutureEpoch
		slashed := false
		// Mark indices divisible by two as activated.
		if i%2 == 0 {
			activationEpoch = helpers.DelayedActivationExitEpoch(0)
		} else if i%3 == 0 {
			// Mark indices divisible by 3 as slashed.
			withdrawableEpoch = params.BeaconConfig().EpochsPerSlashingsVector
			slashed = true
		} else if i%5 == 0 {
			// Mark indices divisible by 5 as exited.
			exitEpoch = 0
			withdrawableEpoch = params.BeaconConfig().MinValidatorWithdrawabilityDelay
		}
		headState.Validators[i] = &ethpb.Validator{
			ActivationEpoch:   activationEpoch,
			PublicKey:         []byte(strconv.Itoa(i)),
			WithdrawableEpoch: withdrawableEpoch,
			Slashed:           slashed,
			ExitEpoch:         exitEpoch,
		}
	}
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{Epoch: 0},
		},
	}
	res, err := bs.GetValidatorActiveSetChanges(ctx, &ethpb.GetValidatorActiveSetChangesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	wantedActive := [][]byte{
		[]byte("0"),
		[]byte("2"),
		[]byte("4"),
	}
	wantedSlashed := [][]byte{
		[]byte("3"),
	}
	wantedExited := [][]byte{
		[]byte("5"),
	}
	wanted := &ethpb.ActiveSetChanges{
		Epoch:               0,
		ActivatedPublicKeys: wantedActive,
		ExitedPublicKeys:    wantedExited,
		SlashedPublicKeys:   wantedSlashed,
	}
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
}

func TestServer_GetValidatorActiveSetChanges_FromArchive(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()
	validators := make([]*ethpb.Validator, 6)
	headState := &pbp2p.BeaconState{
		Slot:       helpers.StartSlot(100),
		Validators: validators,
	}
	activatedIndices := make([]uint64, 0)
	slashedIndices := make([]uint64, 0)
	exitedIndices := make([]uint64, 0)
	for i := 0; i < len(validators); i++ {
		// Mark indices divisible by two as activated.
		if i%2 == 0 {
			activatedIndices = append(activatedIndices, uint64(i))
		} else if i%3 == 0 {
			// Mark indices divisible by 3 as slashed.
			slashedIndices = append(slashedIndices, uint64(i))
		} else if i%5 == 0 {
			// Mark indices divisible by 5 as exited.
			exitedIndices = append(exitedIndices, uint64(i))
		}
		headState.Validators[i] = &ethpb.Validator{
			PublicKey: []byte(strconv.Itoa(i)),
		}
	}
	archivedChanges := &ethpb.ArchivedActiveSetChanges{
		Activated: activatedIndices,
		Exited:    exitedIndices,
		Slashed:   slashedIndices,
	}
	// We store the changes during the genesis epoch.
	if err := db.SaveArchivedActiveValidatorChanges(ctx, 0, archivedChanges); err != nil {
		t.Fatal(err)
	}
	// We store the same changes during epoch 5 for further testing.
	if err := db.SaveArchivedActiveValidatorChanges(ctx, 5, archivedChanges); err != nil {
		t.Fatal(err)
	}
	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}
	res, err := bs.GetValidatorActiveSetChanges(ctx, &ethpb.GetValidatorActiveSetChangesRequest{
		QueryFilter: &ethpb.GetValidatorActiveSetChangesRequest_Genesis{Genesis: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantedActive := [][]byte{
		[]byte("0"),
		[]byte("2"),
		[]byte("4"),
	}
	wantedSlashed := [][]byte{
		[]byte("3"),
	}
	wantedExited := [][]byte{
		[]byte("5"),
	}
	wanted := &ethpb.ActiveSetChanges{
		Epoch:               0,
		ActivatedPublicKeys: wantedActive,
		ExitedPublicKeys:    wantedExited,
		SlashedPublicKeys:   wantedSlashed,
	}
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
	res, err = bs.GetValidatorActiveSetChanges(ctx, &ethpb.GetValidatorActiveSetChangesRequest{
		QueryFilter: &ethpb.GetValidatorActiveSetChangesRequest_Epoch{Epoch: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	wanted.Epoch = 5
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
}

func TestServer_GetValidatorQueue_PendingActivation(t *testing.T) {
	headState := &pbp2p.BeaconState{
		Validators: []*ethpb.Validator{
			{
				ActivationEpoch:            helpers.DelayedActivationExitEpoch(0),
				ActivationEligibilityEpoch: 3,
				PublicKey:                  []byte("3"),
			},
			{
				ActivationEpoch:            helpers.DelayedActivationExitEpoch(0),
				ActivationEligibilityEpoch: 2,
				PublicKey:                  []byte("2"),
			},
			{
				ActivationEpoch:            helpers.DelayedActivationExitEpoch(0),
				ActivationEligibilityEpoch: 1,
				PublicKey:                  []byte("1"),
			},
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
		},
	}
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}
	res, err := bs.GetValidatorQueue(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	// We verify the keys are properly sorted by the validators' activation eligibility epoch.
	wanted := [][]byte{
		[]byte("1"),
		[]byte("2"),
		[]byte("3"),
	}
	activeValidatorCount, err := helpers.ActiveValidatorCount(headState, helpers.CurrentEpoch(headState))
	if err != nil {
		t.Fatal(err)
	}
	wantChurn, err := helpers.ValidatorChurnLimit(activeValidatorCount)
	if err != nil {
		t.Fatal(err)
	}
	if res.ChurnLimit != wantChurn {
		t.Errorf("Wanted churn %d, received %d", wantChurn, res.ChurnLimit)
	}
	if !reflect.DeepEqual(res.ActivationPublicKeys, wanted) {
		t.Errorf("Wanted %v, received %v", wanted, res.ActivationPublicKeys)
	}
}

func TestServer_GetValidatorQueue_PendingExit(t *testing.T) {
	headState := &pbp2p.BeaconState{
		Validators: []*ethpb.Validator{
			{
				ActivationEpoch:   0,
				ExitEpoch:         4,
				WithdrawableEpoch: 3,
				PublicKey:         []byte("3"),
			},
			{
				ActivationEpoch:   0,
				ExitEpoch:         4,
				WithdrawableEpoch: 2,
				PublicKey:         []byte("2"),
			},
			{
				ActivationEpoch:   0,
				ExitEpoch:         4,
				WithdrawableEpoch: 1,
				PublicKey:         []byte("1"),
			},
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
		},
	}
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}
	res, err := bs.GetValidatorQueue(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	// We verify the keys are properly sorted by the validators' withdrawable epoch.
	wanted := [][]byte{
		[]byte("1"),
		[]byte("2"),
		[]byte("3"),
	}
	activeValidatorCount, err := helpers.ActiveValidatorCount(headState, helpers.CurrentEpoch(headState))
	if err != nil {
		t.Fatal(err)
	}
	wantChurn, err := helpers.ValidatorChurnLimit(activeValidatorCount)
	if err != nil {
		t.Fatal(err)
	}
	if res.ChurnLimit != wantChurn {
		t.Errorf("Wanted churn %d, received %d", wantChurn, res.ChurnLimit)
	}
	if !reflect.DeepEqual(res.ExitPublicKeys, wanted) {
		t.Errorf("Wanted %v, received %v", wanted, res.ExitPublicKeys)
	}
}

func TestServer_GetValidatorParticipation_CannotRequestFutureEpoch(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: &pbp2p.BeaconState{Slot: 0},
		},
	}

	wanted := "Cannot retrieve information about an epoch in the future"
	if _, err := bs.GetValidatorParticipation(
		ctx,
		&ethpb.GetValidatorParticipationRequest{
			QueryFilter: &ethpb.GetValidatorParticipationRequest_Epoch{
				Epoch: 1,
			},
		},
	); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_GetValidatorParticipation_FromArchive(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()
	epoch := uint64(4)
	part := &ethpb.ValidatorParticipation{
		GlobalParticipationRate: 1.0,
		VotedEther:              20,
		EligibleEther:           20,
	}
	if err := db.SaveArchivedValidatorParticipation(ctx, epoch-2, part); err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: &pbp2p.BeaconState{Slot: helpers.StartSlot(epoch + 1)},
		},
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: epoch + 1,
			},
		},
	}
	if _, err := bs.GetValidatorParticipation(ctx, &ethpb.GetValidatorParticipationRequest{
		QueryFilter: &ethpb.GetValidatorParticipationRequest_Epoch{
			Epoch: epoch + 2,
		},
	}); err == nil {
		t.Error("Expected error when requesting future epoch, received nil")
	}
	// We request data from epoch 0, which we didn't archive, so we should expect an error.
	if _, err := bs.GetValidatorParticipation(ctx, &ethpb.GetValidatorParticipationRequest{
		QueryFilter: &ethpb.GetValidatorParticipationRequest_Genesis{
			Genesis: true,
		},
	}); err == nil {
		t.Error("Expected error when data from archive is not found, received nil")
	}

	want := &ethpb.ValidatorParticipationResponse{
		Epoch:         epoch - 2,
		Finalized:     true,
		Participation: part,
	}
	res, err := bs.GetValidatorParticipation(ctx, &ethpb.GetValidatorParticipationRequest{
		QueryFilter: &ethpb.GetValidatorParticipationRequest_Epoch{
			Epoch: epoch - 2,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(want, res) {
		t.Errorf("Wanted %v, received %v", want, res)
	}
}

func TestServer_GetValidatorParticipation_CurrentEpoch(t *testing.T) {
	helpers.ClearAllCaches()
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	epoch := uint64(1)
	attestedBalance := uint64(1)
	validatorCount := uint64(100)

	validators := make([]*ethpb.Validator, validatorCount)
	balances := make([]uint64, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	atts := []*pbp2p.PendingAttestation{{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{}}}}

	s := &pbp2p.BeaconState{
		Slot:                       epoch*params.BeaconConfig().SlotsPerEpoch + 1,
		Validators:                 validators,
		Balances:                   balances,
		BlockRoots:                 make([][]byte, 128),
		Slashings:                  []uint64{0, 1e9, 1e9},
		RandaoMixes:                make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CurrentEpochAttestations:   atts,
		FinalizedCheckpoint:        &ethpb.Checkpoint{},
		JustificationBits:          bitfield.Bitvector4{0x00},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{},
	}

	bs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mock.ChainService{State: s},
	}

	res, err := bs.GetValidatorParticipation(ctx, &ethpb.GetValidatorParticipationRequest{
		QueryFilter: &ethpb.GetValidatorParticipationRequest_Epoch{
			Epoch: epoch,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	wanted := &ethpb.ValidatorParticipation{
		VotedEther:              attestedBalance,
		EligibleEther:           validatorCount * params.BeaconConfig().MaxEffectiveBalance,
		GlobalParticipationRate: float32(attestedBalance) / float32(validatorCount*params.BeaconConfig().MaxEffectiveBalance),
	}

	if !reflect.DeepEqual(res.Participation, wanted) {
		t.Error("Incorrect validator participation respond")
	}
}

func setupValidators(t *testing.T, db db.Database, count int) ([]*ethpb.Validator, []uint64) {
	ctx := context.Background()
	balances := make([]uint64, count)
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex(ctx, [48]byte{byte(i)}, uint64(i)); err != nil {
			t.Fatal(err)
		}
		balances[i] = uint64(i)
		validators = append(validators, &ethpb.Validator{
			PublicKey: []byte{byte(i)},
		})
	}
	blk := &ethpb.BeaconBlock{
		Slot: 0,
	}
	blockRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, blockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(
		context.Background(),
		&pbp2p.BeaconState{Validators: validators, Balances: balances},
		blockRoot,
	); err != nil {
		t.Fatal(err)
	}
	return validators, balances
}
