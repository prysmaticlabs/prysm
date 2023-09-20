package validator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	mock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/builder"
	builderTest "github.com/prysmaticlabs/prysm/v4/beacon-chain/builder/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache"
	b "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	dbutil "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	mockExecution "github.com/prysmaticlabs/prysm/v4/beacon-chain/execution/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/blstoexec"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/synccommittee"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/voluntaryexits"
	mockp2p "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/core"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stategen"
	mockSync "github.com/prysmaticlabs/prysm/v4/beacon-chain/sync/initial-sync/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServer_GetBeaconBlock_Phase0(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()

	beaconState, privKeys := util.DeterministicGenesisState(t, 64)
	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	genesis := b.NewGenesisBlock(stateRoot[:])
	genBlk := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       genesis.Block.Slot,
			ParentRoot: genesis.Block.ParentRoot,
			StateRoot:  genesis.Block.StateRoot,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: genesis.Block.Body.RandaoReveal,
				Graffiti:     genesis.Block.Body.Graffiti,
				Eth1Data:     genesis.Block.Body.Eth1Data,
			},
		},
		Signature: genesis.Signature,
	}
	util.SaveBlock(t, ctx, db, genBlk)

	parentRoot, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	proposerServer := getProposerServer(db, beaconState, parentRoot[:])

	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)

	graffiti := bytesutil.ToBytes32([]byte("eth2"))
	req := &ethpb.BlockRequest{
		Slot:         1,
		RandaoReveal: randaoReveal,
		Graffiti:     graffiti[:],
	}
	proposerSlashings, attSlashings := injectSlashings(t, beaconState, privKeys, proposerServer)

	block, err := proposerServer.GetBeaconBlock(ctx, req)
	require.NoError(t, err)
	phase0Blk, ok := block.GetBlock().(*ethpb.GenericBeaconBlock_Phase0)
	require.Equal(t, true, ok)
	assert.Equal(t, req.Slot, phase0Blk.Phase0.Slot)
	assert.DeepEqual(t, parentRoot[:], phase0Blk.Phase0.ParentRoot, "Expected block to have correct parent root")
	assert.DeepEqual(t, randaoReveal, phase0Blk.Phase0.Body.RandaoReveal, "Expected block to have correct randao reveal")
	assert.DeepEqual(t, req.Graffiti, phase0Blk.Phase0.Body.Graffiti, "Expected block to have correct Graffiti")
	assert.Equal(t, params.BeaconConfig().MaxProposerSlashings, uint64(len(phase0Blk.Phase0.Body.ProposerSlashings)))
	assert.DeepEqual(t, proposerSlashings, phase0Blk.Phase0.Body.ProposerSlashings)
	assert.Equal(t, params.BeaconConfig().MaxAttesterSlashings, uint64(len(phase0Blk.Phase0.Body.AttesterSlashings)))
	assert.DeepEqual(t, attSlashings, phase0Blk.Phase0.Body.AttesterSlashings)
}

func TestServer_GetBeaconBlock_Altair(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.AltairForkEpoch = 1
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

	altairSlot, err := slots.EpochStart(params.BeaconConfig().AltairForkEpoch)
	require.NoError(t, err)

	var scBits [fieldparams.SyncAggregateSyncCommitteeBytesLength]byte
	genAltair := &ethpb.SignedBeaconBlockAltair{
		Block: &ethpb.BeaconBlockAltair{
			Slot:       altairSlot + 1,
			ParentRoot: parentRoot[:],
			StateRoot:  genesis.Block.StateRoot,
			Body: &ethpb.BeaconBlockBodyAltair{
				RandaoReveal:  genesis.Block.Body.RandaoReveal,
				Graffiti:      genesis.Block.Body.Graffiti,
				Eth1Data:      genesis.Block.Body.Eth1Data,
				SyncAggregate: &ethpb.SyncAggregate{SyncCommitteeBits: scBits[:], SyncCommitteeSignature: make([]byte, 96)},
			},
		},
		Signature: genesis.Signature,
	}

	blkRoot, err := genAltair.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, blkRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, blkRoot), "Could not save genesis state")

	proposerServer := getProposerServer(db, beaconState, parentRoot[:])

	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)

	graffiti := bytesutil.ToBytes32([]byte("eth2"))
	require.NoError(t, err)
	req := &ethpb.BlockRequest{
		Slot:         altairSlot + 1,
		RandaoReveal: randaoReveal,
		Graffiti:     graffiti[:],
	}
	proposerSlashings, attSlashings := injectSlashings(t, beaconState, privKeys, proposerServer)

	block, err := proposerServer.GetBeaconBlock(ctx, req)
	require.NoError(t, err)
	altairBlk, ok := block.GetBlock().(*ethpb.GenericBeaconBlock_Altair)
	require.Equal(t, true, ok)

	assert.Equal(t, req.Slot, altairBlk.Altair.Slot)
	assert.DeepEqual(t, parentRoot[:], altairBlk.Altair.ParentRoot, "Expected block to have correct parent root")
	assert.DeepEqual(t, randaoReveal, altairBlk.Altair.Body.RandaoReveal, "Expected block to have correct randao reveal")
	assert.DeepEqual(t, req.Graffiti, altairBlk.Altair.Body.Graffiti, "Expected block to have correct Graffiti")
	assert.Equal(t, params.BeaconConfig().MaxProposerSlashings, uint64(len(altairBlk.Altair.Body.ProposerSlashings)))
	assert.DeepEqual(t, proposerSlashings, altairBlk.Altair.Body.ProposerSlashings)
	assert.Equal(t, params.BeaconConfig().MaxAttesterSlashings, uint64(len(altairBlk.Altair.Body.AttesterSlashings)))
	assert.DeepEqual(t, attSlashings, altairBlk.Altair.Body.AttesterSlashings)
}

func TestServer_GetBeaconBlock_Bellatrix(t *testing.T) {
	db := dbutil.SetupDB(t)
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

	proposerServer := getProposerServer(db, beaconState, parentRoot[:])
	proposerServer.CoreService.Eth1BlockFetcher = c
	proposerServer.CoreService.ExecutionEngineCaller = &mockExecution.EngineClient{
		PayloadIDBytes:   &enginev1.PayloadIDBytes{1},
		ExecutionPayload: payload,
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
	cfg = params.MinimalSpecConfig().Copy()
	cfg.DefaultFeeRecipient = common.Address{'b'}
	params.OverrideBeaconConfig(cfg)
	_, err = proposerServer.GetBeaconBlock(ctx, req)
	require.NoError(t, err)
	require.LogsDoNotContain(t, newHook, "Fee recipient is currently using the burn address")
}

func TestServer_GetBeaconBlock_Capella(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()
	transition.SkipSlotCache.Disable()

	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.CapellaForkEpoch = 3
	cfg.BellatrixForkEpoch = 2
	cfg.AltairForkEpoch = 1
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

	capellaSlot, err := slots.EpochStart(params.BeaconConfig().CapellaForkEpoch)
	require.NoError(t, err)

	var scBits [fieldparams.SyncAggregateSyncCommitteeBytesLength]byte
	blk := &ethpb.SignedBeaconBlockCapella{
		Block: &ethpb.BeaconBlockCapella{
			Slot:       capellaSlot + 1,
			ParentRoot: parentRoot[:],
			StateRoot:  genesis.Block.StateRoot,
			Body: &ethpb.BeaconBlockBodyCapella{
				RandaoReveal:  genesis.Block.Body.RandaoReveal,
				Graffiti:      genesis.Block.Body.Graffiti,
				Eth1Data:      genesis.Block.Body.Eth1Data,
				SyncAggregate: &ethpb.SyncAggregate{SyncCommitteeBits: scBits[:], SyncCommitteeSignature: make([]byte, 96)},
				ExecutionPayload: &enginev1.ExecutionPayloadCapella{
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

	random, err := helpers.RandaoMix(beaconState, slots.ToEpoch(beaconState.Slot()))
	require.NoError(t, err)
	timeStamp, err := slots.ToTime(beaconState.GenesisTime(), capellaSlot+1)
	require.NoError(t, err)
	payload := &enginev1.ExecutionPayloadCapella{
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

	proposerServer := getProposerServer(db, beaconState, parentRoot[:])
	proposerServer.CoreService.ExecutionEngineCaller = &mockExecution.EngineClient{
		PayloadIDBytes:          &enginev1.PayloadIDBytes{1},
		ExecutionPayloadCapella: payload,
	}

	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)

	graffiti := bytesutil.ToBytes32([]byte("eth2"))
	require.NoError(t, err)
	req := &ethpb.BlockRequest{
		Slot:         capellaSlot + 1,
		RandaoReveal: randaoReveal,
		Graffiti:     graffiti[:],
	}

	copiedState := beaconState.Copy()
	copiedState, err = transition.ProcessSlots(ctx, copiedState, capellaSlot+1)
	require.NoError(t, err)
	change, err := util.GenerateBLSToExecutionChange(copiedState, privKeys[1], 0)
	require.NoError(t, err)
	proposerServer.CoreService.BLSChangesPool.InsertBLSToExecChange(change)

	got, err := proposerServer.GetBeaconBlock(ctx, req)
	require.NoError(t, err)
	require.Equal(t, 1, len(got.GetCapella().Body.BlsToExecutionChanges))
	require.DeepEqual(t, change, got.GetCapella().Body.BlsToExecutionChanges[0])
}

func TestServer_GetBeaconBlock_Deneb(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()
	transition.SkipSlotCache.Disable()

	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.DenebForkEpoch = 4
	cfg.CapellaForkEpoch = 3
	cfg.BellatrixForkEpoch = 2
	cfg.AltairForkEpoch = 1
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

	denebSlot, err := slots.EpochStart(params.BeaconConfig().DenebForkEpoch)
	require.NoError(t, err)

	var scBits [fieldparams.SyncAggregateSyncCommitteeBytesLength]byte
	blk := &ethpb.SignedBeaconBlockDeneb{
		Block: &ethpb.BeaconBlockDeneb{
			Slot:       denebSlot + 1,
			ParentRoot: parentRoot[:],
			StateRoot:  genesis.Block.StateRoot,
			Body: &ethpb.BeaconBlockBodyDeneb{
				RandaoReveal:  genesis.Block.Body.RandaoReveal,
				Graffiti:      genesis.Block.Body.Graffiti,
				Eth1Data:      genesis.Block.Body.Eth1Data,
				SyncAggregate: &ethpb.SyncAggregate{SyncCommitteeBits: scBits[:], SyncCommitteeSignature: make([]byte, 96)},
				ExecutionPayload: &enginev1.ExecutionPayloadDeneb{
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

	random, err := helpers.RandaoMix(beaconState, slots.ToEpoch(beaconState.Slot()))
	require.NoError(t, err)
	timeStamp, err := slots.ToTime(beaconState.GenesisTime(), denebSlot+1)
	require.NoError(t, err)
	payload := &enginev1.ExecutionPayloadDeneb{
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
		BlobGasUsed:   4,
		ExcessBlobGas: 5,
	}

	kc := make([][]byte, 0)
	kc = append(kc, bytesutil.PadTo([]byte("kc"), 48))
	kc = append(kc, bytesutil.PadTo([]byte("kc1"), 48))
	kc = append(kc, bytesutil.PadTo([]byte("kc2"), 48))
	proofs := [][]byte{[]byte("proof"), []byte("proof1"), []byte("proof2")}
	blobs := [][]byte{[]byte("blob"), []byte("blob1"), []byte("blob2")}
	bundle := &enginev1.BlobsBundle{KzgCommitments: kc, Proofs: proofs, Blobs: blobs}
	proposerServer := getProposerServer(db, beaconState, parentRoot[:])
	proposerServer.CoreService.ExecutionEngineCaller = &mockExecution.EngineClient{
		PayloadIDBytes:        &enginev1.PayloadIDBytes{1},
		ExecutionPayloadDeneb: payload,
		BlobsBundle:           bundle,
	}

	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)

	graffiti := bytesutil.ToBytes32([]byte("eth2"))
	require.NoError(t, err)
	req := &ethpb.BlockRequest{
		Slot:         denebSlot + 1,
		RandaoReveal: randaoReveal,
		Graffiti:     graffiti[:],
	}

	copiedState := beaconState.Copy()
	copiedState, err = transition.ProcessSlots(ctx, copiedState, denebSlot+1)
	require.NoError(t, err)
	change, err := util.GenerateBLSToExecutionChange(copiedState, privKeys[1], 0)
	require.NoError(t, err)
	proposerServer.CoreService.BLSChangesPool.InsertBLSToExecChange(change)

	got, err := proposerServer.GetBeaconBlock(ctx, req)
	require.NoError(t, err)
	require.DeepEqual(t, got.GetDeneb().Block.Body.BlobKzgCommitments, kc)

	require.Equal(t, 3, len(got.GetDeneb().Blobs))
	blockRoot, err := got.GetDeneb().Block.HashTreeRoot()
	require.NoError(t, err)
	require.DeepEqual(t, blockRoot[:], got.GetDeneb().Blobs[0].BlockRoot)
}

func TestServer_GetBeaconBlock_Optimistic(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.BellatrixForkEpoch = 2
	cfg.AltairForkEpoch = 1
	params.OverrideBeaconConfig(cfg)

	bellatrixSlot, err := slots.EpochStart(params.BeaconConfig().BellatrixForkEpoch)
	require.NoError(t, err)

	mockChainService := &mock.ChainService{ForkChoiceStore: doublylinkedtree.New()}
	proposerServer := &Server{
		CoreService: &core.Service{
			OptimisticModeFetcher: &mock.ChainService{Optimistic: true},
			SyncChecker:           &mockSync.Sync{},
			ForkFetcher:           mockChainService,
			ForkchoiceFetcher:     mockChainService,
			TimeFetcher:           &mock.ChainService{},
		},
	}
	req := &ethpb.BlockRequest{
		Slot: bellatrixSlot + 1,
	}
	_, err = proposerServer.GetBeaconBlock(context.Background(), req)
	s, ok := status.FromError(err)
	require.Equal(t, true, ok)
	require.DeepEqual(t, codes.Unavailable, s.Code())
	require.ErrorContains(t, errOptimisticMode.Error(), err)
}

func getProposerServer(db db.HeadAccessDatabase, headState state.BeaconState, headRoot []byte) *Server {
	mockChainService := &mock.ChainService{State: headState, Root: headRoot, ForkChoiceStore: doublylinkedtree.New()}
	return &Server{
		CoreService: &core.Service{
			HeadFetcher:           mockChainService,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockReceiver:         mockChainService,
			ChainStartFetcher:     &mockExecution.Chain{},
			Eth1InfoFetcher:       &mockExecution.Chain{},
			Eth1BlockFetcher:      &mockExecution.Chain{},
			FinalizationFetcher:   mockChainService,
			ForkFetcher:           mockChainService,
			ForkchoiceFetcher:     mockChainService,
			MockEth1Votes:         true,
			AttPool:               attestations.NewPool(),
			SlashingsPool:         slashings.NewPool(),
			ExitPool:              voluntaryexits.NewPool(),
			StateGen:              stategen.New(db, doublylinkedtree.New()),
			SyncCommitteePool:     synccommittee.NewStore(),
			OptimisticModeFetcher: &mock.ChainService{},
			TimeFetcher: &testutil.MockGenesisTimeFetcher{
				Genesis: time.Now(),
			},
			ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
			BeaconDB:               db,
			BLSChangesPool:         blstoexec.NewPool(),
			BlockBuilder:           &builderTest.MockBuilderService{HasConfigured: true},
		},
	}
}

func injectSlashings(t *testing.T, st state.BeaconState, keys []bls.SecretKey, server *Server) ([]*ethpb.ProposerSlashing, []*ethpb.AttesterSlashing) {
	proposerSlashings := make([]*ethpb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
	for i := primitives.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
		proposerSlashing, err := util.GenerateProposerSlashingForValidator(st, keys[i], i /* validator index */)
		require.NoError(t, err)
		proposerSlashings[i] = proposerSlashing
		err = server.CoreService.SlashingsPool.InsertProposerSlashing(context.Background(), st, proposerSlashing)
		require.NoError(t, err)
	}

	attSlashings := make([]*ethpb.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
	for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings; i++ {
		attesterSlashing, err := util.GenerateAttesterSlashingForValidator(st, keys[i+params.BeaconConfig().MaxProposerSlashings], primitives.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings) /* validator index */)
		require.NoError(t, err)
		attSlashings[i] = attesterSlashing
		err = server.CoreService.SlashingsPool.InsertAttesterSlashing(context.Background(), st, attesterSlashing)
		require.NoError(t, err)
	}
	return proposerSlashings, attSlashings
}

func TestProposer_ProposeBlock_OK(t *testing.T) {
	tests := []struct {
		name  string
		block func([32]byte) *ethpb.GenericSignedBeaconBlock
		err   string
	}{
		{
			name: "phase0",
			block: func(parent [32]byte) *ethpb.GenericSignedBeaconBlock {
				blockToPropose := util.NewBeaconBlock()
				blockToPropose.Block.Slot = 5
				blockToPropose.Block.ParentRoot = parent[:]
				blk := &ethpb.GenericSignedBeaconBlock_Phase0{Phase0: blockToPropose}
				return &ethpb.GenericSignedBeaconBlock{Block: blk}
			},
		},
		{
			name: "altair",
			block: func(parent [32]byte) *ethpb.GenericSignedBeaconBlock {
				blockToPropose := util.NewBeaconBlockAltair()
				blockToPropose.Block.Slot = 5
				blockToPropose.Block.ParentRoot = parent[:]
				blk := &ethpb.GenericSignedBeaconBlock_Altair{Altair: blockToPropose}
				return &ethpb.GenericSignedBeaconBlock{Block: blk}
			},
		},
		{
			name: "bellatrix",
			block: func(parent [32]byte) *ethpb.GenericSignedBeaconBlock {
				blockToPropose := util.NewBeaconBlockBellatrix()
				blockToPropose.Block.Slot = 5
				blockToPropose.Block.ParentRoot = parent[:]
				blk := &ethpb.GenericSignedBeaconBlock_Bellatrix{Bellatrix: blockToPropose}
				return &ethpb.GenericSignedBeaconBlock{Block: blk}
			},
		},
		{
			name: "blind capella",
			block: func(parent [32]byte) *ethpb.GenericSignedBeaconBlock {
				blockToPropose := util.NewBlindedBeaconBlockCapella()
				blockToPropose.Block.Slot = 5
				blockToPropose.Block.ParentRoot = parent[:]
				txRoot, err := ssz.TransactionsRoot([][]byte{})
				require.NoError(t, err)
				withdrawalsRoot, err := ssz.WithdrawalSliceRoot([]*enginev1.Withdrawal{}, fieldparams.MaxWithdrawalsPerPayload)
				require.NoError(t, err)
				blockToPropose.Block.Body.ExecutionPayloadHeader.TransactionsRoot = txRoot[:]
				blockToPropose.Block.Body.ExecutionPayloadHeader.WithdrawalsRoot = withdrawalsRoot[:]
				blk := &ethpb.GenericSignedBeaconBlock_BlindedCapella{BlindedCapella: blockToPropose}
				return &ethpb.GenericSignedBeaconBlock{Block: blk}
			},
		},
		{
			name: "bellatrix",
			block: func(parent [32]byte) *ethpb.GenericSignedBeaconBlock {
				blockToPropose := util.NewBeaconBlockBellatrix()
				blockToPropose.Block.Slot = 5
				blockToPropose.Block.ParentRoot = parent[:]
				blk := &ethpb.GenericSignedBeaconBlock_Bellatrix{Bellatrix: blockToPropose}
				return &ethpb.GenericSignedBeaconBlock{Block: blk}
			},
		},
		{
			name: "deneb block no blob",
			block: func(parent [32]byte) *ethpb.GenericSignedBeaconBlock {
				blockToPropose := util.NewBeaconBlockDeneb()
				blockToPropose.Block.Slot = 5
				blockToPropose.Block.ParentRoot = parent[:]
				blk := &ethpb.GenericSignedBeaconBlock_Deneb{Deneb: &ethpb.SignedBeaconBlockAndBlobsDeneb{
					Block: blockToPropose,
				}}
				return &ethpb.GenericSignedBeaconBlock{Block: blk}
			},
		},
		{
			name: "deneb block has blobs",
			block: func(parent [32]byte) *ethpb.GenericSignedBeaconBlock {
				blockToPropose := util.NewBeaconBlockDeneb()
				blockToPropose.Block.Slot = 5
				blockToPropose.Block.ParentRoot = parent[:]
				blk := &ethpb.GenericSignedBeaconBlock_Deneb{Deneb: &ethpb.SignedBeaconBlockAndBlobsDeneb{
					Block: blockToPropose,
					Blobs: []*ethpb.SignedBlobSidecar{
						{Message: &ethpb.BlobSidecar{Index: 0, Slot: 5, BlockParentRoot: parent[:]}},
						{Message: &ethpb.BlobSidecar{Index: 1, Slot: 5, BlockParentRoot: parent[:]}},
						{Message: &ethpb.BlobSidecar{Index: 2, Slot: 5, BlockParentRoot: parent[:]}},
						{Message: &ethpb.BlobSidecar{Index: 3, Slot: 5, BlockParentRoot: parent[:]}},
					},
				}}
				return &ethpb.GenericSignedBeaconBlock{Block: blk}
			},
		},
		{
			name: "deneb block has too many blobs",
			err:  "too many blobs in block: 7",
			block: func(parent [32]byte) *ethpb.GenericSignedBeaconBlock {
				blockToPropose := util.NewBeaconBlockDeneb()
				blockToPropose.Block.Slot = 5
				blockToPropose.Block.ParentRoot = parent[:]
				blk := &ethpb.GenericSignedBeaconBlock_Deneb{Deneb: &ethpb.SignedBeaconBlockAndBlobsDeneb{
					Block: blockToPropose,
					Blobs: []*ethpb.SignedBlobSidecar{
						{Message: &ethpb.BlobSidecar{Index: 0, Slot: 5, BlockParentRoot: parent[:]}},
						{Message: &ethpb.BlobSidecar{Index: 1, Slot: 5, BlockParentRoot: parent[:]}},
						{Message: &ethpb.BlobSidecar{Index: 2, Slot: 5, BlockParentRoot: parent[:]}},
						{Message: &ethpb.BlobSidecar{Index: 3, Slot: 5, BlockParentRoot: parent[:]}},
						{Message: &ethpb.BlobSidecar{Index: 4, Slot: 5, BlockParentRoot: parent[:]}},
						{Message: &ethpb.BlobSidecar{Index: 5, Slot: 5, BlockParentRoot: parent[:]}},
						{Message: &ethpb.BlobSidecar{Index: 6, Slot: 5, BlockParentRoot: parent[:]}},
					},
				}}
				return &ethpb.GenericSignedBeaconBlock{Block: blk}
			},
		},
		{
			name: "blind capella",
			block: func(parent [32]byte) *ethpb.GenericSignedBeaconBlock {
				blockToPropose := util.NewBlindedBeaconBlockDeneb()
				blockToPropose.Message.Slot = 5
				blockToPropose.Message.ParentRoot = parent[:]
				txRoot, err := ssz.TransactionsRoot([][]byte{})
				require.NoError(t, err)
				withdrawalsRoot, err := ssz.WithdrawalSliceRoot([]*enginev1.Withdrawal{}, fieldparams.MaxWithdrawalsPerPayload)
				require.NoError(t, err)
				blockToPropose.Message.Body.ExecutionPayloadHeader.TransactionsRoot = txRoot[:]
				blockToPropose.Message.Body.ExecutionPayloadHeader.WithdrawalsRoot = withdrawalsRoot[:]
				blk := &ethpb.GenericSignedBeaconBlock_BlindedDeneb{BlindedDeneb: &ethpb.SignedBlindedBeaconBlockAndBlobsDeneb{
					SignedBlindedBlock: blockToPropose,
					SignedBlindedBlobSidecars: []*ethpb.SignedBlindedBlobSidecar{
						{
							Message: &ethpb.BlindedBlobSidecar{
								BlockRoot:       []byte{0x01},
								Slot:            2,
								BlockParentRoot: []byte{0x03},
								ProposerIndex:   3,
								BlobRoot:        []byte{0x04},
								KzgCommitment:   []byte{0x05},
								KzgProof:        []byte{0x06},
							},
							Signature: []byte{0x07},
						},
					},
				}}
				return &ethpb.GenericSignedBeaconBlock{Block: blk}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			numDeposits := uint64(64)
			beaconState, _ := util.DeterministicGenesisState(t, numDeposits)
			bsRoot, err := beaconState.HashTreeRoot(ctx)
			require.NoError(t, err)

			c := &mock.ChainService{Root: bsRoot[:], State: beaconState}
			db := dbutil.SetupDB(t)
			proposerServer := &Server{
				CoreService: &core.Service{
					BlockReceiver: c,
					BlockNotifier: c.BlockNotifier(),
					P2P:           mockp2p.NewTestP2P(t),
					BlockBuilder:  &builderTest.MockBuilderService{HasConfigured: true, PayloadCapella: emptyPayloadCapella(), PayloadDeneb: emptyPayloadDeneb(), BlobBundle: &enginev1.BlobsBundle{KzgCommitments: [][]byte{{0x01}}, Proofs: [][]byte{{0x02}}, Blobs: [][]byte{{0x03}}}},
					BeaconDB:      db,
				},
			}
			blockToPropose := tt.block(bsRoot)
			res, err := proposerServer.ProposeBeaconBlock(context.Background(), blockToPropose)
			if tt.err != "" { // Expecting an error
				require.ErrorContains(t, tt.err, err)
			} else {
				assert.NoError(t, err, "Could not propose block correctly")
				if res == nil || len(res.BlockRoot) == 0 {
					t.Error("No block root was returned")
				}
			}
			if tt.name == "deneb block has blobs" {
				scs, err := db.BlobSidecarsBySlot(ctx, blockToPropose.GetDeneb().Block.Block.Slot)
				require.NoError(t, err)
				assert.Equal(t, 4, len(scs))
				for i, sc := range scs {
					require.Equal(t, uint64(i), sc.Index)
				}
			}
		})
	}
}

func TestProposer_GetFeeRecipientByPubKey(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()
	numDeposits := uint64(64)
	beaconState, _ := util.DeterministicGenesisState(t, numDeposits)
	bsRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err)
	proposerServer := &Server{
		BeaconDB:    db,
		HeadFetcher: &mock.ChainService{Root: bsRoot[:], State: beaconState},
	}
	pubkey, err := hexutil.Decode("0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a")
	require.NoError(t, err)
	resp, err := proposerServer.GetFeeRecipientByPubKey(ctx, &ethpb.FeeRecipientByPubKeyRequest{
		PublicKey: pubkey,
	})
	require.NoError(t, err)

	require.Equal(t, params.BeaconConfig().DefaultFeeRecipient.Hex(), hexutil.Encode(resp.FeeRecipient))
	params.BeaconConfig().DefaultFeeRecipient = common.HexToAddress("0x046Fb65722E7b2455012BFEBf6177F1D2e9728D9")
	resp, err = proposerServer.GetFeeRecipientByPubKey(ctx, &ethpb.FeeRecipientByPubKeyRequest{
		PublicKey: beaconState.Validators()[0].PublicKey,
	})
	require.NoError(t, err)

	require.Equal(t, params.BeaconConfig().DefaultFeeRecipient.Hex(), common.BytesToAddress(resp.FeeRecipient).Hex())
	index, err := proposerServer.ValidatorIndex(ctx, &ethpb.ValidatorIndexRequest{
		PublicKey: beaconState.Validators()[0].PublicKey,
	})
	require.NoError(t, err)
	err = proposerServer.BeaconDB.SaveFeeRecipientsByValidatorIDs(ctx, []primitives.ValidatorIndex{index.Index}, []common.Address{common.HexToAddress("0x055Fb65722E7b2455012BFEBf6177F1D2e9728D8")})
	require.NoError(t, err)
	resp, err = proposerServer.GetFeeRecipientByPubKey(ctx, &ethpb.FeeRecipientByPubKeyRequest{
		PublicKey: beaconState.Validators()[0].PublicKey,
	})
	require.NoError(t, err)

	require.Equal(t, common.HexToAddress("0x055Fb65722E7b2455012BFEBf6177F1D2e9728D8").Hex(), common.BytesToAddress(resp.FeeRecipient).Hex())
}

func TestProposer_PrepareBeaconProposer(t *testing.T) {
	type args struct {
		request *ethpb.PrepareBeaconProposerRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr string
	}{
		{
			name: "Happy Path",
			args: args{
				request: &ethpb.PrepareBeaconProposerRequest{
					Recipients: []*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer{
						{
							FeeRecipient:   make([]byte, fieldparams.FeeRecipientLength),
							ValidatorIndex: 1,
						},
					},
				},
			},
			wantErr: "",
		},
		{
			name: "invalid fee recipient length",
			args: args{
				request: &ethpb.PrepareBeaconProposerRequest{
					Recipients: []*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer{
						{
							FeeRecipient:   make([]byte, fieldparams.BLSPubkeyLength),
							ValidatorIndex: 1,
						},
					},
				},
			},
			wantErr: "Invalid fee recipient address",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := dbutil.SetupDB(t)
			ctx := context.Background()
			s := &Server{BeaconDB: db}
			_, err := s.PrepareBeaconProposer(ctx, tt.args.request)
			if tt.wantErr != "" {
				require.ErrorContains(t, tt.wantErr, err)
				return
			}
			require.NoError(t, err)
			address, err := s.BeaconDB.FeeRecipientByValidatorID(ctx, 1)
			require.NoError(t, err)
			require.Equal(t, common.BytesToAddress(tt.args.request.Recipients[0].FeeRecipient), address)

		})
	}
}

func TestProposer_PrepareBeaconProposerOverlapping(t *testing.T) {
	hook := logTest.NewGlobal()
	db := dbutil.SetupDB(t)
	ctx := context.Background()
	proposerServer := &Server{BeaconDB: db}

	// New validator
	f := bytesutil.PadTo([]byte{0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}, fieldparams.FeeRecipientLength)
	req := &ethpb.PrepareBeaconProposerRequest{
		Recipients: []*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer{
			{FeeRecipient: f, ValidatorIndex: 1},
		},
	}
	_, err := proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsContain(t, hook, "Updated fee recipient addresses for validator indices")

	// Same validator
	hook.Reset()
	_, err = proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsDoNotContain(t, hook, "Updated fee recipient addresses for validator indices")

	// Same validator with different fee recipient
	hook.Reset()
	f = bytesutil.PadTo([]byte{0x01, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}, fieldparams.FeeRecipientLength)
	req = &ethpb.PrepareBeaconProposerRequest{
		Recipients: []*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer{
			{FeeRecipient: f, ValidatorIndex: 1},
		},
	}
	_, err = proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsContain(t, hook, "Updated fee recipient addresses for validator indices")

	// More than one validator
	hook.Reset()
	f = bytesutil.PadTo([]byte{0x01, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}, fieldparams.FeeRecipientLength)
	req = &ethpb.PrepareBeaconProposerRequest{
		Recipients: []*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer{
			{FeeRecipient: f, ValidatorIndex: 1},
			{FeeRecipient: f, ValidatorIndex: 2},
		},
	}
	_, err = proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsContain(t, hook, "Updated fee recipient addresses for validator indices")

	// Same validators
	hook.Reset()
	_, err = proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsDoNotContain(t, hook, "Updated fee recipient addresses for validator indices")
}

func BenchmarkServer_PrepareBeaconProposer(b *testing.B) {
	db := dbutil.SetupDB(b)
	ctx := context.Background()
	proposerServer := &Server{BeaconDB: db}

	f := bytesutil.PadTo([]byte{0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}, fieldparams.FeeRecipientLength)
	recipients := make([]*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer, 0)
	for i := 0; i < 10000; i++ {
		recipients = append(recipients, &ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer{FeeRecipient: f, ValidatorIndex: primitives.ValidatorIndex(i)})
	}

	req := &ethpb.PrepareBeaconProposerRequest{
		Recipients: recipients,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := proposerServer.PrepareBeaconProposer(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestProposer_SubmitValidatorRegistrations(t *testing.T) {
	ctx := context.Background()
	proposerServer := &Server{}
	reg := &ethpb.SignedValidatorRegistrationsV1{}
	_, err := proposerServer.SubmitValidatorRegistrations(ctx, reg)
	require.ErrorContains(t, builder.ErrNoBuilder.Error(), err)
	proposerServer = &Server{BlockBuilder: &builderTest.MockBuilderService{}}
	_, err = proposerServer.SubmitValidatorRegistrations(ctx, reg)
	require.ErrorContains(t, builder.ErrNoBuilder.Error(), err)
	proposerServer = &Server{BlockBuilder: &builderTest.MockBuilderService{HasConfigured: true}}
	_, err = proposerServer.SubmitValidatorRegistrations(ctx, reg)
	require.NoError(t, err)
	proposerServer = &Server{BlockBuilder: &builderTest.MockBuilderService{HasConfigured: true, ErrRegisterValidator: errors.New("bad")}}
	_, err = proposerServer.SubmitValidatorRegistrations(ctx, reg)
	require.ErrorContains(t, "bad", err)
}

func emptyPayloadCapella() *enginev1.ExecutionPayloadCapella {
	return &enginev1.ExecutionPayloadCapella{
		ParentHash:    make([]byte, fieldparams.RootLength),
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:    make([]byte, fieldparams.RootLength),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
		Transactions:  make([][]byte, 0),
		Withdrawals:   make([]*enginev1.Withdrawal, 0),
	}
}

func emptyPayloadDeneb() *enginev1.ExecutionPayloadDeneb {
	return &enginev1.ExecutionPayloadDeneb{
		ParentHash:    make([]byte, fieldparams.RootLength),
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:    make([]byte, fieldparams.RootLength),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
		Transactions:  make([][]byte, 0),
		Withdrawals:   make([]*enginev1.Withdrawal, 0),
	}
}
