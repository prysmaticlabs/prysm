package client

import (
	"context"
	"errors"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestAttestToBlockHead_CrosslinkCommitteeRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, m, finish := setup(t)
	defer finish()
	m.attesterClient.EXPECT().CrosslinkCommitteesAtSlot(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.CrosslinkCommitteeRequest{}),
	).Return(nil /*Crosslinks Response*/, errors.New("something bad happened"))

	validator.AttestToBlockHead(context.Background(), 30)
	testutil.AssertLogsContain(t, hook, "Could not fetch crosslink committees at slot 30")
}

func TestAttestToBlockHead_CrosslinkCommitteeRequestEmptyCommittee(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, m, finish := setup(t)
	defer finish()
	m.attesterClient.EXPECT().CrosslinkCommitteesAtSlot(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.CrosslinkCommitteeRequest{}),
	).Return(&pb.CrosslinkCommitteeResponse{
		Committee: []uint64{},
	}, nil)

	validator.AttestToBlockHead(context.Background(), 30)
	testutil.AssertLogsContain(t, hook, "Received an empty committee assignment")
}

func TestAttestToBlockHead_AttestationInfoAtSlotRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, m, finish := setup(t)
	defer finish()
	m.attesterClient.EXPECT().CrosslinkCommitteesAtSlot(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&pb.CrosslinkCommitteeResponse{
		Shard:     5,
		Committee: []uint64{1, 2, 3, 4},
	}, nil)
	m.attesterClient.EXPECT().AttestationInfoAtSlot(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.AttestationInfoRequest{}),
	).Return(nil /* Attestation Info Response*/, errors.New("something bad happened"))

	validator.AttestToBlockHead(context.Background(), 30)
	testutil.AssertLogsContain(t, hook, "Could not fetch necessary info to produce attestation at slot 30")
}

func TestAttestToBlockHead_ValidatorIndexRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, m, finish := setup(t)
	defer finish()
	m.attesterClient.EXPECT().CrosslinkCommitteesAtSlot(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.CrosslinkCommitteeRequest{}),
	).Return(&pb.CrosslinkCommitteeResponse{
		Shard:     5,
		Committee: []uint64{1, 2, 3, 4},
	}, nil)
	m.attesterClient.EXPECT().AttestationInfoAtSlot(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.AttestationInfoRequest{}),
	).Return(&pb.AttestationInfoResponse{
		BeaconBlockRootHash32:     []byte{},
		EpochBoundaryRootHash32:   []byte{},
		JustifiedBlockRootHash32:  []byte{},
		LatestCrosslinkRootHash32: []byte{},
		JustifiedEpoch:            0,
	}, nil)
	m.validatorClient.EXPECT().ValidatorIndex(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.ValidatorIndexRequest{}),
	).Return(nil /* Validator Index Response*/, errors.New("something bad happened"))

	validator.AttestToBlockHead(context.Background(), 30)
	testutil.AssertLogsContain(t, hook, "Could not fetch validator index")
}

func TestAttestToBlockHead_AttestHeadRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, m, finish := setup(t)
	defer finish()
	m.attesterClient.EXPECT().CrosslinkCommitteesAtSlot(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.CrosslinkCommitteeRequest{}),
	).Return(&pb.CrosslinkCommitteeResponse{
		Shard:     5,
		Committee: make([]uint64, 111),
	}, nil)
	m.attesterClient.EXPECT().AttestationInfoAtSlot(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.AttestationInfoRequest{}),
	).Return(&pb.AttestationInfoResponse{
		BeaconBlockRootHash32:     []byte{},
		EpochBoundaryRootHash32:   []byte{},
		JustifiedBlockRootHash32:  []byte{},
		LatestCrosslinkRootHash32: []byte{},
		JustifiedEpoch:            0,
	}, nil)
	m.validatorClient.EXPECT().ValidatorIndex(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.ValidatorIndexRequest{}),
	).Return(&pb.ValidatorIndexResponse{
		Index: 0,
	}, nil)
	m.attesterClient.EXPECT().AttestHead(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pbp2p.Attestation{}),
	).Return(nil, errors.New("something went wrong"))

	validator.AttestToBlockHead(context.Background(), 30)
	testutil.AssertLogsContain(t, hook, "Could not submit attestation to beacon node")
}

func TestAttestToBlockHead_AttestsCorrectly(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, m, finish := setup(t)
	defer finish()
	validatorIndex := uint64(5)
	committee := []uint64{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	m.attesterClient.EXPECT().CrosslinkCommitteesAtSlot(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.CrosslinkCommitteeRequest{}),
	).Return(&pb.CrosslinkCommitteeResponse{
		Shard:     5,
		Committee: committee,
	}, nil)
	m.attesterClient.EXPECT().AttestationInfoAtSlot(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.AttestationInfoRequest{}),
	).Return(&pb.AttestationInfoResponse{
		BeaconBlockRootHash32:     []byte("A"),
		EpochBoundaryRootHash32:   []byte("B"),
		JustifiedBlockRootHash32:  []byte("C"),
		LatestCrosslinkRootHash32: []byte("D"),
		JustifiedEpoch:            3,
	}, nil)
	m.validatorClient.EXPECT().ValidatorIndex(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.ValidatorIndexRequest{}),
	).Return(&pb.ValidatorIndexResponse{
		Index: uint64(validatorIndex),
	}, nil)

	var generatedAttestation *pbp2p.Attestation
	m.attesterClient.EXPECT().AttestHead(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pbp2p.Attestation{}),
	).Do(func(_ context.Context, att *pbp2p.Attestation) {
		generatedAttestation = att
	}).Return(&pb.AttestResponse{}, nil /* error */)

	validator.AttestToBlockHead(context.Background(), 30)

	aggregationBitfield := make([]byte, (len(committee)+7)/8)
	// Validator index is at index 4 in the mocked committee defined in this test.
	indexIntoCommittee := uint64(4)
	aggregationBitfield[indexIntoCommittee/8] |= 1 << (indexIntoCommittee % 8)
	expectedAttestation := &pbp2p.Attestation{
		Data: &pbp2p.AttestationData{
			Slot:                      30,
			Shard:                     5,
			BeaconBlockRootHash32:     []byte("A"),
			EpochBoundaryRootHash32:   []byte("B"),
			JustifiedBlockRootHash32:  []byte("C"),
			LatestCrosslinkRootHash32: []byte("D"),
			ShardBlockRootHash32:      params.BeaconConfig().ZeroHash[:],
			JustifiedEpoch:            3,
		},
		CustodyBitfield:     make([]byte, (len(committee)+7)/8),
		AggregationBitfield: aggregationBitfield,
		AggregateSignature:  []byte("signed"),
	}
	if !proto.Equal(generatedAttestation, expectedAttestation) {
		t.Errorf("Incorrectly attested head, wanted %v, received %v", expectedAttestation, generatedAttestation)
	}
	testutil.AssertLogsContain(t, hook, "Submitted attestation successfully")
}
