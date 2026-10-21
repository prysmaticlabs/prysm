package blockchain

import (
	"testing"
	"time"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/heze"
	mockExecution "github.com/OffchainLabs/prysm/v7/beacon-chain/execution/testing"
	state_native "github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native"
	"github.com/OffchainLabs/prysm/v7/config/features"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	enginev1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/OffchainLabs/prysm/v7/testing/require"
	"github.com/OffchainLabs/prysm/v7/testing/util"
	"github.com/OffchainLabs/prysm/v7/time/slots"
)

func TestRefreshInclusionListPayload(t *testing.T) {
	// Build for every proposer, so the test does not depend on which validator proposes the next slot.
	resetCfg := features.InitWithReset(&features.Flags{PrepareAllPayloads: true})
	defer resetCfg()

	oldPid := primitives.PayloadID{9, 9, 9, 9, 9, 9, 9, 9}
	newPid := &enginev1.PayloadIDBytes{1, 2, 3, 4, 5, 6, 7, 8}
	slotDuration := params.BeaconConfig().SlotDuration()

	t.Run("heze restarts the build with the inclusion lists of the slot", func(t *testing.T) {
		setHezeAtGenesis(t)
		headRoot := bytesutil.ToBytes32([]byte("heze-headroot"))

		// The head is the block of the current slot, and this node proposes the next slot.
		headSlot := 2 * params.BeaconConfig().SlotsPerEpoch
		nextSlot := headSlot + 1
		gloasSt, _ := util.DeterministicGenesisStateGloas(t, 64)
		require.NoError(t, gloasSt.SetDepositRequestsStartIndex(gloasSt.Eth1DepositIndex()))
		st, err := heze.UpgradeToHeze(gloasSt)
		require.NoError(t, err)
		require.NoError(t, st.SetSlot(headSlot))
		dependentRoot := [32]byte{'d'}
		dependentSlot, err := slots.ShufflingDependentSlot(slots.ToEpoch(headSlot))
		require.NoError(t, err)
		require.NoError(t, st.UpdateBlockRootAtIndex(uint64(dependentSlot%params.BeaconConfig().SlotsPerHistoricalRoot), dependentRoot))

		engine := &mockExecution.EngineClient{PayloadIDBytes: newPid}
		service, tr := setupGloasService(t, engine)
		service.head = &head{root: headRoot, state: st, slot: headSlot}
		service.SetGenesisTime(time.Now().Add(-(time.Duration(headSlot)*slotDuration + slotDuration/2)))
		service.SetForkChoiceGenesisTime(service.genesisTime)
		require.Equal(t, headSlot, service.CurrentSlot())

		// The build that started when the head was set saw none of the inclusion lists below.
		service.cfg.PayloadIDCache.Set(nextSlot, headRoot, false, oldPid)
		for i, l := range []struct {
			tx     string
			timely bool
		}{
			{tx: "timely-tx", timely: true},
			{tx: "late-tx", timely: false},
		} {
			il := &ethpb.InclusionList{Slot: headSlot, ValidatorIndex: primitives.ValidatorIndex(i), DependentRoot: dependentRoot[:], Transactions: ilTransactions(t, []byte(l.tx))}
			service.InclusionListStore().Process(&ethpb.SignedInclusionList{Message: il, Signature: make([]byte, 96)}, l.timely)
		}

		service.refreshInclusionListPayload(tr.ctx)

		require.Equal(t, version.Heze, engine.FetchedAttributes.Version())
		v5, err := engine.FetchedAttributes.PbV5()
		require.NoError(t, err)
		require.DeepEqual(t, [][]byte{[]byte("timely-tx"), []byte("late-tx")}, v5.InclusionListTransactions)
		cachedPid, has := service.cfg.PayloadIDCache.PayloadID(nextSlot, headRoot, false)
		require.Equal(t, true, has)
		require.Equal(t, primitives.PayloadID(newPid[:]), cachedPid)
	})

	t.Run("pre-heze keeps the build", func(t *testing.T) {
		params.SetupTestConfigCleanup(t)
		cfg := params.BeaconConfig().Copy()
		cfg.HezeForkEpoch = 100
		cfg.InitializeForkSchedule()
		params.OverrideBeaconConfig(cfg)

		headRoot := bytesutil.ToBytes32([]byte("headroot"))

		engine := &mockExecution.EngineClient{PayloadIDBytes: newPid}
		service, tr := setupGloasService(t, engine)
		blockHash := bytesutil.ToBytes32([]byte("hash1"))
		base, blk := testGloasState(t, 1, params.BeaconConfig().ZeroHash, blockHash)
		base.LatestBlockHash = blockHash[:]
		st, err := state_native.InitializeFromProtoUnsafeGloas(base)
		require.NoError(t, err)
		insertGloasBlock(t, service, base, blk, headRoot)
		service.head = &head{root: headRoot, state: st, slot: 1}
		service.SetGenesisTime(time.Now().Add(-3 * slotDuration / 2))
		service.SetForkChoiceGenesisTime(service.genesisTime)
		service.cfg.PayloadIDCache.Set(service.CurrentSlot()+1, headRoot, false, oldPid)

		service.refreshInclusionListPayload(tr.ctx)

		require.IsNil(t, engine.FetchedAttributes)
		cachedPid, has := service.cfg.PayloadIDCache.PayloadID(service.CurrentSlot()+1, headRoot, false)
		require.Equal(t, true, has)
		require.Equal(t, oldPid, cachedPid)
	})
}
