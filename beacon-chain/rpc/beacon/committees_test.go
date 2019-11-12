package beacon

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestServer_ListBeaconCommittees_Pagination_OutOfRange(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	numValidators := 1
	setupValidators(t, db, numValidators)

	headState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
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
