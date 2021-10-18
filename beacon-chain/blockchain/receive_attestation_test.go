package blockchain

import (
	"context"
	"testing"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
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
	beaconDB := testDB.SetupDB(t)

	cfg := &config{BeaconDB: beaconDB, ForkChoiceStore: protoarray.New(0, 0, [32]byte{})}
	service, err := NewService(ctx)
	require.NoError(t, err)
	service.cfg = cfg

	b32 := util.NewBeaconBlock()
	b32.Block.Slot = 32
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b32)))
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)
	b33 := util.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b33)))
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
	beaconDB := testDB.SetupDB(t)

	cfg := &config{BeaconDB: beaconDB, ForkChoiceStore: protoarray.New(0, 0, [32]byte{})}
	service, err := NewService(ctx)
	require.NoError(t, err)
	service.cfg = cfg

	b32 := util.NewBeaconBlock()
	b32.Block.Slot = 32
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b32)))
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)
	b33 := util.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b33)))
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
	beaconDB := testDB.SetupDB(t)

	cfg := &config{
		BeaconDB:        beaconDB,
		ForkChoiceStore: protoarray.New(0, 0, [32]byte{}),
		StateGen:        stategen.New(beaconDB),
		AttPool:         attestations.NewPool(),
	}
	service, err := NewService(ctx)
	require.NoError(t, err)
	service.genesisTime = prysmTime.Now().Add(-1 * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second)
	service.cfg = cfg
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
	require.NoError(t, service.cfg.ForkChoiceStore.ProcessBlock(ctx, 0, tRoot, tRoot, tRoot, 1, 1))
	require.NoError(t, service.cfg.AttPool.SaveForkchoiceAttestations(atts))
	service.processAttestations(ctx)
	require.Equal(t, 0, len(service.cfg.AttPool.ForkchoiceAttestations()))
	require.LogsDoNotContain(t, hook, "Could not process attestation for fork choice")
}
