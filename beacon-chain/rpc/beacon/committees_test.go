package beacon

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestServer_ListBeaconCommittees_Pagination_OutOfRange(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	numValidators := 1
	headState := setupActiveValidators(t, db, numValidators)
	headState.RandaoMixes = make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(headState.RandaoMixes); i++ {
		headState.RandaoMixes[i] = make([]byte, 32)
	}

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}

	numCommittees := uint64(0)
	for slot := uint64(0); slot < params.BeaconConfig().SlotsPerEpoch; slot++ {
		var countAtSlot = uint64(numValidators) / params.BeaconConfig().SlotsPerEpoch / params.BeaconConfig().TargetCommitteeSize
		if countAtSlot > params.BeaconConfig().MaxCommitteesPerSlot {
			countAtSlot = params.BeaconConfig().MaxCommitteesPerSlot
		}
		if countAtSlot == 0 {
			countAtSlot = 1
		}
		numCommittees += countAtSlot
	}

	req := &ethpb.ListCommitteesRequest{PageToken: strconv.Itoa(1), PageSize: 100}
	wanted := fmt.Sprintf("page start %d >= list %d", req.PageSize, numCommittees)
	if _, err := bs.ListBeaconCommittees(context.Background(), req); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_ListBeaconCommittees_Pagination_ExceedsMaxPageSize(t *testing.T) {
	bs := &Server{}
	exceedsMax := int32(params.BeaconConfig().MaxPageSize + 1)

	wanted := fmt.Sprintf(
		"requested page size %d can not be greater than max size %d",
		exceedsMax,
		params.BeaconConfig().MaxPageSize,
	)
	req := &ethpb.ListCommitteesRequest{PageToken: strconv.Itoa(0), PageSize: exceedsMax}
	if _, err := bs.ListBeaconCommittees(context.Background(), req); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_ListBeaconCommittees_Pagination_CustomPageSize(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	numValidators := 64
	headState := setupActiveValidators(t, db, numValidators)

	headState.RandaoMixes = make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(headState.RandaoMixes); i++ {
		headState.RandaoMixes[i] = make([]byte, 32)
	}

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}

	activeIndices, err := helpers.ActiveValidatorIndices(headState, 0)
	if err != nil {
		t.Fatal(err)
	}
	attesterSeed, err := helpers.Seed(headState, 0, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		t.Fatal(err)
	}
	wanted := make([]*ethpb.BeaconCommittees_CommitteeItem, 0)
	for slot := uint64(0); slot < params.BeaconConfig().SlotsPerEpoch; slot++ {
		var countAtSlot = uint64(numValidators) / params.BeaconConfig().SlotsPerEpoch / params.BeaconConfig().TargetCommitteeSize
		if countAtSlot > params.BeaconConfig().MaxCommitteesPerSlot {
			countAtSlot = params.BeaconConfig().MaxCommitteesPerSlot
		}
		if countAtSlot == 0 {
			countAtSlot = 1
		}
		for i := uint64(0); i < countAtSlot; i++ {
			epochOffset := i + (slot%params.BeaconConfig().SlotsPerEpoch)*countAtSlot
			totalCount := countAtSlot * params.BeaconConfig().SlotsPerEpoch
			committee, err := helpers.ComputeCommittee(activeIndices, attesterSeed, epochOffset, totalCount)
			if err != nil {
				t.Fatal(err)
			}
			wanted = append(wanted, &ethpb.BeaconCommittees_CommitteeItem{
				Committee: committee,
				Slot:      slot,
			})
		}
	}

	tests := []struct {
		req *ethpb.ListCommitteesRequest
		res *ethpb.BeaconCommittees
	}{
		{
			req: &ethpb.ListCommitteesRequest{
				PageToken: strconv.Itoa(1), PageSize: 3,
			},
			res: &ethpb.BeaconCommittees{
				Epoch:                0,
				Committees:           wanted[3:6],
				ActiveValidatorCount: uint64(numValidators),
				NextPageToken:        strconv.Itoa(2),
				TotalSize:            int32(len(wanted)),
			},
		},
		//{req: &ethpb.GetValidatorBalancesRequest{PageToken: strconv.Itoa(1), PageSize: 3},
		//	res: &ethpb.ValidatorBalances{
		//		Balances: []*ethpb.ValidatorBalances_Balance{
		//			{PublicKey: []byte{3}, Index: 3, Balance: uint64(3)},
		//			{PublicKey: []byte{4}, Index: 4, Balance: uint64(4)},
		//			{PublicKey: []byte{5}, Index: 5, Balance: uint64(5)}},
		//		NextPageToken: strconv.Itoa(2),
		//		TotalSize:     int32(count)}},
		//{req: &ethpb.GetValidatorBalancesRequest{PageToken: strconv.Itoa(10), PageSize: 5},
		//	res: &ethpb.ValidatorBalances{
		//		Balances: []*ethpb.ValidatorBalances_Balance{
		//			{PublicKey: []byte{50}, Index: 50, Balance: uint64(50)},
		//			{PublicKey: []byte{51}, Index: 51, Balance: uint64(51)},
		//			{PublicKey: []byte{52}, Index: 52, Balance: uint64(52)},
		//			{PublicKey: []byte{53}, Index: 53, Balance: uint64(53)},
		//			{PublicKey: []byte{54}, Index: 54, Balance: uint64(54)}},
		//		NextPageToken: strconv.Itoa(11),
		//		TotalSize:     int32(count)}},
		//{req: &ethpb.GetValidatorBalancesRequest{PageToken: strconv.Itoa(33), PageSize: 3},
		//	res: &ethpb.ValidatorBalances{
		//		Balances: []*ethpb.ValidatorBalances_Balance{
		//			{PublicKey: []byte{99}, Index: 99, Balance: uint64(99)},
		//			{PublicKey: []byte{100}, Index: 100, Balance: uint64(100)},
		//			{PublicKey: []byte{101}, Index: 101, Balance: uint64(101)},
		//		},
		//		NextPageToken: strconv.Itoa(34),
		//		TotalSize:     int32(count)}},
		//{req: &ethpb.GetValidatorBalancesRequest{PageSize: 2},
		//	res: &ethpb.ValidatorBalances{
		//		Balances: []*ethpb.ValidatorBalances_Balance{
		//			{PublicKey: []byte{0}, Index: 0, Balance: uint64(0)},
		//			{PublicKey: []byte{1}, Index: 1, Balance: uint64(1)}},
		//		NextPageToken: strconv.Itoa(1),
		//		TotalSize:     int32(count)}},
	}
	for _, test := range tests {
		res, err := bs.ListBeaconCommittees(context.Background(), test.req)
		if err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(res, test.res) {
			t.Errorf("Expected %v, received %v", test.res, res)
		}
	}
}

func setupActiveValidators(t *testing.T, db db.Database, count int) *pbp2p.BeaconState {
	ctx := context.Background()
	balances := make([]uint64, count)
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex(ctx, [48]byte{byte(i)}, uint64(i)); err != nil {
			t.Fatal(err)
		}
		balances[i] = uint64(i)
		validators = append(validators, &ethpb.Validator{
			PublicKey:       []byte{byte(i)},
			ActivationEpoch: 0,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
		})
	}
	return &pbp2p.BeaconState{Validators: validators, Balances: balances}
}
