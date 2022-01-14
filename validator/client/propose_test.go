package client

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/eth2-types"
	lruwrpr "github.com/prysmaticlabs/prysm/cache/lru"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/mock"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	testing2 "github.com/prysmaticlabs/prysm/validator/db/testing"
	"github.com/prysmaticlabs/prysm/validator/graffiti"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mocks struct {
	validatorClient *mock.MockBeaconNodeValidatorClient
	nodeClient      *mock.MockNodeClient
	slasherClient   *mock.MockSlasherClient
	signExitFunc    func(context.Context, *validatorpb.SignRequest) (bls.Signature, error)
}

type mockSignature struct{}

func (mockSignature) Verify(bls.PublicKey, []byte) bool {
	return true
}
func (mockSignature) AggregateVerify([]bls.PublicKey, [][32]byte) bool {
	return true
}
func (mockSignature) FastAggregateVerify([]bls.PublicKey, [32]byte) bool {
	return true
}
func (mockSignature) Eth2FastAggregateVerify([]bls.PublicKey, [32]byte) bool {
	return true
}
func (mockSignature) Marshal() []byte {
	return make([]byte, 32)
}
func (m mockSignature) Copy() bls.Signature {
	return m
}

func setup(t *testing.T) (*validator, *mocks, bls.SecretKey, func()) {
	validatorKey, err := bls.RandKey()
	require.NoError(t, err)
	return setupWithKey(t, validatorKey)
}

func setupWithKey(t *testing.T, validatorKey bls.SecretKey) (*validator, *mocks, bls.SecretKey, func()) {
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	valDB := testing2.SetupDB(t, [][fieldparams.BLSPubkeyLength]byte{pubKey})
	ctrl := gomock.NewController(t)
	m := &mocks{
		validatorClient: mock.NewMockBeaconNodeValidatorClient(ctrl),
		nodeClient:      mock.NewMockNodeClient(ctrl),
		slasherClient:   mock.NewMockSlasherClient(ctrl),
		signExitFunc: func(ctx context.Context, req *validatorpb.SignRequest) (bls.Signature, error) {
			return mockSignature{}, nil
		},
	}

	aggregatedSlotCommitteeIDCache := lruwrpr.New(int(params.BeaconConfig().MaxCommitteesPerSlot))
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	km := &mockKeymanager{
		keysMap: map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey{
			pubKey: validatorKey,
		},
	}
	validator := &validator{
		db:                             valDB,
		keyManager:                     km,
		validatorClient:                m.validatorClient,
		slashingProtectionClient:       m.slasherClient,
		graffiti:                       []byte{},
		attLogs:                        make(map[[32]byte]*attSubmitted),
		aggregatedSlotCommitteeIDCache: aggregatedSlotCommitteeIDCache,
	}

	return validator, m, validatorKey, ctrl.Finish
}

func TestProposeBlock_DoesNotProposeGenesisBlock(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, _, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	validator.ProposeBlock(context.Background(), 0, pubKey)

	require.LogsContain(t, hook, "Assigned to genesis slot, skipping proposal")
}

func TestProposeBlock_DomainDataFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 1, pubKey)
	require.LogsContain(t, hook, "Failed to sign randao reveal")
}

func TestProposeBlock_DomainDataIsNil(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(nil /*response*/, nil)

	validator.ProposeBlock(context.Background(), 1, pubKey)
	require.LogsContain(t, hook, domainDataErr)
}

func TestProposeBlock_RequestBlockFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(), // block request
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 1, pubKey)
	require.LogsContain(t, hook, "Failed to request block from beacon node")
}

func TestProposeBlockAltair_RequestBlockFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.AltairForkEpoch = 2
	params.OverrideBeaconConfig(cfg)
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().GetBeaconBlock(
		gomock.Any(), // ctx
		gomock.Any(), // block request
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 2*params.BeaconConfig().SlotsPerEpoch, pubKey)
	require.LogsContain(t, hook, "Failed to request block from beacon node")
}

func TestProposeBlockBellatrix_RequestBlockFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.BellatrixForkEpoch = 2
	params.OverrideBeaconConfig(cfg)
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().GetBeaconBlock(
		gomock.Any(), // ctx
		gomock.Any(), // block request
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 2*params.BeaconConfig().SlotsPerEpoch, pubKey)
	require.LogsContain(t, hook, "Failed to request block from beacon node")
}

func TestProposeBlock_ProposeBlockFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(util.NewBeaconBlock().Block, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 1, pubKey)
	require.LogsContain(t, hook, "Failed to propose block")
}

func TestProposeBlockAltair_ProposeBlockFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.AltairForkEpoch = 2
	params.OverrideBeaconConfig(cfg)
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().GetBeaconBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Altair{Altair: util.NewBeaconBlockAltair().Block},
	}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBeaconBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.GenericSignedBeaconBlock{}),
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 2*params.BeaconConfig().SlotsPerEpoch, pubKey)
	require.LogsContain(t, hook, "Failed to propose block")
}

func TestProposeBlockBellatrix_ProposeBlockFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.BellatrixForkEpoch = 2
	params.OverrideBeaconConfig(cfg)
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().GetBeaconBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Merge{Merge: util.NewBeaconBlockMerge().Block},
	}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBeaconBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.GenericSignedBeaconBlock{}),
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 2*params.BeaconConfig().SlotsPerEpoch, pubKey)
	require.LogsContain(t, hook, "Failed to propose block")
}

func TestProposeBlock_BlocksDoubleProposal(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	dummyRoot := [32]byte{}
	// Save a dummy proposal history at slot 0.
	err := validator.db.SaveProposalHistoryForSlot(context.Background(), pubKey, 0, dummyRoot[:])
	require.NoError(t, err)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Times(1).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	testBlock := util.NewBeaconBlock()
	slot := params.BeaconConfig().SlotsPerEpoch*5 + 2
	testBlock.Block.Slot = slot
	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(testBlock.Block, nil /*err*/)

	secondTestBlock := util.NewBeaconBlock()
	secondTestBlock.Block.Slot = slot
	graffiti := [32]byte{}
	copy(graffiti[:], "someothergraffiti")
	secondTestBlock.Block.Body.Graffiti = graffiti[:]
	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(secondTestBlock.Block, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Times(3).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(&ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil /*error*/)

	validator.ProposeBlock(context.Background(), slot, pubKey)
	require.LogsDoNotContain(t, hook, failedBlockSignLocalErr)

	validator.ProposeBlock(context.Background(), slot, pubKey)
	require.LogsContain(t, hook, failedBlockSignLocalErr)
}

func TestProposeBlockAltair_BlocksDoubleProposal(t *testing.T) {
	hook := logTest.NewGlobal()
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.AltairForkEpoch = 2
	params.OverrideBeaconConfig(cfg)
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	dummyRoot := [32]byte{}
	// Save a dummy proposal history at slot 0.
	err := validator.db.SaveProposalHistoryForSlot(context.Background(), pubKey, 0, dummyRoot[:])
	require.NoError(t, err)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Times(1).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	testBlock := util.NewBeaconBlockAltair()
	slot := params.BeaconConfig().SlotsPerEpoch*5 + 2
	testBlock.Block.Slot = slot
	m.validatorClient.EXPECT().GetBeaconBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Altair{Altair: testBlock.Block},
	}, nil /*err*/)

	secondTestBlock := util.NewBeaconBlockAltair()
	secondTestBlock.Block.Slot = slot
	graffiti := [32]byte{}
	copy(graffiti[:], "someothergraffiti")
	secondTestBlock.Block.Body.Graffiti = graffiti[:]
	m.validatorClient.EXPECT().GetBeaconBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Altair{Altair: secondTestBlock.Block},
	}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Times(3).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBeaconBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.GenericSignedBeaconBlock{}),
	).Return(&ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil /*error*/)

	validator.ProposeBlock(context.Background(), slot, pubKey)
	require.LogsDoNotContain(t, hook, failedBlockSignLocalErr)

	validator.ProposeBlock(context.Background(), slot, pubKey)
	require.LogsContain(t, hook, failedBlockSignLocalErr)
}

func TestProposeBlockBellatrix_BlocksDoubleProposal(t *testing.T) {
	hook := logTest.NewGlobal()
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.BellatrixForkEpoch = 2
	params.OverrideBeaconConfig(cfg)
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	dummyRoot := [32]byte{}
	// Save a dummy proposal history at slot 0.
	err := validator.db.SaveProposalHistoryForSlot(context.Background(), pubKey, 0, dummyRoot[:])
	require.NoError(t, err)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Times(1).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	testBlock := util.NewBeaconBlockMerge()
	slot := params.BeaconConfig().SlotsPerEpoch*5 + 2
	testBlock.Block.Slot = slot
	m.validatorClient.EXPECT().GetBeaconBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Merge{Merge: testBlock.Block},
	}, nil /*err*/)

	secondTestBlock := util.NewBeaconBlockMerge()
	secondTestBlock.Block.Slot = slot
	graffiti := [32]byte{}
	copy(graffiti[:], "someothergraffiti")
	secondTestBlock.Block.Body.Graffiti = graffiti[:]
	m.validatorClient.EXPECT().GetBeaconBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Merge{Merge: secondTestBlock.Block},
	}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Times(3).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBeaconBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.GenericSignedBeaconBlock{}),
	).Return(&ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil /*error*/)

	validator.ProposeBlock(context.Background(), slot, pubKey)
	require.LogsDoNotContain(t, hook, failedBlockSignLocalErr)

	validator.ProposeBlock(context.Background(), slot, pubKey)
	require.LogsContain(t, hook, failedBlockSignLocalErr)
}

func TestProposeBlock_BlocksDoubleProposal_After54KEpochs(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	dummyRoot := [32]byte{}
	// Save a dummy proposal history at slot 0.
	err := validator.db.SaveProposalHistoryForSlot(context.Background(), pubKey, 0, dummyRoot[:])
	require.NoError(t, err)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Times(1).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	testBlock := util.NewBeaconBlock()
	farFuture := params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().WeakSubjectivityPeriod + 9))
	testBlock.Block.Slot = farFuture
	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(testBlock.Block, nil /*err*/)

	secondTestBlock := util.NewBeaconBlock()
	secondTestBlock.Block.Slot = farFuture
	graffiti := [32]byte{}
	copy(graffiti[:], "someothergraffiti")
	secondTestBlock.Block.Body.Graffiti = graffiti[:]
	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(secondTestBlock.Block, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Times(3).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(&ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil /*error*/)

	validator.ProposeBlock(context.Background(), farFuture, pubKey)
	require.LogsDoNotContain(t, hook, failedBlockSignLocalErr)

	validator.ProposeBlock(context.Background(), farFuture, pubKey)
	require.LogsContain(t, hook, failedBlockSignLocalErr)
}

func TestProposeBlock_AllowsPastProposals(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	// Save a dummy proposal history at slot 0.
	err := validator.db.SaveProposalHistoryForSlot(context.Background(), pubKey, 0, []byte{})
	require.NoError(t, err)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Times(2).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	farAhead := params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().WeakSubjectivityPeriod + 9))
	blk := util.NewBeaconBlock()
	blk.Block.Slot = farAhead
	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(blk.Block, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Times(2).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Times(2).Return(&ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil /*error*/)

	validator.ProposeBlock(context.Background(), farAhead, pubKey)
	require.LogsDoNotContain(t, hook, failedBlockSignLocalErr)

	past := params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().WeakSubjectivityPeriod - 400))
	blk2 := util.NewBeaconBlock()
	blk2.Block.Slot = past
	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(blk2.Block, nil /*err*/)
	validator.ProposeBlock(context.Background(), past, pubKey)
	require.LogsDoNotContain(t, hook, failedBlockSignLocalErr)
}

func TestProposeBlock_AllowsSameEpoch(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	// Save a dummy proposal history at slot 0.
	err := validator.db.SaveProposalHistoryForSlot(context.Background(), pubKey, 0, []byte{})
	require.NoError(t, err)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Times(2).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	farAhead := params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().WeakSubjectivityPeriod + 9))
	blk := util.NewBeaconBlock()
	blk.Block.Slot = farAhead
	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(blk.Block, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Times(2).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Times(2).Return(&ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil /*error*/)

	validator.ProposeBlock(context.Background(), farAhead, pubKey)
	require.LogsDoNotContain(t, hook, failedBlockSignLocalErr)

	blk2 := util.NewBeaconBlock()
	blk2.Block.Slot = farAhead - 4
	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(blk2.Block, nil /*err*/)

	validator.ProposeBlock(context.Background(), farAhead-4, pubKey)
	require.LogsDoNotContain(t, hook, failedBlockSignLocalErr)
}

func TestProposeBlock_BroadcastsBlock(t *testing.T) {
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(util.NewBeaconBlock().Block, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(&ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil /*error*/)

	validator.ProposeBlock(context.Background(), 1, pubKey)
}

func TestProposeBlock_BroadcastsBlock_WithGraffiti(t *testing.T) {
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	validator.graffiti = []byte("12345678901234567890123456789012")

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	blk := util.NewBeaconBlock()
	blk.Block.Body.Graffiti = validator.graffiti
	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(blk.Block, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	var sentBlock *ethpb.SignedBeaconBlock

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).DoAndReturn(func(ctx context.Context, block *ethpb.SignedBeaconBlock, arg2 ...grpc.CallOption) (*ethpb.ProposeResponse, error) {
		sentBlock = block
		return &ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil
	})

	validator.ProposeBlock(context.Background(), 1, pubKey)
	assert.Equal(t, string(validator.graffiti), string(sentBlock.Block.Body.Graffiti))
}

func TestProposeBlockAltair_BroadcastsBlock_WithGraffiti(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.AltairForkEpoch = 2
	params.OverrideBeaconConfig(cfg)
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	validator.graffiti = []byte("12345678901234567890123456789012")

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	blk := util.NewBeaconBlockAltair()
	blk.Block.Body.Graffiti = validator.graffiti
	m.validatorClient.EXPECT().GetBeaconBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Altair{
			Altair: blk.Block,
		},
	}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	var sentBlock *ethpb.GenericSignedBeaconBlock

	m.validatorClient.EXPECT().ProposeBeaconBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.GenericSignedBeaconBlock{}),
	).DoAndReturn(func(ctx context.Context, block *ethpb.GenericSignedBeaconBlock, opts ...grpc.CallOption) (*ethpb.ProposeResponse, error) {
		sentBlock = block
		return &ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil
	})

	validator.ProposeBlock(context.Background(), 2*params.BeaconConfig().SlotsPerEpoch, pubKey)
	altairBlk := sentBlock.Block.(*ethpb.GenericSignedBeaconBlock_Altair).Altair.Block
	assert.Equal(t, string(validator.graffiti), string(altairBlk.Body.Graffiti))
}

func TestProposeBlockBellatrix_BroadcastsBlock_WithGraffiti(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.BellatrixForkEpoch = 2
	params.OverrideBeaconConfig(cfg)
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	validator.graffiti = []byte("12345678901234567890123456789012")

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	blk := util.NewBeaconBlockMerge()
	blk.Block.Body.Graffiti = validator.graffiti
	m.validatorClient.EXPECT().GetBeaconBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Merge{
			Merge: blk.Block,
		},
	}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	var sentBlock *ethpb.GenericSignedBeaconBlock

	m.validatorClient.EXPECT().ProposeBeaconBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.GenericSignedBeaconBlock{}),
	).DoAndReturn(func(ctx context.Context, block *ethpb.GenericSignedBeaconBlock, opts ...grpc.CallOption) (*ethpb.ProposeResponse, error) {
		sentBlock = block
		return &ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil
	})

	validator.ProposeBlock(context.Background(), 2*params.BeaconConfig().SlotsPerEpoch, pubKey)
	altairBlk := sentBlock.Block.(*ethpb.GenericSignedBeaconBlock_Merge).Merge.Block
	assert.Equal(t, string(validator.graffiti), string(altairBlk.Body.Graffiti))
}

func TestProposeExit_ValidatorIndexFailed(t *testing.T) {
	_, m, validatorKey, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().ValidatorIndex(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, errors.New("uh oh"))

	err := ProposeExit(
		context.Background(),
		m.validatorClient,
		m.nodeClient,
		m.signExitFunc,
		validatorKey.PublicKey().Marshal(),
	)
	assert.NotNil(t, err)
	assert.ErrorContains(t, "uh oh", err)
	assert.ErrorContains(t, "gRPC call to get validator index failed", err)
}

func TestProposeExit_GetGenesisFailed(t *testing.T) {
	_, m, validatorKey, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().
		ValidatorIndex(gomock.Any(), gomock.Any()).
		Return(nil, nil)

	m.nodeClient.EXPECT().
		GetGenesis(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("uh oh"))

	err := ProposeExit(
		context.Background(),
		m.validatorClient,
		m.nodeClient,
		m.signExitFunc,
		validatorKey.PublicKey().Marshal(),
	)
	assert.NotNil(t, err)
	assert.ErrorContains(t, "uh oh", err)
	assert.ErrorContains(t, "gRPC call to get genesis time failed", err)
}

func TestProposeExit_DomainDataFailed(t *testing.T) {
	_, m, validatorKey, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().
		ValidatorIndex(gomock.Any(), gomock.Any()).
		Return(&ethpb.ValidatorIndexResponse{Index: 1}, nil)

	// Any time in the past will suffice
	genesisTime := &timestamppb.Timestamp{
		Seconds: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	}

	m.nodeClient.EXPECT().
		GetGenesis(gomock.Any(), gomock.Any()).
		Return(&ethpb.Genesis{GenesisTime: genesisTime}, nil)

	m.validatorClient.EXPECT().
		DomainData(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("uh oh"))

	err := ProposeExit(
		context.Background(),
		m.validatorClient,
		m.nodeClient,
		m.signExitFunc,
		validatorKey.PublicKey().Marshal(),
	)
	assert.NotNil(t, err)
	assert.ErrorContains(t, domainDataErr, err)
	assert.ErrorContains(t, "uh oh", err)
	assert.ErrorContains(t, "failed to sign voluntary exit", err)
}

func TestProposeExit_DomainDataIsNil(t *testing.T) {
	_, m, validatorKey, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().
		ValidatorIndex(gomock.Any(), gomock.Any()).
		Return(&ethpb.ValidatorIndexResponse{Index: 1}, nil)

	// Any time in the past will suffice
	genesisTime := &timestamppb.Timestamp{
		Seconds: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	}

	m.nodeClient.EXPECT().
		GetGenesis(gomock.Any(), gomock.Any()).
		Return(&ethpb.Genesis{GenesisTime: genesisTime}, nil)

	m.validatorClient.EXPECT().
		DomainData(gomock.Any(), gomock.Any()).
		Return(nil, nil)

	err := ProposeExit(
		context.Background(),
		m.validatorClient,
		m.nodeClient,
		m.signExitFunc,
		validatorKey.PublicKey().Marshal(),
	)
	assert.NotNil(t, err)
	assert.ErrorContains(t, domainDataErr, err)
	assert.ErrorContains(t, "failed to sign voluntary exit", err)
}

func TestProposeBlock_ProposeExitFailed(t *testing.T) {
	_, m, validatorKey, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().
		ValidatorIndex(gomock.Any(), gomock.Any()).
		Return(&ethpb.ValidatorIndexResponse{Index: 1}, nil)

	// Any time in the past will suffice
	genesisTime := &timestamppb.Timestamp{
		Seconds: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	}

	m.nodeClient.EXPECT().
		GetGenesis(gomock.Any(), gomock.Any()).
		Return(&ethpb.Genesis{GenesisTime: genesisTime}, nil)

	m.validatorClient.EXPECT().
		DomainData(gomock.Any(), gomock.Any()).
		Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil)

	m.validatorClient.EXPECT().
		ProposeExit(gomock.Any(), gomock.AssignableToTypeOf(&ethpb.SignedVoluntaryExit{})).
		Return(nil, errors.New("uh oh"))

	err := ProposeExit(
		context.Background(),
		m.validatorClient,
		m.nodeClient,
		m.signExitFunc,
		validatorKey.PublicKey().Marshal(),
	)
	assert.NotNil(t, err)
	assert.ErrorContains(t, "uh oh", err)
	assert.ErrorContains(t, "failed to propose voluntary exit", err)
}

func TestProposeExit_BroadcastsBlock(t *testing.T) {
	_, m, validatorKey, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().
		ValidatorIndex(gomock.Any(), gomock.Any()).
		Return(&ethpb.ValidatorIndexResponse{Index: 1}, nil)

	// Any time in the past will suffice
	genesisTime := &timestamppb.Timestamp{
		Seconds: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	}

	m.nodeClient.EXPECT().
		GetGenesis(gomock.Any(), gomock.Any()).
		Return(&ethpb.Genesis{GenesisTime: genesisTime}, nil)

	m.validatorClient.EXPECT().
		DomainData(gomock.Any(), gomock.Any()).
		Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil)

	m.validatorClient.EXPECT().
		ProposeExit(gomock.Any(), gomock.AssignableToTypeOf(&ethpb.SignedVoluntaryExit{})).
		Return(&ethpb.ProposeExitResponse{}, nil)

	assert.NoError(t, ProposeExit(
		context.Background(),
		m.validatorClient,
		m.nodeClient,
		m.signExitFunc,
		validatorKey.PublicKey().Marshal(),
	))
}

func TestSignBlock(t *testing.T) {
	validator, m, _, finish := setup(t)
	defer finish()

	secretKey, err := bls.SecretKeyFromBytes(bytesutil.PadTo([]byte{1}, 32))
	require.NoError(t, err, "Failed to generate key from bytes")
	publicKey := secretKey.PublicKey()
	proposerDomain := make([]byte, 32)
	m.validatorClient.EXPECT().
		DomainData(gomock.Any(), gomock.Any()).
		Return(&ethpb.DomainResponse{SignatureDomain: proposerDomain}, nil)
	ctx := context.Background()
	blk := util.NewBeaconBlock()
	blk.Block.Slot = 1
	blk.Block.ProposerIndex = 100
	var pubKey [fieldparams.BLSPubkeyLength]byte
	copy(pubKey[:], publicKey.Marshal())
	km := &mockKeymanager{
		keysMap: map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey{
			pubKey: secretKey,
		},
	}
	validator.keyManager = km
	sig, domain, err := validator.signBlock(ctx, pubKey, 0, 0, wrapper.WrappedPhase0BeaconBlock(blk.Block))
	require.NoError(t, err, "%x,%x,%v", sig, domain.SignatureDomain, err)
	require.Equal(t, "a049e1dc723e5a8b5bd14f292973572dffd53785ddb337"+
		"82f20bf762cbe10ee7b9b4f5ae1ad6ff2089d352403750bed402b94b58469c072536"+
		"faa9a09a88beaff697404ca028b1c7052b0de37dbcff985dfa500459783370312bdd"+
		"36d6e0f224", hex.EncodeToString(sig))
	// proposer domain
	require.DeepEqual(t, proposerDomain, domain.SignatureDomain)
}

func TestSignAltairBlock(t *testing.T) {
	validator, m, _, finish := setup(t)
	defer finish()

	secretKey, err := bls.SecretKeyFromBytes(bytesutil.PadTo([]byte{1}, 32))
	require.NoError(t, err, "Failed to generate key from bytes")
	publicKey := secretKey.PublicKey()
	proposerDomain := make([]byte, 32)
	m.validatorClient.EXPECT().
		DomainData(gomock.Any(), gomock.Any()).
		Return(&ethpb.DomainResponse{SignatureDomain: proposerDomain}, nil)
	ctx := context.Background()
	blk := util.NewBeaconBlockAltair()
	blk.Block.Slot = 1
	blk.Block.ProposerIndex = 100
	var pubKey [fieldparams.BLSPubkeyLength]byte
	copy(pubKey[:], publicKey.Marshal())
	km := &mockKeymanager{
		keysMap: map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey{
			pubKey: secretKey,
		},
	}
	validator.keyManager = km
	wb, err := wrapper.WrappedAltairBeaconBlock(blk.Block)
	require.NoError(t, err)
	sig, domain, err := validator.signBlock(ctx, pubKey, 0, 0, wb)
	require.NoError(t, err, "%x,%x,%v", sig, domain.SignatureDomain, err)
	require.DeepEqual(t, proposerDomain, domain.SignatureDomain)
}

func TestSignBellatrixBlock(t *testing.T) {
	validator, m, _, finish := setup(t)
	defer finish()

	secretKey, err := bls.SecretKeyFromBytes(bytesutil.PadTo([]byte{1}, 32))
	require.NoError(t, err, "Failed to generate key from bytes")
	publicKey := secretKey.PublicKey()
	proposerDomain := make([]byte, 32)
	m.validatorClient.EXPECT().
		DomainData(gomock.Any(), gomock.Any()).
		Return(&ethpb.DomainResponse{SignatureDomain: proposerDomain}, nil)
	ctx := context.Background()
	blk := util.NewBeaconBlockMerge()
	blk.Block.Slot = 1
	blk.Block.ProposerIndex = 100
	var pubKey [fieldparams.BLSPubkeyLength]byte
	copy(pubKey[:], publicKey.Marshal())
	km := &mockKeymanager{
		keysMap: map[[fieldparams.BLSPubkeyLength]byte]bls.SecretKey{
			pubKey: secretKey,
		},
	}
	validator.keyManager = km
	wb, err := wrapper.WrappedMergeBeaconBlock(blk.Block)
	require.NoError(t, err)
	sig, domain, err := validator.signBlock(ctx, pubKey, 0, wb)
	require.NoError(t, err, "%x,%x,%v", sig, domain.SignatureDomain, err)
	require.DeepEqual(t, proposerDomain, domain.SignatureDomain)
}

func TestGetGraffiti_Ok(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := &mocks{
		validatorClient: mock.NewMockBeaconNodeValidatorClient(ctrl),
	}
	pubKey := [fieldparams.BLSPubkeyLength]byte{'a'}
	tests := []struct {
		name string
		v    *validator
		want []byte
	}{
		{name: "use default cli graffiti",
			v: &validator{
				graffiti: []byte{'b'},
				graffitiStruct: &graffiti.Graffiti{
					Default: "c",
					Random:  []string{"d", "e"},
					Specific: map[types.ValidatorIndex]string{
						1: "f",
						2: "g",
					},
				},
			},
			want: []byte{'b'},
		},
		{name: "use default file graffiti",
			v: &validator{
				validatorClient: m.validatorClient,
				graffitiStruct: &graffiti.Graffiti{
					Default: "c",
				},
			},
			want: []byte{'c'},
		},
		{name: "use random file graffiti",
			v: &validator{
				validatorClient: m.validatorClient,
				graffitiStruct: &graffiti.Graffiti{
					Random:  []string{"d"},
					Default: "c",
				},
			},
			want: []byte{'d'},
		},
		{name: "use validator file graffiti, has validator",
			v: &validator{
				validatorClient: m.validatorClient,
				graffitiStruct: &graffiti.Graffiti{
					Random:  []string{"d"},
					Default: "c",
					Specific: map[types.ValidatorIndex]string{
						1: "f",
						2: "g",
					},
				},
			},
			want: []byte{'g'},
		},
		{name: "use validator file graffiti, none specified",
			v: &validator{
				validatorClient: m.validatorClient,
				graffitiStruct:  &graffiti.Graffiti{},
			},
			want: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.name, "use default cli graffiti") {
				m.validatorClient.EXPECT().
					ValidatorIndex(gomock.Any(), &ethpb.ValidatorIndexRequest{PublicKey: pubKey[:]}).
					Return(&ethpb.ValidatorIndexResponse{Index: 2}, nil)
			}
			got, err := tt.v.getGraffiti(context.Background(), pubKey)
			require.NoError(t, err)
			require.DeepEqual(t, tt.want, got)
		})
	}
}

func TestGetGraffitiOrdered_Ok(t *testing.T) {
	pubKey := [fieldparams.BLSPubkeyLength]byte{'a'}
	valDB := testing2.SetupDB(t, [][fieldparams.BLSPubkeyLength]byte{pubKey})
	ctrl := gomock.NewController(t)
	m := &mocks{
		validatorClient: mock.NewMockBeaconNodeValidatorClient(ctrl),
	}
	m.validatorClient.EXPECT().
		ValidatorIndex(gomock.Any(), &ethpb.ValidatorIndexRequest{PublicKey: pubKey[:]}).
		Times(5).
		Return(&ethpb.ValidatorIndexResponse{Index: 2}, nil)

	v := &validator{
		db:              valDB,
		validatorClient: m.validatorClient,
		graffitiStruct: &graffiti.Graffiti{
			Ordered: []string{"a", "b", "c"},
			Default: "d",
		},
	}
	for _, want := range [][]byte{{'a'}, {'b'}, {'c'}, {'d'}, {'d'}} {
		got, err := v.getGraffiti(context.Background(), pubKey)
		require.NoError(t, err)
		require.DeepEqual(t, want, got)
	}
}
