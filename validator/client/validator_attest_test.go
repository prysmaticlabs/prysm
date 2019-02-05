package client

import (
	"context"
	"errors"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"testing"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/validator/internal"
	logTest "github.com/sirupsen/logrus/hooks/test"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"

)

type mocks struct {
	attesterClient *internal.MockAttesterServiceClient
	validatorClient   *internal.MockValidatorServiceClient
}

func setup(t *testing.T) (*validator, *mocks, func()) {
	ctrl := gomock.NewController(t)
	m := &mocks{
		attesterClient: internal.NewMockAttesterServiceClient(ctrl),
		validatorClient:   internal.NewMockValidatorServiceClient(ctrl),
	}

	validator := &validator{
		attesterClient:  m.attesterClient,
		validatorClient:    m.validatorClient,
	}

	return validator, m, ctrl.Finish
}

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

	validator.AttestToBlockHead(context.Background(), 30)

	testutil.AssertLogsContain(t, hook, "Could not fetch validator index")
}
