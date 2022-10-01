package validator

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	blockchainTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	builderTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/builder/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache"
	consensusblocks "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	prysmtime "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/time"
	dbTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	mockExecution "github.com/prysmaticlabs/prysm/v3/beacon-chain/execution/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/synccommittee"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
	mockSync "github.com/prysmaticlabs/prysm/v3/beacon-chain/sync/initial-sync/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestServer_GetEip4844BeaconBlock_NoBlobKZGs(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()
	hook := logTest.NewGlobal()

	terminalBlockHash := bytesutil.PadTo([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, 32)
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.Eip4844ForkEpoch = 3
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

	eip4844Slot, err := slots.EpochStart(params.BeaconConfig().Eip4844ForkEpoch)
	require.NoError(t, err)

	emptyPayload := &v1.ExecutionPayload4844{
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
	blk := &ethpb.SignedBeaconBlockWithBlobKZGs{
		Block: &ethpb.BeaconBlockWithBlobKZGs{
			Slot:       eip4844Slot + 1,
			ParentRoot: parentRoot[:],
			StateRoot:  genesis.Block.StateRoot,
			Body: &ethpb.BeaconBlockBodyWithBlobKZGs{
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
			PayloadIDBytes:       &v1.PayloadIDBytes{1},
			ExecutionPayload4844: emptyPayload,
			BlobsBundle:          &v1.BlobsBundle{BlockHash: emptyPayload.BlockHash},
			// TODO(inphi): Add blob kzgs
		},
		BeaconDB:               db,
		ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
		BlockBuilder:           &builderTest.MockBuilderService{},
	}
	proposerServer.ProposerSlotIndexCache.SetProposerAndPayloadIDs(25, 60, [8]byte{'a'}, parentRoot)

	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)

	block, err := proposerServer.getEip4844BeaconBlock(ctx, &ethpb.BlockRequest{
		Slot:         eip4844Slot + 1,
		RandaoReveal: randaoReveal,
	})
	require.NoError(t, err)
	eip4844Blk, ok := block.GetBlock().(*ethpb.GenericBeaconBlock_Eip4844)
	require.Equal(t, true, ok)
	require.LogsContain(t, hook, "Computed state root")
	require.DeepEqual(t, emptyPayload, eip4844Blk.Eip4844.Body.ExecutionPayload) // Payload should equal.
}

func TestServer_GetEip4844BeaconBlock_WithBlobKZGs(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()
	hook := logTest.NewGlobal()

	terminalBlockHash := bytesutil.PadTo([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, 32)
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.Eip4844ForkEpoch = 3
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

	eip4844Slot, err := slots.EpochStart(params.BeaconConfig().Eip4844ForkEpoch)
	require.NoError(t, err)

	blob := make([][]byte, 4096)
	for i := range blob {
		blob[i] = make([]byte, 32)
	}
	blobs := []*v1.Blob{{Blob: blob}}

	kzgs := make([][]byte, 1)
	var kzg [48]byte
	kzgs[0] = kzg[:]

	vh := types.KZGCommitment(kzg).ComputeVersionedHash()
	tx := &types.SignedBlobTx{
		Message: types.BlobTxMessage{
			BlobVersionedHashes: []common.Hash{vh},
		},
	}
	w := new(bytes.Buffer)
	require.NoError(t, w.WriteByte(types.BlobTxType))
	require.NoError(t, types.EncodeSSZ(w, tx))

	random, err := helpers.RandaoMix(beaconState, prysmtime.CurrentEpoch(beaconState))
	require.NoError(t, err)
	tstamp, err := slots.ToTime(beaconState.GenesisTime(), eip4844Slot+1)
	require.NoError(t, err)

	payload := &v1.ExecutionPayload4844{
		ParentHash:    make([]byte, fieldparams.RootLength),
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
		PrevRandao:    random,
		Timestamp:     uint64(tstamp.Unix()),
		Transactions:  [][]byte{w.Bytes()},
	}

	var scBits [fieldparams.SyncAggregateSyncCommitteeBytesLength]byte
	blk := &ethpb.SignedBeaconBlockWithBlobKZGs{
		Block: &ethpb.BeaconBlockWithBlobKZGs{
			Slot:       eip4844Slot + 1,
			ParentRoot: parentRoot[:],
			StateRoot:  genesis.Block.StateRoot,
			Body: &ethpb.BeaconBlockBodyWithBlobKZGs{
				RandaoReveal:     genesis.Block.Body.RandaoReveal,
				Graffiti:         genesis.Block.Body.Graffiti,
				Eth1Data:         genesis.Block.Body.Eth1Data,
				SyncAggregate:    &ethpb.SyncAggregate{SyncCommitteeBits: scBits[:], SyncCommitteeSignature: make([]byte, 96)},
				ExecutionPayload: payload,
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
			PayloadIDBytes:       &v1.PayloadIDBytes{1},
			ExecutionPayload4844: payload,
			BlobsBundle: &v1.BlobsBundle{
				BlockHash:       payload.BlockHash,
				Blobs:           blobs,
				Kzgs:            kzgs,
				AggregatedProof: make([]byte, 48),
			},
		},
		BeaconDB:               db,
		ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
		BlockBuilder:           &builderTest.MockBuilderService{},
	}
	proposerServer.ProposerSlotIndexCache.SetProposerAndPayloadIDs(25, 60, [8]byte{'a'}, parentRoot)

	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)

	block, err := proposerServer.getEip4844BeaconBlock(ctx, &ethpb.BlockRequest{
		Slot:         eip4844Slot + 1,
		RandaoReveal: randaoReveal,
	})
	require.NoError(t, err)
	eip4844Blk, ok := block.GetBlock().(*ethpb.GenericBeaconBlock_Eip4844)
	require.Equal(t, true, ok)
	require.LogsContain(t, hook, "Computed state root")
	require.DeepEqual(t, payload, eip4844Blk.Eip4844.Body.ExecutionPayload) // Payload should equal.

	root, err := eip4844Blk.Eip4844.HashTreeRoot()
	require.NoError(t, err)
	require.DeepEqual(t, root[:], block.Sidecar.BeaconBlockRoot)
}
