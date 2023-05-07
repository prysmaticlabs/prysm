package blockchain

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	prysmTime "github.com/prysmaticlabs/prysm/v4/time"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var (
	_ = AttestationReceiver(&Service{})
	_ = AttestationStateFetcher(&Service{})
)

func TestAttestationCheckPtState_FarFutureSlot(t *testing.T) {
	helpers.ClearCache()
	service, _ := minimalTestService(t)

	service.genesisTime = time.Now()

	e := primitives.Epoch(slots.MaxSlotBuffer/uint64(params.BeaconConfig().SlotsPerEpoch) + 1)
	_, err := service.AttestationTargetState(context.Background(), &ethpb.Checkpoint{Epoch: e})
	require.ErrorContains(t, "exceeds max allowed value relative to the local clock", err)
}

func TestVerifyLMDFFGConsistent_NotOK(t *testing.T) {
	service, tr := minimalTestService(t)
	ctx := tr.ctx

	b32 := util.NewBeaconBlock()
	b32.Block.Slot = 32
	util.SaveBlock(t, ctx, service.cfg.BeaconDB, b32)
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)
	b33 := util.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	util.SaveBlock(t, ctx, service.cfg.BeaconDB, b33)
	r33, err := b33.Block.HashTreeRoot()
	require.NoError(t, err)

	wanted := "FFG and LMD votes are not consistent"
	a := util.NewAttestation()
	a.Data.Target.Epoch = 1
	a.Data.Target.Root = []byte{'a'}
	a.Data.BeaconBlockRoot = r33[:]
	require.ErrorContains(t, wanted, service.VerifyLmdFfgConsistency(context.Background(), a))
}

func TestVerifyLMDFFGConsistent_OK(t *testing.T) {
	service, tr := minimalTestService(t)
	ctx := tr.ctx

	b32 := util.NewBeaconBlock()
	b32.Block.Slot = 32
	util.SaveBlock(t, ctx, service.cfg.BeaconDB, b32)
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)
	b33 := util.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	util.SaveBlock(t, ctx, service.cfg.BeaconDB, b33)
	r33, err := b33.Block.HashTreeRoot()
	require.NoError(t, err)

	a := util.NewAttestation()
	a.Data.Target.Epoch = 1
	a.Data.Target.Root = r32[:]
	a.Data.BeaconBlockRoot = r33[:]
	err = service.VerifyLmdFfgConsistency(context.Background(), a)
	require.NoError(t, err, "Could not verify LMD and FFG votes to be consistent")
}

func TestProcessAttestations_Ok(t *testing.T) {
	service, tr := minimalTestService(t)
	hook := logTest.NewGlobal()
	ctx := tr.ctx

	service.genesisTime = prysmTime.Now().Add(-1 * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second)
	genesisState, pks := util.DeterministicGenesisState(t, 64)
	require.NoError(t, genesisState.SetGenesisTime(uint64(prysmTime.Now().Unix())-params.BeaconConfig().SecondsPerSlot))
	require.NoError(t, service.saveGenesisData(ctx, genesisState))
	atts, err := util.GenerateAttestations(genesisState, pks, 1, 0, false)
	require.NoError(t, err)
	tRoot := bytesutil.ToBytes32(atts[0].Data.Target.Root)
	copied := genesisState.Copy()
	copied, err = transition.ProcessSlots(ctx, copied, 1)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, copied, tRoot))
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	state, blkRoot, err := prepareForkchoiceState(ctx, 0, tRoot, tRoot, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, service.cfg.ForkChoiceStore.InsertNode(ctx, state, blkRoot))
	require.NoError(t, service.cfg.AttPool.SaveForkchoiceAttestations(atts))
	service.processAttestations(ctx, 0)
	require.Equal(t, 0, len(service.cfg.AttPool.ForkchoiceAttestations()))
	require.LogsDoNotContain(t, hook, "Could not process attestation for fork choice")
}

func TestService_ProcessAttestationsAndUpdateHead(t *testing.T) {
	service, tr := minimalTestService(t)
	ctx, fcs := tr.ctx, tr.fcs

	service.genesisTime = prysmTime.Now().Add(-2 * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second)
	genesisState, pks := util.DeterministicGenesisState(t, 64)
	require.NoError(t, service.saveGenesisData(ctx, genesisState))
	ojc := &ethpb.Checkpoint{Epoch: 0, Root: service.originBlockRoot[:]}
	require.NoError(t, fcs.UpdateJustifiedCheckpoint(ctx, &forkchoicetypes.Checkpoint{Epoch: 0, Root: service.originBlockRoot}))
	copied := genesisState.Copy()
	// Generate a new block for attesters to attest
	blk, err := util.GenerateFullBlock(copied, pks, util.DefaultBlockGenConfig(), 1)
	require.NoError(t, err)
	tRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := blocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)
	require.NoError(t, service.onBlock(ctx, wsb, tRoot))
	copied, err = service.cfg.StateGen.StateByRoot(ctx, tRoot)
	require.NoError(t, err)
	require.Equal(t, 2, fcs.NodeCount())
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb))

	// Generate attestations for this block in Slot 1
	atts, err := util.GenerateAttestations(copied, pks, 1, 1, false)
	require.NoError(t, err)
	require.NoError(t, service.cfg.AttPool.SaveForkchoiceAttestations(atts))
	// Verify the target is in forkchoice
	require.Equal(t, true, fcs.HasNode(bytesutil.ToBytes32(atts[0].Data.BeaconBlockRoot)))
	require.Equal(t, tRoot, bytesutil.ToBytes32(atts[0].Data.BeaconBlockRoot))
	require.Equal(t, true, fcs.HasNode(service.originBlockRoot))

	// Insert a new block to forkchoice
	b, err := util.GenerateFullBlock(genesisState, pks, util.DefaultBlockGenConfig(), 2)
	require.NoError(t, err)
	b.Block.ParentRoot = service.originBlockRoot[:]
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, ctx, service.cfg.BeaconDB, b)
	state, blkRoot, err := prepareForkchoiceState(ctx, 2, r, service.originBlockRoot, [32]byte{'b'}, ojc, ojc)
	require.NoError(t, err)
	require.NoError(t, service.cfg.ForkChoiceStore.InsertNode(ctx, state, blkRoot))
	require.Equal(t, 3, fcs.NodeCount())
	service.head.root = r // Old head

	require.Equal(t, 1, len(service.cfg.AttPool.ForkchoiceAttestations()))
	service.UpdateHead(ctx, 0)
	require.Equal(t, tRoot, service.headRoot())
	require.Equal(t, 0, len(service.cfg.AttPool.ForkchoiceAttestations())) // Validate att pool is empty
}

func TestService_UpdateHead_NoAtts(t *testing.T) {
	service, tr := minimalTestService(t)
	ctx, fcs := tr.ctx, tr.fcs

	service.genesisTime = prysmTime.Now().Add(-2 * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second)
	genesisState, pks := util.DeterministicGenesisState(t, 64)
	require.NoError(t, service.saveGenesisData(ctx, genesisState))
	require.NoError(t, fcs.UpdateJustifiedCheckpoint(ctx, &forkchoicetypes.Checkpoint{Epoch: 0, Root: service.originBlockRoot}))
	copied := genesisState.Copy()
	// Generate a new block
	blk, err := util.GenerateFullBlock(copied, pks, util.DefaultBlockGenConfig(), 1)
	require.NoError(t, err)
	tRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := blocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)
	require.NoError(t, service.onBlock(ctx, wsb, tRoot))
	require.Equal(t, 2, fcs.NodeCount())
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb))
	require.Equal(t, tRoot, service.head.root)

	// Insert a new block to forkchoice
	ojc := &ethpb.Checkpoint{Epoch: 0, Root: params.BeaconConfig().ZeroHash[:]}
	b, err := util.GenerateFullBlock(genesisState, pks, util.DefaultBlockGenConfig(), 2)
	require.NoError(t, err)
	b.Block.ParentRoot = service.originBlockRoot[:]
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, ctx, service.cfg.BeaconDB, b)
	state, blkRoot, err := prepareForkchoiceState(ctx, 2, r, service.originBlockRoot, [32]byte{'b'}, ojc, ojc)
	require.NoError(t, err)
	require.NoError(t, service.cfg.ForkChoiceStore.InsertNode(ctx, state, blkRoot))
	require.Equal(t, 3, fcs.NodeCount())

	require.Equal(t, 0, service.cfg.AttPool.ForkchoiceAttestationCount())
	service.UpdateHead(ctx, 0)
	require.Equal(t, r, service.headRoot())

	require.Equal(t, 0, len(service.cfg.AttPool.ForkchoiceAttestations())) // Validate att pool is empty
}
