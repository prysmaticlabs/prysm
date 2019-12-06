package client

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/internal"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mocks struct {
	proposerClient   *internal.MockProposerServiceClient
	validatorClient  *internal.MockValidatorServiceClient
	attesterClient   *internal.MockAttesterServiceClient
	aggregatorClient *internal.MockAggregatorServiceClient
}

func setup(t *testing.T) (*validator, *mocks, func()) {
	ctrl := gomock.NewController(t)
	m := &mocks{
		proposerClient:   internal.NewMockProposerServiceClient(ctrl),
		validatorClient:  internal.NewMockValidatorServiceClient(ctrl),
		attesterClient:   internal.NewMockAttesterServiceClient(ctrl),
		aggregatorClient: internal.NewMockAggregatorServiceClient(ctrl),
	}
	validator := &validator{
		proposerClient:   m.proposerClient,
		attesterClient:   m.attesterClient,
		validatorClient:  m.validatorClient,
		aggregatorClient: m.aggregatorClient,
		keys:             keyMap,
	}

	return validator, m, ctrl.Finish
}

func TestProposeBlock_DoesNotProposeGenesisBlock(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, _, finish := setup(t)
	defer finish()
	validator.ProposeBlock(context.Background(), 0, validatorPubKey)

	testutil.AssertLogsContain(t, hook, "Assigned to genesis slot, skipping proposal")
}

func TestProposeBlock_DomainDataFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Failed to sign randao reveal")
}

func TestProposeBlock_RequestBlockFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&pb.DomainResponse{}, nil /*err*/)

	m.proposerClient.EXPECT().RequestBlock(
		gomock.Any(), // ctx
		gomock.Any(), // block request
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Failed to request block from beacon node")
}

func TestProposeBlock_ProposeBlockFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&pb.DomainResponse{}, nil /*err*/)

	m.proposerClient.EXPECT().RequestBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&pb.DomainResponse{}, nil /*err*/)

	m.proposerClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.BeaconBlock{}),
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Failed to propose block")
}

func TestProposeBlock_BroadcastsBlock(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&pb.DomainResponse{}, nil /*err*/)

	m.proposerClient.EXPECT().RequestBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&pb.DomainResponse{}, nil /*err*/)

	m.proposerClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.BeaconBlock{}),
	).Return(&pb.ProposeResponse{}, nil /*error*/)

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)
}
