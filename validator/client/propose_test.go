package client

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	lruwrpr "github.com/prysmaticlabs/prysm/v3/cache/lru"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	blocktest "github.com/prysmaticlabs/prysm/v3/consensus-types/blocks/testing"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/mock"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	testing2 "github.com/prysmaticlabs/prysm/v3/validator/db/testing"
	"github.com/prysmaticlabs/prysm/v3/validator/graffiti"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mocks struct {
	validatorClient *mock.MockBeaconNodeValidatorClient
	nodeClient      *mock.MockNodeClient
	slasherClient   *mock.MockSlasherClient
	signfunc        func(context.Context, *validatorpb.SignRequest) (bls.Signature, error)
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
		signfunc: func(ctx context.Context, req *validatorpb.SignRequest) (bls.Signature, error) {
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
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.AltairForkEpoch = 2
	cfg.BellatrixForkEpoch = 4
	params.OverrideBeaconConfig(cfg)

	tests := []struct {
		name string
		slot types.Slot
	}{
		{
			name: "phase 0",
			slot: 1,
		},
		{
			name: "altair",
			slot: params.BeaconConfig().SlotsPerEpoch.Mul(uint64(cfg.AltairForkEpoch)),
		},
		{
			name: "bellatrix",
			slot: params.BeaconConfig().SlotsPerEpoch.Mul(uint64(cfg.BellatrixForkEpoch)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := logTest.NewGlobal()
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
				gomock.AssignableToTypeOf(&ethpb.BlockRequest{}),
			).Return(nil /*response*/, errors.New("uh oh"))

			validator.ProposeBlock(context.Background(), tt.slot, pubKey)
			require.LogsContain(t, hook, "Failed to request block from beacon node")
		})
	}
}

func TestProposeBlock_ProposeBlockFailed(t *testing.T) {
	tests := []struct {
		name  string
		block *ethpb.GenericBeaconBlock
	}{
		{
			name: "phase0",
			block: &ethpb.GenericBeaconBlock{
				Block: &ethpb.GenericBeaconBlock_Phase0{
					Phase0: util.NewBeaconBlock().Block,
				},
			},
		},
		{
			name: "altair",
			block: &ethpb.GenericBeaconBlock{
				Block: &ethpb.GenericBeaconBlock_Altair{
					Altair: util.NewBeaconBlockAltair().Block,
				},
			},
		},
		{
			name: "bellatrix",
			block: &ethpb.GenericBeaconBlock{
				Block: &ethpb.GenericBeaconBlock_Bellatrix{
					Bellatrix: util.NewBeaconBlockBellatrix().Block,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := logTest.NewGlobal()
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
				gomock.AssignableToTypeOf(&ethpb.BlockRequest{}),
			).Return(tt.block, nil /*err*/)

			m.validatorClient.EXPECT().DomainData(
				gomock.Any(), // ctx
				gomock.Any(), // epoch
			).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

			m.validatorClient.EXPECT().ProposeBeaconBlock(
				gomock.Any(), // ctx
				gomock.AssignableToTypeOf(&ethpb.GenericSignedBeaconBlock{}),
			).Return(nil /*response*/, errors.New("uh oh"))

			validator.ProposeBlock(context.Background(), 1, pubKey)
			require.LogsContain(t, hook, "Failed to propose block")
		})
	}
}

func TestProposeBlock_BlocksDoubleProposal(t *testing.T) {
	slot := params.BeaconConfig().SlotsPerEpoch.Mul(5).Add(2)
	blockGraffiti := [32]byte{}
	copy(blockGraffiti[:], "someothergraffiti")

	tests := []struct {
		name   string
		blocks []*ethpb.GenericBeaconBlock
	}{
		{
			name: "phase0",
			blocks: func() []*ethpb.GenericBeaconBlock {
				block0, block1 := util.NewBeaconBlock(), util.NewBeaconBlock()
				block1.Block.Body.Graffiti = blockGraffiti[:]

				var blocks []*ethpb.GenericBeaconBlock
				for _, block := range []*ethpb.SignedBeaconBlock{block0, block1} {
					block.Block.Slot = slot
					blocks = append(blocks, &ethpb.GenericBeaconBlock{
						Block: &ethpb.GenericBeaconBlock_Phase0{
							Phase0: block.Block,
						},
					})
				}
				return blocks
			}(),
		},
		{
			name: "altair",
			blocks: func() []*ethpb.GenericBeaconBlock {
				block0, block1 := util.NewBeaconBlockAltair(), util.NewBeaconBlockAltair()
				block1.Block.Body.Graffiti = blockGraffiti[:]

				var blocks []*ethpb.GenericBeaconBlock
				for _, block := range []*ethpb.SignedBeaconBlockAltair{block0, block1} {
					block.Block.Slot = slot
					blocks = append(blocks, &ethpb.GenericBeaconBlock{
						Block: &ethpb.GenericBeaconBlock_Altair{
							Altair: block.Block,
						},
					})
				}
				return blocks
			}(),
		},
		{
			name: "bellatrix",
			blocks: func() []*ethpb.GenericBeaconBlock {
				block0, block1 := util.NewBeaconBlockBellatrix(), util.NewBeaconBlockBellatrix()
				block1.Block.Body.Graffiti = blockGraffiti[:]

				var blocks []*ethpb.GenericBeaconBlock
				for _, block := range []*ethpb.SignedBeaconBlockBellatrix{block0, block1} {
					block.Block.Slot = slot
					blocks = append(blocks, &ethpb.GenericBeaconBlock{
						Block: &ethpb.GenericBeaconBlock_Bellatrix{
							Bellatrix: block.Block,
						},
					})
				}
				return blocks
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			m.validatorClient.EXPECT().GetBeaconBlock(
				gomock.Any(), // ctx
				gomock.AssignableToTypeOf(&ethpb.BlockRequest{}),
			).Return(tt.blocks[0], nil /*err*/)

			m.validatorClient.EXPECT().GetBeaconBlock(
				gomock.Any(), // ctx
				gomock.AssignableToTypeOf(&ethpb.BlockRequest{}),
			).Return(tt.blocks[1], nil /*err*/)

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
		})
	}
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
	m.validatorClient.EXPECT().GetBeaconBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.BlockRequest{}),
	).Return(&ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Phase0{
			Phase0: testBlock.Block,
		},
	}, nil /*err*/)

	secondTestBlock := util.NewBeaconBlock()
	secondTestBlock.Block.Slot = farFuture
	blockGraffiti := [32]byte{}
	copy(blockGraffiti[:], "someothergraffiti")
	secondTestBlock.Block.Body.Graffiti = blockGraffiti[:]
	m.validatorClient.EXPECT().GetBeaconBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.BlockRequest{}),
	).Return(&ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Phase0{
			Phase0: secondTestBlock.Block,
		},
	}, nil /*err*/)
	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Times(3).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBeaconBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.GenericSignedBeaconBlock{}),
	).Return(&ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil /*error*/)

	validator.ProposeBlock(context.Background(), farFuture, pubKey)
	require.LogsDoNotContain(t, hook, failedBlockSignLocalErr)

	validator.ProposeBlock(context.Background(), farFuture, pubKey)
	require.LogsContain(t, hook, failedBlockSignLocalErr)
}

func TestProposeBlock_AllowsPastProposals(t *testing.T) {
	slot := params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().WeakSubjectivityPeriod + 9))

	tests := []struct {
		name     string
		pastSlot types.Slot
	}{
		{
			name:     "400 slots ago",
			pastSlot: slot.Sub(400),
		},
		{
			name:     "same epoch",
			pastSlot: slot.Sub(4),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			blk := util.NewBeaconBlock()
			blk.Block.Slot = slot
			m.validatorClient.EXPECT().GetBeaconBlock(
				gomock.Any(), // ctx
				gomock.AssignableToTypeOf(&ethpb.BlockRequest{}),
			).Return(&ethpb.GenericBeaconBlock{
				Block: &ethpb.GenericBeaconBlock_Phase0{
					Phase0: blk.Block,
				},
			}, nil /*err*/)

			m.validatorClient.EXPECT().DomainData(
				gomock.Any(), // ctx
				gomock.Any(), // epoch
			).Times(2).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

			m.validatorClient.EXPECT().ProposeBeaconBlock(
				gomock.Any(), // ctx
				gomock.AssignableToTypeOf(&ethpb.GenericSignedBeaconBlock{}),
			).Times(2).Return(&ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil /*error*/)

			validator.ProposeBlock(context.Background(), slot, pubKey)
			require.LogsDoNotContain(t, hook, failedBlockSignLocalErr)

			blk2 := util.NewBeaconBlock()
			blk2.Block.Slot = tt.pastSlot
			m.validatorClient.EXPECT().GetBeaconBlock(
				gomock.Any(), // ctx
				gomock.AssignableToTypeOf(&ethpb.BlockRequest{}),
			).Return(&ethpb.GenericBeaconBlock{
				Block: &ethpb.GenericBeaconBlock_Phase0{
					Phase0: blk2.Block,
				},
			}, nil /*err*/)
			validator.ProposeBlock(context.Background(), tt.pastSlot, pubKey)
			require.LogsDoNotContain(t, hook, failedBlockSignLocalErr)
		})
	}
}

func TestProposeBlock_BroadcastsBlock(t *testing.T) {
	testProposeBlock(t, make([]byte, 32) /*graffiti*/)
}

func TestProposeBlock_BroadcastsBlock_WithGraffiti(t *testing.T) {
	blockGraffiti := []byte("12345678901234567890123456789012")
	testProposeBlock(t, blockGraffiti)
}

func testProposeBlock(t *testing.T, graffiti []byte) {
	tests := []struct {
		name  string
		block *ethpb.GenericBeaconBlock
	}{
		{
			name: "phase0",
			block: &ethpb.GenericBeaconBlock{
				Block: &ethpb.GenericBeaconBlock_Phase0{
					Phase0: func() *ethpb.BeaconBlock {
						blk := util.NewBeaconBlock()
						blk.Block.Body.Graffiti = graffiti
						return blk.Block
					}(),
				},
			},
		},
		{
			name: "altair",
			block: &ethpb.GenericBeaconBlock{
				Block: &ethpb.GenericBeaconBlock_Altair{
					Altair: func() *ethpb.BeaconBlockAltair {
						blk := util.NewBeaconBlockAltair()
						blk.Block.Body.Graffiti = graffiti
						return blk.Block
					}(),
				},
			},
		},
		{
			name: "bellatrix",
			block: &ethpb.GenericBeaconBlock{
				Block: &ethpb.GenericBeaconBlock_Bellatrix{
					Bellatrix: func() *ethpb.BeaconBlockBellatrix {
						blk := util.NewBeaconBlockBellatrix()
						blk.Block.Body.Graffiti = graffiti
						return blk.Block
					}(),
				},
			},
		},
		{
			name: "bellatrix blind block",
			block: &ethpb.GenericBeaconBlock{
				Block: &ethpb.GenericBeaconBlock_BlindedBellatrix{
					BlindedBellatrix: func() *ethpb.BlindedBeaconBlockBellatrix {
						blk := util.NewBlindedBeaconBlockBellatrix()
						blk.Block.Body.Graffiti = graffiti
						return blk.Block
					}(),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := logTest.NewGlobal()
			validator, m, validatorKey, finish := setup(t)
			defer finish()
			pubKey := [fieldparams.BLSPubkeyLength]byte{}
			copy(pubKey[:], validatorKey.PublicKey().Marshal())

			validator.graffiti = graffiti

			m.validatorClient.EXPECT().DomainData(
				gomock.Any(), // ctx
				gomock.Any(), // epoch
			).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

			m.validatorClient.EXPECT().GetBeaconBlock(
				gomock.Any(), // ctx
				gomock.AssignableToTypeOf(&ethpb.BlockRequest{}),
			).DoAndReturn(func(ctx context.Context, req *ethpb.BlockRequest, arg2 ...grpc.CallOption) (*ethpb.GenericBeaconBlock, error) {
				assert.DeepEqual(t, graffiti, req.Graffiti, "Unexpected graffiti in request")

				return tt.block, nil
			})

			m.validatorClient.EXPECT().DomainData(
				gomock.Any(), // ctx
				gomock.Any(), // epoch
			).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

			var sentBlock interfaces.SignedBeaconBlock
			var err error

			m.validatorClient.EXPECT().ProposeBeaconBlock(
				gomock.Any(), // ctx
				gomock.AssignableToTypeOf(&ethpb.GenericSignedBeaconBlock{}),
			).DoAndReturn(func(ctx context.Context, block *ethpb.GenericSignedBeaconBlock, opts ...grpc.CallOption) (*ethpb.ProposeResponse, error) {
				sentBlock, err = blocktest.NewSignedBeaconBlockFromGeneric(block)
				assert.NoError(t, err, "Unexpected error unwrapping block")
				return &ethpb.ProposeResponse{BlockRoot: make([]byte, 32)}, nil
			})

			validator.ProposeBlock(context.Background(), 1, pubKey)
			g := sentBlock.Block().Body().Graffiti()
			assert.Equal(t, string(validator.graffiti), string(g[:]))
			require.LogsContain(t, hook, "Submitted new block")
		})
	}
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
		m.signfunc,
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
		m.signfunc,
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
		m.signfunc,
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
		m.signfunc,
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
		m.signfunc,
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
		m.signfunc,
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
	b, err := blocks.NewBeaconBlock(blk.Block)
	require.NoError(t, err)
	sig, blockRoot, err := validator.signBlock(ctx, pubKey, 0, 0, b)
	require.NoError(t, err, "%x,%v", sig, err)
	require.Equal(t, "a049e1dc723e5a8b5bd14f292973572dffd53785ddb337"+
		"82f20bf762cbe10ee7b9b4f5ae1ad6ff2089d352403750bed402b94b58469c072536"+
		"faa9a09a88beaff697404ca028b1c7052b0de37dbcff985dfa500459783370312bdd"+
		"36d6e0f224", hex.EncodeToString(sig))

	// Verify the returned block root matches the expected root using the proposer signature
	// domain.
	wantedBlockRoot, err := signing.ComputeSigningRoot(b, proposerDomain)
	if err != nil {
		require.NoError(t, err)
	}
	require.DeepEqual(t, wantedBlockRoot, blockRoot)
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
	wb, err := blocks.NewBeaconBlock(blk.Block)
	require.NoError(t, err)
	sig, blockRoot, err := validator.signBlock(ctx, pubKey, 0, 0, wb)
	require.NoError(t, err, "%x,%v", sig, err)
	// Verify the returned block root matches the expected root using the proposer signature
	// domain.
	wantedBlockRoot, err := signing.ComputeSigningRoot(wb, proposerDomain)
	if err != nil {
		require.NoError(t, err)
	}
	require.DeepEqual(t, wantedBlockRoot, blockRoot)
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
	blk := util.NewBeaconBlockBellatrix()
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
	wb, err := blocks.NewBeaconBlock(blk.Block)
	require.NoError(t, err)
	sig, blockRoot, err := validator.signBlock(ctx, pubKey, 0, 0, wb)
	require.NoError(t, err, "%x,%v", sig, err)
	// Verify the returned block root matches the expected root using the proposer signature
	// domain.
	wantedBlockRoot, err := signing.ComputeSigningRoot(wb, proposerDomain)
	if err != nil {
		require.NoError(t, err)
	}
	require.DeepEqual(t, wantedBlockRoot, blockRoot)
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
