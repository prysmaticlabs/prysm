package rpc

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
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

type mockPool struct{}

func (m *mockPool) AttestationPool(ctx context.Context, expectedSlot uint64) ([]*ethpb.Attestation, error) {
	return []*ethpb.Attestation{
		{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: []byte("1"),
			},
		},
		{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: []byte("2"),
			},
		},
	}, nil
}

func TestBeaconChainServer_ListAttestationsNoPagination(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	count := uint64(10)
	atts := make([]*ethpb.Attestation, 0, count)
	for i := uint64(0); i < count; i++ {
		attExample := &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard: i,
				},
			},
		}
		if err := db.SaveAttestation(ctx, attExample); err != nil {
			t.Fatal(err)
		}
		atts = append(atts, attExample)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	received, err := bs.ListAttestations(ctx, &ethpb.ListAttestationsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(atts, received.Attestations) {
		t.Fatalf("incorrect attestations response: wanted %v, received %v", atts, received.Attestations)
	}
}

func TestBeaconChainServer_ListAttestationsPagination(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	count := uint64(100)
	atts := make([]*ethpb.Attestation, 0, count)
	for i := uint64(0); i < count; i++ {
		attExample := &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard: i,
				},
			},
		}
		if err := db.SaveAttestation(ctx, attExample); err != nil {
			t.Fatal(err)
		}
		atts = append(atts, attExample)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	tests := []struct {
		req *ethpb.ListAttestationsRequest
		res *ethpb.ListAttestationsResponse
	}{
		{req: &ethpb.ListAttestationsRequest{PageToken: strconv.Itoa(1), PageSize: 3},
			res: &ethpb.ListAttestationsResponse{
				Attestations: []*ethpb.Attestation{
					{Data: &ethpb.AttestationData{
						Crosslink: &ethpb.Crosslink{Shard: 3},
					}},
					{Data: &ethpb.AttestationData{
						Crosslink: &ethpb.Crosslink{Shard: 4},
					}},
					{Data: &ethpb.AttestationData{
						Crosslink: &ethpb.Crosslink{Shard: 5},
					}},
				},
				NextPageToken: strconv.Itoa(2),
				TotalSize:     int32(count)}},
		{req: &ethpb.ListAttestationsRequest{PageToken: strconv.Itoa(10), PageSize: 5},
			res: &ethpb.ListAttestationsResponse{
				Attestations: []*ethpb.Attestation{
					{Data: &ethpb.AttestationData{
						Crosslink: &ethpb.Crosslink{Shard: 50},
					}},
					{Data: &ethpb.AttestationData{
						Crosslink: &ethpb.Crosslink{Shard: 51},
					}},
					{Data: &ethpb.AttestationData{
						Crosslink: &ethpb.Crosslink{Shard: 52},
					}},
					{Data: &ethpb.AttestationData{
						Crosslink: &ethpb.Crosslink{Shard: 53},
					}},
					{Data: &ethpb.AttestationData{
						Crosslink: &ethpb.Crosslink{Shard: 54},
					}},
				},
				NextPageToken: strconv.Itoa(11),
				TotalSize:     int32(count)}},
		{req: &ethpb.ListAttestationsRequest{PageToken: strconv.Itoa(33), PageSize: 3},
			res: &ethpb.ListAttestationsResponse{
				Attestations: []*ethpb.Attestation{
					{Data: &ethpb.AttestationData{
						Crosslink: &ethpb.Crosslink{Shard: 99},
					}},
				},
				NextPageToken: strconv.Itoa(34),
				TotalSize:     int32(count)}},
		{req: &ethpb.ListAttestationsRequest{PageSize: 2},
			res: &ethpb.ListAttestationsResponse{
				Attestations: []*ethpb.Attestation{
					{Data: &ethpb.AttestationData{
						Crosslink: &ethpb.Crosslink{Shard: 0},
					}},
					{Data: &ethpb.AttestationData{
						Crosslink: &ethpb.Crosslink{Shard: 1},
					}},
				},
				NextPageToken: strconv.Itoa(1),
				TotalSize:     int32(count)}},
	}
	for _, test := range tests {
		res, err := bs.ListAttestations(ctx, test.req)
		if err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(res, test.res) {
			t.Error("Incorrect attestations response")
		}
	}
}

func TestBeaconChainServer_ListAttestationsPaginationOutOfRange(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	count := uint64(1)
	atts := make([]*ethpb.Attestation, 0, count)
	for i := uint64(0); i < count; i++ {
		attExample := &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard: i,
				},
			},
		}
		if err := db.SaveAttestation(ctx, attExample); err != nil {
			t.Fatal(err)
		}
		atts = append(atts, attExample)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	req := &ethpb.ListAttestationsRequest{PageToken: strconv.Itoa(1), PageSize: 100}
	wanted := fmt.Sprintf("page start %d >= list %d", req.PageSize, len(atts))
	if _, err := bs.ListAttestations(ctx, req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestBeaconChainServer_ListAttestationsExceedsMaxPageSize(t *testing.T) {
	ctx := context.Background()
	bs := &BeaconChainServer{}
	exceedsMax := int32(params.BeaconConfig().MaxPageSize + 1)

	wanted := fmt.Sprintf("requested page size %d can not be greater than max size %d", exceedsMax, params.BeaconConfig().MaxPageSize)
	req := &ethpb.ListAttestationsRequest{PageToken: strconv.Itoa(0), PageSize: exceedsMax}
	if _, err := bs.ListAttestations(ctx, req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestBeaconChainServer_ListAttestationsDefaultPageSize(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	count := uint64(params.BeaconConfig().DefaultPageSize)
	atts := make([]*ethpb.Attestation, 0, count)
	for i := uint64(0); i < count; i++ {
		attExample := &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard: i,
				},
			},
		}
		if err := db.SaveAttestation(ctx, attExample); err != nil {
			t.Fatal(err)
		}
		atts = append(atts, attExample)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	req := &ethpb.ListAttestationsRequest{}
	res, err := bs.ListAttestations(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	j := params.BeaconConfig().DefaultPageSize
	if !reflect.DeepEqual(res.Attestations, atts[i:j]) {
		t.Error("Incorrect attestations response")
	}
}

func TestBeaconChainServer_AttestationPool(t *testing.T) {
	ctx := context.Background()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	bs := &BeaconChainServer{
		pool:     &mockPool{},
		beaconDB: db,
	}
	if err := bs.beaconDB.SaveBlock(&ethpb.BeaconBlock{Slot: 10}); err != nil {
		t.Fatal(err)
	}
	if err := bs.beaconDB.UpdateChainHead(ctx, &ethpb.BeaconBlock{Slot: 10}, &pbp2p.BeaconState{Slot: 10}); err != nil {
		t.Fatal(err)
	}
	res, err := bs.AttestationPool(ctx, &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	want, _ := bs.pool.AttestationPool(ctx, 10)
	if !reflect.DeepEqual(res.Attestations, want) {
		t.Errorf("Wanted AttestationPool() = %v, received %v", want, res.Attestations)
	}
}

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
	wanted := fmt.Sprintf("page start %d >= list %d", req.PageSize, len(validators))
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

	wanted := fmt.Sprintf("page start %d >= list %d", 0, 0)
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

func TestBeaconChainServer_GetValidatorsParticipation(t *testing.T) {
	helpers.ClearAllCaches()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

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

	atts := []*pbp2p.PendingAttestation{{Data: &ethpb.AttestationData{Crosslink: &ethpb.Crosslink{Shard: 0}, Target: &ethpb.Checkpoint{}}}}
	var crosslinks []*ethpb.Crosslink
	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
		crosslinks = append(crosslinks, &ethpb.Crosslink{
			StartEpoch: 0,
			DataRoot:   []byte{'A'},
		})
	}

	s := &pbp2p.BeaconState{
		Slot:                       epoch*params.BeaconConfig().SlotsPerEpoch + 1,
		Validators:                 validators,
		Balances:                   balances,
		BlockRoots:                 make([][]byte, 128),
		Slashings:                  []uint64{0, 1e9, 1e9},
		RandaoMixes:                make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots:           make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CompactCommitteesRoots:     make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CurrentCrosslinks:          crosslinks,
		CurrentEpochAttestations:   atts,
		FinalizedCheckpoint:        &ethpb.Checkpoint{},
		JustificationBits:          bitfield.Bitvector4{0x00},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{},
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	if err := bs.beaconDB.SaveState(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	if err := bs.beaconDB.SaveFinalizedBlock(&ethpb.BeaconBlock{Slot: 1}); err != nil {
		t.Fatal(err)
	}

	res, err := bs.GetValidatorParticipation(context.Background(), &ethpb.GetValidatorParticipationRequest{Epoch: epoch})
	if err != nil {
		t.Fatal(err)
	}

	wanted := &ethpb.ValidatorParticipation{
		Epoch:                   epoch,
		VotedEther:              attestedBalance,
		EligibleEther:           validatorCount * params.BeaconConfig().MaxEffectiveBalance,
		GlobalParticipationRate: float32(attestedBalance) / float32(validatorCount*params.BeaconConfig().MaxEffectiveBalance),
	}

	if !reflect.DeepEqual(res, wanted) {
		t.Error("Incorrect validator participation respond")
	}
}

func TestBeaconChainServer_ListBlocksPagination(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	count := uint64(100)
	blks := make([]*ethpb.BeaconBlock, count)
	for i := uint64(0); i < count; i++ {
		b := &ethpb.BeaconBlock{
			Slot: i,
		}
		if err := db.SaveBlock(b); err != nil {
			t.Fatal(err)
		}
		blks[i] = b
	}

	root6, err := ssz.SigningRoot(&ethpb.BeaconBlock{Slot: 6})
	if err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	tests := []struct {
		req *ethpb.ListBlocksRequest
		res *ethpb.ListBlocksResponse
	}{
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Slot{Slot: 5},
			PageSize:    3},
			res: &ethpb.ListBlocksResponse{
				Blocks:        []*ethpb.BeaconBlock{{Slot: 5}},
				NextPageToken: strconv.Itoa(1),
				TotalSize:     1}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Root{Root: root6[:]},
			PageSize:    3},
			res: &ethpb.ListBlocksResponse{
				Blocks:    []*ethpb.BeaconBlock{{Slot: 6}},
				TotalSize: 1}},
		{req: &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Root{Root: root6[:]}},
			res: &ethpb.ListBlocksResponse{
				Blocks:    []*ethpb.BeaconBlock{{Slot: 6}},
				TotalSize: 1}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: 0},
			PageSize:    100},
			res: &ethpb.ListBlocksResponse{
				Blocks:        blks[0:params.BeaconConfig().SlotsPerEpoch],
				NextPageToken: strconv.Itoa(1),
				TotalSize:     int32(params.BeaconConfig().SlotsPerEpoch)}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(1),
			QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: 5},
			PageSize:    3},
			res: &ethpb.ListBlocksResponse{
				Blocks:        blks[43:46],
				NextPageToken: strconv.Itoa(2),
				TotalSize:     int32(params.BeaconConfig().SlotsPerEpoch)}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(1),
			QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: 11},
			PageSize:    7},
			res: &ethpb.ListBlocksResponse{
				Blocks:        blks[95:96],
				NextPageToken: strconv.Itoa(2),
				TotalSize:     int32(params.BeaconConfig().SlotsPerEpoch)}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: 12},
			PageSize:    4},
			res: &ethpb.ListBlocksResponse{
				Blocks:        blks[96:100],
				NextPageToken: strconv.Itoa(1),
				TotalSize:     int32(params.BeaconConfig().SlotsPerEpoch / 2)}},
	}

	for _, test := range tests {
		res, err := bs.ListBlocks(ctx, test.req)
		if err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(res, test.res) {
			t.Error("Incorrect blocks response")
		}
	}
}

func TestBeaconChainServer_ListBlocksErrors(t *testing.T) {
	ctx := context.Background()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	bs := &BeaconChainServer{beaconDB: db}
	exceedsMax := int32(params.BeaconConfig().MaxPageSize + 1)

	wanted := fmt.Sprintf("requested page size %d can not be greater than max size %d", exceedsMax, params.BeaconConfig().MaxPageSize)
	req := &ethpb.ListBlocksRequest{PageToken: strconv.Itoa(0), PageSize: exceedsMax}
	if _, err := bs.ListBlocks(ctx, req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}

	wanted = "must satisfy one of the filter requirement"
	req = &ethpb.ListBlocksRequest{}
	if _, err := bs.ListBlocks(ctx, req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Epoch{}}
	res, err := bs.ListBlocks(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Blocks) != 0 {
		t.Errorf("wanted empty list, got a list of %d", len(res.Blocks))
	}
	if res.TotalSize != 0 {
		t.Errorf("wanted total size 0, got size %d", res.TotalSize)

	}

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Slot{}}
	res, err = bs.ListBlocks(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Blocks) != 0 {
		t.Errorf("wanted empty list, got a list of %d", len(res.Blocks))
	}
	if res.TotalSize != 0 {
		t.Errorf("wanted total size 0, got size %d", res.TotalSize)

	}

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Root{Root: []byte{'A'}}}
	res, err = bs.ListBlocks(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Blocks) != 0 {
		t.Errorf("wanted empty list, got a list of %d", len(res.Blocks))
	}
	if res.TotalSize != 0 {
		t.Errorf("wanted total size 0, got size %d", res.TotalSize)

	}

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Root{Root: []byte{'A'}}}
	res, err = bs.ListBlocks(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Blocks) != 0 {
		t.Errorf("wanted empty list, got a list of %d", len(res.Blocks))
	}
	if res.TotalSize != 0 {
		t.Errorf("wanted total size 0, got size %d", res.TotalSize)

	}
}
