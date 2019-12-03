package beacon

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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
		"Requested page size %d can not be greater than max size %d",
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

	numValidators := 128
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
				PageSize: 2,
			},
			res: &ethpb.BeaconCommittees{
				Epoch:                0,
				Committees:           wanted[0:2],
				ActiveValidatorCount: uint64(numValidators),
				NextPageToken:        strconv.Itoa(1),
				TotalSize:            int32(len(wanted)),
			},
		},
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
		{
			req: &ethpb.ListCommitteesRequest{
				PageToken: strconv.Itoa(3), PageSize: 5,
			},
			res: &ethpb.BeaconCommittees{
				Epoch:                0,
				Committees:           wanted[15:20],
				ActiveValidatorCount: uint64(numValidators),
				NextPageToken:        strconv.Itoa(4),
				TotalSize:            int32(len(wanted)),
			},
		},
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

func TestServer_ListBeaconCommittees_FromArchive(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()

	numValidators := 128
	headState := setupActiveValidators(t, db, numValidators)

	headState.RandaoMixes = make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(headState.RandaoMixes); i++ {
		headState.RandaoMixes[i] = make([]byte, 32)
	}

	headState.Slot = params.BeaconConfig().SlotsPerEpoch * 10

	// Store the genesis seed.
	seed, err := helpers.Seed(headState, 0, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveArchivedCommitteeInfo(ctx, 0, &pbp2p.ArchivedCommitteeInfo{
		AttesterSeed: seed[:],
	}); err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}

	activeIndices, err := helpers.ActiveValidatorIndices(headState, 0)
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
			committee, err := helpers.ComputeCommittee(activeIndices, seed, epochOffset, totalCount)
			if err != nil {
				t.Fatal(err)
			}
			wanted = append(wanted, &ethpb.BeaconCommittees_CommitteeItem{
				Committee: committee,
				Slot:      slot,
			})
		}
	}
	res1, err := bs.ListBeaconCommittees(context.Background(), &ethpb.ListCommitteesRequest{
		QueryFilter: &ethpb.ListCommitteesRequest_Genesis{
			Genesis: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	res2, err := bs.ListBeaconCommittees(context.Background(), &ethpb.ListCommitteesRequest{
		QueryFilter: &ethpb.ListCommitteesRequest_Epoch{
			Epoch: 0,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res1, res2) {
		t.Fatal(err)
	}
	wantedRes := &ethpb.BeaconCommittees{
		Epoch:                0,
		Committees:           wanted,
		ActiveValidatorCount: uint64(numValidators),
		NextPageToken:        strconv.Itoa(1),
		TotalSize:            int32(len(wanted)),
	}
	if !reflect.DeepEqual(wantedRes, res1) {
		t.Errorf("Wanted %v, received %v", wantedRes, res1)
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
