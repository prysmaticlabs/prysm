package validator

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	mock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache"
	b "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	dbutil "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	mockExecution "github.com/prysmaticlabs/prysm/v3/beacon-chain/execution/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/synccommittee"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
	mockSync "github.com/prysmaticlabs/prysm/v3/beacon-chain/sync/initial-sync/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestProposer_GetBeaconBlock_BellatrixEpoch(t *testing.T) {
	db := dbutil.SetupDB(t)
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

	genesis := b.NewGenesisBlock(stateRoot[:])
	util.SaveBlock(t, ctx, db, genesis)

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	bellatrixSlot, err := slots.EpochStart(params.BeaconConfig().BellatrixForkEpoch)
	require.NoError(t, err)

	var scBits [fieldparams.SyncAggregateSyncCommitteeBytesLength]byte
	blk := &ethpb.SignedBeaconBlockBellatrix{
		Block: &ethpb.BeaconBlockBellatrix{
			Slot:       bellatrixSlot + 1,
			ParentRoot: parentRoot[:],
			StateRoot:  genesis.Block.StateRoot,
			Body: &ethpb.BeaconBlockBodyBellatrix{
				RandaoReveal:  genesis.Block.Body.RandaoReveal,
				Graffiti:      genesis.Block.Body.Graffiti,
				Eth1Data:      genesis.Block.Body.Eth1Data,
				SyncAggregate: &ethpb.SyncAggregate{SyncCommitteeBits: scBits[:], SyncCommitteeSignature: make([]byte, 96)},
				ExecutionPayload: &enginev1.ExecutionPayload{
					ParentHash:    make([]byte, fieldparams.RootLength),
					FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
					StateRoot:     make([]byte, fieldparams.RootLength),
					ReceiptsRoot:  make([]byte, fieldparams.RootLength),
					LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
					PrevRandao:    make([]byte, fieldparams.RootLength),
					BaseFeePerGas: make([]byte, fieldparams.RootLength),
					BlockHash:     make([]byte, fieldparams.RootLength),
				},
			},
		},
		Signature: genesis.Signature,
	}

	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, blkRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, blkRoot), "Could not save genesis state")

	c := mockExecution.New()
	c.HashesByHeight[0] = terminalBlockHash
	random, err := helpers.RandaoMix(beaconState, slots.ToEpoch(beaconState.Slot()))
	require.NoError(t, err)
	timeStamp, err := slots.ToTime(beaconState.GenesisTime(), bellatrixSlot+1)
	require.NoError(t, err)

	payload := &enginev1.ExecutionPayload{
		ParentHash:    make([]byte, fieldparams.RootLength),
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:    random,
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
		Transactions:  make([][]byte, 0),
		ExtraData:     make([]byte, 0),
		BlockNumber:   1,
		GasLimit:      2,
		GasUsed:       3,
		Timestamp:     uint64(timeStamp.Unix()),
	}
	proposerServer := &Server{
		HeadFetcher:       &mock.ChainService{State: beaconState, Root: parentRoot[:], Optimistic: false},
		TimeFetcher:       &mock.ChainService{Genesis: time.Now()},
		SyncChecker:       &mockSync.Sync{IsSyncing: false},
		BlockReceiver:     &mock.ChainService{},
		HeadUpdater:       &mock.ChainService{},
		ChainStartFetcher: &mockExecution.Chain{},
		Eth1InfoFetcher:   &mockExecution.Chain{},
		Eth1BlockFetcher:  c,
		MockEth1Votes:     true,
		AttPool:           attestations.NewPool(),
		SlashingsPool:     slashings.NewPool(),
		ExitPool:          voluntaryexits.NewPool(),
		StateGen:          stategen.New(db),
		SyncCommitteePool: synccommittee.NewStore(),
		ExecutionEngineCaller: &mockExecution.EngineClient{
			PayloadIDBytes:   &enginev1.PayloadIDBytes{1},
			ExecutionPayload: payload,
		},
		BeaconDB:               db,
		ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
	}

	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)

	graffiti := bytesutil.ToBytes32([]byte("eth2"))
	require.NoError(t, err)
	req := &ethpb.BlockRequest{
		Slot:         bellatrixSlot + 1,
		RandaoReveal: randaoReveal,
		Graffiti:     graffiti[:],
	}

	block, err := proposerServer.GetBeaconBlock(ctx, req)
	require.NoError(t, err)
	bellatrixBlk, ok := block.GetBlock().(*ethpb.GenericBeaconBlock_Bellatrix)
	require.Equal(t, true, ok)

	assert.Equal(t, req.Slot, bellatrixBlk.Bellatrix.Slot)
	assert.DeepEqual(t, parentRoot[:], bellatrixBlk.Bellatrix.ParentRoot, "Expected block to have correct parent root")
	assert.DeepEqual(t, randaoReveal, bellatrixBlk.Bellatrix.Body.RandaoReveal, "Expected block to have correct randao reveal")
	assert.DeepEqual(t, req.Graffiti, bellatrixBlk.Bellatrix.Body.Graffiti, "Expected block to have correct Graffiti")

	require.LogsContain(t, hook, "Fee recipient is currently using the burn address")
	require.DeepEqual(t, payload, bellatrixBlk.Bellatrix.Body.ExecutionPayload) // Payload should equal.

	// Operator sets default fee recipient to not be burned through beacon node cli.
	newHook := logTest.NewGlobal()
	params.SetupTestConfigCleanup(t)
	cfg = params.BeaconConfig().Copy()
	cfg.DefaultFeeRecipient = common.Address{'b'}
	params.OverrideBeaconConfig(cfg)
	_, err = proposerServer.GetBeaconBlock(ctx, req)
	require.NoError(t, err)
	require.LogsDoNotContain(t, newHook, "Fee recipient is currently using the burn address")
}
