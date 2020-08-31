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
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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
	require.NoError(t, err)
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

	require.LogsContain(t, hook, "Assigned to genesis slot, skipping proposal")
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
	require.LogsContain(t, hook, "Failed to sign randao reveal")
}

func TestProposeBlock_DomainDataIsNil(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(nil /*response*/, nil)

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)
	require.LogsContain(t, hook, domainDataErr)
}

func TestProposeBlock_RequestBlockFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(), // block request
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)
	require.LogsContain(t, hook, "Failed to request block from beacon node")
}

func TestProposeBlock_ProposeBlockFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(testutil.NewBeaconBlock().Block, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)
	require.LogsContain(t, hook, "Failed to propose block")
}

func TestProposeBlock_BlocksDoubleProposal(t *testing.T) {
	cfg := &featureconfig.Flags{
		LocalProtection: true,
	}
	reset := featureconfig.InitWithReset(cfg)
	defer reset()
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Times(2).Return(testutil.NewBeaconBlock().Block, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(&ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil /*error*/)

	slot := params.BeaconConfig().SlotsPerEpoch*5 + 2
	validator.ProposeBlock(context.Background(), slot, validatorPubKey)
	require.LogsDoNotContain(t, hook, failedPreBlockSignLocalErr)

	validator.ProposeBlock(context.Background(), slot, validatorPubKey)
	require.LogsContain(t, hook, failedPreBlockSignLocalErr)
}

func TestProposeBlock_BlocksDoubleProposal_After54KEpochs(t *testing.T) {
	cfg := &featureconfig.Flags{
		LocalProtection: true,
	}
	reset := featureconfig.InitWithReset(cfg)
	defer reset()
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Times(2).Return(testutil.NewBeaconBlock().Block, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(&ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil /*error*/)

	farFuture := (params.BeaconConfig().WeakSubjectivityPeriod + 9) * params.BeaconConfig().SlotsPerEpoch
	validator.ProposeBlock(context.Background(), farFuture, validatorPubKey)
	require.LogsDoNotContain(t, hook, failedPreBlockSignLocalErr)

	validator.ProposeBlock(context.Background(), farFuture, validatorPubKey)
	require.LogsContain(t, hook, failedPreBlockSignLocalErr)
}

func TestProposeBlock_AllowsPastProposals(t *testing.T) {
	cfg := &featureconfig.Flags{
		LocalProtection: true,
	}
	reset := featureconfig.InitWithReset(cfg)
	defer reset()
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	farAhead := (params.BeaconConfig().WeakSubjectivityPeriod + 9) * params.BeaconConfig().SlotsPerEpoch
	blk := testutil.NewBeaconBlock()
	blk.Block.Slot = farAhead
	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(blk.Block, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Times(2).Return(&ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil /*error*/)

	validator.ProposeBlock(context.Background(), farAhead, validatorPubKey)
	require.LogsDoNotContain(t, hook, failedPreBlockSignLocalErr)

	past := (params.BeaconConfig().WeakSubjectivityPeriod - 400) * params.BeaconConfig().SlotsPerEpoch
	blk2 := testutil.NewBeaconBlock()
	blk2.Block.Slot = past
	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(blk2.Block, nil /*err*/)
	validator.ProposeBlock(context.Background(), past, validatorPubKey)
	require.LogsDoNotContain(t, hook, failedPreBlockSignLocalErr)
}

func TestProposeBlock_AllowsSameEpoch(t *testing.T) {
	cfg := &featureconfig.Flags{
		LocalProtection: true,
	}
	reset := featureconfig.InitWithReset(cfg)
	defer reset()
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	farAhead := (params.BeaconConfig().WeakSubjectivityPeriod + 9) * params.BeaconConfig().SlotsPerEpoch
	blk := testutil.NewBeaconBlock()
	blk.Block.Slot = farAhead
	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(blk.Block, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Times(2).Return(&ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil /*error*/)

	pubKey := validatorPubKey
	validator.ProposeBlock(context.Background(), farAhead, pubKey)
	require.LogsDoNotContain(t, hook, failedPreBlockSignLocalErr)

	blk2 := testutil.NewBeaconBlock()
	blk2.Block.Slot = farAhead - 4
	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(blk2.Block, nil /*err*/)

	validator.ProposeBlock(context.Background(), farAhead-4, pubKey)
	require.LogsDoNotContain(t, hook, failedPreBlockSignLocalErr)
}

func TestProposeBlock_BroadcastsBlock(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(testutil.NewBeaconBlock().Block, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(&ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil /*error*/)

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)
}

func TestProposeBlock_BroadcastsBlock_WithGraffiti(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	validator.graffiti = []byte("12345678901234567890123456789012")

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	blk := testutil.NewBeaconBlock()
	blk.Block.Body.Graffiti = validator.graffiti
	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(blk.Block, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	var sentBlock *ethpb.SignedBeaconBlock

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).DoAndReturn(func(ctx context.Context, block *ethpb.SignedBeaconBlock) (*ethpb.ProposeResponse, error) {
		sentBlock = block
		return &ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil
	})

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)
	assert.Equal(t, string(validator.graffiti), string(sentBlock.Block.Body.Graffiti))
}

func TestProposeExit_DomainDataFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(nil /*response*/, errors.New("uh oh"))

	exit := &ethpb.VoluntaryExit{Epoch: 1, ValidatorIndex: 1}

	err := validator.ProposeExit(context.Background(), exit, validatorPubKey)
	assert.NotNil(t, err)
	assert.ErrorContains(t, domainDataErr, err)
	assert.ErrorContains(t, "uh oh", err)
	assert.LogsContain(t, hook, "Failed to sign voluntary exit")
}

func TestProposeExit_DomainDataIsNil(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(nil /*response*/, nil)

	exit := &ethpb.VoluntaryExit{Epoch: 1, ValidatorIndex: 1}

	err := validator.ProposeExit(context.Background(), exit, validatorPubKey)
	assert.NotNil(t, err)
	assert.ErrorContains(t, domainDataErr, err)
	assert.LogsContain(t, hook, "Failed to sign voluntary exit")
}

func TestProposeBlock_ProposeExitFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeExit(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedVoluntaryExit{}),
	).Return(nil /*response*/, errors.New("uh oh"))

	exit := &ethpb.VoluntaryExit{Epoch: 1, ValidatorIndex: 1}

	err := validator.ProposeExit(context.Background(), exit, validatorPubKey)
	assert.NotNil(t, err)
	assert.ErrorContains(t, "uh oh", err)
	assert.LogsContain(t, hook, "Failed to propose voluntary exit")
}

func TestProposeExit_BroadcastsBlock(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeExit(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedVoluntaryExit{}),
	).Return(&ethpb.ProposeExitResponse{}, nil /*error*/)

	exit := &ethpb.VoluntaryExit{Epoch: 1, ValidatorIndex: 1}

	assert.NoError(t, validator.ProposeExit(context.Background(), exit, validatorPubKey))
}
