package client

import (
	"context"
	"errors"
	"testing"

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
		gomock.Any(),
	).Return(nil /*Crosslinks Response*/, errors.New("something bad happened"))

	validator.AttestToBlockHead(context.Background(), 30)

	testutil.AssertLogsContain(t, hook, "Could not fetch crosslink committees at slot 30")
}

func TestAttestToBlockHead_AttestationInfoAtSlotRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, m, finish := setup(t)
	defer finish()
	m.attesterClient.EXPECT().CrosslinkCommitteesAtSlot(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&pb.CrosslinkCommitteeResponse{
		Shard: 5,
	}, nil)
	m.attesterClient.EXPECT().AttestationInfoAtSlot(
		gomock.Any(), // ctx
		gomock.Any(),
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
		gomock.Any(),
	).Return(&pb.CrosslinkCommitteeResponse{
		Shard: 5,
	}, nil)
	m.attesterClient.EXPECT().AttestationInfoAtSlot(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&pb.AttestationInfoResponse{
		BeaconBlockRootHash32:    []byte{},
		EpochBoundaryRootHash32:  []byte{},
		JustifiedBlockRootHash32: []byte{},
		JustifiedEpoch:           0,
	}, nil)
	m.validatorClient.EXPECT().ValidatorIndex(
		gomock.Any(), // ctx
		gomock.Any(),
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
		gomock.Any(),
	).Return(&pb.CrosslinkCommitteeResponse{
		Shard:     5,
		Committee: make([]uint64, 111),
	}, nil)
	m.attesterClient.EXPECT().AttestationInfoAtSlot(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&pb.AttestationInfoResponse{
		BeaconBlockRootHash32:    []byte{},
		EpochBoundaryRootHash32:  []byte{},
		JustifiedBlockRootHash32: []byte{},
		JustifiedEpoch:           0,
	}, nil)
	m.validatorClient.EXPECT().ValidatorIndex(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&pb.ValidatorIndexResponse{
		Index: 0,
	}, nil)
	m.attesterClient.EXPECT().AttestHead(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(nil, errors.New("something went wrong"))

	validator.AttestToBlockHead(context.Background(), 30)

	testutil.AssertLogsContain(t, hook, "Could not submit attestation to beacon node")
}

func TestAttestToBlockHead_AttestsCorrectly(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, m, finish := setup(t)
	defer finish()
	validatorIndex := 5
	m.attesterClient.EXPECT().CrosslinkCommitteesAtSlot(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&pb.CrosslinkCommitteeResponse{
		Shard:     5,
		Committee: []uint64{0, 3, 4, 2, 5, 6, 8, 9, 10},
	}, nil)
	m.attesterClient.EXPECT().AttestationInfoAtSlot(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&pb.AttestationInfoResponse{
		BeaconBlockRootHash32:    []byte{},
		EpochBoundaryRootHash32:  []byte{},
		JustifiedBlockRootHash32: []byte{},
		JustifiedEpoch:           0,
	}, nil)
	m.validatorClient.EXPECT().ValidatorIndex(
		gomock.Any(), // ctx
		gomock.Any(),
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
	if generatedAttestation.Data.Shard != 5 {
		t.Errorf(
			"Incorrect shard in attestation data, wanted %d, received %d",
			5,
			generatedAttestation.Data.Shard,
		)
	}
	if generatedAttestation.Data.Slot != 30 {
		t.Errorf(
			"Incorrect slot in attestation data, wanted %d, received %d",
			30,
			generatedAttestation.Data.Slot,
		)
	}
	testutil.AssertLogsContain(t, hook, "Submitted attestation successfully")
}
