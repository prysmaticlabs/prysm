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
				BeaconBlockRoot: []byte("root"),
				Crosslink: &ethpb.Crosslink{
					Shard: i,
				},
			},
			AggregationBits: bitfield.Bitlist{0b11},
			CustodyBits:     bitfield.NewBitlist(1),
		}
		if err := db.SaveAttestation(ctx, attExample); err != nil {
			t.Fatal(err)
		}
		atts = append(atts, attExample)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	received, err := bs.ListAttestations(ctx, &ethpb.ListAttestationsRequest{
		QueryFilter: &ethpb.ListAttestationsRequest_HeadBlockRoot{
			HeadBlockRoot: []byte("root"),
		},
	})
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
			AggregationBits: bitfield.Bitlist{0b11},
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
			AggregationBits: bitfield.Bitlist{0b11},
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
			AggregationBits: bitfield.Bitlist{0b11},
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
				BeaconBlockRoot: []byte("root"),
				Crosslink: &ethpb.Crosslink{
					Shard: i,
				},
			},
			AggregationBits: bitfield.Bitlist{0b11},
			CustodyBits:     bitfield.NewBitlist(1),
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
		{
			req: &ethpb.ListAttestationsRequest{
				QueryFilter: &ethpb.ListAttestationsRequest_HeadBlockRoot{
					HeadBlockRoot: []byte("root"),
				},
				PageToken: strconv.Itoa(1),
				PageSize:  3,
			},
			res: &ethpb.ListAttestationsResponse{
				Attestations: []*ethpb.Attestation{
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Crosslink:       &ethpb.Crosslink{Shard: 3},
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Crosslink:       &ethpb.Crosslink{Shard: 4},
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Crosslink:       &ethpb.Crosslink{Shard: 5},
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
				},
				NextPageToken: strconv.Itoa(2),
				TotalSize:     int32(count)}},
		{
			req: &ethpb.ListAttestationsRequest{
				QueryFilter: &ethpb.ListAttestationsRequest_HeadBlockRoot{
					HeadBlockRoot: []byte("root"),
				},
				PageToken: strconv.Itoa(10),
				PageSize:  5,
			},
			res: &ethpb.ListAttestationsResponse{
				Attestations: []*ethpb.Attestation{
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Crosslink:       &ethpb.Crosslink{Shard: 50},
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Crosslink:       &ethpb.Crosslink{Shard: 51},
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Crosslink:       &ethpb.Crosslink{Shard: 52},
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Crosslink:       &ethpb.Crosslink{Shard: 53},
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Crosslink:       &ethpb.Crosslink{Shard: 54},
					}, AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits: bitfield.NewBitlist(1)},
				},
				NextPageToken: strconv.Itoa(11),
				TotalSize:     int32(count)}},
		{
			req: &ethpb.ListAttestationsRequest{
				QueryFilter: &ethpb.ListAttestationsRequest_HeadBlockRoot{
					HeadBlockRoot: []byte("root"),
				},
				PageToken: strconv.Itoa(33),
				PageSize:  3,
			},
			res: &ethpb.ListAttestationsResponse{
				Attestations: []*ethpb.Attestation{
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Crosslink:       &ethpb.Crosslink{Shard: 99},
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
				},
				NextPageToken: strconv.Itoa(34),
				TotalSize:     int32(count)}},
		{
			req: &ethpb.ListAttestationsRequest{
				QueryFilter: &ethpb.ListAttestationsRequest_HeadBlockRoot{
					HeadBlockRoot: []byte("root"),
				},
				PageSize: 2,
			},
			res: &ethpb.ListAttestationsResponse{
				Attestations: []*ethpb.Attestation{
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Crosslink:       &ethpb.Crosslink{Shard: 0},
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Crosslink:       &ethpb.Crosslink{Shard: 1},
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1),
					},
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
			t.Errorf("Incorrect attestations response, wanted %v, received %v", test.res, res)
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
				BeaconBlockRoot: []byte("root"),
				Crosslink: &ethpb.Crosslink{
					Shard: i,
				},
			},
			AggregationBits: bitfield.Bitlist{0b11},
		}
		if err := db.SaveAttestation(ctx, attExample); err != nil {
			t.Fatal(err)
		}
		atts = append(atts, attExample)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	req := &ethpb.ListAttestationsRequest{
		QueryFilter: &ethpb.ListAttestationsRequest_HeadBlockRoot{
			HeadBlockRoot: []byte("root"),
		},
		PageToken: strconv.Itoa(1),
		PageSize:  100,
	}
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
				BeaconBlockRoot: []byte("root"),
				Crosslink: &ethpb.Crosslink{
					Shard: i,
				},
			},
			AggregationBits: bitfield.Bitlist{0b11},
			CustodyBits:     bitfield.NewBitlist(1),
		}
		if err := db.SaveAttestation(ctx, attExample); err != nil {
			t.Fatal(err)
		}
		atts = append(atts, attExample)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	req := &ethpb.ListAttestationsRequest{
		QueryFilter: &ethpb.ListAttestationsRequest_HeadBlockRoot{
			HeadBlockRoot: []byte("root"),
		},
	}
	res, err := bs.ListAttestations(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	j := params.BeaconConfig().DefaultPageSize
	if !reflect.DeepEqual(res.Attestations, atts[i:j]) {
		t.Log(res.Attestations, atts[i:j])
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
	want, _ := bs.pool.AttestationPoolNoVerify(ctx)
	if !reflect.DeepEqual(res.Attestations, want) {
		t.Errorf("Wanted AttestationPool() = %v, received %v", want, res.Attestations)
	}
}

func TestBeaconChainServer_ListValidatorBalances(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	setupValidators(t, db, 100)

	headState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB:    db,
		headFetcher: &mock.ChainService{State: headState},
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
	setupValidators(t, db, 1)

	headState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB:    db,
		headFetcher: &mock.ChainService{State: headState},
	}

	req := &ethpb.GetValidatorBalancesRequest{Indices: []uint64{uint64(1)}}
	wanted := "does not exist"
	if _, err := bs.ListValidatorBalances(context.Background(), req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestBeaconChainServer_ListValidatorBalancesFromArchive(t *testing.T) {
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
	bs := &BeaconChainServer{
		beaconDB: db,
		headFetcher: &mock.ChainService{
			State: &pbp2p.BeaconState{
				Slot:       params.BeaconConfig().SlotsPerEpoch * 3,
				Validators: validators,
				Balances:   newerBalances,
			},
		},
	}

	req := &ethpb.GetValidatorBalancesRequest{
		QueryFilter: &ethpb.GetValidatorBalancesRequest_Epoch{Epoch: 0},
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

func TestBeaconChainServer_ListValidatorBalancesFromArchive_NewValidatorNotFound(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()
	epoch := uint64(0)
	_, balances := setupValidators(t, db, 100)

	if err := db.SaveArchivedBalances(ctx, epoch, balances); err != nil {
		t.Fatal(err)
	}

	newValidators, newBalances := setupValidators(t, db, 200)
	bs := &BeaconChainServer{
		beaconDB: db,
		headFetcher: &mock.ChainService{
			State: &pbp2p.BeaconState{
				Slot:       params.BeaconConfig().SlotsPerEpoch * 3,
				Validators: newValidators,
				Balances:   newBalances,
			},
		},
	}

	req := &ethpb.GetValidatorBalancesRequest{
		QueryFilter: &ethpb.GetValidatorBalancesRequest_Epoch{Epoch: 0},
		Indices:     []uint64{1, 150, 161},
	}
	if _, err := bs.ListValidatorBalances(context.Background(), req); !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Wanted out of range error for including newer validators in the arguments, received %v", err)
	}
}

func TestBeaconChainServer_GetValidators_NoPagination(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	validators, _ := setupValidators(t, db, 100)
	headState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		headFetcher: &mock.ChainService{
			State: headState,
		},
		finalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
	}

	received, err := bs.GetValidators(context.Background(), &ethpb.GetValidatorsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(validators, received.Validators) {
		t.Fatal("Incorrect respond of validators")
	}
}

func TestBeaconChainServer_GetValidators_Pagination(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	count := 100
	setupValidators(t, db, count)

	headState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		headFetcher: &mock.ChainService{
			State: headState,
		},
		finalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
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

func TestBeaconChainServer_GetValidators_PaginationOutOfRange(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	count := 1
	validators, _ := setupValidators(t, db, count)
	headState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		headFetcher: &mock.ChainService{
			State: headState,
		},
		finalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
	}

	req := &ethpb.GetValidatorsRequest{PageToken: strconv.Itoa(1), PageSize: 100}
	wanted := fmt.Sprintf("page start %d >= list %d", req.PageSize, len(validators))
	if _, err := bs.GetValidators(context.Background(), req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestBeaconChainServer_GetValidators_ExceedsMaxPageSize(t *testing.T) {
	bs := &BeaconChainServer{}
	exceedsMax := int32(params.BeaconConfig().MaxPageSize + 1)

	wanted := fmt.Sprintf("requested page size %d can not be greater than max size %d", exceedsMax, params.BeaconConfig().MaxPageSize)
	req := &ethpb.GetValidatorsRequest{PageToken: strconv.Itoa(0), PageSize: exceedsMax}
	if _, err := bs.GetValidators(context.Background(), req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestBeaconChainServer_GetValidators_DefaultPageSize(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	validators, _ := setupValidators(t, db, 1000)
	headState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		headFetcher: &mock.ChainService{
			State: headState,
		},
		finalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
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

func TestBeaconChainServer_GetValidators_FromOldEpoch(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	numEpochs := 30
	validators := make([]*ethpb.Validator, numEpochs)
	for i := 0; i < numEpochs; i++ {
		validators[i] = &ethpb.Validator{
			ActivationEpoch: uint64(i),
		}
	}

	bs := &BeaconChainServer{
		headFetcher: &mock.ChainService{
			State: &pbp2p.BeaconState{
				Validators: validators,
			},
		},
		finalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 200,
			},
		},
	}

	req := &ethpb.GetValidatorsRequest{
		QueryFilter: &ethpb.GetValidatorsRequest_Genesis{
			Genesis: true,
		},
	}
	res, err := bs.GetValidators(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Validators) != 1 {
		t.Errorf("Wanted 1 validator at genesis, received %d", len(res.Validators))
	}

	req = &ethpb.GetValidatorsRequest{
		QueryFilter: &ethpb.GetValidatorsRequest_Epoch{
			Epoch: 20,
		},
	}
	res, err = bs.GetValidators(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.Validators, validators[:21]) {
		t.Errorf("Incorrect number of validators, wanted %d received %d", 20, len(res.Validators))
	}
}

func TestBeaconChainServer_GetValidatorActiveSetChanges(t *testing.T) {
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
	bs := &BeaconChainServer{
		headFetcher: &mock.ChainService{
			State: headState,
		},
		finalizationFetcher: &mock.ChainService{
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

func TestBeaconChainServer_GetValidatorActiveSetChanges_FromArchive(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()
	validators := make([]*ethpb.Validator, 6)
	headState := &pbp2p.BeaconState{
		Slot:       0,
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
	bs := &BeaconChainServer{
		beaconDB: db,
		headFetcher: &mock.ChainService{
			State: headState,
		},
		finalizationFetcher: &mock.ChainService{
			// Pick an epoch far in the future so that we trigger fetching from the archive.
			FinalizedCheckPoint: &ethpb.Checkpoint{Epoch: 100},
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

func TestBeaconChainServer_GetValidatorQueue_PendingActivation(t *testing.T) {
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
	bs := &BeaconChainServer{
		headFetcher: &mock.ChainService{
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
	wantChurn, err := helpers.ValidatorChurnLimit(headState)
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

func TestBeaconChainServer_GetValidatorQueue_PendingExit(t *testing.T) {
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
	bs := &BeaconChainServer{
		headFetcher: &mock.ChainService{
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
	wantChurn, err := helpers.ValidatorChurnLimit(headState)
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

func TestBeaconChainServer_ListAssignmentsInputOutOfRange(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	setupValidators(t, db, 1)
	headState, err := db.HeadState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	bs := &BeaconChainServer{
		beaconDB: db,
		headFetcher: &mock.ChainService{
			State: headState,
		},
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

func TestBeaconChainServer_ListAssignmentsExceedsMaxPageSize(t *testing.T) {
	bs := &BeaconChainServer{}
	exceedsMax := int32(params.BeaconConfig().MaxPageSize + 1)

	wanted := fmt.Sprintf("requested page size %d can not be greater than max size %d", exceedsMax, params.BeaconConfig().MaxPageSize)
	req := &ethpb.ListValidatorAssignmentsRequest{
		PageToken: strconv.Itoa(0),
		PageSize:  exceedsMax,
	}
	if _, err := bs.ListValidatorAssignments(context.Background(), req); err != nil && !strings.Contains(err.Error(), wanted) {
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
		headFetcher: &mock.ChainService{
			State: s,
		},
		finalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
	}

	res, err := bs.ListValidatorAssignments(context.Background(), &ethpb.ListValidatorAssignmentsRequest{
		QueryFilter: &ethpb.ListValidatorAssignmentsRequest_Genesis{Genesis: true},
	})
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

func TestBeaconChainServer_ListAssignmentsDefaultPageSize_FromArchive(t *testing.T) {
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

	// We tell the beacon chain server that our finalized epoch is 10 so that when
	// we request assignments for epoch 0, it looks within the archived data.
	bs := &BeaconChainServer{
		beaconDB: db,
		headFetcher: &mock.ChainService{
			State: s,
		},
		finalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 10,
			},
		},
	}

	// We then store archived data into the DB.
	currentEpoch := helpers.CurrentEpoch(s)
	committeeCount, err := helpers.CommitteeCount(s, currentEpoch)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := helpers.Seed(s, currentEpoch)
	if err != nil {
		t.Fatal(err)
	}
	startShard, err := helpers.StartShard(s, currentEpoch)
	if err != nil {
		t.Fatal(err)
	}
	proposerIndex, err := helpers.BeaconProposerIndex(s)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveArchivedCommitteeInfo(context.Background(), 0, &ethpb.ArchivedCommitteeInfo{
		Seed:           seed[:],
		StartShard:     startShard,
		CommitteeCount: committeeCount,
		ProposerIndex:  proposerIndex,
	}); err != nil {
		t.Fatal(err)
	}

	res, err := bs.ListValidatorAssignments(context.Background(), &ethpb.ListValidatorAssignmentsRequest{
		QueryFilter: &ethpb.ListValidatorAssignmentsRequest_Genesis{Genesis: true},
	})
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

func TestBeaconChainServer_ListAssignmentsFilterPubkeysIndices_NoPagination(t *testing.T) {
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
		headFetcher: &mock.ChainService{
			State: s,
		},
		finalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
	}

	req := &ethpb.ListValidatorAssignmentsRequest{PublicKeys: [][]byte{{1}, {2}}, Indices: []uint64{2, 3}}
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

func TestBeaconChainServer_ListAssignmentsCanFilterPubkeysIndices_WithPagination(t *testing.T) {
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
		headFetcher: &mock.ChainService{
			State: s,
		},
		finalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
	}

	req := &ethpb.ListValidatorAssignmentsRequest{Indices: []uint64{1, 2, 3, 4, 5, 6}, PageSize: 2, PageToken: "1"}
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
		t.Error("Did not get wanted assignments")
	}

	// Test the wrap around scenario.
	assignments = nil
	req = &ethpb.ListValidatorAssignmentsRequest{Indices: []uint64{1, 2, 3, 4, 5, 6}, PageSize: 5, PageToken: "1"}
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

func TestBeaconChainServer_GetValidatorsParticipation_FromArchive(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()
	epoch := uint64(4)
	part := &ethpb.ValidatorParticipation{
		GlobalParticipationRate: 1.0,
		VotedEther:              20,
		EligibleEther:           20,
	}
	if err := db.SaveArchivedValidatorParticipation(ctx, epoch, part); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
		headFetcher: &mock.ChainService{
			State: &pbp2p.BeaconState{Slot: helpers.StartSlot(epoch + 1)},
		},
		finalizationFetcher: &mock.ChainService{
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
		Epoch:         epoch,
		Finalized:     true,
		Participation: part,
	}
	res, err := bs.GetValidatorParticipation(ctx, &ethpb.GetValidatorParticipationRequest{
		QueryFilter: &ethpb.GetValidatorParticipationRequest_Epoch{
			Epoch: epoch,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(want, res) {
		t.Errorf("Wanted %v, received %v", want, res)
	}
}

func TestBeaconChainServer_GetValidatorsParticipation_CurrentEpoch(t *testing.T) {
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

	res, err := bs.GetValidatorParticipation(ctx, &ethpb.GetValidatorParticipationRequest{})
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
