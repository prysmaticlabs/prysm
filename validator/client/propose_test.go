package client

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	lru "github.com/hashicorp/golang-lru"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	testing2 "github.com/prysmaticlabs/prysm/validator/db/testing"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mocks struct {
	validatorClient *mock.MockBeaconNodeValidatorClient
}

func setup(t *testing.T) (*validator, *mocks, func()) {
	valDB := testing2.SetupDB(t, [][48]byte{validatorPubKey})
	ctrl := gomock.NewController(t)
	m := &mocks{
		validatorClient: mock.NewMockBeaconNodeValidatorClient(ctrl),
	}

	aggregatedSlotCommitteeIDCache, err := lru.New(int(params.BeaconConfig().MaxCommitteesPerSlot))
	if err != nil {
		t.Fatal(err)
	}
	cleanMap := make(map[uint64]uint64)
	cleanMap[0] = params.BeaconConfig().FarFutureEpoch
	clean := &slashpb.AttestationHistory{
		TargetToSource: cleanMap,
	}
	attHistoryByPubKey := make(map[[48]byte]*slashpb.AttestationHistory)
	attHistoryByPubKey[validatorPubKey] = clean

	validator := &validator{
		db:                             valDB,
		validatorClient:                m.validatorClient,
		keyManager:                     testKeyManager,
		graffiti:                       []byte{},
		attLogs:                        make(map[[32]byte]*attSubmitted),
		aggregatedSlotCommitteeIDCache: aggregatedSlotCommitteeIDCache,
		attesterHistoryByPubKey:        attHistoryByPubKey,
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
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
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
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Failed to propose block")
}

func TestProposeBlock_BlocksDoubleProposal(t *testing.T) {
	cfg := &featureconfig.Flags{
		ProtectProposer: true,
	}
	reset := featureconfig.InitWithReset(cfg)
	defer reset()
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Times(2).Return(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(&ethpb.ProposeResponse{}, nil /*error*/)

	slot := params.BeaconConfig().SlotsPerEpoch*5 + 2
	validator.ProposeBlock(context.Background(), slot, validatorPubKey)
	testutil.AssertLogsDoNotContain(t, hook, "Tried to sign a double proposal")

	validator.ProposeBlock(context.Background(), slot, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Tried to sign a double proposal")
}

func TestProposeBlock_BlocksDoubleProposal_After54KEpochs(t *testing.T) {
	cfg := &featureconfig.Flags{
		ProtectProposer: true,
	}
	reset := featureconfig.InitWithReset(cfg)
	defer reset()
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Times(2).Return(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(&ethpb.ProposeResponse{}, nil /*error*/)

	farFuture := (params.BeaconConfig().WeakSubjectivityPeriod + 9) * params.BeaconConfig().SlotsPerEpoch
	validator.ProposeBlock(context.Background(), farFuture, validatorPubKey)
	testutil.AssertLogsDoNotContain(t, hook, "Tried to sign a double proposal")

	validator.ProposeBlock(context.Background(), farFuture, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Tried to sign a double proposal")
}

func TestProposeBlock_AllowsPastProposals(t *testing.T) {
	cfg := &featureconfig.Flags{
		ProtectProposer: true,
	}
	reset := featureconfig.InitWithReset(cfg)
	defer reset()
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Times(2).Return(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Times(2).Return(&ethpb.ProposeResponse{}, nil /*error*/)

	farAhead := (params.BeaconConfig().WeakSubjectivityPeriod + 9) * params.BeaconConfig().SlotsPerEpoch
	validator.ProposeBlock(context.Background(), farAhead, validatorPubKey)
	testutil.AssertLogsDoNotContain(t, hook, "Tried to sign a double proposal")

	past := (params.BeaconConfig().WeakSubjectivityPeriod - 400) * params.BeaconConfig().SlotsPerEpoch
	validator.ProposeBlock(context.Background(), past, validatorPubKey)
	testutil.AssertLogsDoNotContain(t, hook, "Tried to sign a double proposal")
}

func TestProposeBlock_AllowsSameEpoch(t *testing.T) {
	cfg := &featureconfig.Flags{
		ProtectProposer: true,
	}
	reset := featureconfig.InitWithReset(cfg)
	defer reset()
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Times(2).Return(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Times(2).Return(&ethpb.ProposeResponse{}, nil /*error*/)

	pubKey := validatorPubKey
	farAhead := (params.BeaconConfig().WeakSubjectivityPeriod + 9) * params.BeaconConfig().SlotsPerEpoch
	validator.ProposeBlock(context.Background(), farAhead, pubKey)
	testutil.AssertLogsDoNotContain(t, hook, "Tried to sign a double proposal")

	validator.ProposeBlock(context.Background(), farAhead-4, pubKey)
	testutil.AssertLogsDoNotContain(t, hook, "Tried to sign a double proposal")
}

func TestProposeBlock_BroadcastsBlock(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(&ethpb.ProposeResponse{}, nil /*error*/)

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)
}

func TestProposeBlock_BroadcastsBlock_WithGraffiti(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	validator.graffiti = []byte("12345678901234567890123456789012")

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{Graffiti: validator.graffiti}}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	var sentBlock *ethpb.SignedBeaconBlock

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).DoAndReturn(func(ctx context.Context, block *ethpb.SignedBeaconBlock) (*ethpb.ProposeResponse, error) {
		sentBlock = block
		return &ethpb.ProposeResponse{}, nil
	})

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)

	if string(sentBlock.Block.Body.Graffiti) != string(validator.graffiti) {
		t.Errorf("Block was broadcast with the wrong graffiti field, wanted \"%v\", got \"%v\"", string(validator.graffiti), string(sentBlock.Block.Body.Graffiti))
	}
}
