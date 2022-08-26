package validator

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	blockchainTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	builderTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/builder/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/altair"
	consensusblocks "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	prysmtime "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/time"
	dbTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	mockExecution "github.com/prysmaticlabs/prysm/v3/beacon-chain/execution/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/synccommittee"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
	v3 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v3"
	mockSync "github.com/prysmaticlabs/prysm/v3/beacon-chain/sync/initial-sync/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestServer_buildHeaderBlock(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, keys := util.DeterministicGenesisStateAltair(t, 16384)
	sCom, err := altair.NextSyncCommittee(context.Background(), beaconState)
	require.NoError(t, err)
	require.NoError(t, beaconState.SetCurrentSyncCommittee(sCom))
	copiedState := beaconState.Copy()

	proposerServer := &Server{
		BeaconDB: db,
		StateGen: stategen.New(db),
	}
	b, err := util.GenerateFullBlockAltair(copiedState, keys, util.DefaultBlockGenConfig(), 1)
	require.NoError(t, err)
	r := bytesutil.ToBytes32(b.Block.ParentRoot)
	util.SaveBlock(t, ctx, proposerServer.BeaconDB, b)
	require.NoError(t, proposerServer.BeaconDB.SaveState(ctx, beaconState, r))

	b1, err := util.GenerateFullBlockAltair(copiedState, keys, util.DefaultBlockGenConfig(), 2)
	require.NoError(t, err)

	vs := &Server{StateGen: stategen.New(db), BeaconDB: db}
	h := &v1.ExecutionPayloadHeader{
		BlockNumber:      123,
		GasLimit:         456,
		GasUsed:          789,
		ParentHash:       make([]byte, fieldparams.RootLength),
		FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:        make([]byte, fieldparams.RootLength),
		ReceiptsRoot:     make([]byte, fieldparams.RootLength),
		LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:       make([]byte, fieldparams.RootLength),
		BaseFeePerGas:    make([]byte, fieldparams.RootLength),
		BlockHash:        make([]byte, fieldparams.RootLength),
		TransactionsRoot: make([]byte, fieldparams.RootLength),
		ExtraData:        make([]byte, 0),
	}
	got, err := vs.buildBlindBlock(ctx, b1.Block, h)
	require.NoError(t, err)
	require.DeepEqual(t, h, got.GetBlindedBellatrix().Body.ExecutionPayloadHeader)

	_, err = vs.buildBlindBlock(ctx, nil, h)
	require.ErrorContains(t, "nil block", err)

	_, err = vs.buildBlindBlock(ctx, b1.Block, nil)
	require.ErrorContains(t, "nil header", err)
}

func TestServer_getPayloadHeader(t *testing.T) {
	emptyRoot, err := ssz.TransactionsRoot([][]byte{})
	require.NoError(t, err)
	ti, err := slots.ToTime(uint64(time.Now().Unix()), 0)
	require.NoError(t, err)

	sk, err := bls.RandKey()
	require.NoError(t, err)
	bid := &ethpb.BuilderBid{
		Header: &v1.ExecutionPayloadHeader{
			FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
			StateRoot:        make([]byte, fieldparams.RootLength),
			ReceiptsRoot:     make([]byte, fieldparams.RootLength),
			LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
			PrevRandao:       make([]byte, fieldparams.RootLength),
			BaseFeePerGas:    make([]byte, fieldparams.RootLength),
			BlockHash:        make([]byte, fieldparams.RootLength),
			TransactionsRoot: bytesutil.PadTo([]byte{1}, fieldparams.RootLength),
			ParentHash:       params.BeaconConfig().ZeroHash[:],
			Timestamp:        uint64(ti.Unix()),
		},
		Pubkey: sk.PublicKey().Marshal(),
		Value:  bytesutil.PadTo([]byte{1, 2, 3}, 32),
	}
	d := params.BeaconConfig().DomainApplicationBuilder
	domain, err := signing.ComputeDomain(d, nil, nil)
	require.NoError(t, err)
	sr, err := signing.ComputeSigningRoot(bid, domain)
	require.NoError(t, err)
	sBid := &ethpb.SignedBuilderBid{
		Message:   bid,
		Signature: sk.Sign(sr[:]).Marshal(),
	}

	require.NoError(t, err)
	tests := []struct {
		name           string
		head           interfaces.SignedBeaconBlock
		mock           *builderTest.MockBuilderService
		fetcher        *blockchainTest.ChainService
		err            string
		returnedHeader *v1.ExecutionPayloadHeader
	}{
		{
			name: "head is not bellatrix ready",
			mock: &builderTest.MockBuilderService{},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.SignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
					require.NoError(t, err)
					return wb
				}(),
			},
		},
		{
			name: "get header failed",
			mock: &builderTest.MockBuilderService{
				ErrGetHeader: errors.New("can't get header"),
			},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.SignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
					require.NoError(t, err)
					return wb
				}(),
			},
			err: "can't get header",
		},
		{
			name: "0 bid",
			mock: &builderTest.MockBuilderService{
				Bid: &ethpb.SignedBuilderBid{
					Message: &ethpb.BuilderBid{
						Header: &v1.ExecutionPayloadHeader{
							BlockNumber: 123,
						},
					},
				},
			},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.SignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
					require.NoError(t, err)
					return wb
				}(),
			},
			err: "builder returned header with 0 bid amount",
		},
		{
			name: "invalid tx root",
			mock: &builderTest.MockBuilderService{
				Bid: &ethpb.SignedBuilderBid{
					Message: &ethpb.BuilderBid{
						Value: []byte{1},
						Header: &v1.ExecutionPayloadHeader{
							BlockNumber:      123,
							TransactionsRoot: emptyRoot[:],
						},
					},
				},
			},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.SignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
					require.NoError(t, err)
					return wb
				}(),
			},
			err: "builder returned header with an empty tx root",
		},
		{
			name: "can get header",
			mock: &builderTest.MockBuilderService{
				Bid: sBid,
			},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.SignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
					require.NoError(t, err)
					return wb
				}(),
			},
			returnedHeader: bid.Header,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vs := &Server{BlockBuilder: tc.mock, HeadFetcher: tc.fetcher, TimeFetcher: &blockchainTest.ChainService{
				Genesis: time.Now(),
			}}
			h, err := vs.getPayloadHeaderFromBuilder(context.Background(), 0, 0)
			if tc.err != "" {
				require.ErrorContains(t, tc.err, err)
			} else {
				require.NoError(t, err)
				require.DeepEqual(t, tc.returnedHeader, h)
			}
		})
	}
}

func TestServer_getBuilderBlock(t *testing.T) {
	p := emptyPayload()
	p.GasLimit = 123

	tests := []struct {
		name        string
		blk         interfaces.SignedBeaconBlock
		mock        *builderTest.MockBuilderService
		err         string
		returnedBlk interfaces.SignedBeaconBlock
	}{
		{
			name: "nil block",
			blk:  nil,
			err:  "signed beacon block can't be nil",
		},
		{
			name: "old block version",
			blk: func() interfaces.SignedBeaconBlock {
				wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
				require.NoError(t, err)
				return wb
			}(),
			returnedBlk: func() interfaces.SignedBeaconBlock {
				wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
				require.NoError(t, err)
				return wb
			}(),
		},
		{
			name: "not configured",
			blk: func() interfaces.SignedBeaconBlock {
				wb, err := blocks.NewSignedBeaconBlock(util.NewBlindedBeaconBlockBellatrix())
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				HasConfigured: false,
			},
			returnedBlk: func() interfaces.SignedBeaconBlock {
				wb, err := blocks.NewSignedBeaconBlock(util.NewBlindedBeaconBlockBellatrix())
				require.NoError(t, err)
				return wb
			}(),
		},
		{
			name: "submit blind block error",
			blk: func() interfaces.SignedBeaconBlock {
				b := util.NewBlindedBeaconBlockBellatrix()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				HasConfigured:         true,
				ErrSubmitBlindedBlock: errors.New("can't submit"),
			},
			err: "can't submit",
		},
		{
			name: "head and payload root mismatch",
			blk: func() interfaces.SignedBeaconBlock {
				b := util.NewBlindedBeaconBlockBellatrix()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				HasConfigured: true,
				Payload:       p,
			},
			returnedBlk: func() interfaces.SignedBeaconBlock {
				b := util.NewBeaconBlockBellatrix()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				b.Block.Body.ExecutionPayload = p
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			err: "header and payload root do not match",
		},
		{
			name: "can get payload",
			blk: func() interfaces.SignedBeaconBlock {
				b := util.NewBlindedBeaconBlockBellatrix()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				txRoot, err := ssz.TransactionsRoot([][]byte{})
				require.NoError(t, err)
				b.Block.Body.ExecutionPayloadHeader = &v1.ExecutionPayloadHeader{
					ParentHash:       make([]byte, fieldparams.RootLength),
					FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
					StateRoot:        make([]byte, fieldparams.RootLength),
					ReceiptsRoot:     make([]byte, fieldparams.RootLength),
					LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
					PrevRandao:       make([]byte, fieldparams.RootLength),
					BaseFeePerGas:    make([]byte, fieldparams.RootLength),
					BlockHash:        make([]byte, fieldparams.RootLength),
					TransactionsRoot: txRoot[:],
					GasLimit:         123,
				}
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				HasConfigured: true,
				Payload:       p,
			},
			returnedBlk: func() interfaces.SignedBeaconBlock {
				b := util.NewBeaconBlockBellatrix()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				b.Block.Body.ExecutionPayload = p
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vs := &Server{BlockBuilder: tc.mock}
			gotBlk, err := vs.unblindBuilderBlock(context.Background(), tc.blk)
			if tc.err != "" {
				require.ErrorContains(t, tc.err, err)
			} else {
				require.NoError(t, err)
				require.DeepEqual(t, tc.returnedBlk, gotBlk)
			}
		})
	}
}

func TestServer_readyForBuilder(t *testing.T) {
	ctx := context.Background()
	vs := &Server{BeaconDB: dbTest.SetupDB(t)}
	cs := &blockchainTest.ChainService{FinalizedCheckPoint: &ethpb.Checkpoint{}} // Checkpoint root is zeros.
	vs.FinalizationFetcher = cs
	ready, err := vs.readyForBuilder(ctx)
	require.NoError(t, err)
	require.Equal(t, false, ready)

	b := util.NewBeaconBlockBellatrix()
	wb, err := blocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	wbr, err := wb.Block().HashTreeRoot()
	require.NoError(t, err)

	b1 := util.NewBeaconBlockBellatrix()
	b1.Block.Body.ExecutionPayload.BlockNumber = 1 // Execution enabled.
	wb1, err := blocks.NewSignedBeaconBlock(b1)
	require.NoError(t, err)
	wbr1, err := wb1.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, vs.BeaconDB.SaveBlock(ctx, wb))
	require.NoError(t, vs.BeaconDB.SaveBlock(ctx, wb1))

	// Ready is false given finalized block does not have execution.
	cs = &blockchainTest.ChainService{FinalizedCheckPoint: &ethpb.Checkpoint{Root: wbr[:]}}
	vs.FinalizationFetcher = cs
	ready, err = vs.readyForBuilder(ctx)
	require.NoError(t, err)
	require.Equal(t, false, ready)

	// Ready is true given finalized block has execution.
	cs = &blockchainTest.ChainService{FinalizedCheckPoint: &ethpb.Checkpoint{Root: wbr1[:]}}
	vs.FinalizationFetcher = cs
	ready, err = vs.readyForBuilder(ctx)
	require.NoError(t, err)
	require.Equal(t, true, ready)
}

func TestServer_getAndBuildHeaderBlock(t *testing.T) {
	ctx := context.Background()
	vs := &Server{}

	// Nil builder
	ready, _, err := vs.GetAndBuildBlindBlock(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, false, ready)

	// Not configured
	vs.BlockBuilder = &builderTest.MockBuilderService{}
	ready, _, err = vs.GetAndBuildBlindBlock(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, false, ready)

	// Block is not ready
	vs.BlockBuilder = &builderTest.MockBuilderService{HasConfigured: true}
	vs.FinalizationFetcher = &blockchainTest.ChainService{FinalizedCheckPoint: &ethpb.Checkpoint{}}
	ready, _, err = vs.GetAndBuildBlindBlock(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, false, ready)

	// Failed to get header
	b1 := util.NewBeaconBlockBellatrix()
	b1.Block.Body.ExecutionPayload.BlockNumber = 1 // Execution enabled.
	wb1, err := blocks.NewSignedBeaconBlock(b1)
	require.NoError(t, err)
	wbr1, err := wb1.Block().HashTreeRoot()
	require.NoError(t, err)
	vs.BeaconDB = dbTest.SetupDB(t)
	require.NoError(t, vs.BeaconDB.SaveBlock(ctx, wb1))
	vs.FinalizationFetcher = &blockchainTest.ChainService{FinalizedCheckPoint: &ethpb.Checkpoint{Root: wbr1[:]}}
	vs.HeadFetcher = &blockchainTest.ChainService{Block: wb1}
	vs.BlockBuilder = &builderTest.MockBuilderService{HasConfigured: true, ErrGetHeader: errors.New("could not get payload")}
	vs.ForkFetcher = &blockchainTest.ChainService{ForkChoiceStore: protoarray.New()}
	ready, _, err = vs.GetAndBuildBlindBlock(ctx, &ethpb.BeaconBlockAltair{})
	require.ErrorContains(t, "could not get payload", err)
	require.Equal(t, false, ready)

	// Block built and validated!
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, keys := util.DeterministicGenesisStateAltair(t, 16384)
	sCom, err := altair.NextSyncCommittee(context.Background(), beaconState)
	require.NoError(t, err)
	require.NoError(t, beaconState.SetCurrentSyncCommittee(sCom))
	copiedState := beaconState.Copy()

	b, err := util.GenerateFullBlockAltair(copiedState, keys, util.DefaultBlockGenConfig(), 1)
	require.NoError(t, err)
	r := bytesutil.ToBytes32(b.Block.ParentRoot)
	util.SaveBlock(t, ctx, vs.BeaconDB, b)
	require.NoError(t, vs.BeaconDB.SaveState(ctx, beaconState, r))

	altairBlk, err := util.GenerateFullBlockAltair(copiedState, keys, util.DefaultBlockGenConfig(), 2)
	require.NoError(t, err)

	ts := uint64(time.Now().Unix()) + uint64(altairBlk.Block.Slot)*params.BeaconConfig().SecondsPerSlot
	h := &v1.ExecutionPayloadHeader{
		BlockNumber:      123,
		GasLimit:         456,
		GasUsed:          789,
		ParentHash:       make([]byte, fieldparams.RootLength),
		FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:        make([]byte, fieldparams.RootLength),
		ReceiptsRoot:     make([]byte, fieldparams.RootLength),
		LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:       make([]byte, fieldparams.RootLength),
		BaseFeePerGas:    make([]byte, fieldparams.RootLength),
		BlockHash:        make([]byte, fieldparams.RootLength),
		TransactionsRoot: make([]byte, fieldparams.RootLength),
		Timestamp:        ts,
	}

	vs.StateGen = stategen.New(vs.BeaconDB)
	vs.GenesisFetcher = &blockchainTest.ChainService{}
	vs.ForkFetcher = &blockchainTest.ChainService{Fork: &ethpb.Fork{}}

	sk, err := bls.RandKey()
	require.NoError(t, err)
	bid := &ethpb.BuilderBid{
		Header: &v1.ExecutionPayloadHeader{
			BlockNumber:      123,
			GasLimit:         456,
			GasUsed:          789,
			ParentHash:       make([]byte, fieldparams.RootLength),
			FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
			StateRoot:        make([]byte, fieldparams.RootLength),
			ReceiptsRoot:     make([]byte, fieldparams.RootLength),
			LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
			PrevRandao:       make([]byte, fieldparams.RootLength),
			BaseFeePerGas:    make([]byte, fieldparams.RootLength),
			BlockHash:        make([]byte, fieldparams.RootLength),
			TransactionsRoot: make([]byte, fieldparams.RootLength),
			Timestamp:        ts,
		},
		Pubkey: sk.PublicKey().Marshal(),
		Value:  bytesutil.PadTo([]byte{1, 2, 3}, 32),
	}
	d := params.BeaconConfig().DomainApplicationBuilder
	domain, err := signing.ComputeDomain(d, nil, nil)
	require.NoError(t, err)
	sr, err := signing.ComputeSigningRoot(bid, domain)
	require.NoError(t, err)
	sBid := &ethpb.SignedBuilderBid{
		Message:   bid,
		Signature: sk.Sign(sr[:]).Marshal(),
	}
	vs.BlockBuilder = &builderTest.MockBuilderService{HasConfigured: true, Bid: sBid}
	vs.TimeFetcher = &blockchainTest.ChainService{Genesis: time.Now()}
	vs.ForkFetcher = &blockchainTest.ChainService{ForkChoiceStore: protoarray.New()}
	ready, builtBlk, err := vs.GetAndBuildBlindBlock(ctx, altairBlk.Block)
	require.NoError(t, err)
	require.Equal(t, true, ready)
	require.DeepEqual(t, h, builtBlk.GetBlindedBellatrix().Body.ExecutionPayloadHeader)
}

func TestServer_GetBellatrixBeaconBlock_HappyCase(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()
	hook := logTest.NewGlobal()

	terminalBlockHash := bytesutil.PadTo([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, 32)
	params.SetupTestConfigCleanup(t)
	cfg := params.MainnetConfig().Copy()
	cfg.BellatrixForkEpoch = 2
	cfg.AltairForkEpoch = 1
	cfg.TerminalBlockHash = common.BytesToHash(terminalBlockHash)
	cfg.TerminalBlockHashActivationEpoch = 2
	params.OverrideBeaconConfig(cfg)

	beaconState, privKeys := util.DeterministicGenesisState(t, 64)
	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	genesis := consensusblocks.NewGenesisBlock(stateRoot[:])
	wsb, err := blocks.NewSignedBeaconBlock(genesis)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wsb), "Could not save genesis block")

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	bellatrixSlot, err := slots.EpochStart(params.BeaconConfig().BellatrixForkEpoch)
	require.NoError(t, err)

	emptyPayload := &v1.ExecutionPayload{
		ParentHash:    make([]byte, fieldparams.RootLength),
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:    make([]byte, fieldparams.RootLength),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
	}
	blk := &ethpb.SignedBeaconBlockBellatrix{
		Block: &ethpb.BeaconBlockBellatrix{
			Slot:       bellatrixSlot + 1,
			ParentRoot: parentRoot[:],
			StateRoot:  genesis.Block.StateRoot,
			Body: &ethpb.BeaconBlockBodyBellatrix{
				RandaoReveal:     genesis.Block.Body.RandaoReveal,
				Graffiti:         genesis.Block.Body.Graffiti,
				Eth1Data:         genesis.Block.Body.Eth1Data,
				SyncAggregate:    &ethpb.SyncAggregate{SyncCommitteeBits: bitfield.NewBitvector512(), SyncCommitteeSignature: make([]byte, 96)},
				ExecutionPayload: emptyPayload,
			},
		},
		Signature: genesis.Signature,
	}

	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, blkRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, blkRoot), "Could not save genesis state")

	proposerServer := &Server{
		HeadFetcher:       &blockchainTest.ChainService{State: beaconState, Root: parentRoot[:], Optimistic: false},
		TimeFetcher:       &blockchainTest.ChainService{Genesis: time.Now()},
		SyncChecker:       &mockSync.Sync{IsSyncing: false},
		BlockReceiver:     &blockchainTest.ChainService{},
		HeadUpdater:       &blockchainTest.ChainService{},
		ChainStartFetcher: &mockExecution.Chain{},
		Eth1InfoFetcher:   &mockExecution.Chain{},
		MockEth1Votes:     true,
		AttPool:           attestations.NewPool(),
		SlashingsPool:     slashings.NewPool(),
		ExitPool:          voluntaryexits.NewPool(),
		StateGen:          stategen.New(db),
		SyncCommitteePool: synccommittee.NewStore(),
		ExecutionEngineCaller: &mockExecution.EngineClient{
			PayloadIDBytes:   &v1.PayloadIDBytes{1},
			ExecutionPayload: emptyPayload,
		},
		BeaconDB:               db,
		ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
		BlockBuilder:           &builderTest.MockBuilderService{},
	}
	proposerServer.ProposerSlotIndexCache.SetProposerAndPayloadIDs(65, 40, [8]byte{'a'}, parentRoot)

	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)

	block, err := proposerServer.getBellatrixBeaconBlock(ctx, &ethpb.BlockRequest{
		Slot:         bellatrixSlot + 1,
		RandaoReveal: randaoReveal,
	})
	require.NoError(t, err)
	bellatrixBlk, ok := block.GetBlock().(*ethpb.GenericBeaconBlock_Bellatrix)
	require.Equal(t, true, ok)
	require.LogsContain(t, hook, "Computed state root")
	require.DeepEqual(t, emptyPayload, bellatrixBlk.Bellatrix.Body.ExecutionPayload) // Payload should equal.
}

func TestServer_GetBellatrixBeaconBlock_BuilderCase(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()
	hook := logTest.NewGlobal()

	params.SetupTestConfigCleanup(t)
	cfg := params.MainnetConfig().Copy()
	cfg.BellatrixForkEpoch = 2
	cfg.AltairForkEpoch = 1
	params.OverrideBeaconConfig(cfg)

	beaconState, privKeys := util.DeterministicGenesisState(t, 64)
	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	genesis := consensusblocks.NewGenesisBlock(stateRoot[:])
	wsb, err := blocks.NewSignedBeaconBlock(genesis)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wsb), "Could not save genesis block")

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	bellatrixSlot, err := slots.EpochStart(params.BeaconConfig().BellatrixForkEpoch)
	require.NoError(t, err)

	emptyPayload := &v1.ExecutionPayload{
		ParentHash:    make([]byte, fieldparams.RootLength),
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:    make([]byte, fieldparams.RootLength),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
	}
	blk := &ethpb.SignedBeaconBlockBellatrix{
		Block: &ethpb.BeaconBlockBellatrix{
			Slot:       bellatrixSlot + 1,
			ParentRoot: parentRoot[:],
			StateRoot:  genesis.Block.StateRoot,
			Body: &ethpb.BeaconBlockBodyBellatrix{
				RandaoReveal:     genesis.Block.Body.RandaoReveal,
				Graffiti:         genesis.Block.Body.Graffiti,
				Eth1Data:         genesis.Block.Body.Eth1Data,
				SyncAggregate:    &ethpb.SyncAggregate{SyncCommitteeBits: bitfield.NewBitvector512(), SyncCommitteeSignature: make([]byte, 96)},
				ExecutionPayload: emptyPayload,
			},
		},
		Signature: genesis.Signature,
	}

	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, blkRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, blkRoot), "Could not save genesis state")

	b1 := util.NewBeaconBlockBellatrix()
	b1.Block.Body.ExecutionPayload.BlockNumber = 1 // Execution enabled.
	wb1, err := blocks.NewSignedBeaconBlock(b1)
	require.NoError(t, err)
	wbr1, err := wb1.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wb1))

	random, err := helpers.RandaoMix(beaconState, prysmtime.CurrentEpoch(beaconState))
	require.NoError(t, err)

	tstamp, err := slots.ToTime(beaconState.GenesisTime(), bellatrixSlot+1)
	require.NoError(t, err)
	h := &v1.ExecutionPayloadHeader{
		BlockNumber:      123,
		GasLimit:         456,
		GasUsed:          789,
		ParentHash:       make([]byte, fieldparams.RootLength),
		FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:        make([]byte, fieldparams.RootLength),
		ReceiptsRoot:     make([]byte, fieldparams.RootLength),
		LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:       random,
		BaseFeePerGas:    make([]byte, fieldparams.RootLength),
		BlockHash:        make([]byte, fieldparams.RootLength),
		TransactionsRoot: make([]byte, fieldparams.RootLength),
		ExtraData:        make([]byte, 0),
		Timestamp:        uint64(tstamp.Unix()),
	}

	proposerServer := &Server{
		FinalizationFetcher: &blockchainTest.ChainService{FinalizedCheckPoint: &ethpb.Checkpoint{Root: wbr1[:]}},
		HeadFetcher:         &blockchainTest.ChainService{State: beaconState, Root: parentRoot[:], Optimistic: false, Block: wb1},
		TimeFetcher:         &blockchainTest.ChainService{Genesis: time.Unix(int64(beaconState.GenesisTime()), 0)},
		SyncChecker:         &mockSync.Sync{IsSyncing: false},
		BlockReceiver:       &blockchainTest.ChainService{},
		HeadUpdater:         &blockchainTest.ChainService{},
		ForkFetcher:         &blockchainTest.ChainService{Fork: &ethpb.Fork{}},
		GenesisFetcher:      &blockchainTest.ChainService{},
		ChainStartFetcher:   &mockExecution.Chain{},
		Eth1InfoFetcher:     &mockExecution.Chain{},
		MockEth1Votes:       true,
		AttPool:             attestations.NewPool(),
		SlashingsPool:       slashings.NewPool(),
		ExitPool:            voluntaryexits.NewPool(),
		StateGen:            stategen.New(db),
		SyncCommitteePool:   synccommittee.NewStore(),
		ExecutionEngineCaller: &mockExecution.EngineClient{
			PayloadIDBytes:   &v1.PayloadIDBytes{1},
			ExecutionPayload: emptyPayload,
		},
		BeaconDB: db,
	}
	sk, err := bls.RandKey()
	require.NoError(t, err)
	bid := &ethpb.BuilderBid{
		Header: h,
		Pubkey: sk.PublicKey().Marshal(),
		Value:  bytesutil.PadTo([]byte{1, 2, 3}, 32),
	}
	d := params.BeaconConfig().DomainApplicationBuilder
	domain, err := signing.ComputeDomain(d, nil, nil)
	require.NoError(t, err)
	sr, err := signing.ComputeSigningRoot(bid, domain)
	require.NoError(t, err)
	sBid := &ethpb.SignedBuilderBid{
		Message:   bid,
		Signature: sk.Sign(sr[:]).Marshal(),
	}
	proposerServer.BlockBuilder = &builderTest.MockBuilderService{HasConfigured: true, Bid: sBid}
	proposerServer.ForkFetcher = &blockchainTest.ChainService{ForkChoiceStore: protoarray.New()}
	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)

	require.NoError(t, proposerServer.BeaconDB.SaveRegistrationsByValidatorIDs(ctx, []types.ValidatorIndex{40},
		[]*ethpb.ValidatorRegistrationV1{{FeeRecipient: bytesutil.PadTo([]byte{}, fieldparams.FeeRecipientLength), Pubkey: bytesutil.PadTo([]byte{}, fieldparams.BLSPubkeyLength)}}))

	params.SetupTestConfigCleanup(t)
	cfg.MaxBuilderConsecutiveMissedSlots = bellatrixSlot + 1
	cfg.MaxBuilderEpochMissedSlots = 32
	params.OverrideBeaconConfig(cfg)

	block, err := proposerServer.getBellatrixBeaconBlock(ctx, &ethpb.BlockRequest{
		Slot:         bellatrixSlot + 1,
		RandaoReveal: randaoReveal,
	})
	require.NoError(t, err)
	bellatrixBlk, ok := block.GetBlock().(*ethpb.GenericBeaconBlock_BlindedBellatrix)
	require.Equal(t, true, ok)
	require.LogsContain(t, hook, "Computed state root")
	require.DeepEqual(t, h, bellatrixBlk.BlindedBellatrix.Body.ExecutionPayloadHeader) // Payload header should equal.
}

func TestServer_validatorRegistered(t *testing.T) {
	proposerServer := &Server{}
	ctx := context.Background()

	reg, err := proposerServer.validatorRegistered(ctx, 0)
	require.ErrorContains(t, "nil beacon db", err)
	require.Equal(t, false, reg)

	proposerServer.BeaconDB = dbTest.SetupDB(t)
	reg, err = proposerServer.validatorRegistered(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, false, reg)

	f := bytesutil.PadTo([]byte{}, fieldparams.FeeRecipientLength)
	p := bytesutil.PadTo([]byte{}, fieldparams.BLSPubkeyLength)
	require.NoError(t, proposerServer.BeaconDB.SaveRegistrationsByValidatorIDs(ctx, []types.ValidatorIndex{0, 1},
		[]*ethpb.ValidatorRegistrationV1{{FeeRecipient: f, Pubkey: p}, {FeeRecipient: f, Pubkey: p}}))

	reg, err = proposerServer.validatorRegistered(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, true, reg)
	reg, err = proposerServer.validatorRegistered(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, true, reg)
}

func TestServer_validateBuilderSignature(t *testing.T) {
	s := &Server{}
	sk, err := bls.RandKey()
	require.NoError(t, err)
	bid := &ethpb.BuilderBid{
		Header: &v1.ExecutionPayloadHeader{
			ParentHash:       make([]byte, fieldparams.RootLength),
			FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
			StateRoot:        make([]byte, fieldparams.RootLength),
			ReceiptsRoot:     make([]byte, fieldparams.RootLength),
			LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
			PrevRandao:       make([]byte, fieldparams.RootLength),
			BaseFeePerGas:    make([]byte, fieldparams.RootLength),
			BlockHash:        make([]byte, fieldparams.RootLength),
			TransactionsRoot: make([]byte, fieldparams.RootLength),
			BlockNumber:      1,
		},
		Pubkey: sk.PublicKey().Marshal(),
		Value:  bytesutil.PadTo([]byte{1, 2, 3}, 32),
	}
	d := params.BeaconConfig().DomainApplicationBuilder
	domain, err := signing.ComputeDomain(d, nil, nil)
	require.NoError(t, err)
	sr, err := signing.ComputeSigningRoot(bid, domain)
	require.NoError(t, err)
	sBid := &ethpb.SignedBuilderBid{
		Message:   bid,
		Signature: sk.Sign(sr[:]).Marshal(),
	}
	require.NoError(t, s.validateBuilderSignature(sBid))

	sBid.Message.Value = make([]byte, 32)
	require.ErrorIs(t, s.validateBuilderSignature(sBid), signing.ErrSigFailedToVerify)
}

func TestServer_circuitBreakBuilder(t *testing.T) {
	hook := logTest.NewGlobal()
	s := &Server{}
	_, err := s.circuitBreakBuilder(0)
	require.ErrorContains(t, "no fork choicer configured", err)

	s.ForkFetcher = &blockchainTest.ChainService{ForkChoiceStore: protoarray.New()}
	s.ForkFetcher.ForkChoicer().SetGenesisTime(uint64(time.Now().Unix()))
	b, err := s.circuitBreakBuilder(params.BeaconConfig().MaxBuilderConsecutiveMissedSlots + 1)
	require.NoError(t, err)
	require.Equal(t, true, b)
	require.LogsContain(t, hook, "Builder circuit breaker activated due to missing consecutive slot")

	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ctx := context.Background()
	st, blkRoot, err := createState(1, [32]byte{'a'}, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, s.ForkFetcher.ForkChoicer().InsertNode(ctx, st, blkRoot))
	b, err = s.circuitBreakBuilder(params.BeaconConfig().MaxBuilderConsecutiveMissedSlots + 1)
	require.NoError(t, err)
	require.Equal(t, false, b)

	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	st, blkRoot, err = createState(params.BeaconConfig().SlotsPerEpoch, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, s.ForkFetcher.ForkChoicer().InsertNode(ctx, st, blkRoot))
	b, err = s.circuitBreakBuilder(params.BeaconConfig().SlotsPerEpoch + 1)
	require.NoError(t, err)
	require.Equal(t, true, b)
	require.LogsContain(t, hook, "Builder circuit breaker activated due to missing enough slots last epoch")

	want := params.BeaconConfig().SlotsPerEpoch - params.BeaconConfig().MaxBuilderEpochMissedSlots
	for i := types.Slot(2); i <= want+2; i++ {
		st, blkRoot, err = createState(i, [32]byte{byte(i)}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, ojc, ofc)
		require.NoError(t, err)
		require.NoError(t, s.ForkFetcher.ForkChoicer().InsertNode(ctx, st, blkRoot))
	}
	b, err = s.circuitBreakBuilder(params.BeaconConfig().SlotsPerEpoch + 1)
	require.NoError(t, err)
	require.Equal(t, false, b)
}

func createState(
	slot types.Slot,
	blockRoot [32]byte,
	parentRoot [32]byte,
	payloadHash [32]byte,
	justified *ethpb.Checkpoint,
	finalized *ethpb.Checkpoint,
) (state.BeaconState, [32]byte, error) {

	base := &ethpb.BeaconStateBellatrix{
		Slot:                       slot,
		RandaoMixes:                make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		BlockRoots:                 make([][]byte, 1),
		CurrentJustifiedCheckpoint: justified,
		FinalizedCheckpoint:        finalized,
		LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader{
			BlockHash: payloadHash[:],
		},
		LatestBlockHeader: &ethpb.BeaconBlockHeader{
			ParentRoot: parentRoot[:],
		},
	}

	base.BlockRoots[0] = append(base.BlockRoots[0], blockRoot[:]...)
	st, err := v3.InitializeFromProto(base)
	return st, blockRoot, err
}
