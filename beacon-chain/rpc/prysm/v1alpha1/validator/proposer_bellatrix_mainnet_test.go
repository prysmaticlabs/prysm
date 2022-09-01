package validator

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	blockchainTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	builderTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/builder/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache"
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
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
	mockSync "github.com/prysmaticlabs/prysm/v3/beacon-chain/sync/initial-sync/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestServer_GetBellatrixBeaconBlock_HappyCase(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()
	hook := logTest.NewGlobal()

	terminalBlockHash := bytesutil.PadTo([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, 32)
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
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
	var scBits [fieldparams.SyncAggregateSyncCommitteeBytesLength]byte
	blk := &ethpb.SignedBeaconBlockBellatrix{
		Block: &ethpb.BeaconBlockBellatrix{
			Slot:       bellatrixSlot + 1,
			ParentRoot: parentRoot[:],
			StateRoot:  genesis.Block.StateRoot,
			Body: &ethpb.BeaconBlockBodyBellatrix{
				RandaoReveal:     genesis.Block.Body.RandaoReveal,
				Graffiti:         genesis.Block.Body.Graffiti,
				Eth1Data:         genesis.Block.Body.Eth1Data,
				SyncAggregate:    &ethpb.SyncAggregate{SyncCommitteeBits: scBits[:], SyncCommitteeSignature: make([]byte, 96)},
				ExecutionPayload: emptyPayload,
			},
		},
		Signature: genesis.Signature,
	}

	blkRoot, err := blk.Block.HashTreeRoot()
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
	cfg := params.BeaconConfig().Copy()
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
	var scBits [fieldparams.SyncAggregateSyncCommitteeBytesLength]byte
	blk := &ethpb.SignedBeaconBlockBellatrix{
		Block: &ethpb.BeaconBlockBellatrix{
			Slot:       bellatrixSlot + 1,
			ParentRoot: parentRoot[:],
			StateRoot:  genesis.Block.StateRoot,
			Body: &ethpb.BeaconBlockBodyBellatrix{
				RandaoReveal:     genesis.Block.Body.RandaoReveal,
				Graffiti:         genesis.Block.Body.Graffiti,
				Eth1Data:         genesis.Block.Body.Eth1Data,
				SyncAggregate:    &ethpb.SyncAggregate{SyncCommitteeBits: scBits[:], SyncCommitteeSignature: make([]byte, 96)},
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
