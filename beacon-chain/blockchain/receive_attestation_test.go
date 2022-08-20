package blockchain

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	testDB "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	prysmTime "github.com/prysmaticlabs/prysm/v3/time"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var (
	_ = AttestationReceiver(&Service{})
	_ = AttestationStateFetcher(&Service{})
)

func TestAttestationCheckPtState_FarFutureSlot(t *testing.T) {
	helpers.ClearCache()
	beaconDB := testDB.SetupDB(t)

	chainService := setupBeaconChain(t, beaconDB)
	chainService.genesisTime = time.Now()

	e := types.Epoch(slots.MaxSlotBuffer/uint64(params.BeaconConfig().SlotsPerEpoch) + 1)
	_, err := chainService.AttestationTargetState(context.Background(), &ethpb.Checkpoint{Epoch: e})
	require.ErrorContains(t, "exceeds max allowed value relative to the local clock", err)
}

func TestVerifyLMDFFGConsistent_NotOK(t *testing.T) {
	ctx := context.Background()
	opts := testServiceOptsWithDB(t)

	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

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
	ctx := context.Background()

	opts := testServiceOptsWithDB(t)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

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
	hook := logTest.NewGlobal()
	ctx := context.Background()
	opts := testServiceOptsWithDB(t)
	opts = append(opts, WithAttestationPool(attestations.NewPool()))

	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
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
	service.processAttestations(ctx)
	require.Equal(t, 0, len(service.cfg.AttPool.ForkchoiceAttestations()))
	require.LogsDoNotContain(t, hook, "Could not process attestation for fork choice")
}

func TestNotifyEngineIfChangedHead(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	opts := testServiceOptsWithDB(t)

	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	service.cfg.ProposerSlotIndexCache = cache.NewProposerPayloadIDsCache()
	require.NoError(t, service.notifyEngineIfChangedHead(ctx, service.headRoot()))
	hookErr := "could not notify forkchoice update"
	invalidStateErr := "Could not get state from db"
	require.LogsDoNotContain(t, hook, invalidStateErr)
	require.LogsDoNotContain(t, hook, hookErr)
	gb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
	require.NoError(t, err)
	require.NoError(t, service.saveInitSyncBlock(ctx, [32]byte{'a'}, gb))
	require.NoError(t, service.notifyEngineIfChangedHead(ctx, [32]byte{'a'}))
	require.LogsContain(t, hook, invalidStateErr)

	hook.Reset()
	service.head = &head{
		root:  [32]byte{'a'},
		block: nil, /* should not panic if notify head uses correct head */
	}

	// Block in Cache
	b := util.NewBeaconBlock()
	b.Block.Slot = 2
	wsb, err := blocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	r1, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.saveInitSyncBlock(ctx, r1, wsb))
	st, _ := util.DeterministicGenesisState(t, 1)
	service.head = &head{
		slot:  1,
		root:  r1,
		block: wsb,
		state: st,
	}
	service.cfg.ProposerSlotIndexCache.SetProposerAndPayloadIDs(2, 1, [8]byte{1}, [32]byte{2})
	require.NoError(t, service.notifyEngineIfChangedHead(ctx, r1))
	require.LogsDoNotContain(t, hook, invalidStateErr)
	require.LogsDoNotContain(t, hook, hookErr)

	// Block in DB
	b = util.NewBeaconBlock()
	b.Block.Slot = 3
	util.SaveBlock(t, ctx, service.cfg.BeaconDB, b)
	r1, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	st, _ = util.DeterministicGenesisState(t, 1)
	service.head = &head{
		slot:  1,
		root:  r1,
		block: wsb,
		state: st,
	}
	service.cfg.ProposerSlotIndexCache.SetProposerAndPayloadIDs(2, 1, [8]byte{1}, [32]byte{2})
	require.NoError(t, service.notifyEngineIfChangedHead(ctx, r1))
	require.LogsDoNotContain(t, hook, invalidStateErr)
	require.LogsDoNotContain(t, hook, hookErr)
	vId, payloadID, has := service.cfg.ProposerSlotIndexCache.GetProposerPayloadIDs(2, [32]byte{2})
	require.Equal(t, true, has)
	require.Equal(t, types.ValidatorIndex(1), vId)
	require.Equal(t, [8]byte{1}, payloadID)

	// Test zero headRoot returns immediately.
	headRoot := service.headRoot()
	require.NoError(t, service.notifyEngineIfChangedHead(ctx, [32]byte{}))
	require.Equal(t, service.headRoot(), headRoot)
}

func TestService_ProcessAttestationsAndUpdateHead(t *testing.T) {
	ctx := context.Background()
	opts := testServiceOptsWithDB(t)
	fcs := doublylinkedtree.New()
	opts = append(opts,
		WithAttestationPool(attestations.NewPool()),
		WithStateNotifier(&mockBeaconNode{}),
		WithForkChoiceStore(fcs),
	)

	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	service.genesisTime = prysmTime.Now().Add(-2 * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second)
	genesisState, pks := util.DeterministicGenesisState(t, 64)
	require.NoError(t, service.saveGenesisData(ctx, genesisState))
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

	// Generate attestatios for this block in Slot 1
	atts, err := util.GenerateAttestations(copied, pks, 1, 1, false)
	require.NoError(t, err)
	require.NoError(t, service.cfg.AttPool.SaveForkchoiceAttestations(atts))
	// Verify the target is in forchoice
	require.Equal(t, true, fcs.HasNode(bytesutil.ToBytes32(atts[0].Data.BeaconBlockRoot)))

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
	service.head.root = r // Old head

	require.Equal(t, 1, len(service.cfg.AttPool.ForkchoiceAttestations()))
	require.NoError(t, err, service.UpdateHead(ctx))

	require.Equal(t, 0, len(service.cfg.AttPool.ForkchoiceAttestations())) // Validate att pool is empty
	require.Equal(t, tRoot, service.head.root)                             // Validate head is the new one
}
