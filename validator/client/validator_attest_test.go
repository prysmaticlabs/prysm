package client

import (
	"context"
	"errors"
	"github.com/golang/mock/gomock"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"testing"
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
		BeaconBlockRootHash32: []byte{},
		EpochBoundaryRootHash32: []byte{},
		JustifiedBlockRootHash32: []byte{},
		JustifiedEpoch: 0,
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
		Shard: 5,
		Committee: make([]uint64, 111),
	}, nil)
	m.attesterClient.EXPECT().AttestationInfoAtSlot(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&pb.AttestationInfoResponse{
		BeaconBlockRootHash32: []byte{},
		EpochBoundaryRootHash32: []byte{},
		JustifiedBlockRootHash32: []byte{},
		JustifiedEpoch: 0,
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
