package client

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/internal"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mocks struct {
	proposerClient  *internal.MockProposerServiceClient
	beaconClient    *internal.MockBeaconServiceClient
	validatorClient *internal.MockValidatorServiceClient
	attesterClient  *internal.MockAttesterServiceClient
}

func setup(t *testing.T) (*validator, *mocks, func()) {
	ctrl := gomock.NewController(t)
	m := &mocks{
		proposerClient:  internal.NewMockProposerServiceClient(ctrl),
		beaconClient:    internal.NewMockBeaconServiceClient(ctrl),
		validatorClient: internal.NewMockValidatorServiceClient(ctrl),
		attesterClient:  internal.NewMockAttesterServiceClient(ctrl),
	}
	validator := &validator{
		proposerClient:  m.proposerClient,
		beaconClient:    m.beaconClient,
		attesterClient:  m.attesterClient,
		validatorClient: m.validatorClient,
		keys:            keyMap,
	}

	return validator, m, ctrl.Finish
}

func TestProposeBlock_DoesNotProposeGenesisBlock(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, _, finish := setup(t)
	defer finish()
	validator.ProposeBlock(context.Background(), 0, hex.EncodeToString(validatorKey.PublicKey.Marshal()))

	testutil.AssertLogsContain(t, hook, "Assigned to genesis slot, skipping proposal")
}

func TestProposeBlock_LogsCanonicalHeadFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(nil /*beaconBlock*/, errors.New("something bad happened"))

	validator.ProposeBlock(context.Background(), 55, hex.EncodeToString(validatorKey.PublicKey.Marshal()))

	testutil.AssertLogsContain(t, hook, "something bad happened")
}

func TestProposeBlock_DomainDataFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 1, hex.EncodeToString(validatorKey.PublicKey.Marshal()))
	testutil.AssertLogsContain(t, hook, "Failed to get domain data from beacon node's state")
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

	validator.ProposeBlock(context.Background(), 1, hex.EncodeToString(validatorKey.PublicKey.Marshal()))
	testutil.AssertLogsContain(t, hook, "Failed to request block from beacon node")
}

func TestProposeBlock_ProposeBlockFailed(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&pb.DomainResponse{}, nil /*err*/)

	m.proposerClient.EXPECT().RequestBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&pbp2p.BeaconBlock{Body: &pbp2p.BeaconBlockBody{}}, nil /*err*/)

	m.proposerClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Return(&pb.ProposeResponse{}, nil /*error*/)

	validator.ProposeBlock(context.Background(), 1, hex.EncodeToString(validatorKey.PublicKey.Marshal()))
}

func TestProposeBlock_BroadcastsBlock(t *testing.T) {
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
	).Return(&pbp2p.BeaconBlock{Body: &pbp2p.BeaconBlockBody{}}, nil /*err*/)

	m.proposerClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 1, hex.EncodeToString(validatorKey.PublicKey.Marshal()))
	testutil.AssertLogsContain(t, hook, "Failed to request block from beacon node")
}
