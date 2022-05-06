package blockchain

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	prysmTime "github.com/prysmaticlabs/prysm/time"
	"github.com/prysmaticlabs/prysm/time/slots"
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
	wsb, err := wrapper.WrappedSignedBeaconBlock(b32)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb))
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)
	b33 := util.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	wsb, err = wrapper.WrappedSignedBeaconBlock(b33)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb))
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
	wsb, err := wrapper.WrappedSignedBeaconBlock(b32)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb))
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)
	b33 := util.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	wsb, err = wrapper.WrappedSignedBeaconBlock(b33)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb))
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
	require.NoError(t, service.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 0, tRoot, tRoot, params.BeaconConfig().ZeroHash, 1, 1))
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
	service.notifyEngineIfChangedHead(ctx, service.headRoot())
	hookErr := "could not notify forkchoice update"
	finalizedErr := "could not get finalized checkpoint"
	require.LogsDoNotContain(t, hook, finalizedErr)
	require.LogsDoNotContain(t, hook, hookErr)
	gb, err := wrapper.WrappedSignedBeaconBlock(util.NewBeaconBlock())
	require.NoError(t, err)
	service.saveInitSyncBlock([32]byte{'a'}, gb)
	service.notifyEngineIfChangedHead(ctx, [32]byte{'a'})
	require.LogsContain(t, hook, finalizedErr)

	hook.Reset()
	service.head = &head{
		root:  [32]byte{'a'},
		block: nil, /* should not panic if notify head uses correct head */
	}

	// Block in Cache
	b := util.NewBeaconBlock()
	b.Block.Slot = 2
	wsb, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	r1, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	service.saveInitSyncBlock(r1, wsb)
	finalized := &ethpb.Checkpoint{Root: r1[:], Epoch: 0}
	st, _ := util.DeterministicGenesisState(t, 1)
	service.head = &head{
		slot:  1,
		root:  r1,
		block: wsb,
		state: st,
	}
	service.cfg.ProposerSlotIndexCache.SetProposerAndPayloadIDs(2, 1, [8]byte{1})
	service.store.SetFinalizedCheckpt(finalized)
	service.notifyEngineIfChangedHead(ctx, r1)
	require.LogsDoNotContain(t, hook, finalizedErr)
	require.LogsDoNotContain(t, hook, hookErr)

	// Block in DB
	b = util.NewBeaconBlock()
	b.Block.Slot = 3
	wsb, err = wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb))
	r1, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	finalized = &ethpb.Checkpoint{Root: r1[:], Epoch: 0}
	st, _ = util.DeterministicGenesisState(t, 1)
	service.head = &head{
		slot:  1,
		root:  r1,
		block: wsb,
		state: st,
	}
	service.cfg.ProposerSlotIndexCache.SetProposerAndPayloadIDs(2, 1, [8]byte{1})
	service.store.SetFinalizedCheckpt(finalized)
	service.notifyEngineIfChangedHead(ctx, r1)
	require.LogsDoNotContain(t, hook, finalizedErr)
	require.LogsDoNotContain(t, hook, hookErr)
	vId, payloadID, has := service.cfg.ProposerSlotIndexCache.GetProposerPayloadIDs(2)
	require.Equal(t, true, has)
	require.Equal(t, types.ValidatorIndex(1), vId)
	require.Equal(t, [8]byte{1}, payloadID)
}

func TestService_ProcessAttestationsAndUpdateHead(t *testing.T) {
	ctx := context.Background()
	opts := testServiceOptsWithDB(t)
	opts = append(opts, WithAttestationPool(attestations.NewPool()), WithStateNotifier(&mockBeaconNode{}))

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
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Root: tRoot[:]}))
	require.NoError(t, service.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 0, tRoot, tRoot, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, service.cfg.AttPool.SaveForkchoiceAttestations(atts))
	b := util.NewBeaconBlock()
	wb, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wb))
	service.head.root = r // Old head
	require.Equal(t, 1, len(service.cfg.AttPool.ForkchoiceAttestations()))
	require.NoError(t, err, service.ProcessAttestationsAndUpdateHead(ctx))
	require.Equal(t, tRoot, service.head.root)                             // Validate head is the new one
	require.Equal(t, 0, len(service.cfg.AttPool.ForkchoiceAttestations())) // Validate att pool is empty
}
