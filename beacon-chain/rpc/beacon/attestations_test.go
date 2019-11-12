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
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockOps "github.com/prysmaticlabs/prysm/beacon-chain/operations/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestServer_ListAttestations_NoPagination(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()

	count := uint64(10)
	atts := make([]*ethpb.Attestation, 0, count)
	for i := uint64(0); i < count; i++ {
		attExample := &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: []byte("root"),
				Slot:            i,
			},
			AggregationBits: bitfield.Bitlist{0b11},
			CustodyBits:     bitfield.NewBitlist(1),
		}
		if err := db.SaveAttestation(ctx, attExample); err != nil {
			t.Fatal(err)
		}
		atts = append(atts, attExample)
	}

	bs := &Server{
		BeaconDB: db,
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

func TestServer_ListAttestations_FiltersCorrectly(t *testing.T) {
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
				Slot: 3,
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
				Slot: 4,
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
				Slot: 5,
			},
			AggregationBits: bitfield.Bitlist{0b11},
		},
	}

	if err := db.SaveAttestations(ctx, atts); err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		BeaconDB: db,
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

func TestServer_ListAttestations_Pagination_CustomPageParameters(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()

	count := uint64(100)
	atts := make([]*ethpb.Attestation, 0, count)
	for i := uint64(0); i < count; i++ {
		attExample := &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: []byte("root"),
				Slot:            i,
			},
			AggregationBits: bitfield.Bitlist{0b11},
			CustodyBits:     bitfield.NewBitlist(1),
		}
		if err := db.SaveAttestation(ctx, attExample); err != nil {
			t.Fatal(err)
		}
		atts = append(atts, attExample)
	}

	bs := &Server{
		BeaconDB: db,
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
						Slot:            3,
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Slot:            4,
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Slot:            5,
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
						Slot:            50,
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Slot:            51,
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Slot:            52,
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Slot:            53,
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Slot:            54,
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
						Slot:            99,
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
					},
						AggregationBits: bitfield.Bitlist{0b11},
						CustodyBits:     bitfield.NewBitlist(1)},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Slot:            1,
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

func TestServer_ListAttestations_Pagination_OutOfRange(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()

	count := uint64(1)
	atts := make([]*ethpb.Attestation, 0, count)
	for i := uint64(0); i < count; i++ {
		attExample := &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: []byte("root"),
				Slot:            i,
			},
			AggregationBits: bitfield.Bitlist{0b11},
		}
		if err := db.SaveAttestation(ctx, attExample); err != nil {
			t.Fatal(err)
		}
		atts = append(atts, attExample)
	}

	bs := &Server{
		BeaconDB: db,
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

func TestServer_ListAttestations_Pagination_ExceedsMaxPageSize(t *testing.T) {
	ctx := context.Background()
	bs := &Server{}
	exceedsMax := int32(params.BeaconConfig().MaxPageSize + 1)

	wanted := fmt.Sprintf("requested page size %d can not be greater than max size %d", exceedsMax, params.BeaconConfig().MaxPageSize)
	req := &ethpb.ListAttestationsRequest{PageToken: strconv.Itoa(0), PageSize: exceedsMax}
	if _, err := bs.ListAttestations(ctx, req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_ListAttestations_Pagination_DefaultPageSize(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()

	count := uint64(params.BeaconConfig().DefaultPageSize)
	atts := make([]*ethpb.Attestation, 0, count)
	for i := uint64(0); i < count; i++ {
		attExample := &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: []byte("root"),
				Slot:            i,
			},
			AggregationBits: bitfield.Bitlist{0b11},
			CustodyBits:     bitfield.NewBitlist(1),
		}
		if err := db.SaveAttestation(ctx, attExample); err != nil {
			t.Fatal(err)
		}
		atts = append(atts, attExample)
	}

	bs := &Server{
		BeaconDB: db,
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

func TestServer_AttestationPool(t *testing.T) {
	ctx := context.Background()
	block := &ethpb.BeaconBlock{
		Slot: 10,
	}
	bs := &Server{
		Pool: &mockOps.Operations{
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
		HeadFetcher: &mock.ChainService{
			Block: block,
		},
	}
	res, err := bs.AttestationPool(ctx, &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	want, _ := bs.Pool.AttestationPoolNoVerify(ctx)
	if !reflect.DeepEqual(res.Attestations, want) {
		t.Errorf("Wanted AttestationPool() = %v, received %v", want, res.Attestations)
	}
}
