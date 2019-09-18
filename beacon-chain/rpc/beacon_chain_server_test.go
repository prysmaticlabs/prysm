package rpc

import (
	"bytes"
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
	mockOps "github.com/prysmaticlabs/prysm/beacon-chain/operations/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestBeaconChainServer_ListAttestationsNoPagination(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
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

func TestBeaconChainServer_ListAttestations_FiltersCorrectly(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()

	someRoot := []byte{1, 2, 3}
	sourceRoot := []byte{4, 5, 6}
	sourceEpoch := uint64(5)
	targetRoot := []byte{7, 8, 9}
	targetEpoch := uint64(7)

	unknownRoot := []byte{1, 1, 1}
	atts := []*ethpb.Attestation{
		{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: someRoot,
				Source: &ethpb.Checkpoint{
					Root:  sourceRoot,
					Epoch: sourceEpoch,
				},
				Target: &ethpb.Checkpoint{
					Root:  targetRoot,
					Epoch: targetEpoch,
				},
				Crosslink: &ethpb.Crosslink{
					Shard: 3,
				},
			},
		},
		{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: unknownRoot,
				Source: &ethpb.Checkpoint{
					Root:  sourceRoot,
					Epoch: sourceEpoch,
				},
				Target: &ethpb.Checkpoint{
					Root:  targetRoot,
					Epoch: targetEpoch,
				},
				Crosslink: &ethpb.Crosslink{
					Shard: 4,
				},
			},
		},
		{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: someRoot,
				Source: &ethpb.Checkpoint{
					Root:  unknownRoot,
					Epoch: sourceEpoch,
				},
				Target: &ethpb.Checkpoint{
					Root:  unknownRoot,
					Epoch: targetEpoch,
				},
				Crosslink: &ethpb.Crosslink{
					Shard: 5,
				},
			},
		},
	}

	if err := db.SaveAttestations(ctx, atts); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	received, err := bs.ListAttestations(ctx, &ethpb.ListAttestationsRequest{
		QueryFilter: &ethpb.ListAttestationsRequest_HeadBlockRoot{HeadBlockRoot: someRoot},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(received.Attestations) != 2 {
		t.Errorf("Wanted 2 matching attestations with root %#x, received %d", someRoot, len(received.Attestations))
	}
	received, err = bs.ListAttestations(ctx, &ethpb.ListAttestationsRequest{
		QueryFilter: &ethpb.ListAttestationsRequest_SourceEpoch{SourceEpoch: sourceEpoch},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(received.Attestations) != 3 {
		t.Errorf("Wanted 3 matching attestations with source epoch %d, received %d", sourceEpoch, len(received.Attestations))
	}
	received, err = bs.ListAttestations(ctx, &ethpb.ListAttestationsRequest{
		QueryFilter: &ethpb.ListAttestationsRequest_SourceRoot{SourceRoot: sourceRoot},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(received.Attestations) != 2 {
		t.Errorf("Wanted 2 matching attestations with source root %#x, received %d", sourceRoot, len(received.Attestations))
	}
	received, err = bs.ListAttestations(ctx, &ethpb.ListAttestationsRequest{
		QueryFilter: &ethpb.ListAttestationsRequest_TargetEpoch{TargetEpoch: targetEpoch},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(received.Attestations) != 3 {
		t.Errorf("Wanted 3 matching attestations with target epoch %d, received %d", targetEpoch, len(received.Attestations))
	}
	received, err = bs.ListAttestations(ctx, &ethpb.ListAttestationsRequest{
		QueryFilter: &ethpb.ListAttestationsRequest_TargetRoot{TargetRoot: targetRoot},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(received.Attestations) != 2 {
		t.Errorf("Wanted 2 matching attestations with target root %#x, received %d", targetRoot, len(received.Attestations))
	}
}

func TestBeaconChainServer_ListAttestationsPagination(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
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
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
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
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
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
	block := &ethpb.BeaconBlock{
		Slot: 10,
	}
	bs := &BeaconChainServer{
		pool: &mockOps.Operations{
			Attestations: []*ethpb.Attestation{
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
			},
		},
		headFetcher: &mock.ChainService{
			Block: block,
		},
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
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	setupValidators(t, db, 100)

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
		{req: &ethpb.GetValidatorBalancesRequest{PublicKeys: [][]byte{{}}, Indices: []uint64{3, 4}}, // Public key has a blank value
			res: &ethpb.ValidatorBalances{Balances: []*ethpb.ValidatorBalances_Balance{
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
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	_, balances := setupValidators(t, db, 1)

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	req := &ethpb.GetValidatorBalancesRequest{Indices: []uint64{uint64(1)}}
	wanted := fmt.Sprintf("validator index %d >= balance list %d", 1, len(balances))
	if _, err := bs.ListValidatorBalances(context.Background(), req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestBeaconChainServer_GetValidatorsNoPagination(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	validators, _ := setupValidators(t, db, 100)
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
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	count := 100
	setupValidators(t, db, count)

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
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	count := 1
	validators, _ := setupValidators(t, db, count)
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
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	validators, _ := setupValidators(t, db, 1000)
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
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	setupValidators(t, db, 1)
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
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	count := 1000
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex(ctx, [48]byte{byte(i)}, uint64(i)); err != nil {
			t.Fatal(err)
		}
		// Mark the validators with index divisible by 3 inactive.
		if i%3 == 0 {
			validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}, ExitEpoch: 0})
		} else {
			validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}, ExitEpoch: params.BeaconConfig().FarFutureEpoch})
		}
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

	s := &pbp2p.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)}
	if err := db.SaveState(ctx, s, blockRoot); err != nil {
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
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	count := 100
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex(ctx, [48]byte{byte(i)}, uint64(i)); err != nil {
			t.Fatal(err)
		}
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}, ExitEpoch: params.BeaconConfig().FarFutureEpoch})
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

	s := &pbp2p.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)}
	if err := db.SaveState(ctx, s, blockRoot); err != nil {
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
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	count := 100
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex(ctx, [48]byte{byte(i)}, uint64(i)); err != nil {
			t.Fatal(err)
		}
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}, ExitEpoch: params.BeaconConfig().FarFutureEpoch})
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

	s := &pbp2p.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)}
	if err := db.SaveState(ctx, s, blockRoot); err != nil {
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
		beaconDB:    db,
		headFetcher: &mock.ChainService{State: s},
	}

	res, err := bs.GetValidatorParticipation(ctx, &ethpb.GetValidatorParticipationRequest{Epoch: epoch})
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
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()

	count := uint64(100)
	blks := make([]*ethpb.BeaconBlock, count)
	for i := uint64(0); i < count; i++ {
		b := &ethpb.BeaconBlock{
			Slot: i,
		}
		blks[i] = b
	}
	if err := db.SaveBlocks(ctx, blks); err != nil {
		t.Fatal(err)
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
			t.Errorf("Incorrect blocks response, wanted %d, received %d", len(test.res.Blocks), len(res.Blocks))
		}
	}
}

func TestBeaconChainServer_ListBlocksErrors(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()

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

func TestBeaconChainServer_GetChainHead(t *testing.T) {
	s := &pbp2p.BeaconState{
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Epoch: 3, Root: []byte{'A'}},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Epoch: 2, Root: []byte{'B'}},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Epoch: 1, Root: []byte{'C'}},
	}

	bs := &BeaconChainServer{headFetcher: &mock.ChainService{State: s}}

	head, err := bs.GetChainHead(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if head.PreviousJustifiedSlot != 3*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Wanted PreviousJustifiedSlot: %d, got: %d",
			3*params.BeaconConfig().SlotsPerEpoch, head.PreviousJustifiedSlot)
	}
	if head.JustifiedSlot != 2*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Wanted JustifiedSlot: %d, got: %d",
			2*params.BeaconConfig().SlotsPerEpoch, head.JustifiedSlot)
	}
	if head.FinalizedSlot != 1*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Wanted FinalizedSlot: %d, got: %d",
			1*params.BeaconConfig().SlotsPerEpoch, head.FinalizedSlot)
	}
	if !bytes.Equal([]byte{'A'}, head.PreviousJustifiedBlockRoot) {
		t.Errorf("Wanted PreviousJustifiedBlockRoot: %v, got: %v",
			[]byte{'A'}, head.PreviousJustifiedBlockRoot)
	}
	if !bytes.Equal([]byte{'B'}, head.JustifiedBlockRoot) {
		t.Errorf("Wanted JustifiedBlockRoot: %v, got: %v",
			[]byte{'B'}, head.JustifiedBlockRoot)
	}
	if !bytes.Equal([]byte{'C'}, head.FinalizedBlockRoot) {
		t.Errorf("Wanted FinalizedBlockRoot: %v, got: %v",
			[]byte{'C'}, head.FinalizedBlockRoot)
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
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}})
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
