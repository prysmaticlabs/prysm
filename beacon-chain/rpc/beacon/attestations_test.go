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
	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	mockRPC "github.com/prysmaticlabs/prysm/beacon-chain/rpc/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	mocktick "github.com/prysmaticlabs/prysm/shared/slotutil/testing"
)

func TestServer_ListAttestations_NoResults(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	st, err := stateTrie.InitializeFromProto(&pbp2p.BeaconState{
		Slot: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: st,
		},
	}
	wanted := &ethpb.ListAttestationsResponse{
		Attestations:  make([]*ethpb.Attestation, 0),
		TotalSize:     int32(0),
		NextPageToken: strconv.Itoa(0),
	}
	res, err := bs.ListAttestations(ctx, &ethpb.ListAttestationsRequest{
		QueryFilter: &ethpb.ListAttestationsRequest_SourceEpoch{SourceEpoch: 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
}

func TestServer_ListAttestations_Genesis(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	st, err := stateTrie.InitializeFromProto(&pbp2p.BeaconState{
		Slot: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: st,
		},
	}

	// Should throw an error if no genesis data is found.
	if _, err := bs.ListAttestations(ctx, &ethpb.ListAttestationsRequest{
		QueryFilter: &ethpb.ListAttestationsRequest_Genesis{
			Genesis: true,
		},
	}); err != nil && !strings.Contains(err.Error(), "Could not find genesis") {
		t.Fatal(err)
	}

	parentRoot := [32]byte{1, 2, 3}
	blk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{
		Slot:       0,
		ParentRoot: parentRoot[:],
	},
	}
	root, err := ssz.HashTreeRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, root); err != nil {
		t.Fatal(err)
	}
	att := &ethpb.Attestation{
		AggregationBits: bitfield.Bitlist{0b11},
		Data: &ethpb.AttestationData{
			Slot:            0,
			BeaconBlockRoot: root[:],
		},
	}
	if err := db.SaveAttestation(ctx, att); err != nil {
		t.Fatal(err)
	}
	wanted := &ethpb.ListAttestationsResponse{
		Attestations:  []*ethpb.Attestation{att},
		NextPageToken: "",
		TotalSize:     1,
	}

	res, err := bs.ListAttestations(ctx, &ethpb.ListAttestationsRequest{
		QueryFilter: &ethpb.ListAttestationsRequest_Genesis{
			Genesis: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}

	// Should throw an error if there is more than 1 block
	// for the genesis slot.
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}
	if _, err := bs.ListAttestations(ctx, &ethpb.ListAttestationsRequest{
		QueryFilter: &ethpb.ListAttestationsRequest_Genesis{
			Genesis: true,
		},
	}); err != nil && !strings.Contains(err.Error(), "Found more than 1") {
		t.Fatal(err)
	}
}

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
						AggregationBits: bitfield.Bitlist{0b11}},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Slot:            4,
					},
						AggregationBits: bitfield.Bitlist{0b11}},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Slot:            5,
					},
						AggregationBits: bitfield.Bitlist{0b11}},
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
						AggregationBits: bitfield.Bitlist{0b11}},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Slot:            51,
					},
						AggregationBits: bitfield.Bitlist{0b11}},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Slot:            52,
					},
						AggregationBits: bitfield.Bitlist{0b11}},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Slot:            53,
					},
						AggregationBits: bitfield.Bitlist{0b11}},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Slot:            54,
					}, AggregationBits: bitfield.Bitlist{0b11}},
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
						AggregationBits: bitfield.Bitlist{0b11}},
				},
				NextPageToken: "",
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
						AggregationBits: bitfield.Bitlist{0b11}},
					{Data: &ethpb.AttestationData{
						BeaconBlockRoot: []byte("root"),
						Slot:            1,
					},
						AggregationBits: bitfield.Bitlist{0b11},
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
	exceedsMax := int32(flags.Get().MaxPageSize + 1)

	wanted := fmt.Sprintf("Requested page size %d can not be greater than max size %d", exceedsMax, flags.Get().MaxPageSize)
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

func TestServer_AttestationPool_Pagination_ExceedsMaxPageSize(t *testing.T) {
	ctx := context.Background()
	bs := &Server{}
	exceedsMax := int32(flags.Get().MaxPageSize + 1)

	wanted := fmt.Sprintf("Requested page size %d can not be greater than max size %d", exceedsMax, flags.Get().MaxPageSize)
	req := &ethpb.AttestationPoolRequest{PageToken: strconv.Itoa(0), PageSize: exceedsMax}
	if _, err := bs.AttestationPool(ctx, req); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_AttestationPool_Pagination_OutOfRange(t *testing.T) {
	ctx := context.Background()
	bs := &Server{
		AttestationsPool: attestations.NewPool(),
	}

	atts := []*ethpb.Attestation{
		{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1101}},
		{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1101}},
		{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b1101}},
	}
	if err := bs.AttestationsPool.SaveAggregatedAttestations(atts); err != nil {
		t.Fatal(err)
	}

	req := &ethpb.AttestationPoolRequest{
		PageToken: strconv.Itoa(1),
		PageSize:  100,
	}
	wanted := fmt.Sprintf("page start %d >= list %d", req.PageSize, len(atts))
	if _, err := bs.AttestationPool(ctx, req); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_AttestationPool_Pagination_DefaultPageSize(t *testing.T) {
	ctx := context.Background()
	bs := &Server{
		AttestationsPool: attestations.NewPool(),
	}

	atts := make([]*ethpb.Attestation, params.BeaconConfig().DefaultPageSize+1)
	for i := 0; i < len(atts); i++ {
		atts[i] = &ethpb.Attestation{
			Data:            &ethpb.AttestationData{Slot: uint64(i)},
			AggregationBits: bitfield.Bitlist{0b1101},
		}
	}
	if err := bs.AttestationsPool.SaveAggregatedAttestations(atts); err != nil {
		t.Fatal(err)
	}

	req := &ethpb.AttestationPoolRequest{}
	res, err := bs.AttestationPool(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Attestations) != params.BeaconConfig().DefaultPageSize {
		t.Errorf(
			"Wanted %d attestations in response, received %d",
			params.BeaconConfig().DefaultPageSize,
			len(res.Attestations),
		)
	}
	if int(res.TotalSize) != params.BeaconConfig().DefaultPageSize+1 {
		t.Errorf("Wanted total size %d, received %d", params.BeaconConfig().DefaultPageSize+1, res.TotalSize)
	}
}

func TestServer_AttestationPool_Pagination_CustomPageSize(t *testing.T) {
	ctx := context.Background()
	bs := &Server{
		AttestationsPool: attestations.NewPool(),
	}

	numAtts := 100
	atts := make([]*ethpb.Attestation, numAtts)
	for i := 0; i < len(atts); i++ {
		atts[i] = &ethpb.Attestation{
			Data:            &ethpb.AttestationData{Slot: uint64(i)},
			AggregationBits: bitfield.Bitlist{0b1101},
		}
	}
	if err := bs.AttestationsPool.SaveAggregatedAttestations(atts); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		req *ethpb.AttestationPoolRequest
		res *ethpb.AttestationPoolResponse
	}{
		{
			req: &ethpb.AttestationPoolRequest{
				PageToken: strconv.Itoa(1),
				PageSize:  3,
			},
			res: &ethpb.AttestationPoolResponse{
				NextPageToken: "2",
				TotalSize:     int32(numAtts),
			},
		},
		{
			req: &ethpb.AttestationPoolRequest{
				PageToken: strconv.Itoa(3),
				PageSize:  30,
			},
			res: &ethpb.AttestationPoolResponse{
				NextPageToken: "",
				TotalSize:     int32(numAtts),
			},
		},
		{
			req: &ethpb.AttestationPoolRequest{
				PageToken: strconv.Itoa(0),
				PageSize:  int32(numAtts),
			},
			res: &ethpb.AttestationPoolResponse{
				NextPageToken: "1",
				TotalSize:     int32(numAtts),
			},
		},
	}
	for _, tt := range tests {
		res, err := bs.AttestationPool(ctx, tt.req)
		if err != nil {
			t.Fatal(err)
		}
		if res.TotalSize != tt.res.TotalSize {
			t.Errorf("Wanted total size %d, received %d", tt.res.TotalSize, res.TotalSize)
		}
		if res.NextPageToken != tt.res.NextPageToken {
			t.Errorf("Wanted next page token %s, received %s", tt.res.NextPageToken, res.NextPageToken)
		}
	}
}

func TestServer_StreamAttestations_ContextCanceled(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
	ticker := &mocktick.MockTicker{
		Channel: make(chan uint64),
	}
	server := &Server{
		Ctx:        ctx,
		SlotTicker: ticker,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mockRPC.NewMockBeaconChain_StreamAttestationsServer(ctrl)
	mockStream.EXPECT().Context().Return(ctx)
	go func(tt *testing.T) {
		if err := server.StreamAttestations(
			&ptypes.Empty{},
			mockStream,
		); !strings.Contains(err.Error(), "Context canceled") {
			tt.Errorf("Expected context canceled error got: %v", err)
		}
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestServer_StreamAttestations_OnSlotTick(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	ticker := &mocktick.MockTicker{
		Channel: make(chan uint64),
	}
	server := &Server{
		Ctx:              ctx,
		SlotTicker:       ticker,
		AttestationsPool: attestations.NewPool(),
	}

	atts := []*ethpb.Attestation{
		{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1101}},
		{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1101}},
		{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b1101}},
	}
	if err := server.AttestationsPool.SaveAggregatedAttestations(atts); err != nil {
		t.Fatal(err)
	}

	mockStream := mockRPC.NewMockBeaconChain_StreamAttestationsServer(ctrl)
	mockStream.EXPECT().Send(atts[0])
	mockStream.EXPECT().Send(atts[1])
	mockStream.EXPECT().Send(atts[2]).Do(func(arg0 interface{}) {
		exitRoutine <- true
	})
	mockStream.EXPECT().Context().Return(ctx).AnyTimes()

	go func(tt *testing.T) {
		if err := server.StreamAttestations(&ptypes.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
	}(t)
	ticker.Channel <- 0
	<-exitRoutine
}
