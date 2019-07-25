package rpc

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestBeaconChainServer_ListValidatorBalances(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 100
	balances := make([]uint64, count)
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		balances[i] = uint64(i)
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}})
	}

	if err := db.SaveState(
		context.Background(),
		&pbp2p.BeaconState{Validators: validators, Balances: balances}); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	tests := []struct {
		req *ethpb.GetValidatorBalancesRequest
		res *ethpb.ValidatorBalances
	}{
		{req: &ethpb.GetValidatorBalancesRequest{PublicKeys: [][]byte{{99}}},
			res: &ethpb.ValidatorBalances{Balances: []*ethpb.ValidatorBalances_Balance{{
				Index: 99, PublicKey: []byte{99}, Balance: 99}},
			}},
		{req: &ethpb.GetValidatorBalancesRequest{Indices: []uint64{1, 2, 3}},
			res: &ethpb.ValidatorBalances{Balances: []*ethpb.ValidatorBalances_Balance{
				{Index: 1, PublicKey: []byte{1}, Balance: 1},
				{Index: 2, PublicKey: []byte{2}, Balance: 2},
				{Index: 3, PublicKey: []byte{3}, Balance: 3}},
			}},
		{req: &ethpb.GetValidatorBalancesRequest{PublicKeys: [][]byte{{10}, {11}, {12}}},
			res: &ethpb.ValidatorBalances{Balances: []*ethpb.ValidatorBalances_Balance{
				{Index: 10, PublicKey: []byte{10}, Balance: 10},
				{Index: 11, PublicKey: []byte{11}, Balance: 11},
				{Index: 12, PublicKey: []byte{12}, Balance: 12}},
			}},
		{req: &ethpb.GetValidatorBalancesRequest{PublicKeys: [][]byte{{2}, {3}}, Indices: []uint64{3, 4}}, // Duplication
			res: &ethpb.ValidatorBalances{Balances: []*ethpb.ValidatorBalances_Balance{
				{Index: 2, PublicKey: []byte{2}, Balance: 2},
				{Index: 3, PublicKey: []byte{3}, Balance: 3},
				{Index: 4, PublicKey: []byte{4}, Balance: 4}},
			}},
	}

	for _, test := range tests {
		res, err := bs.ListValidatorBalances(context.Background(), test.req)
		if err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(res, test.res) {
			t.Error("Incorrect respond of validator balances")
		}
	}
}

func TestBeaconChainServer_ListValidatorBalancesOutOfRange(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 1
	balances := make([]uint64, count)
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		balances[i] = uint64(i)
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}})
	}

	if err := db.SaveState(
		context.Background(),
		&pbp2p.BeaconState{Validators: validators, Balances: balances}); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	req := &ethpb.GetValidatorBalancesRequest{Indices: []uint64{uint64(count)}}
	wanted := fmt.Sprintf("validator index %d >= balance list %d", count, len(balances))
	if _, err := bs.ListValidatorBalances(context.Background(), req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestBeaconChainServer_GetValidatorsNoPagination(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 100
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}})
	}

	if err := db.SaveState(
		context.Background(),
		&pbp2p.BeaconState{Validators: validators}); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	received, err := bs.GetValidators(context.Background(), &ethpb.GetValidatorsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(validators, received.Validators) {
		fmt.Println(received.Validators)
		t.Fatal("Incorrect respond of validators")
	}
}

func TestBeaconChainServer_GetValidatorsPagination(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 100
	balances := make([]uint64, count)
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		balances[i] = uint64(i)
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}})
	}

	if err := db.SaveState(
		context.Background(),
		&pbp2p.BeaconState{Validators: validators, Balances: balances}); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	tests := []struct {
		req *ethpb.GetValidatorsRequest
		res *ethpb.Validators
	}{
		{req: &ethpb.GetValidatorsRequest{PageToken: strconv.Itoa(1), PageSize: 3},
			res: &ethpb.Validators{
				Validators: []*ethpb.Validator{
					{PublicKey: []byte{3}},
					{PublicKey: []byte{4}},
					{PublicKey: []byte{5}}},
				NextPageToken: strconv.Itoa(2),
				TotalSize:     int32(count)}},
		{req: &ethpb.GetValidatorsRequest{PageToken: strconv.Itoa(10), PageSize: 5},
			res: &ethpb.Validators{
				Validators: []*ethpb.Validator{
					{PublicKey: []byte{50}},
					{PublicKey: []byte{51}},
					{PublicKey: []byte{52}},
					{PublicKey: []byte{53}},
					{PublicKey: []byte{54}}},
				NextPageToken: strconv.Itoa(11),
				TotalSize:     int32(count)}},
		{req: &ethpb.GetValidatorsRequest{PageToken: strconv.Itoa(33), PageSize: 3},
			res: &ethpb.Validators{
				Validators: []*ethpb.Validator{
					{PublicKey: []byte{99}}},
				NextPageToken: strconv.Itoa(34),
				TotalSize:     int32(count)}},
		{req: &ethpb.GetValidatorsRequest{PageSize: 2},
			res: &ethpb.Validators{
				Validators: []*ethpb.Validator{
					{PublicKey: []byte{0}},
					{PublicKey: []byte{1}}},
				NextPageToken: strconv.Itoa(1),
				TotalSize:     int32(count)}},
	}
	for _, test := range tests {
		res, err := bs.GetValidators(context.Background(), test.req)
		if err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(res, test.res) {
			t.Error("Incorrect respond of validators")
		}
	}
}

func TestBeaconChainServer_GetValidatorsPaginationOutOfRange(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 1
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}})
	}

	if err := db.SaveState(
		context.Background(),
		&pbp2p.BeaconState{Validators: validators}); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	req := &ethpb.GetValidatorsRequest{PageToken: strconv.Itoa(1), PageSize: 100}
	wanted := fmt.Sprintf("page start %d >= validator list %d", req.PageSize, len(validators))
	if _, err := bs.GetValidators(context.Background(), req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestBeaconChainServer_GetValidatorsExceedsMaxPageSize(t *testing.T) {
	bs := &BeaconChainServer{}
	exceedsMax := int32(params.BeaconConfig().MaxPageSize + 1)

	wanted := fmt.Sprintf("requested page size %d can not be greater than max size %d", exceedsMax, params.BeaconConfig().MaxPageSize)
	req := &ethpb.GetValidatorsRequest{PageToken: strconv.Itoa(0), PageSize: exceedsMax}
	if _, err := bs.GetValidators(context.Background(), req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestBeaconChainServer_GetValidatorsDefaultPageSize(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 1000
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}})
	}

	if err := db.SaveState(
		context.Background(),
		&pbp2p.BeaconState{Validators: validators}); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	req := &ethpb.GetValidatorsRequest{}
	res, err := bs.GetValidators(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	j := params.BeaconConfig().DefaultPageSize
	if !reflect.DeepEqual(res.Validators, validators[i:j]) {
		t.Error("Incorrect respond of validators")
	}
}

func TestBeaconChainServer_ListAssignmentsInputOutOfRange(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 1
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}})
	}

	if err := db.SaveState(
		context.Background(),
		&pbp2p.BeaconState{Validators: validators}); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{beaconDB: db}

	wanted := fmt.Sprintf("page start %d >= validator list %d", 0, 0)
	if _, err := bs.ListValidatorAssignments(context.Background(), &ethpb.ListValidatorAssignmentsRequest{Epoch: 0}); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestBeaconChainServer_ListAssignmentsExceedsMaxPageSize(t *testing.T) {
	bs := &BeaconChainServer{}
	exceedsMax := int32(params.BeaconConfig().MaxPageSize + 1)

	wanted := fmt.Sprintf("requested page size %d can not be greater than max size %d", exceedsMax, params.BeaconConfig().MaxPageSize)
	req := &ethpb.ListValidatorAssignmentsRequest{PageToken: strconv.Itoa(0), PageSize: exceedsMax}
	if _, err := bs.ListValidatorAssignments(context.Background(), req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestBeaconChainServer_ListAssignmentsDefaultPageSize(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 1000
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		// Mark the validators with index divisible by 3 inactive.
		if i%3 == 0 {
			validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}, ExitEpoch: 0})
		} else {
			validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}, ExitEpoch: params.BeaconConfig().FarFutureEpoch})
		}
	}

	s := &pbp2p.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)}
	if err := db.SaveState(context.Background(), s); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	res, err := bs.ListValidatorAssignments(context.Background(), &ethpb.ListValidatorAssignmentsRequest{Epoch: 0})
	if err != nil {
		t.Fatal(err)
	}

	// Construct the wanted assignments
	var wanted []*ethpb.ValidatorAssignments_CommitteeAssignment

	activeIndices, err := helpers.ActiveValidatorIndices(s, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range activeIndices[0:params.BeaconConfig().DefaultPageSize] {
		committee, shard, slot, isProposer, err := helpers.CommitteeAssignment(s, 0, index)
		if err != nil {
			t.Fatal(err)
		}
		wanted = append(wanted, &ethpb.ValidatorAssignments_CommitteeAssignment{
			CrosslinkCommittees: committee,
			Shard:               shard,
			Slot:                slot,
			Proposer:            isProposer,
			PublicKey:           s.Validators[index].PublicKey,
		})
	}

	if !reflect.DeepEqual(res.Assignments, wanted) {
		t.Error("Did not receive wanted assignments")
	}
}

func TestBeaconChainServer_ListAssignmentsFilterPubkeysIndicesNoPage(t *testing.T) {
	helpers.ClearAllCaches()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 100
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}, ExitEpoch: params.BeaconConfig().FarFutureEpoch})
	}

	s := &pbp2p.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)}
	if err := db.SaveState(context.Background(), s); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	req := &ethpb.ListValidatorAssignmentsRequest{Epoch: 0, PublicKeys: [][]byte{{1}, {2}}, Indices: []uint64{2, 3}}
	res, err := bs.ListValidatorAssignments(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Construct the wanted assignments
	var wanted []*ethpb.ValidatorAssignments_CommitteeAssignment

	activeIndices, err := helpers.ActiveValidatorIndices(s, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range activeIndices[1:4] {
		committee, shard, slot, isProposer, err := helpers.CommitteeAssignment(s, 0, index)
		if err != nil {
			t.Fatal(err)
		}
		wanted = append(wanted, &ethpb.ValidatorAssignments_CommitteeAssignment{
			CrosslinkCommittees: committee,
			Shard:               shard,
			Slot:                slot,
			Proposer:            isProposer,
			PublicKey:           s.Validators[index].PublicKey,
		})
	}

	if !reflect.DeepEqual(res.Assignments, wanted) {
		t.Error("Did not receive wanted assignments")
	}
}

func TestBeaconChainServer_ListAssignmentsCanFilterPubkeysIndicesWithPages(t *testing.T) {
	helpers.ClearAllCaches()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 100
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}, ExitEpoch: params.BeaconConfig().FarFutureEpoch})
	}

	s := &pbp2p.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)}
	if err := db.SaveState(context.Background(), s); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	req := &ethpb.ListValidatorAssignmentsRequest{Epoch: 0, Indices: []uint64{1, 2, 3, 4, 5, 6}, PageSize: 2, PageToken: "1"}
	res, err := bs.ListValidatorAssignments(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Construct the wanted assignments
	var assignments []*ethpb.ValidatorAssignments_CommitteeAssignment

	activeIndices, err := helpers.ActiveValidatorIndices(s, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range activeIndices[3:5] {
		committee, shard, slot, isProposer, err := helpers.CommitteeAssignment(s, 0, index)
		if err != nil {
			t.Fatal(err)
		}
		assignments = append(assignments, &ethpb.ValidatorAssignments_CommitteeAssignment{
			CrosslinkCommittees: committee,
			Shard:               shard,
			Slot:                slot,
			Proposer:            isProposer,
			PublicKey:           s.Validators[index].PublicKey,
		})
	}

	wantedRes := &ethpb.ValidatorAssignments{
		Assignments:   assignments,
		TotalSize:     int32(len(req.Indices)),
		NextPageToken: "2",
	}

	if !reflect.DeepEqual(res, wantedRes) {
		t.Error("Did not receive wanted assignments")
	}

	// Test the wrap around scenario
	assignments = nil
	req = &ethpb.ListValidatorAssignmentsRequest{Epoch: 0, Indices: []uint64{1, 2, 3, 4, 5, 6}, PageSize: 5, PageToken: "1"}
	res, err = bs.ListValidatorAssignments(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	for _, index := range activeIndices[6:7] {
		committee, shard, slot, isProposer, err := helpers.CommitteeAssignment(s, 0, index)
		if err != nil {
			t.Fatal(err)
		}
		assignments = append(assignments, &ethpb.ValidatorAssignments_CommitteeAssignment{
			CrosslinkCommittees: committee,
			Shard:               shard,
			Slot:                slot,
			Proposer:            isProposer,
			PublicKey:           s.Validators[index].PublicKey,
		})
	}

	wantedRes = &ethpb.ValidatorAssignments{
		Assignments:   assignments,
		TotalSize:     int32(len(req.Indices)),
		NextPageToken: "2",
	}

	if !reflect.DeepEqual(res, wantedRes) {
		t.Error("Did not receive wanted assignments")
	}
}
