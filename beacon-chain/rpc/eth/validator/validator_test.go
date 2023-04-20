package validator

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/go-bitfield"
	mockChain "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	builderTest "github.com/prysmaticlabs/prysm/v4/beacon-chain/builder/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	coreTime "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	dbutil "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	mockExecution "github.com/prysmaticlabs/prysm/v4/beacon-chain/execution/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/blstoexec"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/synccommittee"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/voluntaryexits"
	p2pmock "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/testing"
	p2pType "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/types"
	v1alpha1validator "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/prysm/v1alpha1/validator"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stategen"
	mockSync "github.com/prysmaticlabs/prysm/v4/beacon-chain/sync/initial-sync/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"google.golang.org/protobuf/proto"
)

func TestGetAttesterDuties(t *testing.T) {
	ctx := context.Background()
	genesis := util.NewBeaconBlock()
	depChainStart := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, err := util.DeterministicDepositsAndKeys(depChainStart)
	require.NoError(t, err)
	eth1Data, err := util.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bs, err := transition.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
	require.NoError(t, err, "Could not set up genesis state")
	// Set state to non-epoch start slot.
	require.NoError(t, bs.SetSlot(5))
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	roots := make([][]byte, fieldparams.BlockRootsLength)
	roots[0] = genesisRoot[:]
	require.NoError(t, bs.SetBlockRoots(roots))
	db := dbutil.SetupDB(t)

	// Deactivate last validator.
	vals := bs.Validators()
	vals[len(vals)-1].ExitEpoch = 0
	require.NoError(t, bs.SetValidators(vals))

	pubKeys := make([][]byte, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = deposits[i].Data.PublicKey
	}

	// nextEpochState must not be used for committee calculations when requesting next epoch
	nextEpochState := bs.Copy()
	require.NoError(t, nextEpochState.SetSlot(params.BeaconConfig().SlotsPerEpoch))
	require.NoError(t, nextEpochState.SetValidators(vals[:512]))

	chainSlot := primitives.Slot(0)
	chain := &mockChain.ChainService{
		State: bs, Root: genesisRoot[:], Slot: &chainSlot,
	}
	vs := &Server{
		Stater: &testutil.MockStater{
			StatesBySlot: map[primitives.Slot]state.BeaconState{
				0:                                   bs,
				params.BeaconConfig().SlotsPerEpoch: nextEpochState,
			},
		},
		TimeFetcher:           chain,
		SyncChecker:           &mockSync.Sync{IsSyncing: false},
		OptimisticModeFetcher: chain,
	}

	t.Run("Single validator", func(t *testing.T) {
		req := &ethpbv1.AttesterDutiesRequest{
			Epoch: 0,
			Index: []primitives.ValidatorIndex{0},
		}
		resp, err := vs.GetAttesterDuties(ctx, req)
		require.NoError(t, err)
		assert.DeepEqual(t, genesisRoot[:], resp.DependentRoot)
		require.Equal(t, 1, len(resp.Data))
		duty := resp.Data[0]
		assert.Equal(t, primitives.CommitteeIndex(1), duty.CommitteeIndex)
		assert.Equal(t, primitives.Slot(0), duty.Slot)
		assert.Equal(t, primitives.ValidatorIndex(0), duty.ValidatorIndex)
		assert.DeepEqual(t, pubKeys[0], duty.Pubkey)
		assert.Equal(t, uint64(171), duty.CommitteeLength)
		assert.Equal(t, uint64(3), duty.CommitteesAtSlot)
		assert.Equal(t, primitives.CommitteeIndex(80), duty.ValidatorCommitteeIndex)
	})

	t.Run("Multiple validators", func(t *testing.T) {
		req := &ethpbv1.AttesterDutiesRequest{
			Epoch: 0,
			Index: []primitives.ValidatorIndex{0, 1},
		}
		resp, err := vs.GetAttesterDuties(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 2, len(resp.Data))
	})

	t.Run("Next epoch", func(t *testing.T) {
		req := &ethpbv1.AttesterDutiesRequest{
			Epoch: slots.ToEpoch(bs.Slot()) + 1,
			Index: []primitives.ValidatorIndex{0},
		}
		resp, err := vs.GetAttesterDuties(ctx, req)
		require.NoError(t, err)
		assert.DeepEqual(t, genesisRoot[:], resp.DependentRoot)
		require.Equal(t, 1, len(resp.Data))
		duty := resp.Data[0]
		assert.Equal(t, primitives.CommitteeIndex(0), duty.CommitteeIndex)
		assert.Equal(t, primitives.Slot(62), duty.Slot)
		assert.Equal(t, primitives.ValidatorIndex(0), duty.ValidatorIndex)
		assert.DeepEqual(t, pubKeys[0], duty.Pubkey)
		assert.Equal(t, uint64(170), duty.CommitteeLength)
		assert.Equal(t, uint64(3), duty.CommitteesAtSlot)
		assert.Equal(t, primitives.CommitteeIndex(110), duty.ValidatorCommitteeIndex)
	})

	t.Run("Epoch out of bound", func(t *testing.T) {
		currentEpoch := slots.ToEpoch(bs.Slot())
		req := &ethpbv1.AttesterDutiesRequest{
			Epoch: currentEpoch + 2,
			Index: []primitives.ValidatorIndex{0},
		}
		_, err := vs.GetAttesterDuties(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, fmt.Sprintf("Request epoch %d can not be greater than next epoch %d", currentEpoch+2, currentEpoch+1), err)
	})

	t.Run("Validator index out of bound", func(t *testing.T) {
		req := &ethpbv1.AttesterDutiesRequest{
			Epoch: 0,
			Index: []primitives.ValidatorIndex{primitives.ValidatorIndex(len(pubKeys))},
		}
		_, err := vs.GetAttesterDuties(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Invalid validator index", err)
	})

	t.Run("Inactive validator - no duties", func(t *testing.T) {
		req := &ethpbv1.AttesterDutiesRequest{
			Epoch: 0,
			Index: []primitives.ValidatorIndex{primitives.ValidatorIndex(len(pubKeys) - 1)},
		}
		resp, err := vs.GetAttesterDuties(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 0, len(resp.Data))
	})

	t.Run("execution optimistic", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		blk.Block.Slot = 31
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		chainSlot := primitives.Slot(0)
		chain := &mockChain.ChainService{
			State: bs, Root: genesisRoot[:], Slot: &chainSlot, Optimistic: true,
		}
		vs := &Server{
			Stater:                &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{0: bs}},
			TimeFetcher:           chain,
			OptimisticModeFetcher: chain,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
		}
		req := &ethpbv1.AttesterDutiesRequest{
			Epoch: 0,
			Index: []primitives.ValidatorIndex{0},
		}
		resp, err := vs.GetAttesterDuties(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})
}

func TestGetAttesterDuties_SyncNotReady(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	chainService := &mockChain.ChainService{State: st}
	vs := &Server{
		SyncChecker:           &mockSync.Sync{IsSyncing: true},
		HeadFetcher:           chainService,
		TimeFetcher:           chainService,
		OptimisticModeFetcher: chainService,
	}
	_, err = vs.GetAttesterDuties(context.Background(), &ethpbv1.AttesterDutiesRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestGetProposerDuties(t *testing.T) {
	ctx := context.Background()
	genesis := util.NewBeaconBlock()
	depChainStart := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, err := util.DeterministicDepositsAndKeys(depChainStart)
	require.NoError(t, err)
	eth1Data, err := util.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	roots := make([][]byte, fieldparams.BlockRootsLength)
	roots[0] = genesisRoot[:]
	// We DON'T WANT this root to be returned when testing the next epoch
	roots[31] = []byte("next_epoch_dependent_root")
	db := dbutil.SetupDB(t)

	pubKeys := make([][]byte, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = deposits[i].Data.PublicKey
	}

	t.Run("Ok", func(t *testing.T) {
		bs, err := transition.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
		require.NoError(t, err, "Could not set up genesis state")
		require.NoError(t, bs.SetSlot(params.BeaconConfig().SlotsPerEpoch))
		require.NoError(t, bs.SetBlockRoots(roots))
		chainSlot := primitives.Slot(0)
		chain := &mockChain.ChainService{
			State: bs, Root: genesisRoot[:], Slot: &chainSlot,
		}
		vs := &Server{
			Stater:                 &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{0: bs}},
			HeadFetcher:            chain,
			TimeFetcher:            chain,
			OptimisticModeFetcher:  chain,
			SyncChecker:            &mockSync.Sync{IsSyncing: false},
			ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
		}

		req := &ethpbv1.ProposerDutiesRequest{
			Epoch: 0,
		}
		resp, err := vs.GetProposerDuties(ctx, req)
		require.NoError(t, err)
		assert.DeepEqual(t, genesisRoot[:], resp.DependentRoot)
		assert.Equal(t, 31, len(resp.Data))
		// We expect a proposer duty for slot 11.
		var expectedDuty *ethpbv1.ProposerDuty
		for _, duty := range resp.Data {
			if duty.Slot == 11 {
				expectedDuty = duty
			}
		}
		vid, _, has := vs.ProposerSlotIndexCache.GetProposerPayloadIDs(11, [32]byte{})
		require.Equal(t, true, has)
		require.Equal(t, primitives.ValidatorIndex(9982), vid)
		require.NotNil(t, expectedDuty, "Expected duty for slot 11 not found")
		assert.Equal(t, primitives.ValidatorIndex(9982), expectedDuty.ValidatorIndex)
		assert.DeepEqual(t, pubKeys[9982], expectedDuty.Pubkey)
	})

	t.Run("Next epoch", func(t *testing.T) {
		bs, err := transition.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
		require.NoError(t, err, "Could not set up genesis state")
		require.NoError(t, bs.SetBlockRoots(roots))
		chainSlot := primitives.Slot(0)
		chain := &mockChain.ChainService{
			State: bs, Root: genesisRoot[:], Slot: &chainSlot,
		}
		vs := &Server{
			Stater:                 &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{0: bs}},
			HeadFetcher:            chain,
			TimeFetcher:            chain,
			OptimisticModeFetcher:  chain,
			SyncChecker:            &mockSync.Sync{IsSyncing: false},
			ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
		}

		req := &ethpbv1.ProposerDutiesRequest{
			Epoch: 1,
		}
		resp, err := vs.GetProposerDuties(ctx, req)
		require.NoError(t, err)
		assert.DeepEqual(t, bytesutil.PadTo(genesisRoot[:], 32), resp.DependentRoot)
		assert.Equal(t, 32, len(resp.Data))
		// We expect a proposer duty for slot 43.
		var expectedDuty *ethpbv1.ProposerDuty
		for _, duty := range resp.Data {
			if duty.Slot == 43 {
				expectedDuty = duty
			}
		}
		vid, _, has := vs.ProposerSlotIndexCache.GetProposerPayloadIDs(43, [32]byte{})
		require.Equal(t, true, has)
		require.Equal(t, primitives.ValidatorIndex(4863), vid)
		require.NotNil(t, expectedDuty, "Expected duty for slot 43 not found")
		assert.Equal(t, primitives.ValidatorIndex(4863), expectedDuty.ValidatorIndex)
		assert.DeepEqual(t, pubKeys[4863], expectedDuty.Pubkey)
	})

	t.Run("Prune payload ID cache ok", func(t *testing.T) {
		bs, err := transition.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
		require.NoError(t, err, "Could not set up genesis state")
		require.NoError(t, bs.SetSlot(params.BeaconConfig().SlotsPerEpoch))
		require.NoError(t, bs.SetBlockRoots(roots))
		chainSlot := params.BeaconConfig().SlotsPerEpoch
		chain := &mockChain.ChainService{
			State: bs, Root: genesisRoot[:], Slot: &chainSlot,
		}
		vs := &Server{
			Stater:                 &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{params.BeaconConfig().SlotsPerEpoch: bs}},
			HeadFetcher:            chain,
			TimeFetcher:            chain,
			OptimisticModeFetcher:  chain,
			SyncChecker:            &mockSync.Sync{IsSyncing: false},
			ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
		}

		req := &ethpbv1.ProposerDutiesRequest{
			Epoch: 1,
		}
		vs.ProposerSlotIndexCache.SetProposerAndPayloadIDs(1, 1, [8]byte{1}, [32]byte{2})
		vs.ProposerSlotIndexCache.SetProposerAndPayloadIDs(31, 2, [8]byte{2}, [32]byte{3})
		vs.ProposerSlotIndexCache.SetProposerAndPayloadIDs(32, 4309, [8]byte{3}, [32]byte{4})

		_, err = vs.GetProposerDuties(ctx, req)
		require.NoError(t, err)

		vid, _, has := vs.ProposerSlotIndexCache.GetProposerPayloadIDs(1, [32]byte{})
		require.Equal(t, false, has)
		require.Equal(t, primitives.ValidatorIndex(0), vid)
		vid, _, has = vs.ProposerSlotIndexCache.GetProposerPayloadIDs(2, [32]byte{})
		require.Equal(t, false, has)
		require.Equal(t, primitives.ValidatorIndex(0), vid)
		vid, _, has = vs.ProposerSlotIndexCache.GetProposerPayloadIDs(32, [32]byte{})
		require.Equal(t, true, has)
		require.Equal(t, primitives.ValidatorIndex(4309), vid)
	})

	t.Run("Epoch out of bound", func(t *testing.T) {
		bs, err := transition.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
		require.NoError(t, err, "Could not set up genesis state")
		// Set state to non-epoch start slot.
		require.NoError(t, bs.SetSlot(5))
		require.NoError(t, bs.SetBlockRoots(roots))
		chainSlot := primitives.Slot(0)
		chain := &mockChain.ChainService{
			State: bs, Root: genesisRoot[:], Slot: &chainSlot,
		}
		vs := &Server{
			Stater:                 &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{0: bs}},
			HeadFetcher:            chain,
			TimeFetcher:            chain,
			OptimisticModeFetcher:  chain,
			SyncChecker:            &mockSync.Sync{IsSyncing: false},
			ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
		}

		currentEpoch := slots.ToEpoch(bs.Slot())
		req := &ethpbv1.ProposerDutiesRequest{
			Epoch: currentEpoch + 2,
		}
		_, err = vs.GetProposerDuties(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, fmt.Sprintf("Request epoch %d can not be greater than next epoch %d", currentEpoch+2, currentEpoch+1), err)
	})

	t.Run("execution optimistic", func(t *testing.T) {
		bs, err := transition.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
		require.NoError(t, err, "Could not set up genesis state")
		// Set state to non-epoch start slot.
		require.NoError(t, bs.SetSlot(5))
		require.NoError(t, bs.SetBlockRoots(roots))
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		blk.Block.Slot = 31
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		chainSlot := primitives.Slot(0)
		chain := &mockChain.ChainService{
			State: bs, Root: genesisRoot[:], Slot: &chainSlot, Optimistic: true,
		}
		vs := &Server{
			Stater:                 &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{0: bs}},
			HeadFetcher:            chain,
			TimeFetcher:            chain,
			OptimisticModeFetcher:  chain,
			SyncChecker:            &mockSync.Sync{IsSyncing: false},
			ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
		}
		req := &ethpbv1.ProposerDutiesRequest{
			Epoch: 0,
		}
		resp, err := vs.GetProposerDuties(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})
}

func TestGetProposerDuties_SyncNotReady(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	chainService := &mockChain.ChainService{State: st}
	vs := &Server{
		SyncChecker:           &mockSync.Sync{IsSyncing: true},
		HeadFetcher:           chainService,
		TimeFetcher:           chainService,
		OptimisticModeFetcher: chainService,
	}
	_, err = vs.GetProposerDuties(context.Background(), &ethpbv1.ProposerDutiesRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestGetSyncCommitteeDuties(t *testing.T) {
	ctx := context.Background()
	genesisTime := time.Now()
	numVals := uint64(11)
	st, _ := util.DeterministicGenesisStateAltair(t, numVals)
	require.NoError(t, st.SetGenesisTime(uint64(genesisTime.Unix())))
	vals := st.Validators()
	currCommittee := &ethpbalpha.SyncCommittee{}
	for i := 0; i < 5; i++ {
		currCommittee.Pubkeys = append(currCommittee.Pubkeys, vals[i].PublicKey)
		currCommittee.AggregatePubkey = make([]byte, 48)
	}
	// add one public key twice - this is needed for one of the test cases
	currCommittee.Pubkeys = append(currCommittee.Pubkeys, vals[0].PublicKey)
	require.NoError(t, st.SetCurrentSyncCommittee(currCommittee))
	nextCommittee := &ethpbalpha.SyncCommittee{}
	for i := 5; i < 10; i++ {
		nextCommittee.Pubkeys = append(nextCommittee.Pubkeys, vals[i].PublicKey)
		nextCommittee.AggregatePubkey = make([]byte, 48)

	}
	require.NoError(t, st.SetNextSyncCommittee(nextCommittee))

	mockChainService := &mockChain.ChainService{Genesis: genesisTime}
	vs := &Server{
		Stater:                &testutil.MockStater{BeaconState: st},
		SyncChecker:           &mockSync.Sync{IsSyncing: false},
		TimeFetcher:           mockChainService,
		HeadFetcher:           mockChainService,
		OptimisticModeFetcher: mockChainService,
	}

	t.Run("Single validator", func(t *testing.T) {
		req := &ethpbv2.SyncCommitteeDutiesRequest{
			Epoch: 0,
			Index: []primitives.ValidatorIndex{1},
		}
		resp, err := vs.GetSyncCommitteeDuties(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		require.Equal(t, 1, len(resp.Data))
		duty := resp.Data[0]
		assert.DeepEqual(t, vals[1].PublicKey, duty.Pubkey)
		assert.Equal(t, primitives.ValidatorIndex(1), duty.ValidatorIndex)
		require.Equal(t, 1, len(duty.ValidatorSyncCommitteeIndices))
		assert.Equal(t, uint64(1), duty.ValidatorSyncCommitteeIndices[0])
	})

	t.Run("Epoch not at period start", func(t *testing.T) {
		req := &ethpbv2.SyncCommitteeDutiesRequest{
			Epoch: 1,
			Index: []primitives.ValidatorIndex{1},
		}
		resp, err := vs.GetSyncCommitteeDuties(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		require.Equal(t, 1, len(resp.Data))
		duty := resp.Data[0]
		assert.DeepEqual(t, vals[1].PublicKey, duty.Pubkey)
		assert.Equal(t, primitives.ValidatorIndex(1), duty.ValidatorIndex)
		require.Equal(t, 1, len(duty.ValidatorSyncCommitteeIndices))
		assert.Equal(t, uint64(1), duty.ValidatorSyncCommitteeIndices[0])
	})

	t.Run("Multiple validators", func(t *testing.T) {
		req := &ethpbv2.SyncCommitteeDutiesRequest{
			Epoch: 0,
			Index: []primitives.ValidatorIndex{1, 2},
		}
		resp, err := vs.GetSyncCommitteeDuties(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 2, len(resp.Data))
	})

	t.Run("Validator without duty not returned", func(t *testing.T) {
		req := &ethpbv2.SyncCommitteeDutiesRequest{
			Epoch: 0,
			Index: []primitives.ValidatorIndex{1, 10},
		}
		resp, err := vs.GetSyncCommitteeDuties(ctx, req)
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Data))
		assert.Equal(t, primitives.ValidatorIndex(1), resp.Data[0].ValidatorIndex)
	})

	t.Run("Multiple indices for validator", func(t *testing.T) {
		req := &ethpbv2.SyncCommitteeDutiesRequest{
			Epoch: 0,
			Index: []primitives.ValidatorIndex{0},
		}
		resp, err := vs.GetSyncCommitteeDuties(ctx, req)
		require.NoError(t, err)
		duty := resp.Data[0]
		require.Equal(t, 2, len(duty.ValidatorSyncCommitteeIndices))
		assert.DeepEqual(t, []uint64{0, 5}, duty.ValidatorSyncCommitteeIndices)
	})

	t.Run("Validator index out of bound", func(t *testing.T) {
		req := &ethpbv2.SyncCommitteeDutiesRequest{
			Epoch: 0,
			Index: []primitives.ValidatorIndex{primitives.ValidatorIndex(numVals)},
		}
		_, err := vs.GetSyncCommitteeDuties(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Invalid validator index", err)
	})

	t.Run("next sync committee period", func(t *testing.T) {
		req := &ethpbv2.SyncCommitteeDutiesRequest{
			Epoch: params.BeaconConfig().EpochsPerSyncCommitteePeriod,
			Index: []primitives.ValidatorIndex{5},
		}
		resp, err := vs.GetSyncCommitteeDuties(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		require.Equal(t, 1, len(resp.Data))
		duty := resp.Data[0]
		assert.DeepEqual(t, vals[5].PublicKey, duty.Pubkey)
		assert.Equal(t, primitives.ValidatorIndex(5), duty.ValidatorIndex)
		require.Equal(t, 1, len(duty.ValidatorSyncCommitteeIndices))
		assert.Equal(t, uint64(0), duty.ValidatorSyncCommitteeIndices[0])
	})

	t.Run("epoch too far in the future", func(t *testing.T) {
		req := &ethpbv2.SyncCommitteeDutiesRequest{
			Epoch: params.BeaconConfig().EpochsPerSyncCommitteePeriod * 2,
			Index: []primitives.ValidatorIndex{5},
		}
		_, err := vs.GetSyncCommitteeDuties(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Epoch is too far in the future", err)
	})

	t.Run("correct sync committee is fetched", func(t *testing.T) {
		// in this test we swap validators in the current and next sync committee inside the new state

		newSyncPeriodStartSlot := primitives.Slot(uint64(params.BeaconConfig().EpochsPerSyncCommitteePeriod) * uint64(params.BeaconConfig().SlotsPerEpoch))
		newSyncPeriodSt, _ := util.DeterministicGenesisStateAltair(t, numVals)
		require.NoError(t, newSyncPeriodSt.SetSlot(newSyncPeriodStartSlot))
		require.NoError(t, newSyncPeriodSt.SetGenesisTime(uint64(genesisTime.Unix())))
		vals := newSyncPeriodSt.Validators()
		currCommittee := &ethpbalpha.SyncCommittee{}
		for i := 5; i < 10; i++ {
			currCommittee.Pubkeys = append(currCommittee.Pubkeys, vals[i].PublicKey)
			currCommittee.AggregatePubkey = make([]byte, 48)
		}
		require.NoError(t, newSyncPeriodSt.SetCurrentSyncCommittee(currCommittee))
		nextCommittee := &ethpbalpha.SyncCommittee{}
		for i := 0; i < 5; i++ {
			nextCommittee.Pubkeys = append(nextCommittee.Pubkeys, vals[i].PublicKey)
			nextCommittee.AggregatePubkey = make([]byte, 48)

		}
		require.NoError(t, newSyncPeriodSt.SetNextSyncCommittee(nextCommittee))

		stateFetchFn := func(slot primitives.Slot) state.BeaconState {
			if slot < newSyncPeriodStartSlot {
				return st
			} else {
				return newSyncPeriodSt
			}
		}
		mockChainService := &mockChain.ChainService{Genesis: genesisTime, Slot: &newSyncPeriodStartSlot}
		vs := &Server{
			Stater:                &testutil.MockStater{BeaconState: stateFetchFn(newSyncPeriodStartSlot)},
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			TimeFetcher:           mockChainService,
			HeadFetcher:           mockChainService,
			OptimisticModeFetcher: mockChainService,
		}

		req := &ethpbv2.SyncCommitteeDutiesRequest{
			Epoch: params.BeaconConfig().EpochsPerSyncCommitteePeriod,
			Index: []primitives.ValidatorIndex{8},
		}
		resp, err := vs.GetSyncCommitteeDuties(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		require.Equal(t, 1, len(resp.Data))
		duty := resp.Data[0]
		assert.DeepEqual(t, vals[8].PublicKey, duty.Pubkey)
		assert.Equal(t, primitives.ValidatorIndex(8), duty.ValidatorIndex)
		require.Equal(t, 1, len(duty.ValidatorSyncCommitteeIndices))
		assert.Equal(t, uint64(3), duty.ValidatorSyncCommitteeIndices[0])
	})

	t.Run("execution optimistic", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		require.NoError(t, db.SaveStateSummary(ctx, &ethpbalpha.StateSummary{Slot: 0, Root: []byte("root")}))
		require.NoError(t, db.SaveLastValidatedCheckpoint(ctx, &ethpbalpha.Checkpoint{Epoch: 0, Root: []byte("root")}))

		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		slot, err := slots.EpochStart(1)
		require.NoError(t, err)

		state, err := util.NewBeaconStateBellatrix()
		require.NoError(t, err)
		require.NoError(t, state.SetSlot(slot))

		mockChainService := &mockChain.ChainService{
			Genesis:    genesisTime,
			Optimistic: true,
			Slot:       &slot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{
				Root:  root[:],
				Epoch: 1,
			},
			State: state,
		}
		vs := &Server{
			Stater:                &testutil.MockStater{BeaconState: st},
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			TimeFetcher:           mockChainService,
			HeadFetcher:           mockChainService,
			OptimisticModeFetcher: mockChainService,
			ChainInfoFetcher:      mockChainService,
			BeaconDB:              db,
		}
		req := &ethpbv2.SyncCommitteeDutiesRequest{
			Epoch: 1,
			Index: []primitives.ValidatorIndex{1},
		}
		resp, err := vs.GetSyncCommitteeDuties(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})
}

func TestGetSyncCommitteeDuties_SyncNotReady(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	chainService := &mockChain.ChainService{State: st}
	vs := &Server{
		SyncChecker:           &mockSync.Sync{IsSyncing: true},
		HeadFetcher:           chainService,
		TimeFetcher:           chainService,
		OptimisticModeFetcher: chainService,
	}
	_, err = vs.GetSyncCommitteeDuties(context.Background(), &ethpbv2.SyncCommitteeDutiesRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestSyncCommitteeDutiesLastValidEpoch(t *testing.T) {
	t.Run("first epoch of current period", func(t *testing.T) {
		assert.Equal(t, params.BeaconConfig().EpochsPerSyncCommitteePeriod*2-1, syncCommitteeDutiesLastValidEpoch(0))
	})
	t.Run("last epoch of current period", func(t *testing.T) {
		assert.Equal(
			t,
			params.BeaconConfig().EpochsPerSyncCommitteePeriod*2-1,
			syncCommitteeDutiesLastValidEpoch(params.BeaconConfig().EpochsPerSyncCommitteePeriod-1),
		)
	})
}

func TestProduceBlockV2(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		beaconState, parentRoot, privKeys := util.DeterministicGenesisStateWithGenesisBlock(t, ctx, db, 64)

		mockChainService := &mockChain.ChainService{State: beaconState, Root: parentRoot[:], ForkChoiceStore: doublylinkedtree.New()}
		mockExecutionChain := &mockExecution.Chain{}
		v1Alpha1Server := &v1alpha1validator.Server{
			HeadFetcher:           mockChainService,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockReceiver:         mockChainService,
			ForkFetcher:           mockChainService,
			ForkchoiceFetcher:     mockChainService,
			ChainStartFetcher:     mockExecutionChain,
			Eth1InfoFetcher:       mockExecutionChain,
			Eth1BlockFetcher:      mockExecutionChain,
			MockEth1Votes:         true,
			AttPool:               attestations.NewPool(),
			SlashingsPool:         slashings.NewPool(),
			ExitPool:              voluntaryexits.NewPool(),
			StateGen:              stategen.New(db, mockChainService.ForkChoiceStore),
			TimeFetcher:           mockChainService,
			OptimisticModeFetcher: mockChainService,
		}
		require.NoError(t, db.SaveGenesisData(ctx, beaconState))

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
		for i := primitives.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
			proposerSlashing, err := util.GenerateProposerSlashingForValidator(
				beaconState,
				privKeys[i],
				i, /* validator index */
			)
			require.NoError(t, err)
			proposerSlashings[i] = proposerSlashing
			err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), beaconState, proposerSlashing)
			require.NoError(t, err)
		}

		attSlashings := make([]*ethpbalpha.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
		for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings; i++ {
			attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
				beaconState,
				privKeys[i+params.BeaconConfig().MaxProposerSlashings],
				primitives.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings), /* validator index */
			)
			require.NoError(t, err)
			attSlashings[i] = attesterSlashing
			err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), beaconState, attesterSlashing)
			require.NoError(t, err)
		}

		v1Server := &Server{
			V1Alpha1Server: v1Alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}
		randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))
		req := &ethpbv1.ProduceBlockRequest{
			Slot:         1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		resp, err := v1Server.ProduceBlockV2(ctx, req)
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_PHASE0, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BeaconBlockContainerV2_Phase0Block)
		require.Equal(t, true, ok)
		blk := containerBlock.Phase0Block
		assert.Equal(t, req.Slot, blk.Slot, "Expected block to have slot of 1")
		assert.DeepEqual(t, parentRoot[:], blk.ParentRoot, "Expected block to have correct parent root")
		assert.DeepEqual(t, randaoReveal, blk.Body.RandaoReveal, "Expected block to have correct randao reveal")
		assert.DeepEqual(t, req.Graffiti, blk.Body.Graffiti, "Expected block to have correct graffiti")

		assert.Equal(t, params.BeaconConfig().MaxProposerSlashings, uint64(len(blk.Body.ProposerSlashings)))
		expectedPropSlashings := make([]*ethpbv1.ProposerSlashing, len(proposerSlashings))
		for i, slash := range proposerSlashings {
			expectedPropSlashings[i] = migration.V1Alpha1ProposerSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedPropSlashings, blk.Body.ProposerSlashings)

		assert.Equal(t, params.BeaconConfig().MaxAttesterSlashings, uint64(len(blk.Body.AttesterSlashings)))
		expectedAttSlashings := make([]*ethpbv1.AttesterSlashing, len(attSlashings))
		for i, slash := range attSlashings {
			expectedAttSlashings[i] = migration.V1Alpha1AttSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedAttSlashings, blk.Body.AttesterSlashings)
	})

	t.Run("Altair", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		params.SetupTestConfigCleanup(t)
		bc := params.BeaconConfig().Copy()
		bc.AltairForkEpoch = primitives.Epoch(0)
		params.OverrideBeaconConfig(bc)

		beaconState, privKeys := util.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
		syncCommittee, err := altair.NextSyncCommittee(context.Background(), beaconState)
		require.NoError(t, err)
		require.NoError(t, beaconState.SetCurrentSyncCommittee(syncCommittee))
		require.NoError(t, beaconState.SetNextSyncCommittee(syncCommittee))

		stateRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err, "Could not hash genesis state")
		genesisBlock := util.NewBeaconBlockAltair()
		genesisBlock.Block.StateRoot = stateRoot[:]
		util.SaveBlock(t, ctx, db, genesisBlock)
		parentRoot, err := genesisBlock.Block.HashTreeRoot()
		require.NoError(t, err)

		require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
		require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

		mockChainService := &mockChain.ChainService{State: beaconState, Root: parentRoot[:], ForkChoiceStore: doublylinkedtree.New()}
		mockExecutionChain := &mockExecution.Chain{}
		v1Alpha1Server := &v1alpha1validator.Server{
			HeadFetcher:           mockChainService,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockReceiver:         mockChainService,
			ForkFetcher:           mockChainService,
			ForkchoiceFetcher:     mockChainService,
			ChainStartFetcher:     mockExecutionChain,
			Eth1InfoFetcher:       mockExecutionChain,
			Eth1BlockFetcher:      mockExecutionChain,
			MockEth1Votes:         true,
			AttPool:               attestations.NewPool(),
			SlashingsPool:         slashings.NewPool(),
			ExitPool:              voluntaryexits.NewPool(),
			StateGen:              stategen.New(db, doublylinkedtree.New()),
			SyncCommitteePool:     synccommittee.NewStore(),
			TimeFetcher:           mockChainService,
			OptimisticModeFetcher: mockChainService,
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
		for i := primitives.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
			proposerSlashing, err := util.GenerateProposerSlashingForValidator(
				beaconState,
				privKeys[i],
				i, /* validator index */
			)
			require.NoError(t, err)
			proposerSlashings[i] = proposerSlashing
			err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), beaconState, proposerSlashing)
			require.NoError(t, err)
		}

		attSlashings := make([]*ethpbalpha.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
		for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings; i++ {
			attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
				beaconState,
				privKeys[i+params.BeaconConfig().MaxProposerSlashings],
				primitives.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings), /* validator index */
			)
			require.NoError(t, err)
			attSlashings[i] = attesterSlashing
			err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), beaconState, attesterSlashing)
			require.NoError(t, err)
		}

		aggregationBits := bitfield.NewBitvector128()
		for i := range aggregationBits {
			aggregationBits[i] = 0xAA
		}

		syncCommitteeIndices, err := altair.NextSyncCommitteeIndices(context.Background(), beaconState)
		require.NoError(t, err)
		sigs := make([]bls.Signature, 0, len(syncCommitteeIndices))
		for i, indice := range syncCommitteeIndices {
			if aggregationBits.BitAt(uint64(i)) {
				b := p2pType.SSZBytes(parentRoot[:])
				sb, err := signing.ComputeDomainAndSign(beaconState, coreTime.CurrentEpoch(beaconState), &b, params.BeaconConfig().DomainSyncCommittee, privKeys[indice])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				sigs = append(sigs, sig)
			}
		}
		aggregatedSig := bls.AggregateSignatures(sigs).Marshal()
		contribution := &ethpbalpha.SyncCommitteeContribution{
			Slot:              0,
			BlockRoot:         parentRoot[:],
			SubcommitteeIndex: 0,
			AggregationBits:   aggregationBits,
			Signature:         aggregatedSig,
		}
		require.NoError(t, v1Alpha1Server.SyncCommitteePool.SaveSyncCommitteeContribution(contribution))

		v1Server := &Server{
			V1Alpha1Server: v1Alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}
		randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))

		req := &ethpbv1.ProduceBlockRequest{
			Slot:         1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		resp, err := v1Server.ProduceBlockV2(ctx, req)
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BeaconBlockContainerV2_AltairBlock)
		require.Equal(t, true, ok)
		blk := containerBlock.AltairBlock
		assert.Equal(t, req.Slot, blk.Slot, "Expected block to have slot of 1")
		assert.DeepEqual(t, parentRoot[:], blk.ParentRoot, "Expected block to have correct parent root")
		assert.DeepEqual(t, randaoReveal, blk.Body.RandaoReveal, "Expected block to have correct randao reveal")
		assert.DeepEqual(t, req.Graffiti, blk.Body.Graffiti, "Expected block to have correct graffiti")
		assert.Equal(t, params.BeaconConfig().MaxProposerSlashings, uint64(len(blk.Body.ProposerSlashings)))

		expectedPropSlashings := make([]*ethpbv1.ProposerSlashing, len(proposerSlashings))
		for i, slash := range proposerSlashings {
			expectedPropSlashings[i] = migration.V1Alpha1ProposerSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedPropSlashings, blk.Body.ProposerSlashings)

		assert.Equal(t, params.BeaconConfig().MaxAttesterSlashings, uint64(len(blk.Body.AttesterSlashings)))
		expectedAttSlashings := make([]*ethpbv1.AttesterSlashing, len(attSlashings))
		for i, slash := range attSlashings {
			expectedAttSlashings[i] = migration.V1Alpha1AttSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedAttSlashings, blk.Body.AttesterSlashings)

		expectedBits := bitfield.NewBitvector512()
		for i := 0; i <= 15; i++ {
			expectedBits[i] = 0xAA
		}
		assert.DeepEqual(t, expectedBits, blk.Body.SyncAggregate.SyncCommitteeBits)
		assert.DeepEqual(t, aggregatedSig, blk.Body.SyncAggregate.SyncCommitteeSignature)
	})

	t.Run("Bellatrix", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		params.SetupTestConfigCleanup(t)
		bc := params.BeaconConfig().Copy()
		bc.AltairForkEpoch = primitives.Epoch(0)
		bc.BellatrixForkEpoch = primitives.Epoch(1)
		params.OverrideBeaconConfig(bc)

		beaconState, privKeys := util.DeterministicGenesisStateBellatrix(t, params.BeaconConfig().SyncCommitteeSize)
		require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch))
		syncCommittee, err := altair.NextSyncCommittee(context.Background(), beaconState)
		require.NoError(t, err)
		require.NoError(t, beaconState.SetCurrentSyncCommittee(syncCommittee))
		require.NoError(t, beaconState.SetNextSyncCommittee(syncCommittee))
		require.NoError(t, beaconState.UpdateRandaoMixesAtIndex(1, bytesutil.PadTo([]byte("prev_randao"), 32)))

		payloadHeader, err := blocks.WrappedExecutionPayloadHeader(&enginev1.ExecutionPayloadHeader{
			ParentHash:   bytesutil.PadTo([]byte("parent_hash"), 32),
			FeeRecipient: bytesutil.PadTo([]byte("fee_recipient"), 20),
			StateRoot:    bytesutil.PadTo([]byte("state_root"), 32),
			ReceiptsRoot: bytesutil.PadTo([]byte("receipts_root"), 32),
			LogsBloom:    bytesutil.PadTo([]byte("logs_bloom"), 256),
			PrevRandao:   bytesutil.PadTo([]byte("prev_randao"), 32),
			BlockNumber:  123,
			GasLimit:     123,
			GasUsed:      123,
			// State time at slot 33
			Timestamp:     396,
			ExtraData:     bytesutil.PadTo([]byte("extra_data"), 32),
			BaseFeePerGas: bytesutil.PadTo([]byte("base_fee_per_gas"), 32),
			// Must be equal to payload.ParentHash
			BlockHash:        bytesutil.PadTo([]byte("equal_hash"), 32),
			TransactionsRoot: bytesutil.PadTo([]byte("transactions_root"), 32),
		})
		require.NoError(t, err)
		require.NoError(t, beaconState.SetLatestExecutionPayloadHeader(payloadHeader))

		payload := &enginev1.ExecutionPayload{
			// Must be equal to payloadHeader.BlockHash
			ParentHash:   bytesutil.PadTo([]byte("equal_hash"), 32),
			FeeRecipient: bytesutil.PadTo([]byte("fee_recipient"), 20),
			StateRoot:    bytesutil.PadTo([]byte("state_root"), 32),
			ReceiptsRoot: bytesutil.PadTo([]byte("receipts_root"), 32),
			LogsBloom:    bytesutil.PadTo([]byte("logs_bloom"), 256),
			PrevRandao:   bytesutil.PadTo([]byte("prev_randao"), 32),
			BlockNumber:  123,
			GasLimit:     123,
			GasUsed:      123,
			// State time at slot 33
			Timestamp:     396,
			ExtraData:     bytesutil.PadTo([]byte("extra_data"), 32),
			BaseFeePerGas: bytesutil.PadTo([]byte("base_fee_per_gas"), 32),
			BlockHash:     bytesutil.PadTo([]byte("block_hash"), 32),
			Transactions:  [][]byte{[]byte("transaction1"), []byte("transaction2")},
		}

		stateRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err, "Could not hash genesis state")
		genesisBlock := util.NewBeaconBlockBellatrix()
		genesisBlock.Block.StateRoot = stateRoot[:]
		util.SaveBlock(t, ctx, db, genesisBlock)
		parentRoot, err := genesisBlock.Block.HashTreeRoot()
		require.NoError(t, err)

		require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
		require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

		var payloadIdBytes enginev1.PayloadIDBytes
		copy(payloadIdBytes[:], "payload_id")
		mockChainService := &mockChain.ChainService{State: beaconState, Root: parentRoot[:], ForkChoiceStore: doublylinkedtree.New()}
		mockExecutionChain := &mockExecution.Chain{}
		v1Alpha1Server := &v1alpha1validator.Server{
			ExecutionEngineCaller: &mockExecution.EngineClient{
				OverrideValidHash: bytesutil.ToBytes32([]byte("parent_hash")),
				PayloadIDBytes:    &payloadIdBytes,
				ExecutionPayload:  payload,
			},
			TimeFetcher:            mockChainService,
			HeadFetcher:            mockChainService,
			OptimisticModeFetcher:  mockChainService,
			FinalizationFetcher:    mockChainService,
			SyncChecker:            &mockSync.Sync{IsSyncing: false},
			BlockReceiver:          mockChainService,
			ForkFetcher:            mockChainService,
			ForkchoiceFetcher:      mockChainService,
			ChainStartFetcher:      mockExecutionChain,
			Eth1InfoFetcher:        mockExecutionChain,
			Eth1BlockFetcher:       mockExecutionChain,
			MockEth1Votes:          true,
			AttPool:                attestations.NewPool(),
			SlashingsPool:          slashings.NewPool(),
			ExitPool:               voluntaryexits.NewPool(),
			StateGen:               stategen.New(db, doublylinkedtree.New()),
			SyncCommitteePool:      synccommittee.NewStore(),
			ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
			BlockBuilder: &builderTest.MockBuilderService{
				HasConfigured: false,
			},
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
		for i := primitives.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
			proposerSlashing, err := util.GenerateProposerSlashingForValidator(
				beaconState,
				privKeys[i],
				i,
			)
			require.NoError(t, err)
			proposerSlashings[i] = proposerSlashing
			err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), beaconState, proposerSlashing)
			require.NoError(t, err)
		}

		attSlashings := make([]*ethpbalpha.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
		for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings; i++ {
			attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
				beaconState,
				privKeys[i+params.BeaconConfig().MaxProposerSlashings],
				primitives.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings), /* validator index */
			)
			require.NoError(t, err)
			attSlashings[i] = attesterSlashing
			err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), beaconState, attesterSlashing)
			require.NoError(t, err)
		}

		aggregationBits := bitfield.NewBitvector128()
		for i := range aggregationBits {
			aggregationBits[i] = 0xAA
		}

		syncCommitteeIndices, err := altair.NextSyncCommitteeIndices(context.Background(), beaconState)
		require.NoError(t, err)
		sigs := make([]bls.Signature, 0, len(syncCommitteeIndices))
		for i, indice := range syncCommitteeIndices {
			if aggregationBits.BitAt(uint64(i)) {
				b := p2pType.SSZBytes(parentRoot[:])
				sb, err := signing.ComputeDomainAndSign(beaconState, coreTime.CurrentEpoch(beaconState), &b, params.BeaconConfig().DomainSyncCommittee, privKeys[indice])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				sigs = append(sigs, sig)
			}
		}
		aggregatedSig := bls.AggregateSignatures(sigs).Marshal()
		contribution := &ethpbalpha.SyncCommitteeContribution{
			Slot:              params.BeaconConfig().SlotsPerEpoch,
			BlockRoot:         parentRoot[:],
			SubcommitteeIndex: 0,
			AggregationBits:   aggregationBits,
			Signature:         aggregatedSig,
		}
		require.NoError(t, v1Alpha1Server.SyncCommitteePool.SaveSyncCommitteeContribution(contribution))

		v1Server := &Server{
			V1Alpha1Server: v1Alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}
		randaoReveal, err := util.RandaoReveal(beaconState, 1, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))
		req := &ethpbv1.ProduceBlockRequest{
			Slot:         params.BeaconConfig().SlotsPerEpoch + 1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		v1Server.V1Alpha1Server.BeaconDB = db
		resp, err := v1Server.ProduceBlockV2(ctx, req)
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BeaconBlockContainerV2_BellatrixBlock)
		require.Equal(t, true, ok)
		blk := containerBlock.BellatrixBlock
		assert.Equal(t, req.Slot, blk.Slot, "Expected block to have slot of 1")
		assert.DeepEqual(t, parentRoot[:], blk.ParentRoot, "Expected block to have correct parent root")
		assert.DeepEqual(t, randaoReveal, blk.Body.RandaoReveal, "Expected block to have correct randao reveal")
		assert.DeepEqual(t, req.Graffiti, blk.Body.Graffiti, "Expected block to have correct graffiti")
		assert.Equal(t, params.BeaconConfig().MaxProposerSlashings, uint64(len(blk.Body.ProposerSlashings)))

		expectedPropSlashings := make([]*ethpbv1.ProposerSlashing, len(proposerSlashings))
		for i, slash := range proposerSlashings {
			expectedPropSlashings[i] = migration.V1Alpha1ProposerSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedPropSlashings, blk.Body.ProposerSlashings)

		assert.Equal(t, params.BeaconConfig().MaxAttesterSlashings, uint64(len(blk.Body.AttesterSlashings)))
		expectedAttSlashings := make([]*ethpbv1.AttesterSlashing, len(attSlashings))
		for i, slash := range attSlashings {
			expectedAttSlashings[i] = migration.V1Alpha1AttSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedAttSlashings, blk.Body.AttesterSlashings)

		expectedBits := bitfield.NewBitvector512()
		for i := 0; i <= 15; i++ {
			expectedBits[i] = 0xAA
		}
		assert.DeepEqual(t, expectedBits, blk.Body.SyncAggregate.SyncCommitteeBits)
		assert.DeepEqual(t, aggregatedSig, blk.Body.SyncAggregate.SyncCommitteeSignature)

		assert.DeepEqual(t, payload, blk.Body.ExecutionPayload)
	})

	t.Run("Capella", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		params.SetupTestConfigCleanup(t)
		bc := params.BeaconConfig().Copy()
		bc.AltairForkEpoch = primitives.Epoch(0)
		bc.BellatrixForkEpoch = primitives.Epoch(1)
		bc.CapellaForkEpoch = primitives.Epoch(2)
		params.OverrideBeaconConfig(bc)

		beaconState, privKeys := util.DeterministicGenesisStateCapella(t, params.BeaconConfig().SyncCommitteeSize)
		require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch*2))
		syncCommittee, err := altair.NextSyncCommittee(context.Background(), beaconState)
		require.NoError(t, err)
		require.NoError(t, beaconState.SetCurrentSyncCommittee(syncCommittee))
		require.NoError(t, beaconState.SetNextSyncCommittee(syncCommittee))
		require.NoError(t, beaconState.UpdateRandaoMixesAtIndex(2, bytesutil.PadTo([]byte("prev_randao"), 32)))

		vals := beaconState.Validators()
		creds0 := make([]byte, 32)
		creds0[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
		copy(creds0[state_native.ETH1AddressOffset:], "address0")
		vals[0].WithdrawalCredentials = creds0
		vals[0].EffectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		creds1 := make([]byte, 32)
		creds1[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
		copy(creds1[state_native.ETH1AddressOffset:], "address1")
		vals[1].WithdrawalCredentials = creds1
		vals[1].EffectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		require.NoError(t, beaconState.SetValidators(vals))
		balances := beaconState.Balances()
		balances[0] += 123
		balances[1] += 123
		require.NoError(t, beaconState.SetBalances(balances))
		withdrawals := []*enginev1.Withdrawal{
			{
				Index:          0,
				ValidatorIndex: 0,
				Address:        bytesutil.PadTo([]byte("address0"), 20),
				Amount:         123,
			},
			{
				Index:          1,
				ValidatorIndex: 1,
				Address:        bytesutil.PadTo([]byte("address1"), 20),
				Amount:         123,
			},
		}
		withdrawalsRoot, err := ssz.WithdrawalSliceRoot(withdrawals, 2)
		require.NoError(t, err)

		payloadHeader, err := blocks.WrappedExecutionPayloadHeaderCapella(&enginev1.ExecutionPayloadHeaderCapella{
			ParentHash:   bytesutil.PadTo([]byte("parent_hash"), 32),
			FeeRecipient: bytesutil.PadTo([]byte("fee_recipient"), 20),
			StateRoot:    bytesutil.PadTo([]byte("state_root"), 32),
			ReceiptsRoot: bytesutil.PadTo([]byte("receipts_root"), 32),
			LogsBloom:    bytesutil.PadTo([]byte("logs_bloom"), 256),
			PrevRandao:   bytesutil.PadTo([]byte("prev_randao"), 32),
			BlockNumber:  123,
			GasLimit:     123,
			GasUsed:      123,
			// State time at slot 65
			Timestamp:     780,
			ExtraData:     bytesutil.PadTo([]byte("extra_data"), 32),
			BaseFeePerGas: bytesutil.PadTo([]byte("base_fee_per_gas"), 32),
			// Must be equal to payload.ParentHash
			BlockHash:        bytesutil.PadTo([]byte("equal_hash"), 32),
			TransactionsRoot: bytesutil.PadTo([]byte("transactions_root"), 32),
			WithdrawalsRoot:  withdrawalsRoot[:],
		}, big.NewInt(0))
		require.NoError(t, err)
		require.NoError(t, beaconState.SetLatestExecutionPayloadHeader(payloadHeader))

		payload := &enginev1.ExecutionPayloadCapella{
			// Must be equal to payloadHeader.BlockHash
			ParentHash:   bytesutil.PadTo([]byte("equal_hash"), 32),
			FeeRecipient: bytesutil.PadTo([]byte("fee_recipient"), 20),
			StateRoot:    bytesutil.PadTo([]byte("state_root"), 32),
			ReceiptsRoot: bytesutil.PadTo([]byte("receipts_root"), 32),
			LogsBloom:    bytesutil.PadTo([]byte("logs_bloom"), 256),
			PrevRandao:   bytesutil.PadTo([]byte("prev_randao"), 32),
			BlockNumber:  123,
			GasLimit:     123,
			GasUsed:      123,
			// State time at slot 65
			Timestamp:     780,
			ExtraData:     bytesutil.PadTo([]byte("extra_data"), 32),
			BaseFeePerGas: bytesutil.PadTo([]byte("base_fee_per_gas"), 32),
			BlockHash:     bytesutil.PadTo([]byte("block_hash"), 32),
			Transactions:  [][]byte{[]byte("transaction1"), []byte("transaction2")},
			Withdrawals:   withdrawals,
		}

		stateRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err, "Could not hash genesis state")
		genesisBlock := util.NewBeaconBlockCapella()
		genesisBlock.Block.StateRoot = stateRoot[:]
		util.SaveBlock(t, ctx, db, genesisBlock)
		parentRoot, err := genesisBlock.Block.HashTreeRoot()
		require.NoError(t, err)

		require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
		require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

		var payloadIdBytes enginev1.PayloadIDBytes
		copy(payloadIdBytes[:], "payload_id")
		mockChainService := &mockChain.ChainService{State: beaconState, Root: parentRoot[:], ForkChoiceStore: doublylinkedtree.New()}
		mockExecutionChain := &mockExecution.Chain{}
		v1Alpha1Server := &v1alpha1validator.Server{
			ExecutionEngineCaller: &mockExecution.EngineClient{
				OverrideValidHash:       bytesutil.ToBytes32([]byte("parent_hash")),
				PayloadIDBytes:          &payloadIdBytes,
				ExecutionPayloadCapella: payload,
			},
			TimeFetcher:            mockChainService,
			HeadFetcher:            mockChainService,
			OptimisticModeFetcher:  mockChainService,
			SyncChecker:            &mockSync.Sync{IsSyncing: false},
			BlockReceiver:          mockChainService,
			ForkFetcher:            mockChainService,
			ForkchoiceFetcher:      mockChainService,
			FinalizationFetcher:    mockChainService,
			ChainStartFetcher:      mockExecutionChain,
			Eth1InfoFetcher:        mockExecutionChain,
			Eth1BlockFetcher:       mockExecutionChain,
			MockEth1Votes:          true,
			AttPool:                attestations.NewPool(),
			SlashingsPool:          slashings.NewPool(),
			ExitPool:               voluntaryexits.NewPool(),
			BLSChangesPool:         blstoexec.NewPool(),
			StateGen:               stategen.New(db, doublylinkedtree.New()),
			SyncCommitteePool:      synccommittee.NewStore(),
			ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
			BlockBuilder: &builderTest.MockBuilderService{
				HasConfigured: false,
			},
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
		for i := primitives.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
			proposerSlashing, err := util.GenerateProposerSlashingForValidator(
				beaconState,
				privKeys[i],
				i,
			)
			require.NoError(t, err)
			proposerSlashings[i] = proposerSlashing
			err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), beaconState, proposerSlashing)
			require.NoError(t, err)
		}

		attSlashings := make([]*ethpbalpha.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
		for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings; i++ {
			attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
				beaconState,
				privKeys[i+params.BeaconConfig().MaxProposerSlashings],
				primitives.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings), /* validator index */
			)
			require.NoError(t, err)
			attSlashings[i] = attesterSlashing
			err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), beaconState, attesterSlashing)
			require.NoError(t, err)
		}

		aggregationBits := bitfield.NewBitvector128()
		for i := range aggregationBits {
			aggregationBits[i] = 0xAA
		}

		syncCommitteeIndices, err := altair.NextSyncCommitteeIndices(context.Background(), beaconState)
		require.NoError(t, err)
		sigs := make([]bls.Signature, 0, len(syncCommitteeIndices))
		for i, indice := range syncCommitteeIndices {
			if aggregationBits.BitAt(uint64(i)) {
				b := p2pType.SSZBytes(parentRoot[:])
				sb, err := signing.ComputeDomainAndSign(beaconState, coreTime.CurrentEpoch(beaconState), &b, params.BeaconConfig().DomainSyncCommittee, privKeys[indice])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				sigs = append(sigs, sig)
			}
		}
		aggregatedSig := bls.AggregateSignatures(sigs).Marshal()
		contribution := &ethpbalpha.SyncCommitteeContribution{
			Slot:              params.BeaconConfig().SlotsPerEpoch * 2,
			BlockRoot:         parentRoot[:],
			SubcommitteeIndex: 0,
			AggregationBits:   aggregationBits,
			Signature:         aggregatedSig,
		}
		require.NoError(t, v1Alpha1Server.SyncCommitteePool.SaveSyncCommitteeContribution(contribution))

		v1Server := &Server{
			V1Alpha1Server: v1Alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}
		randaoReveal, err := util.RandaoReveal(beaconState, 2, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))
		req := &ethpbv1.ProduceBlockRequest{
			Slot:         params.BeaconConfig().SlotsPerEpoch*2 + 1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		v1Server.V1Alpha1Server.BeaconDB = db
		resp, err := v1Server.ProduceBlockV2(ctx, req)
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_CAPELLA, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BeaconBlockContainerV2_CapellaBlock)
		require.Equal(t, true, ok)
		blk := containerBlock.CapellaBlock
		assert.Equal(t, req.Slot, blk.Slot, "Expected block to have slot of 1")
		assert.DeepEqual(t, parentRoot[:], blk.ParentRoot, "Expected block to have correct parent root")
		assert.DeepEqual(t, randaoReveal, blk.Body.RandaoReveal, "Expected block to have correct randao reveal")
		assert.DeepEqual(t, req.Graffiti, blk.Body.Graffiti, "Expected block to have correct graffiti")
		assert.Equal(t, params.BeaconConfig().MaxProposerSlashings, uint64(len(blk.Body.ProposerSlashings)))

		expectedPropSlashings := make([]*ethpbv1.ProposerSlashing, len(proposerSlashings))
		for i, slash := range proposerSlashings {
			expectedPropSlashings[i] = migration.V1Alpha1ProposerSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedPropSlashings, blk.Body.ProposerSlashings)

		assert.Equal(t, params.BeaconConfig().MaxAttesterSlashings, uint64(len(blk.Body.AttesterSlashings)))
		expectedAttSlashings := make([]*ethpbv1.AttesterSlashing, len(attSlashings))
		for i, slash := range attSlashings {
			expectedAttSlashings[i] = migration.V1Alpha1AttSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedAttSlashings, blk.Body.AttesterSlashings)

		expectedBits := bitfield.NewBitvector512()
		for i := 0; i <= 15; i++ {
			expectedBits[i] = 0xAA
		}
		assert.DeepEqual(t, expectedBits, blk.Body.SyncAggregate.SyncCommitteeBits)
		assert.DeepEqual(t, aggregatedSig, blk.Body.SyncAggregate.SyncCommitteeSignature)

		assert.DeepEqual(t, payload, blk.Body.ExecutionPayload)
	})
}

func TestProduceBlockV2SSZ(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		ctx := context.Background()

		db := dbutil.SetupDB(t)

		bs, parentRoot, privKeys := util.DeterministicGenesisStateWithGenesisBlock(t, ctx, db, 2)

		mockChainService := &mockChain.ChainService{State: bs, Root: parentRoot[:], ForkChoiceStore: doublylinkedtree.New()}
		v1Alpha1Server := &v1alpha1validator.Server{
			HeadFetcher:         mockChainService,
			SyncChecker:         &mockSync.Sync{IsSyncing: false},
			BlockReceiver:       mockChainService,
			TimeFetcher:         mockChainService,
			ForkFetcher:         mockChainService,
			ForkchoiceFetcher:   mockChainService,
			FinalizationFetcher: mockChainService,
			ChainStartFetcher:   &mockExecution.Chain{},
			Eth1InfoFetcher:     &mockExecution.Chain{},
			Eth1BlockFetcher:    &mockExecution.Chain{},
			MockEth1Votes:       true,
			AttPool:             attestations.NewPool(),
			SlashingsPool:       slashings.NewPool(),
			ExitPool:            voluntaryexits.NewPool(),
			StateGen:            stategen.New(db, doublylinkedtree.New()),
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, 1)
		proposerSlashing, err := util.GenerateProposerSlashingForValidator(
			bs,
			privKeys[0],
			0,
		)
		require.NoError(t, err)
		proposerSlashings[0] = proposerSlashing
		err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), bs, proposerSlashing)
		require.NoError(t, err)

		attSlashings := make([]*ethpbalpha.AttesterSlashing, 1)
		attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
			bs,
			privKeys[1],
			1,
		)
		require.NoError(t, err)
		attSlashings[0] = attesterSlashing
		err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), bs, attesterSlashing)
		require.NoError(t, err)

		v1Server := &Server{
			V1Alpha1Server: v1Alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}
		randaoReveal, err := util.RandaoReveal(bs, 0, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))
		req := &ethpbv1.ProduceBlockRequest{
			Slot:         1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		resp, err := v1Server.ProduceBlockV2SSZ(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, ethpbv2.Version_PHASE0, resp.Version)

		expectedBlock := &ethpbv1.BeaconBlock{
			Slot:          1,
			ProposerIndex: 0,
			ParentRoot:    []byte{164, 47, 20, 157, 2, 202, 58, 173, 57, 154, 254, 181, 153, 74, 40, 89, 159, 74, 62, 247, 28, 222, 153, 182, 168, 79, 170, 149, 80, 99, 97, 32},
			StateRoot:     []byte{224, 94, 112, 87, 163, 57, 192, 233, 181, 179, 212, 9, 226, 214, 65, 238, 189, 114, 223, 101, 215, 146, 163, 140, 92, 242, 35, 82, 222, 154, 127, 136},
			Body: &ethpbv1.BeaconBlockBody{
				RandaoReveal: randaoReveal,
				Eth1Data: &ethpbv1.Eth1Data{
					DepositRoot:  []byte{124, 159, 161, 54, 212, 65, 63, 166, 23, 54, 55, 232, 131, 182, 153, 141, 50, 225, 214, 117, 248, 140, 221, 255, 157, 203, 207, 51, 24, 32, 244, 184},
					DepositCount: 2,
					BlockHash:    []byte{8, 83, 63, 107, 189, 73, 117, 17, 62, 79, 12, 177, 4, 171, 205, 236, 29, 134, 217, 157, 87, 130, 180, 169, 167, 246, 39, 12, 14, 187, 106, 39},
				},
				Graffiti: graffiti[:],
				ProposerSlashings: []*ethpbv1.ProposerSlashing{
					{
						SignedHeader_1: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_1.Header.Slot,
								ProposerIndex: proposerSlashing.Header_1.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_1.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_1.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_1.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_1.Signature,
						},
						SignedHeader_2: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_2.Header.Slot,
								ProposerIndex: proposerSlashing.Header_2.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_2.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_2.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_2.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_2.Signature,
						},
					},
				},
				AttesterSlashings: []*ethpbv1.AttesterSlashing{
					{
						Attestation_1: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_1.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_1.Data.Slot,
								Index:           attesterSlashing.Attestation_1.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_1.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_1.Signature,
						},
						Attestation_2: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_2.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_2.Data.Slot,
								Index:           attesterSlashing.Attestation_2.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_2.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_2.Signature,
						},
					},
				},
			},
		}
		expectedData, err := expectedBlock.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})

	t.Run("Altair", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		params.SetupTestConfigCleanup(t)
		bc := params.BeaconConfig()
		bc.AltairForkEpoch = primitives.Epoch(0)
		params.OverrideBeaconConfig(bc)

		bs, privKeys := util.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
		syncCommittee, err := altair.NextSyncCommittee(context.Background(), bs)
		require.NoError(t, err)
		require.NoError(t, bs.SetCurrentSyncCommittee(syncCommittee))
		require.NoError(t, bs.SetNextSyncCommittee(syncCommittee))

		stateRoot, err := bs.HashTreeRoot(ctx)
		require.NoError(t, err, "Could not hash genesis state")
		genesisBlock := util.NewBeaconBlockAltair()
		genesisBlock.Block.StateRoot = stateRoot[:]
		util.SaveBlock(t, ctx, db, genesisBlock)
		parentRoot, err := genesisBlock.Block.HashTreeRoot()
		require.NoError(t, err)

		require.NoError(t, db.SaveState(ctx, bs, parentRoot), "Could not save genesis state")
		require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

		mockChainService := &mockChain.ChainService{State: bs, Root: parentRoot[:], ForkChoiceStore: doublylinkedtree.New()}
		v1Alpha1Server := &v1alpha1validator.Server{
			HeadFetcher:       mockChainService,
			SyncChecker:       &mockSync.Sync{IsSyncing: false},
			BlockReceiver:     mockChainService,
			TimeFetcher:       mockChainService,
			ForkFetcher:       mockChainService,
			ForkchoiceFetcher: mockChainService,
			ChainStartFetcher: &mockExecution.Chain{},
			Eth1InfoFetcher:   &mockExecution.Chain{},
			Eth1BlockFetcher:  &mockExecution.Chain{},
			MockEth1Votes:     true,
			AttPool:           attestations.NewPool(),
			SlashingsPool:     slashings.NewPool(),
			ExitPool:          voluntaryexits.NewPool(),
			StateGen:          stategen.New(db, doublylinkedtree.New()),
			SyncCommitteePool: synccommittee.NewStore(),
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, 1)
		proposerSlashing, err := util.GenerateProposerSlashingForValidator(
			bs,
			privKeys[0],
			0,
		)
		require.NoError(t, err)
		proposerSlashings[0] = proposerSlashing
		err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), bs, proposerSlashing)
		require.NoError(t, err)

		attSlashings := make([]*ethpbalpha.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
		attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
			bs,
			privKeys[1],
			1,
		)
		require.NoError(t, err)
		attSlashings[0] = attesterSlashing
		err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), bs, attesterSlashing)
		require.NoError(t, err)

		aggregationBits := bitfield.NewBitvector128()
		for i := range aggregationBits {
			aggregationBits[i] = 0xAA
		}

		syncCommitteeIndices, err := altair.NextSyncCommitteeIndices(context.Background(), bs)
		require.NoError(t, err)
		sigs := make([]bls.Signature, 0, len(syncCommitteeIndices))
		for i, indice := range syncCommitteeIndices {
			if aggregationBits.BitAt(uint64(i)) {
				b := p2pType.SSZBytes(parentRoot[:])
				sb, err := signing.ComputeDomainAndSign(bs, coreTime.CurrentEpoch(bs), &b, params.BeaconConfig().DomainSyncCommittee, privKeys[indice])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				sigs = append(sigs, sig)
			}
		}
		aggregatedSig := bls.AggregateSignatures(sigs).Marshal()
		contribution := &ethpbalpha.SyncCommitteeContribution{
			Slot:              0,
			BlockRoot:         parentRoot[:],
			SubcommitteeIndex: 0,
			AggregationBits:   aggregationBits,
			Signature:         aggregatedSig,
		}
		require.NoError(t, v1Alpha1Server.SyncCommitteePool.SaveSyncCommitteeContribution(contribution))

		v1Server := &Server{
			V1Alpha1Server: v1Alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}
		randaoReveal, err := util.RandaoReveal(bs, 0, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))

		req := &ethpbv1.ProduceBlockRequest{
			Slot:         1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		resp, err := v1Server.ProduceBlockV2SSZ(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)

		expectedBlock := &ethpbv2.BeaconBlockAltair{
			Slot:          1,
			ProposerIndex: 19,
			ParentRoot:    []byte{162, 206, 194, 54, 242, 248, 88, 148, 193, 141, 39, 23, 91, 116, 219, 219, 2, 248, 4, 155, 159, 201, 41, 156, 130, 57, 167, 176, 153, 18, 73, 148},
			StateRoot:     []byte{144, 220, 158, 2, 142, 57, 111, 170, 148, 225, 129, 23, 103, 232, 44, 1, 222, 77, 36, 110, 118, 237, 184, 77, 253, 182, 0, 62, 168, 56, 105, 95},
			Body: &ethpbv2.BeaconBlockBodyAltair{
				RandaoReveal: randaoReveal,
				Eth1Data: &ethpbv1.Eth1Data{
					DepositRoot:  []byte{124, 159, 161, 54, 212, 65, 63, 166, 23, 54, 55, 232, 131, 182, 153, 141, 50, 225, 214, 117, 248, 140, 221, 255, 157, 203, 207, 51, 24, 32, 244, 184},
					DepositCount: params.BeaconConfig().SyncCommitteeSize,
					BlockHash:    []byte{8, 83, 63, 107, 189, 73, 117, 17, 62, 79, 12, 177, 4, 171, 205, 236, 29, 134, 217, 157, 87, 130, 180, 169, 167, 246, 39, 12, 14, 187, 106, 39},
				},
				Graffiti: graffiti[:],
				ProposerSlashings: []*ethpbv1.ProposerSlashing{
					{
						SignedHeader_1: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_1.Header.Slot,
								ProposerIndex: proposerSlashing.Header_1.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_1.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_1.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_1.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_1.Signature,
						},
						SignedHeader_2: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_2.Header.Slot,
								ProposerIndex: proposerSlashing.Header_2.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_2.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_2.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_2.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_2.Signature,
						},
					},
				},
				AttesterSlashings: []*ethpbv1.AttesterSlashing{
					{
						Attestation_1: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_1.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_1.Data.Slot,
								Index:           attesterSlashing.Attestation_1.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_1.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_1.Signature,
						},
						Attestation_2: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_2.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_2.Data.Slot,
								Index:           attesterSlashing.Attestation_2.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_2.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_2.Signature,
						},
					},
				},
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits:      []byte{170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					SyncCommitteeSignature: []byte{130, 228, 60, 221, 180, 9, 29, 148, 136, 255, 135, 183, 146, 130, 88, 240, 116, 219, 183, 208, 148, 211, 202, 78, 240, 120, 60, 99, 77, 76, 109, 210, 163, 243, 244, 25, 70, 184, 29, 252, 138, 128, 202, 173, 1, 166, 48, 49, 11, 136, 42, 124, 163, 187, 206, 253, 214, 149, 114, 137, 146, 123, 197, 187, 250, 204, 59, 196, 87, 195, 48, 11, 116, 123, 58, 49, 62, 98, 193, 166, 0, 172, 15, 253, 130, 88, 46, 110, 45, 84, 57, 107, 83, 182, 127, 105},
				},
			},
		}
		expectedData, err := expectedBlock.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})

	t.Run("Bellatrix", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		params.SetupTestConfigCleanup(t)
		bc := params.BeaconConfig()
		bc.AltairForkEpoch = primitives.Epoch(0)
		bc.BellatrixForkEpoch = primitives.Epoch(1)
		params.OverrideBeaconConfig(bc)

		bs, privKeys := util.DeterministicGenesisStateBellatrix(t, params.BeaconConfig().SyncCommitteeSize)
		require.NoError(t, bs.SetSlot(params.BeaconConfig().SlotsPerEpoch))
		syncCommittee, err := altair.NextSyncCommittee(context.Background(), bs)
		require.NoError(t, err)
		require.NoError(t, bs.SetCurrentSyncCommittee(syncCommittee))
		require.NoError(t, bs.SetNextSyncCommittee(syncCommittee))

		stateRoot, err := bs.HashTreeRoot(ctx)
		require.NoError(t, err, "Could not hash genesis state")
		genesisBlock := util.NewBeaconBlockBellatrix()
		genesisBlock.Block.StateRoot = stateRoot[:]
		util.SaveBlock(t, ctx, db, genesisBlock)
		parentRoot, err := genesisBlock.Block.HashTreeRoot()
		require.NoError(t, err)

		require.NoError(t, db.SaveState(ctx, bs, parentRoot), "Could not save genesis state")
		require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

		mockChainService := &mockChain.ChainService{State: bs, Root: parentRoot[:], ForkChoiceStore: doublylinkedtree.New()}
		v1Alpha1Server := &v1alpha1validator.Server{
			ExecutionEngineCaller: &mockExecution.EngineClient{
				ExecutionBlock: &enginev1.ExecutionBlock{
					TotalDifficulty: "0x1",
				},
			},
			TimeFetcher:            &mockChain.ChainService{},
			HeadFetcher:            mockChainService,
			OptimisticModeFetcher:  mockChainService,
			SyncChecker:            &mockSync.Sync{IsSyncing: false},
			BlockReceiver:          mockChainService,
			ForkFetcher:            mockChainService,
			ForkchoiceFetcher:      mockChainService,
			ChainStartFetcher:      &mockExecution.Chain{},
			Eth1InfoFetcher:        &mockExecution.Chain{},
			Eth1BlockFetcher:       &mockExecution.Chain{},
			MockEth1Votes:          true,
			AttPool:                attestations.NewPool(),
			SlashingsPool:          slashings.NewPool(),
			ExitPool:               voluntaryexits.NewPool(),
			StateGen:               stategen.New(db, doublylinkedtree.New()),
			SyncCommitteePool:      synccommittee.NewStore(),
			ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
			BlockBuilder: &builderTest.MockBuilderService{
				HasConfigured: true,
			},
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, 1)
		proposerSlashing, err := util.GenerateProposerSlashingForValidator(
			bs,
			privKeys[0],
			0,
		)
		require.NoError(t, err)
		proposerSlashings[0] = proposerSlashing
		err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), bs, proposerSlashing)
		require.NoError(t, err)

		attSlashings := make([]*ethpbalpha.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
		attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
			bs,
			privKeys[1],
			1,
		)
		require.NoError(t, err)
		attSlashings[0] = attesterSlashing
		err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), bs, attesterSlashing)
		require.NoError(t, err)

		aggregationBits := bitfield.NewBitvector128()
		for i := range aggregationBits {
			aggregationBits[i] = 0xAA
		}

		syncCommitteeIndices, err := altair.NextSyncCommitteeIndices(context.Background(), bs)
		require.NoError(t, err)
		sigs := make([]bls.Signature, 0, len(syncCommitteeIndices))
		for i, indice := range syncCommitteeIndices {
			if aggregationBits.BitAt(uint64(i)) {
				b := p2pType.SSZBytes(parentRoot[:])
				sb, err := signing.ComputeDomainAndSign(bs, coreTime.CurrentEpoch(bs), &b, params.BeaconConfig().DomainSyncCommittee, privKeys[indice])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				sigs = append(sigs, sig)
			}
		}
		aggregatedSig := bls.AggregateSignatures(sigs).Marshal()
		contribution := &ethpbalpha.SyncCommitteeContribution{
			Slot:              params.BeaconConfig().SlotsPerEpoch,
			BlockRoot:         parentRoot[:],
			SubcommitteeIndex: 0,
			AggregationBits:   aggregationBits,
			Signature:         aggregatedSig,
		}
		require.NoError(t, v1Alpha1Server.SyncCommitteePool.SaveSyncCommitteeContribution(contribution))

		v1Server := &Server{
			V1Alpha1Server: v1Alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}
		randaoReveal, err := util.RandaoReveal(bs, 1, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))

		req := &ethpbv1.ProduceBlockRequest{
			Slot:         params.BeaconConfig().SlotsPerEpoch + 1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		v1Server.V1Alpha1Server.BeaconDB = db
		require.NoError(t, v1Alpha1Server.BeaconDB.SaveRegistrationsByValidatorIDs(ctx, []primitives.ValidatorIndex{348},
			[]*ethpbalpha.ValidatorRegistrationV1{{FeeRecipient: bytesutil.PadTo([]byte{}, fieldparams.FeeRecipientLength), Pubkey: bytesutil.PadTo([]byte{}, fieldparams.BLSPubkeyLength)}}))

		resp, err := v1Server.ProduceBlockV2SSZ(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)

		expectedBlock := &ethpbv2.BeaconBlockBellatrix{
			Slot:          33,
			ProposerIndex: 348,
			ParentRoot:    []byte{228, 15, 208, 120, 31, 194, 202, 144, 41, 107, 98, 126, 162, 234, 190, 94, 174, 176, 69, 177, 103, 82, 69, 254, 0, 230, 192, 67, 158, 29, 141, 85},
			StateRoot:     []byte{143, 107, 161, 135, 58, 60, 195, 107, 55, 142, 122, 111, 184, 1, 19, 233, 145, 204, 160, 226, 148, 67, 194, 102, 79, 196, 74, 242, 174, 108, 68, 82},
			Body: &ethpbv2.BeaconBlockBodyBellatrix{
				RandaoReveal: randaoReveal,
				Eth1Data: &ethpbv1.Eth1Data{
					DepositRoot:  []byte{40, 2, 99, 184, 81, 91, 153, 196, 115, 217, 104, 93, 31, 202, 27, 153, 42, 224, 148, 156, 116, 43, 161, 28, 155, 166, 37, 217, 205, 152, 69, 6},
					DepositCount: params.BeaconConfig().SyncCommitteeSize,
					BlockHash:    []byte{226, 231, 104, 45, 7, 68, 48, 54, 228, 109, 84, 245, 125, 45, 227, 127, 135, 155, 63, 38, 241, 251, 129, 192, 248, 49, 9, 120, 146, 18, 34, 228},
				},
				Graffiti: graffiti[:],
				ProposerSlashings: []*ethpbv1.ProposerSlashing{
					{
						SignedHeader_1: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_1.Header.Slot,
								ProposerIndex: proposerSlashing.Header_1.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_1.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_1.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_1.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_1.Signature,
						},
						SignedHeader_2: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_2.Header.Slot,
								ProposerIndex: proposerSlashing.Header_2.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_2.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_2.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_2.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_2.Signature,
						},
					},
				},
				AttesterSlashings: []*ethpbv1.AttesterSlashing{
					{
						Attestation_1: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_1.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_1.Data.Slot,
								Index:           attesterSlashing.Attestation_1.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_1.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_1.Signature,
						},
						Attestation_2: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_2.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_2.Data.Slot,
								Index:           attesterSlashing.Attestation_2.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_2.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_2.Signature,
						},
					},
				},
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits:      []byte{170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					SyncCommitteeSignature: []byte{153, 51, 238, 112, 158, 23, 41, 26, 18, 53, 3, 111, 57, 180, 45, 131, 90, 249, 28, 23, 153, 188, 171, 204, 45, 180, 133, 236, 47, 203, 119, 132, 162, 17, 61, 60, 122, 161, 45, 136, 130, 174, 120, 60, 64, 144, 6, 34, 24, 87, 41, 77, 16, 223, 36, 125, 80, 185, 178, 234, 74, 184, 196, 45, 242, 47, 124, 178, 83, 65, 106, 26, 179, 178, 27, 4, 72, 79, 191, 128, 114, 51, 246, 147, 3, 55, 210, 64, 148, 78, 144, 45, 97, 182, 157, 206},
				},
				ExecutionPayload: &enginev1.ExecutionPayload{
					ParentHash:    make([]byte, 32),
					FeeRecipient:  make([]byte, 20),
					StateRoot:     make([]byte, 32),
					ReceiptsRoot:  make([]byte, 32),
					LogsBloom:     make([]byte, 256),
					PrevRandao:    make([]byte, 32),
					ExtraData:     nil,
					BaseFeePerGas: make([]byte, 32),
					BlockHash:     make([]byte, 32),
					Transactions:  nil,
				},
			},
		}
		expectedData, err := expectedBlock.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})

	t.Run("Capella", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		params.SetupTestConfigCleanup(t)
		bc := params.BeaconConfig()
		bc.AltairForkEpoch = primitives.Epoch(0)
		bc.BellatrixForkEpoch = primitives.Epoch(1)
		bc.CapellaForkEpoch = primitives.Epoch(2)
		params.OverrideBeaconConfig(bc)

		bs, privKeys := util.DeterministicGenesisStateCapella(t, params.BeaconConfig().SyncCommitteeSize)
		require.NoError(t, bs.SetSlot(params.BeaconConfig().SlotsPerEpoch*2))
		syncCommittee, err := altair.NextSyncCommittee(context.Background(), bs)
		require.NoError(t, err)
		require.NoError(t, bs.SetCurrentSyncCommittee(syncCommittee))
		require.NoError(t, bs.SetNextSyncCommittee(syncCommittee))

		stateRoot, err := bs.HashTreeRoot(ctx)
		require.NoError(t, err, "Could not hash genesis state")
		genesisBlock := util.NewBeaconBlockCapella()
		genesisBlock.Block.StateRoot = stateRoot[:]
		util.SaveBlock(t, ctx, db, genesisBlock)
		parentRoot, err := genesisBlock.Block.HashTreeRoot()
		require.NoError(t, err)

		require.NoError(t, db.SaveState(ctx, bs, parentRoot), "Could not save genesis state")
		require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

		prevRandao, err := helpers.RandaoMix(bs, 2)
		require.NoError(t, err)

		mockChainService := &mockChain.ChainService{State: bs, Root: parentRoot[:], ForkChoiceStore: doublylinkedtree.New()}
		v1Alpha1Server := &v1alpha1validator.Server{
			ExecutionEngineCaller: &mockExecution.EngineClient{
				ExecutionBlock: &enginev1.ExecutionBlock{
					TotalDifficulty: "0x1",
				},
				PayloadIDBytes: &enginev1.PayloadIDBytes{},
				ExecutionPayloadCapella: &enginev1.ExecutionPayloadCapella{
					ParentHash:   make([]byte, 32),
					FeeRecipient: make([]byte, 20),
					StateRoot:    make([]byte, 32),
					ReceiptsRoot: make([]byte, 32),
					LogsBloom:    make([]byte, 256),
					PrevRandao:   prevRandao,
					// State time at slot 65
					Timestamp:     780,
					ExtraData:     make([]byte, 32),
					BaseFeePerGas: make([]byte, 32),
					BlockHash:     make([]byte, 32),
				},
			},
			TimeFetcher:            &mockChain.ChainService{},
			HeadFetcher:            mockChainService,
			OptimisticModeFetcher:  mockChainService,
			SyncChecker:            &mockSync.Sync{IsSyncing: false},
			BlockReceiver:          mockChainService,
			ForkFetcher:            mockChainService,
			ForkchoiceFetcher:      mockChainService,
			FinalizationFetcher:    mockChainService,
			ChainStartFetcher:      &mockExecution.Chain{},
			Eth1InfoFetcher:        &mockExecution.Chain{},
			Eth1BlockFetcher:       &mockExecution.Chain{},
			MockEth1Votes:          true,
			AttPool:                attestations.NewPool(),
			SlashingsPool:          slashings.NewPool(),
			ExitPool:               voluntaryexits.NewPool(),
			BLSChangesPool:         blstoexec.NewPool(),
			StateGen:               stategen.New(db, doublylinkedtree.New()),
			SyncCommitteePool:      synccommittee.NewStore(),
			ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
			BlockBuilder: &builderTest.MockBuilderService{
				HasConfigured: true,
			},
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, 1)
		proposerSlashing, err := util.GenerateProposerSlashingForValidator(
			bs,
			privKeys[0],
			0,
		)
		require.NoError(t, err)
		proposerSlashings[0] = proposerSlashing
		err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), bs, proposerSlashing)
		require.NoError(t, err)

		attSlashings := make([]*ethpbalpha.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
		attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
			bs,
			privKeys[1],
			1,
		)
		require.NoError(t, err)
		attSlashings[0] = attesterSlashing
		err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), bs, attesterSlashing)
		require.NoError(t, err)

		aggregationBits := bitfield.NewBitvector128()
		for i := range aggregationBits {
			aggregationBits[i] = 0xAA
		}

		syncCommitteeIndices, err := altair.NextSyncCommitteeIndices(context.Background(), bs)
		require.NoError(t, err)
		sigs := make([]bls.Signature, 0, len(syncCommitteeIndices))
		for i, indice := range syncCommitteeIndices {
			if aggregationBits.BitAt(uint64(i)) {
				b := p2pType.SSZBytes(parentRoot[:])
				sb, err := signing.ComputeDomainAndSign(bs, coreTime.CurrentEpoch(bs), &b, params.BeaconConfig().DomainSyncCommittee, privKeys[indice])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				sigs = append(sigs, sig)
			}
		}
		aggregatedSig := bls.AggregateSignatures(sigs).Marshal()
		contribution := &ethpbalpha.SyncCommitteeContribution{
			Slot:              params.BeaconConfig().SlotsPerEpoch * 2,
			BlockRoot:         parentRoot[:],
			SubcommitteeIndex: 0,
			AggregationBits:   aggregationBits,
			Signature:         aggregatedSig,
		}
		require.NoError(t, v1Alpha1Server.SyncCommitteePool.SaveSyncCommitteeContribution(contribution))

		v1Server := &Server{
			V1Alpha1Server: v1Alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}
		randaoReveal, err := util.RandaoReveal(bs, 2, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))

		req := &ethpbv1.ProduceBlockRequest{
			Slot:         params.BeaconConfig().SlotsPerEpoch*2 + 1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		v1Server.V1Alpha1Server.BeaconDB = db
		require.NoError(t, v1Alpha1Server.BeaconDB.SaveRegistrationsByValidatorIDs(ctx, []primitives.ValidatorIndex{348},
			[]*ethpbalpha.ValidatorRegistrationV1{{FeeRecipient: bytesutil.PadTo([]byte{}, fieldparams.FeeRecipientLength), Pubkey: bytesutil.PadTo([]byte{}, fieldparams.BLSPubkeyLength)}}))

		resp, err := v1Server.ProduceBlockV2SSZ(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, ethpbv2.Version_CAPELLA, resp.Version)

		expectedBlock := &ethpbv2.BeaconBlockCapella{
			Slot:          65,
			ProposerIndex: 372,
			ParentRoot:    []byte{98, 110, 62, 255, 200, 27, 255, 78, 6, 11, 199, 119, 47, 57, 40, 250, 3, 255, 123, 149, 158, 235, 206, 73, 197, 53, 194, 112, 193, 78, 30, 209},
			StateRoot:     []byte{15, 201, 220, 7, 167, 148, 124, 27, 100, 119, 219, 2, 159, 243, 76, 2, 101, 209, 32, 213, 21, 158, 75, 203, 184, 93, 189, 243, 179, 78, 150, 171},
			Body: &ethpbv2.BeaconBlockBodyCapella{
				RandaoReveal: randaoReveal,
				Eth1Data: &ethpbv1.Eth1Data{
					DepositRoot:  []byte{209, 112, 83, 246, 47, 50, 100, 179, 169, 154, 93, 33, 128, 212, 195, 171, 204, 14, 180, 156, 134, 217, 151, 150, 60, 190, 195, 214, 0, 7, 248, 51},
					DepositCount: params.BeaconConfig().SyncCommitteeSize,
					BlockHash:    []byte{81, 55, 109, 120, 46, 24, 197, 204, 4, 239, 233, 63, 224, 130, 143, 78, 98, 6, 148, 78, 34, 173, 90, 38, 115, 7, 146, 56, 102, 19, 237, 43},
				},
				Graffiti: graffiti[:],
				ProposerSlashings: []*ethpbv1.ProposerSlashing{
					{
						SignedHeader_1: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_1.Header.Slot,
								ProposerIndex: proposerSlashing.Header_1.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_1.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_1.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_1.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_1.Signature,
						},
						SignedHeader_2: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_2.Header.Slot,
								ProposerIndex: proposerSlashing.Header_2.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_2.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_2.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_2.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_2.Signature,
						},
					},
				},
				AttesterSlashings: []*ethpbv1.AttesterSlashing{
					{
						Attestation_1: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_1.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_1.Data.Slot,
								Index:           attesterSlashing.Attestation_1.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_1.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_1.Signature,
						},
						Attestation_2: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_2.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_2.Data.Slot,
								Index:           attesterSlashing.Attestation_2.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_2.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_2.Signature,
						},
					},
				},
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits:      []byte{170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					SyncCommitteeSignature: []byte{166, 223, 239, 55, 64, 151, 124, 78, 34, 159, 22, 51, 216, 224, 97, 5, 3, 97, 106, 183, 24, 71, 180, 209, 40, 202, 128, 169, 176, 187, 134, 85, 104, 150, 180, 125, 117, 192, 82, 189, 14, 15, 245, 196, 141, 54, 235, 23, 4, 41, 66, 171, 1, 35, 136, 193, 40, 242, 154, 63, 19, 13, 215, 141, 195, 110, 10, 201, 153, 85, 50, 0, 152, 237, 80, 68, 104, 181, 63, 237, 114, 83, 206, 252, 210, 155, 50, 136, 125, 141, 173, 158, 34, 165, 36, 95},
				},
				ExecutionPayload: &enginev1.ExecutionPayloadCapella{
					ParentHash:   make([]byte, 32),
					FeeRecipient: make([]byte, 20),
					StateRoot:    make([]byte, 32),
					ReceiptsRoot: make([]byte, 32),
					LogsBloom:    make([]byte, 256),
					PrevRandao:   prevRandao,
					// State time at slot 65
					Timestamp:     780,
					ExtraData:     make([]byte, 32),
					BaseFeePerGas: make([]byte, 32),
					BlockHash:     make([]byte, 32),
					Transactions:  nil,
					Withdrawals:   nil,
				},
			},
		}
		expectedData, err := expectedBlock.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
}

func TestProduceBlockV2_SyncNotReady(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	chainService := &mockChain.ChainService{State: st}
	vs := &Server{
		SyncChecker:           &mockSync.Sync{IsSyncing: true},
		HeadFetcher:           chainService,
		TimeFetcher:           chainService,
		OptimisticModeFetcher: chainService,
	}
	_, err = vs.ProduceBlockV2(context.Background(), &ethpbv1.ProduceBlockRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestProduceBlockV2SSZ_SyncNotReady(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	chainService := &mockChain.ChainService{State: st}
	vs := &Server{
		SyncChecker:           &mockSync.Sync{IsSyncing: true},
		HeadFetcher:           chainService,
		TimeFetcher:           chainService,
		OptimisticModeFetcher: chainService,
	}
	_, err = vs.ProduceBlockV2SSZ(context.Background(), &ethpbv1.ProduceBlockRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestProduceBlindedBlock(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		beaconState, parentRoot, privKeys := util.DeterministicGenesisStateWithGenesisBlock(t, ctx, db, 64)

		mockChainService := &mockChain.ChainService{State: beaconState, Root: parentRoot[:], ForkChoiceStore: doublylinkedtree.New()}
		v1Alpha1Server := &v1alpha1validator.Server{
			HeadFetcher:       mockChainService,
			SyncChecker:       &mockSync.Sync{IsSyncing: false},
			BlockReceiver:     mockChainService,
			ForkFetcher:       mockChainService,
			TimeFetcher:       mockChainService,
			ForkchoiceFetcher: mockChainService,
			ChainStartFetcher: &mockExecution.Chain{},
			Eth1InfoFetcher:   &mockExecution.Chain{},
			Eth1BlockFetcher:  &mockExecution.Chain{},
			MockEth1Votes:     true,
			AttPool:           attestations.NewPool(),
			SlashingsPool:     slashings.NewPool(),
			ExitPool:          voluntaryexits.NewPool(),
			StateGen:          stategen.New(db, doublylinkedtree.New()),
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
		for i := primitives.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
			proposerSlashing, err := util.GenerateProposerSlashingForValidator(
				beaconState,
				privKeys[i],
				i, /* validator index */
			)
			require.NoError(t, err)
			proposerSlashings[i] = proposerSlashing
			err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), beaconState, proposerSlashing)
			require.NoError(t, err)
		}

		attSlashings := make([]*ethpbalpha.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
		for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings; i++ {
			attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
				beaconState,
				privKeys[i+params.BeaconConfig().MaxProposerSlashings],
				primitives.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings), /* validator index */
			)
			require.NoError(t, err)
			attSlashings[i] = attesterSlashing
			err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), beaconState, attesterSlashing)
			require.NoError(t, err)
		}

		v1Server := &Server{
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
			V1Alpha1Server: v1Alpha1Server,
		}
		randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))
		req := &ethpbv1.ProduceBlockRequest{
			Slot:         1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		resp, err := v1Server.ProduceBlindedBlock(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, ethpbv2.Version_PHASE0, resp.Version)

		containerBlock, ok := resp.Data.Block.(*ethpbv2.BlindedBeaconBlockContainer_Phase0Block)
		require.Equal(t, true, ok)
		blk := containerBlock.Phase0Block
		assert.Equal(t, req.Slot, blk.Slot, "Expected block to have slot of 1")
		assert.DeepEqual(t, parentRoot[:], blk.ParentRoot, "Expected block to have correct parent root")
		assert.DeepEqual(t, randaoReveal, blk.Body.RandaoReveal, "Expected block to have correct randao reveal")
		assert.DeepEqual(t, req.Graffiti, blk.Body.Graffiti, "Expected block to have correct graffiti")
		assert.Equal(t, params.BeaconConfig().MaxProposerSlashings, uint64(len(blk.Body.ProposerSlashings)))
		expectedPropSlashings := make([]*ethpbv1.ProposerSlashing, len(proposerSlashings))
		for i, slash := range proposerSlashings {
			expectedPropSlashings[i] = migration.V1Alpha1ProposerSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedPropSlashings, blk.Body.ProposerSlashings)
		assert.Equal(t, params.BeaconConfig().MaxAttesterSlashings, uint64(len(blk.Body.AttesterSlashings)))
		expectedAttSlashings := make([]*ethpbv1.AttesterSlashing, len(attSlashings))
		for i, slash := range attSlashings {
			expectedAttSlashings[i] = migration.V1Alpha1AttSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedAttSlashings, blk.Body.AttesterSlashings)
	})

	t.Run("Altair", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		params.SetupTestConfigCleanup(t)
		bc := params.BeaconConfig().Copy()
		bc.AltairForkEpoch = primitives.Epoch(0)
		params.OverrideBeaconConfig(bc)

		beaconState, privKeys := util.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
		syncCommittee, err := altair.NextSyncCommittee(context.Background(), beaconState)
		require.NoError(t, err)
		require.NoError(t, beaconState.SetCurrentSyncCommittee(syncCommittee))
		require.NoError(t, beaconState.SetNextSyncCommittee(syncCommittee))

		stateRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err, "Could not hash genesis state")
		genesisBlock := util.NewBeaconBlockAltair()
		genesisBlock.Block.StateRoot = stateRoot[:]
		util.SaveBlock(t, ctx, db, genesisBlock)
		parentRoot, err := genesisBlock.Block.HashTreeRoot()
		require.NoError(t, err)

		require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
		require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

		mockChainService := &mockChain.ChainService{State: beaconState, Root: parentRoot[:], ForkChoiceStore: doublylinkedtree.New()}
		v1Alpha1Server := &v1alpha1validator.Server{
			HeadFetcher:       mockChainService,
			SyncChecker:       &mockSync.Sync{IsSyncing: false},
			BlockReceiver:     mockChainService,
			ForkFetcher:       mockChainService,
			TimeFetcher:       mockChainService,
			ForkchoiceFetcher: mockChainService,
			ChainStartFetcher: &mockExecution.Chain{},
			Eth1InfoFetcher:   &mockExecution.Chain{},
			Eth1BlockFetcher:  &mockExecution.Chain{},
			MockEth1Votes:     true,
			AttPool:           attestations.NewPool(),
			SlashingsPool:     slashings.NewPool(),
			ExitPool:          voluntaryexits.NewPool(),
			StateGen:          stategen.New(db, doublylinkedtree.New()),
			SyncCommitteePool: synccommittee.NewStore(),
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
		for i := primitives.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
			proposerSlashing, err := util.GenerateProposerSlashingForValidator(
				beaconState,
				privKeys[i],
				i, /* validator index */
			)
			require.NoError(t, err)
			proposerSlashings[i] = proposerSlashing
			err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), beaconState, proposerSlashing)
			require.NoError(t, err)
		}

		attSlashings := make([]*ethpbalpha.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
		for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings; i++ {
			attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
				beaconState,
				privKeys[i+params.BeaconConfig().MaxProposerSlashings],
				primitives.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings), /* validator index */
			)
			require.NoError(t, err)
			attSlashings[i] = attesterSlashing
			err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), beaconState, attesterSlashing)
			require.NoError(t, err)
		}

		aggregationBits := bitfield.NewBitvector128()
		for i := range aggregationBits {
			aggregationBits[i] = 0xAA
		}

		syncCommitteeIndices, err := altair.NextSyncCommitteeIndices(context.Background(), beaconState)
		require.NoError(t, err)
		sigs := make([]bls.Signature, 0, len(syncCommitteeIndices))
		for i, indice := range syncCommitteeIndices {
			if aggregationBits.BitAt(uint64(i)) {
				b := p2pType.SSZBytes(parentRoot[:])
				sb, err := signing.ComputeDomainAndSign(beaconState, coreTime.CurrentEpoch(beaconState), &b, params.BeaconConfig().DomainSyncCommittee, privKeys[indice])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				sigs = append(sigs, sig)
			}
		}
		aggregatedSig := bls.AggregateSignatures(sigs).Marshal()
		contribution := &ethpbalpha.SyncCommitteeContribution{
			Slot:              0,
			BlockRoot:         parentRoot[:],
			SubcommitteeIndex: 0,
			AggregationBits:   aggregationBits,
			Signature:         aggregatedSig,
		}
		require.NoError(t, v1Alpha1Server.SyncCommitteePool.SaveSyncCommitteeContribution(contribution))

		v1Server := &Server{
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
			V1Alpha1Server: v1Alpha1Server,
		}
		randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))

		req := &ethpbv1.ProduceBlockRequest{
			Slot:         1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		resp, err := v1Server.ProduceBlindedBlock(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)

		containerBlock, ok := resp.Data.Block.(*ethpbv2.BlindedBeaconBlockContainer_AltairBlock)
		require.Equal(t, true, ok)
		blk := containerBlock.AltairBlock
		assert.Equal(t, req.Slot, blk.Slot, "Expected block to have slot of 1")
		assert.DeepEqual(t, parentRoot[:], blk.ParentRoot, "Expected block to have correct parent root")
		assert.DeepEqual(t, randaoReveal, blk.Body.RandaoReveal, "Expected block to have correct randao reveal")
		assert.DeepEqual(t, req.Graffiti, blk.Body.Graffiti, "Expected block to have correct graffiti")
		assert.Equal(t, params.BeaconConfig().MaxProposerSlashings, uint64(len(blk.Body.ProposerSlashings)))
		expectedPropSlashings := make([]*ethpbv1.ProposerSlashing, len(proposerSlashings))
		for i, slash := range proposerSlashings {
			expectedPropSlashings[i] = migration.V1Alpha1ProposerSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedPropSlashings, blk.Body.ProposerSlashings)
		assert.Equal(t, params.BeaconConfig().MaxAttesterSlashings, uint64(len(blk.Body.AttesterSlashings)))
		expectedAttSlashings := make([]*ethpbv1.AttesterSlashing, len(attSlashings))
		for i, slash := range attSlashings {
			expectedAttSlashings[i] = migration.V1Alpha1AttSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedAttSlashings, blk.Body.AttesterSlashings)
		expectedBits := bitfield.NewBitvector512()
		for i := 0; i <= 15; i++ {
			expectedBits[i] = 0xAA
		}
		assert.DeepEqual(t, expectedBits, blk.Body.SyncAggregate.SyncCommitteeBits)
		assert.DeepEqual(t, aggregatedSig, blk.Body.SyncAggregate.SyncCommitteeSignature)
	})

	t.Run("Bellatrix", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		params.SetupTestConfigCleanup(t)
		bc := params.BeaconConfig().Copy()
		bc.AltairForkEpoch = primitives.Epoch(0)
		bc.BellatrixForkEpoch = primitives.Epoch(1)
		bc.MaxBuilderConsecutiveMissedSlots = params.BeaconConfig().SlotsPerEpoch + 1
		bc.MaxBuilderEpochMissedSlots = params.BeaconConfig().SlotsPerEpoch
		params.OverrideBeaconConfig(bc)

		beaconState, privKeys := util.DeterministicGenesisStateBellatrix(t, params.BeaconConfig().SyncCommitteeSize)
		require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch))
		syncCommittee, err := altair.NextSyncCommittee(context.Background(), beaconState)
		require.NoError(t, err)
		require.NoError(t, beaconState.SetCurrentSyncCommittee(syncCommittee))
		require.NoError(t, beaconState.SetNextSyncCommittee(syncCommittee))

		ts := time.Now()
		ti := ts.Add(-time.Duration(uint64(params.BeaconConfig().SlotsPerEpoch+1)*params.BeaconConfig().SecondsPerSlot) * time.Second)
		require.NoError(t, beaconState.SetGenesisTime(uint64(ti.Unix())))
		random, err := helpers.RandaoMix(beaconState, coreTime.CurrentEpoch(beaconState))
		require.NoError(t, err)
		stateRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err, "Could not hash genesis state")
		genesisBlock := util.NewBeaconBlockBellatrix()
		genesisBlock.Block.StateRoot = stateRoot[:]
		util.SaveBlock(t, ctx, db, genesisBlock)
		parentRoot, err := genesisBlock.Block.HashTreeRoot()
		require.NoError(t, err)

		fb := util.HydrateSignedBeaconBlockBellatrix(&ethpbalpha.SignedBeaconBlockBellatrix{})
		fb.Block.Body.ExecutionPayload.GasLimit = 123
		wfb, err := blocks.NewSignedBeaconBlock(fb)
		require.NoError(t, err)
		require.NoError(t, db.SaveBlock(ctx, wfb), "Could not save block")
		r, err := wfb.Block().HashTreeRoot()
		require.NoError(t, err)

		sk, err := bls.RandKey()
		require.NoError(t, err)

		require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
		require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

		bid := &ethpbalpha.BuilderBid{
			Header: &enginev1.ExecutionPayloadHeader{
				ParentHash:       make([]byte, fieldparams.RootLength),
				FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
				StateRoot:        make([]byte, fieldparams.RootLength),
				ReceiptsRoot:     make([]byte, fieldparams.RootLength),
				LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
				PrevRandao:       random,
				BaseFeePerGas:    make([]byte, fieldparams.RootLength),
				BlockHash:        make([]byte, fieldparams.RootLength),
				TransactionsRoot: make([]byte, fieldparams.RootLength),
				BlockNumber:      1,
				Timestamp:        uint64(ts.Unix()),
			},
			Pubkey: sk.PublicKey().Marshal(),
			Value:  bytesutil.PadTo([]byte{1, 2, 3}, 32),
		}
		d := params.BeaconConfig().DomainApplicationBuilder
		domain, err := signing.ComputeDomain(d, nil, nil)
		require.NoError(t, err)
		sr, err := signing.ComputeSigningRoot(bid, domain)
		require.NoError(t, err)
		sBid := &ethpbalpha.SignedBuilderBid{
			Message:   bid,
			Signature: sk.Sign(sr[:]).Marshal(),
		}

		fcs := doublylinkedtree.New()
		fcs.SetGenesisTime(uint64(ti.Unix()))
		chainSlot := primitives.Slot(params.BeaconConfig().SlotsPerEpoch + 1)
		mockChainService := &mockChain.ChainService{Slot: &chainSlot, Genesis: ti, State: beaconState, Root: parentRoot[:], ForkChoiceStore: fcs, Block: wfb}
		v1Alpha1Server := &v1alpha1validator.Server{
			BeaconDB:               db,
			ForkFetcher:            mockChainService,
			ForkchoiceFetcher:      mockChainService,
			TimeFetcher:            mockChainService,
			HeadFetcher:            mockChainService,
			OptimisticModeFetcher:  mockChainService,
			SyncChecker:            &mockSync.Sync{IsSyncing: false},
			BlockReceiver:          mockChainService,
			ChainStartFetcher:      &mockExecution.Chain{GenesisState: beaconState},
			Eth1InfoFetcher:        &mockExecution.Chain{},
			Eth1BlockFetcher:       &mockExecution.Chain{},
			MockEth1Votes:          true,
			AttPool:                attestations.NewPool(),
			SlashingsPool:          slashings.NewPool(),
			ExitPool:               voluntaryexits.NewPool(),
			StateGen:               stategen.New(db, doublylinkedtree.New()),
			SyncCommitteePool:      synccommittee.NewStore(),
			ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
			BlockBuilder: &builderTest.MockBuilderService{
				HasConfigured: true,
				Bid:           sBid,
			},
			FinalizationFetcher: &mockChain.ChainService{
				FinalizedCheckPoint: &ethpbalpha.Checkpoint{
					Root: r[:],
				},
			},
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
		for i := primitives.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
			proposerSlashing, err := util.GenerateProposerSlashingForValidator(
				beaconState,
				privKeys[i],
				i,
			)
			require.NoError(t, err)
			proposerSlashings[i] = proposerSlashing
			err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), beaconState, proposerSlashing)
			require.NoError(t, err)
		}

		attSlashings := make([]*ethpbalpha.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
		for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings; i++ {
			attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
				beaconState,
				privKeys[i+params.BeaconConfig().MaxProposerSlashings],
				primitives.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings), /* validator index */
			)
			require.NoError(t, err)
			attSlashings[i] = attesterSlashing
			err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), beaconState, attesterSlashing)
			require.NoError(t, err)
		}

		aggregationBits := bitfield.NewBitvector128()
		for i := range aggregationBits {
			aggregationBits[i] = 0xAA
		}

		syncCommitteeIndices, err := altair.NextSyncCommitteeIndices(context.Background(), beaconState)
		require.NoError(t, err)
		sigs := make([]bls.Signature, 0, len(syncCommitteeIndices))
		for i, indice := range syncCommitteeIndices {
			if aggregationBits.BitAt(uint64(i)) {
				b := p2pType.SSZBytes(parentRoot[:])
				sb, err := signing.ComputeDomainAndSign(beaconState, coreTime.CurrentEpoch(beaconState), &b, params.BeaconConfig().DomainSyncCommittee, privKeys[indice])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				sigs = append(sigs, sig)
			}
		}
		aggregatedSig := bls.AggregateSignatures(sigs).Marshal()
		contribution := &ethpbalpha.SyncCommitteeContribution{
			Slot:              params.BeaconConfig().SlotsPerEpoch,
			BlockRoot:         parentRoot[:],
			SubcommitteeIndex: 0,
			AggregationBits:   aggregationBits,
			Signature:         aggregatedSig,
		}
		require.NoError(t, v1Alpha1Server.SyncCommitteePool.SaveSyncCommitteeContribution(contribution))

		v1Server := &Server{
			V1Alpha1Server:        v1Alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			TimeFetcher:           &mockChain.ChainService{},
			OptimisticModeFetcher: &mockChain.ChainService{},
		}
		randaoReveal, err := util.RandaoReveal(beaconState, 1, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))

		copied := beaconState.Copy()
		require.NoError(t, copied.SetSlot(params.BeaconConfig().SlotsPerEpoch+1))
		idx, err := helpers.BeaconProposerIndex(ctx, copied)
		require.NoError(t, err)
		require.NoError(t,
			db.SaveRegistrationsByValidatorIDs(ctx, []primitives.ValidatorIndex{idx},
				[]*ethpbalpha.ValidatorRegistrationV1{{FeeRecipient: make([]byte, 20), Pubkey: make([]byte, 48)}}))

		req := &ethpbv1.ProduceBlockRequest{
			Slot:         params.BeaconConfig().SlotsPerEpoch + 1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		resp, err := v1Server.ProduceBlindedBlock(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)

		containerBlock, ok := resp.Data.Block.(*ethpbv2.BlindedBeaconBlockContainer_BellatrixBlock)
		require.Equal(t, true, ok)
		blk := containerBlock.BellatrixBlock
		assert.Equal(t, req.Slot, blk.Slot, "Expected block to have slot of 33")
		assert.DeepEqual(t, parentRoot[:], blk.ParentRoot, "Expected block to have correct parent root")
		assert.DeepEqual(t, randaoReveal, blk.Body.RandaoReveal, "Expected block to have correct randao reveal")
		assert.DeepEqual(t, req.Graffiti, blk.Body.Graffiti, "Expected block to have correct graffiti")
		assert.Equal(t, params.BeaconConfig().MaxProposerSlashings, uint64(len(blk.Body.ProposerSlashings)))
		expectedPropSlashings := make([]*ethpbv1.ProposerSlashing, len(proposerSlashings))
		for i, slash := range proposerSlashings {
			expectedPropSlashings[i] = migration.V1Alpha1ProposerSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedPropSlashings, blk.Body.ProposerSlashings)
		assert.Equal(t, params.BeaconConfig().MaxAttesterSlashings, uint64(len(blk.Body.AttesterSlashings)))
		expectedAttSlashings := make([]*ethpbv1.AttesterSlashing, len(attSlashings))
		for i, slash := range attSlashings {
			expectedAttSlashings[i] = migration.V1Alpha1AttSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedAttSlashings, blk.Body.AttesterSlashings)
		expectedBits := bitfield.NewBitvector512()
		for i := 0; i <= 15; i++ {
			expectedBits[i] = 0xAA
		}
		assert.DeepEqual(t, expectedBits, blk.Body.SyncAggregate.SyncCommitteeBits)
		assert.DeepEqual(t, aggregatedSig, blk.Body.SyncAggregate.SyncCommitteeSignature)
	})

	t.Run("Capella", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		params.SetupTestConfigCleanup(t)
		bc := params.BeaconConfig().Copy()
		bc.AltairForkEpoch = primitives.Epoch(0)
		bc.BellatrixForkEpoch = primitives.Epoch(1)
		bc.CapellaForkEpoch = primitives.Epoch(2)
		bc.MaxBuilderConsecutiveMissedSlots = params.BeaconConfig().SlotsPerEpoch*2 + 1
		bc.MaxBuilderEpochMissedSlots = params.BeaconConfig().SlotsPerEpoch * 2
		params.OverrideBeaconConfig(bc)

		beaconState, privKeys := util.DeterministicGenesisStateCapella(t, params.BeaconConfig().SyncCommitteeSize)
		require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch*2))
		syncCommittee, err := altair.NextSyncCommittee(context.Background(), beaconState)
		require.NoError(t, err)
		require.NoError(t, beaconState.SetCurrentSyncCommittee(syncCommittee))
		require.NoError(t, beaconState.SetNextSyncCommittee(syncCommittee))

		ts := time.Now()
		ti := ts.Add(-time.Duration(uint64(2*params.BeaconConfig().SlotsPerEpoch+1)*params.BeaconConfig().SecondsPerSlot) * time.Second)
		require.NoError(t, beaconState.SetGenesisTime(uint64(ti.Unix())))

		stateRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err, "Could not hash genesis state")
		genesisBlock := util.NewBeaconBlockCapella()
		genesisBlock.Block.StateRoot = stateRoot[:]
		util.SaveBlock(t, ctx, db, genesisBlock)
		parentRoot, err := genesisBlock.Block.HashTreeRoot()
		require.NoError(t, err)

		require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
		require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

		fb := util.HydrateSignedBeaconBlockCapella(&ethpbalpha.SignedBeaconBlockCapella{})
		fb.Block.Body.ExecutionPayload.GasLimit = 123
		wfb, err := blocks.NewSignedBeaconBlock(fb)
		require.NoError(t, err)
		require.NoError(t, db.SaveBlock(ctx, wfb), "Could not save block")
		r, err := wfb.Block().HashTreeRoot()
		require.NoError(t, err)

		sk, err := bls.RandKey()
		require.NoError(t, err)
		random, err := helpers.RandaoMix(beaconState, coreTime.CurrentEpoch(beaconState))
		require.NoError(t, err)
		wds, err := beaconState.ExpectedWithdrawals()
		require.NoError(t, err)
		wr, err := ssz.WithdrawalSliceRoot(wds, fieldparams.MaxWithdrawalsPerPayload)
		require.NoError(t, err)
		v := big.NewInt(1e9) // minimal 1 gwei
		bid := &ethpbalpha.BuilderBidCapella{
			Header: &enginev1.ExecutionPayloadHeaderCapella{
				ParentHash:       make([]byte, fieldparams.RootLength),
				FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
				StateRoot:        make([]byte, fieldparams.RootLength),
				ReceiptsRoot:     make([]byte, fieldparams.RootLength),
				LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
				PrevRandao:       random,
				BaseFeePerGas:    make([]byte, fieldparams.RootLength),
				BlockHash:        make([]byte, fieldparams.RootLength),
				TransactionsRoot: make([]byte, fieldparams.RootLength),
				BlockNumber:      1,
				Timestamp:        uint64(ts.Unix()),
				WithdrawalsRoot:  wr[:],
			},
			Pubkey: sk.PublicKey().Marshal(),
			Value:  bytesutil.PadTo(bytesutil.ReverseByteOrder(v.Bytes()), 32),
		}
		d := params.BeaconConfig().DomainApplicationBuilder
		domain, err := signing.ComputeDomain(d, nil, nil)
		require.NoError(t, err)
		sr, err := signing.ComputeSigningRoot(bid, domain)
		require.NoError(t, err)
		sBid := &ethpbalpha.SignedBuilderBidCapella{
			Message:   bid,
			Signature: sk.Sign(sr[:]).Marshal(),
		}
		id := &enginev1.PayloadIDBytes{0x1}

		chainSlot := primitives.Slot(2*params.BeaconConfig().SlotsPerEpoch + 1)
		fcs := doublylinkedtree.New()
		fcs.SetGenesisTime(uint64(ti.Unix()))
		mockChainService := &mockChain.ChainService{Slot: &chainSlot, Genesis: ti, State: beaconState, Root: parentRoot[:], ForkChoiceStore: fcs, Block: wfb}
		v1Alpha1Server := &v1alpha1validator.Server{
			ExecutionEngineCaller:  &mockExecution.EngineClient{PayloadIDBytes: id, ExecutionPayloadCapella: &enginev1.ExecutionPayloadCapella{BlockNumber: 1, Withdrawals: wds}, BlockValue: big.NewInt(0)},
			BeaconDB:               db,
			ForkFetcher:            mockChainService,
			ForkchoiceFetcher:      mockChainService,
			TimeFetcher:            mockChainService,
			HeadFetcher:            mockChainService,
			OptimisticModeFetcher:  mockChainService,
			SyncChecker:            &mockSync.Sync{IsSyncing: false},
			BlockReceiver:          mockChainService,
			ChainStartFetcher:      &mockExecution.Chain{},
			Eth1InfoFetcher:        &mockExecution.Chain{},
			Eth1BlockFetcher:       &mockExecution.Chain{},
			MockEth1Votes:          true,
			AttPool:                attestations.NewPool(),
			SlashingsPool:          slashings.NewPool(),
			ExitPool:               voluntaryexits.NewPool(),
			BLSChangesPool:         blstoexec.NewPool(),
			StateGen:               stategen.New(db, fcs),
			SyncCommitteePool:      synccommittee.NewStore(),
			ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
			BlockBuilder: &builderTest.MockBuilderService{
				HasConfigured: true,
				BidCapella:    sBid,
			},
			FinalizationFetcher: &mockChain.ChainService{
				FinalizedCheckPoint: &ethpbalpha.Checkpoint{
					Root: r[:],
				},
			},
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
		for i := primitives.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
			proposerSlashing, err := util.GenerateProposerSlashingForValidator(
				beaconState,
				privKeys[i],
				i,
			)
			require.NoError(t, err)
			proposerSlashings[i] = proposerSlashing
			err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), beaconState, proposerSlashing)
			require.NoError(t, err)
		}

		attSlashings := make([]*ethpbalpha.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
		for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings; i++ {
			attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
				beaconState,
				privKeys[i+params.BeaconConfig().MaxProposerSlashings],
				primitives.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings), /* validator index */
			)
			require.NoError(t, err)
			attSlashings[i] = attesterSlashing
			err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), beaconState, attesterSlashing)
			require.NoError(t, err)
		}

		aggregationBits := bitfield.NewBitvector128()
		for i := range aggregationBits {
			aggregationBits[i] = 0xAA
		}

		syncCommitteeIndices, err := altair.NextSyncCommitteeIndices(context.Background(), beaconState)
		require.NoError(t, err)
		sigs := make([]bls.Signature, 0, len(syncCommitteeIndices))
		for i, indice := range syncCommitteeIndices {
			if aggregationBits.BitAt(uint64(i)) {
				b := p2pType.SSZBytes(parentRoot[:])
				sb, err := signing.ComputeDomainAndSign(beaconState, coreTime.CurrentEpoch(beaconState), &b, params.BeaconConfig().DomainSyncCommittee, privKeys[indice])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				sigs = append(sigs, sig)
			}
		}
		aggregatedSig := bls.AggregateSignatures(sigs).Marshal()
		contribution := &ethpbalpha.SyncCommitteeContribution{
			Slot:              params.BeaconConfig().SlotsPerEpoch * 2,
			BlockRoot:         parentRoot[:],
			SubcommitteeIndex: 0,
			AggregationBits:   aggregationBits,
			Signature:         aggregatedSig,
		}
		require.NoError(t, v1Alpha1Server.SyncCommitteePool.SaveSyncCommitteeContribution(contribution))

		v1Server := &Server{
			V1Alpha1Server:        v1Alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			TimeFetcher:           &mockChain.ChainService{},
			OptimisticModeFetcher: &mockChain.ChainService{},
		}
		randaoReveal, err := util.RandaoReveal(beaconState, 1, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))

		copied := beaconState.Copy()
		require.NoError(t, copied.SetSlot(params.BeaconConfig().SlotsPerEpoch*2+1))
		idx, err := helpers.BeaconProposerIndex(ctx, copied)
		require.NoError(t, err)
		require.NoError(t,
			db.SaveRegistrationsByValidatorIDs(ctx, []primitives.ValidatorIndex{idx},
				[]*ethpbalpha.ValidatorRegistrationV1{{FeeRecipient: make([]byte, 20), Pubkey: make([]byte, 48)}}))

		req := &ethpbv1.ProduceBlockRequest{
			Slot:         params.BeaconConfig().SlotsPerEpoch*2 + 1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		resp, err := v1Server.ProduceBlindedBlock(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, ethpbv2.Version_CAPELLA, resp.Version)

		containerBlock, ok := resp.Data.Block.(*ethpbv2.BlindedBeaconBlockContainer_CapellaBlock)
		require.Equal(t, true, ok)
		blk := containerBlock.CapellaBlock
		assert.Equal(t, req.Slot, blk.Slot, "Expected block to have slot of 1")
		assert.DeepEqual(t, parentRoot[:], blk.ParentRoot, "Expected block to have correct parent root")
		assert.DeepEqual(t, randaoReveal, blk.Body.RandaoReveal, "Expected block to have correct randao reveal")
		assert.DeepEqual(t, req.Graffiti, blk.Body.Graffiti, "Expected block to have correct graffiti")
		assert.Equal(t, params.BeaconConfig().MaxProposerSlashings, uint64(len(blk.Body.ProposerSlashings)))
		expectedPropSlashings := make([]*ethpbv1.ProposerSlashing, len(proposerSlashings))
		for i, slash := range proposerSlashings {
			expectedPropSlashings[i] = migration.V1Alpha1ProposerSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedPropSlashings, blk.Body.ProposerSlashings)
		assert.Equal(t, params.BeaconConfig().MaxAttesterSlashings, uint64(len(blk.Body.AttesterSlashings)))
		expectedAttSlashings := make([]*ethpbv1.AttesterSlashing, len(attSlashings))
		for i, slash := range attSlashings {
			expectedAttSlashings[i] = migration.V1Alpha1AttSlashingToV1(slash)
		}
		assert.DeepEqual(t, expectedAttSlashings, blk.Body.AttesterSlashings)
		expectedBits := bitfield.NewBitvector512()
		for i := 0; i <= 15; i++ {
			expectedBits[i] = 0xAA
		}
		assert.DeepEqual(t, expectedBits, blk.Body.SyncAggregate.SyncCommitteeBits)
		assert.DeepEqual(t, aggregatedSig, blk.Body.SyncAggregate.SyncCommitteeSignature)
	})
	t.Run("Unsupported Block Type", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		params.SetupTestConfigCleanup(t)
		bc := params.BeaconConfig().Copy()
		bc.AltairForkEpoch = primitives.Epoch(0)
		bc.BellatrixForkEpoch = primitives.Epoch(1)
		bc.CapellaForkEpoch = primitives.Epoch(2)
		params.OverrideBeaconConfig(bc)

		beaconState, privKeys := util.DeterministicGenesisStateCapella(t, params.BeaconConfig().SyncCommitteeSize)
		require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch*2))
		syncCommittee, err := altair.NextSyncCommittee(context.Background(), beaconState)
		require.NoError(t, err)
		require.NoError(t, beaconState.SetCurrentSyncCommittee(syncCommittee))
		require.NoError(t, beaconState.SetNextSyncCommittee(syncCommittee))
		require.NoError(t, beaconState.UpdateRandaoMixesAtIndex(2, bytesutil.PadTo([]byte("prev_randao"), 32)))

		vals := beaconState.Validators()
		creds0 := make([]byte, 32)
		creds0[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
		copy(creds0[state_native.ETH1AddressOffset:], "address0")
		vals[0].WithdrawalCredentials = creds0
		vals[0].EffectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		creds1 := make([]byte, 32)
		creds1[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
		copy(creds1[state_native.ETH1AddressOffset:], "address1")
		vals[1].WithdrawalCredentials = creds1
		vals[1].EffectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		require.NoError(t, beaconState.SetValidators(vals))
		balances := beaconState.Balances()
		balances[0] += 123
		balances[1] += 123
		require.NoError(t, beaconState.SetBalances(balances))
		withdrawals := []*enginev1.Withdrawal{
			{
				Index:          0,
				ValidatorIndex: 0,
				Address:        bytesutil.PadTo([]byte("address0"), 20),
				Amount:         123,
			},
			{
				Index:          1,
				ValidatorIndex: 1,
				Address:        bytesutil.PadTo([]byte("address1"), 20),
				Amount:         123,
			},
		}
		withdrawalsRoot, err := ssz.WithdrawalSliceRoot(withdrawals, 2)
		require.NoError(t, err)

		payloadHeader, err := blocks.WrappedExecutionPayloadHeaderCapella(&enginev1.ExecutionPayloadHeaderCapella{
			ParentHash:   bytesutil.PadTo([]byte("parent_hash"), 32),
			FeeRecipient: bytesutil.PadTo([]byte("fee_recipient"), 20),
			StateRoot:    bytesutil.PadTo([]byte("state_root"), 32),
			ReceiptsRoot: bytesutil.PadTo([]byte("receipts_root"), 32),
			LogsBloom:    bytesutil.PadTo([]byte("logs_bloom"), 256),
			PrevRandao:   bytesutil.PadTo([]byte("prev_randao"), 32),
			BlockNumber:  123,
			GasLimit:     123,
			GasUsed:      123,
			// State time at slot 65
			Timestamp:     780,
			ExtraData:     bytesutil.PadTo([]byte("extra_data"), 32),
			BaseFeePerGas: bytesutil.PadTo([]byte("base_fee_per_gas"), 32),
			// Must be equal to payload.ParentHash
			BlockHash:        bytesutil.PadTo([]byte("equal_hash"), 32),
			TransactionsRoot: bytesutil.PadTo([]byte("transactions_root"), 32),
			WithdrawalsRoot:  withdrawalsRoot[:],
		}, big.NewInt(0))
		require.NoError(t, err)
		require.NoError(t, beaconState.SetLatestExecutionPayloadHeader(payloadHeader))

		payload := &enginev1.ExecutionPayloadCapella{
			// Must be equal to payloadHeader.BlockHash
			ParentHash:   bytesutil.PadTo([]byte("equal_hash"), 32),
			FeeRecipient: bytesutil.PadTo([]byte("fee_recipient"), 20),
			StateRoot:    bytesutil.PadTo([]byte("state_root"), 32),
			ReceiptsRoot: bytesutil.PadTo([]byte("receipts_root"), 32),
			LogsBloom:    bytesutil.PadTo([]byte("logs_bloom"), 256),
			PrevRandao:   bytesutil.PadTo([]byte("prev_randao"), 32),
			BlockNumber:  123,
			GasLimit:     123,
			GasUsed:      123,
			// State time at slot 65
			Timestamp:     780,
			ExtraData:     bytesutil.PadTo([]byte("extra_data"), 32),
			BaseFeePerGas: bytesutil.PadTo([]byte("base_fee_per_gas"), 32),
			BlockHash:     bytesutil.PadTo([]byte("block_hash"), 32),
			Transactions:  [][]byte{[]byte("transaction1"), []byte("transaction2")},
			Withdrawals:   withdrawals,
		}

		stateRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err, "Could not hash genesis state")
		genesisBlock := util.NewBeaconBlockCapella()
		genesisBlock.Block.StateRoot = stateRoot[:]
		util.SaveBlock(t, ctx, db, genesisBlock)
		parentRoot, err := genesisBlock.Block.HashTreeRoot()
		require.NoError(t, err)

		require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
		require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

		var payloadIdBytes enginev1.PayloadIDBytes
		copy(payloadIdBytes[:], "payload_id")
		mockChainService := &mockChain.ChainService{State: beaconState, Root: parentRoot[:], ForkChoiceStore: doublylinkedtree.New()}
		mockExecutionChain := &mockExecution.Chain{}
		v1Alpha1Server := &v1alpha1validator.Server{
			ExecutionEngineCaller: &mockExecution.EngineClient{
				OverrideValidHash:       bytesutil.ToBytes32([]byte("parent_hash")),
				PayloadIDBytes:          &payloadIdBytes,
				ExecutionPayloadCapella: payload,
			},
			BeaconDB:               db,
			TimeFetcher:            mockChainService,
			HeadFetcher:            mockChainService,
			OptimisticModeFetcher:  mockChainService,
			SyncChecker:            &mockSync.Sync{IsSyncing: false},
			BlockReceiver:          mockChainService,
			ForkFetcher:            mockChainService,
			ForkchoiceFetcher:      mockChainService,
			FinalizationFetcher:    mockChainService,
			ChainStartFetcher:      mockExecutionChain,
			Eth1InfoFetcher:        mockExecutionChain,
			Eth1BlockFetcher:       mockExecutionChain,
			MockEth1Votes:          true,
			AttPool:                attestations.NewPool(),
			SlashingsPool:          slashings.NewPool(),
			ExitPool:               voluntaryexits.NewPool(),
			BLSChangesPool:         blstoexec.NewPool(),
			StateGen:               stategen.New(db, doublylinkedtree.New()),
			SyncCommitteePool:      synccommittee.NewStore(),
			ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
			BlockBuilder: &builderTest.MockBuilderService{
				HasConfigured: false,
			},
		}

		v1Server := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			V1Alpha1Server:        v1Alpha1Server,
			OptimisticModeFetcher: &mockChain.ChainService{},
		}
		randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))
		req := &ethpbv1.ProduceBlockRequest{
			Slot:         params.BeaconConfig().SlotsPerEpoch*2 + 1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		_, err = v1Server.ProduceBlindedBlock(ctx, req)
		require.ErrorContains(t, " block was not a supported blinded block type", err)
	})
}

func TestProduceBlindedBlockSSZ(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		ctx := context.Background()

		db := dbutil.SetupDB(t)

		bs, parentRoot, privKeys := util.DeterministicGenesisStateWithGenesisBlock(t, ctx, db, 2)

		mockChainService := &mockChain.ChainService{State: bs, Root: parentRoot[:], ForkChoiceStore: doublylinkedtree.New()}
		v1Alpha1Server := &v1alpha1validator.Server{
			HeadFetcher:       mockChainService,
			SyncChecker:       &mockSync.Sync{IsSyncing: false},
			BlockReceiver:     mockChainService,
			TimeFetcher:       mockChainService,
			ForkFetcher:       mockChainService,
			ForkchoiceFetcher: mockChainService,
			ChainStartFetcher: &mockExecution.Chain{},
			Eth1InfoFetcher:   &mockExecution.Chain{},
			Eth1BlockFetcher:  &mockExecution.Chain{},
			MockEth1Votes:     true,
			AttPool:           attestations.NewPool(),
			SlashingsPool:     slashings.NewPool(),
			ExitPool:          voluntaryexits.NewPool(),
			StateGen:          stategen.New(db, doublylinkedtree.New()),
			BlockBuilder:      &builderTest.MockBuilderService{HasConfigured: true},
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, 1)
		proposerSlashing, err := util.GenerateProposerSlashingForValidator(
			bs,
			privKeys[0],
			0,
		)
		require.NoError(t, err)
		proposerSlashings[0] = proposerSlashing
		err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), bs, proposerSlashing)
		require.NoError(t, err)

		attSlashings := make([]*ethpbalpha.AttesterSlashing, 1)
		attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
			bs,
			privKeys[1],
			1,
		)
		require.NoError(t, err)
		attSlashings[0] = attesterSlashing
		err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), bs, attesterSlashing)
		require.NoError(t, err)

		v1Server := &Server{
			V1Alpha1Server: v1Alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}
		randaoReveal, err := util.RandaoReveal(bs, 0, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))
		req := &ethpbv1.ProduceBlockRequest{
			Slot:         1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		resp, err := v1Server.ProduceBlindedBlockSSZ(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, ethpbv2.Version_PHASE0, resp.Version)

		expectedBlock := &ethpbv1.BeaconBlock{
			Slot:          1,
			ProposerIndex: 0,
			ParentRoot:    []byte{164, 47, 20, 157, 2, 202, 58, 173, 57, 154, 254, 181, 153, 74, 40, 89, 159, 74, 62, 247, 28, 222, 153, 182, 168, 79, 170, 149, 80, 99, 97, 32},
			StateRoot:     []byte{224, 94, 112, 87, 163, 57, 192, 233, 181, 179, 212, 9, 226, 214, 65, 238, 189, 114, 223, 101, 215, 146, 163, 140, 92, 242, 35, 82, 222, 154, 127, 136},
			Body: &ethpbv1.BeaconBlockBody{
				RandaoReveal: randaoReveal,
				Eth1Data: &ethpbv1.Eth1Data{
					DepositRoot:  []byte{124, 159, 161, 54, 212, 65, 63, 166, 23, 54, 55, 232, 131, 182, 153, 141, 50, 225, 214, 117, 248, 140, 221, 255, 157, 203, 207, 51, 24, 32, 244, 184},
					DepositCount: 2,
					BlockHash:    []byte{8, 83, 63, 107, 189, 73, 117, 17, 62, 79, 12, 177, 4, 171, 205, 236, 29, 134, 217, 157, 87, 130, 180, 169, 167, 246, 39, 12, 14, 187, 106, 39},
				},
				Graffiti: graffiti[:],
				ProposerSlashings: []*ethpbv1.ProposerSlashing{
					{
						SignedHeader_1: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_1.Header.Slot,
								ProposerIndex: proposerSlashing.Header_1.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_1.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_1.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_1.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_1.Signature,
						},
						SignedHeader_2: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_2.Header.Slot,
								ProposerIndex: proposerSlashing.Header_2.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_2.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_2.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_2.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_2.Signature,
						},
					},
				},
				AttesterSlashings: []*ethpbv1.AttesterSlashing{
					{
						Attestation_1: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_1.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_1.Data.Slot,
								Index:           attesterSlashing.Attestation_1.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_1.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_1.Signature,
						},
						Attestation_2: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_2.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_2.Data.Slot,
								Index:           attesterSlashing.Attestation_2.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_2.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_2.Signature,
						},
					},
				},
			},
		}
		expectedData, err := expectedBlock.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})

	t.Run("Altair", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		params.SetupTestConfigCleanup(t)
		bc := params.BeaconConfig()
		bc.AltairForkEpoch = primitives.Epoch(0)
		params.OverrideBeaconConfig(bc)

		bs, privKeys := util.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
		syncCommittee, err := altair.NextSyncCommittee(context.Background(), bs)
		require.NoError(t, err)
		require.NoError(t, bs.SetCurrentSyncCommittee(syncCommittee))
		require.NoError(t, bs.SetNextSyncCommittee(syncCommittee))

		stateRoot, err := bs.HashTreeRoot(ctx)
		require.NoError(t, err, "Could not hash genesis state")
		genesisBlock := util.NewBeaconBlockAltair()
		genesisBlock.Block.StateRoot = stateRoot[:]
		util.SaveBlock(t, ctx, db, genesisBlock)
		parentRoot, err := genesisBlock.Block.HashTreeRoot()
		require.NoError(t, err)

		require.NoError(t, db.SaveState(ctx, bs, parentRoot), "Could not save genesis state")
		require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

		mockChainService := &mockChain.ChainService{State: bs, Root: parentRoot[:], ForkChoiceStore: doublylinkedtree.New()}
		v1Alpha1Server := &v1alpha1validator.Server{
			HeadFetcher:       mockChainService,
			SyncChecker:       &mockSync.Sync{IsSyncing: false},
			BlockReceiver:     &mockChain.ChainService{},
			TimeFetcher:       mockChainService,
			ForkFetcher:       mockChainService,
			ForkchoiceFetcher: mockChainService,
			ChainStartFetcher: &mockExecution.Chain{},
			Eth1InfoFetcher:   &mockExecution.Chain{},
			Eth1BlockFetcher:  &mockExecution.Chain{},
			MockEth1Votes:     true,
			AttPool:           attestations.NewPool(),
			SlashingsPool:     slashings.NewPool(),
			ExitPool:          voluntaryexits.NewPool(),
			StateGen:          stategen.New(db, doublylinkedtree.New()),
			SyncCommitteePool: synccommittee.NewStore(),
			BlockBuilder:      &builderTest.MockBuilderService{HasConfigured: true},
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, 1)
		proposerSlashing, err := util.GenerateProposerSlashingForValidator(
			bs,
			privKeys[0],
			0,
		)
		require.NoError(t, err)
		proposerSlashings[0] = proposerSlashing
		err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), bs, proposerSlashing)
		require.NoError(t, err)

		attSlashings := make([]*ethpbalpha.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
		attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
			bs,
			privKeys[1],
			1,
		)
		require.NoError(t, err)
		attSlashings[0] = attesterSlashing
		err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), bs, attesterSlashing)
		require.NoError(t, err)

		aggregationBits := bitfield.NewBitvector128()
		for i := range aggregationBits {
			aggregationBits[i] = 0xAA
		}

		syncCommitteeIndices, err := altair.NextSyncCommitteeIndices(context.Background(), bs)
		require.NoError(t, err)
		sigs := make([]bls.Signature, 0, len(syncCommitteeIndices))
		for i, indice := range syncCommitteeIndices {
			if aggregationBits.BitAt(uint64(i)) {
				b := p2pType.SSZBytes(parentRoot[:])
				sb, err := signing.ComputeDomainAndSign(bs, coreTime.CurrentEpoch(bs), &b, params.BeaconConfig().DomainSyncCommittee, privKeys[indice])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				sigs = append(sigs, sig)
			}
		}
		aggregatedSig := bls.AggregateSignatures(sigs).Marshal()
		contribution := &ethpbalpha.SyncCommitteeContribution{
			Slot:              0,
			BlockRoot:         parentRoot[:],
			SubcommitteeIndex: 0,
			AggregationBits:   aggregationBits,
			Signature:         aggregatedSig,
		}
		require.NoError(t, v1Alpha1Server.SyncCommitteePool.SaveSyncCommitteeContribution(contribution))

		v1Server := &Server{
			V1Alpha1Server: v1Alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}
		randaoReveal, err := util.RandaoReveal(bs, 0, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))

		req := &ethpbv1.ProduceBlockRequest{
			Slot:         1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		resp, err := v1Server.ProduceBlindedBlockSSZ(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)

		expectedBlock := &ethpbv2.BeaconBlockAltair{
			Slot:          1,
			ProposerIndex: 19,
			ParentRoot:    []byte{162, 206, 194, 54, 242, 248, 88, 148, 193, 141, 39, 23, 91, 116, 219, 219, 2, 248, 4, 155, 159, 201, 41, 156, 130, 57, 167, 176, 153, 18, 73, 148},
			StateRoot:     []byte{144, 220, 158, 2, 142, 57, 111, 170, 148, 225, 129, 23, 103, 232, 44, 1, 222, 77, 36, 110, 118, 237, 184, 77, 253, 182, 0, 62, 168, 56, 105, 95},
			Body: &ethpbv2.BeaconBlockBodyAltair{
				RandaoReveal: randaoReveal,
				Eth1Data: &ethpbv1.Eth1Data{
					DepositRoot:  []byte{124, 159, 161, 54, 212, 65, 63, 166, 23, 54, 55, 232, 131, 182, 153, 141, 50, 225, 214, 117, 248, 140, 221, 255, 157, 203, 207, 51, 24, 32, 244, 184},
					DepositCount: params.BeaconConfig().SyncCommitteeSize,
					BlockHash:    []byte{8, 83, 63, 107, 189, 73, 117, 17, 62, 79, 12, 177, 4, 171, 205, 236, 29, 134, 217, 157, 87, 130, 180, 169, 167, 246, 39, 12, 14, 187, 106, 39},
				},
				Graffiti: graffiti[:],
				ProposerSlashings: []*ethpbv1.ProposerSlashing{
					{
						SignedHeader_1: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_1.Header.Slot,
								ProposerIndex: proposerSlashing.Header_1.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_1.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_1.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_1.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_1.Signature,
						},
						SignedHeader_2: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_2.Header.Slot,
								ProposerIndex: proposerSlashing.Header_2.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_2.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_2.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_2.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_2.Signature,
						},
					},
				},
				AttesterSlashings: []*ethpbv1.AttesterSlashing{
					{
						Attestation_1: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_1.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_1.Data.Slot,
								Index:           attesterSlashing.Attestation_1.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_1.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_1.Signature,
						},
						Attestation_2: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_2.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_2.Data.Slot,
								Index:           attesterSlashing.Attestation_2.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_2.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_2.Signature,
						},
					},
				},
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits:      []byte{170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					SyncCommitteeSignature: []byte{130, 228, 60, 221, 180, 9, 29, 148, 136, 255, 135, 183, 146, 130, 88, 240, 116, 219, 183, 208, 148, 211, 202, 78, 240, 120, 60, 99, 77, 76, 109, 210, 163, 243, 244, 25, 70, 184, 29, 252, 138, 128, 202, 173, 1, 166, 48, 49, 11, 136, 42, 124, 163, 187, 206, 253, 214, 149, 114, 137, 146, 123, 197, 187, 250, 204, 59, 196, 87, 195, 48, 11, 116, 123, 58, 49, 62, 98, 193, 166, 0, 172, 15, 253, 130, 88, 46, 110, 45, 84, 57, 107, 83, 182, 127, 105},
				},
			},
		}
		expectedData, err := expectedBlock.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})

	t.Run("Bellatrix", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		params.SetupTestConfigCleanup(t)
		bc := params.BeaconConfig()
		bc.AltairForkEpoch = primitives.Epoch(0)
		bc.BellatrixForkEpoch = primitives.Epoch(1)
		params.OverrideBeaconConfig(bc)

		bs, privKeys := util.DeterministicGenesisStateBellatrix(t, params.BeaconConfig().SyncCommitteeSize)
		require.NoError(t, bs.SetSlot(params.BeaconConfig().SlotsPerEpoch))
		syncCommittee, err := altair.NextSyncCommittee(context.Background(), bs)
		require.NoError(t, err)
		require.NoError(t, bs.SetCurrentSyncCommittee(syncCommittee))
		require.NoError(t, bs.SetNextSyncCommittee(syncCommittee))

		stateRoot, err := bs.HashTreeRoot(ctx)
		require.NoError(t, err, "Could not hash genesis state")
		genesisBlock := util.NewBeaconBlockBellatrix()
		genesisBlock.Block.StateRoot = stateRoot[:]
		util.SaveBlock(t, ctx, db, genesisBlock)
		parentRoot, err := genesisBlock.Block.HashTreeRoot()
		require.NoError(t, err)

		require.NoError(t, db.SaveState(ctx, bs, parentRoot), "Could not save genesis state")
		require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

		mockChainService := &mockChain.ChainService{State: bs, Root: parentRoot[:], ForkChoiceStore: doublylinkedtree.New()}
		v1Alpha1Server := &v1alpha1validator.Server{
			ExecutionEngineCaller: &mockExecution.EngineClient{
				ExecutionBlock: &enginev1.ExecutionBlock{
					TotalDifficulty: "0x1",
				},
			},
			TimeFetcher:            mockChainService,
			HeadFetcher:            mockChainService,
			OptimisticModeFetcher:  mockChainService,
			SyncChecker:            &mockSync.Sync{IsSyncing: false},
			BlockReceiver:          mockChainService,
			ForkFetcher:            mockChainService,
			ForkchoiceFetcher:      mockChainService,
			ChainStartFetcher:      &mockExecution.Chain{},
			Eth1InfoFetcher:        &mockExecution.Chain{},
			Eth1BlockFetcher:       &mockExecution.Chain{},
			MockEth1Votes:          true,
			AttPool:                attestations.NewPool(),
			SlashingsPool:          slashings.NewPool(),
			ExitPool:               voluntaryexits.NewPool(),
			StateGen:               stategen.New(db, doublylinkedtree.New()),
			SyncCommitteePool:      synccommittee.NewStore(),
			ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
			BlockBuilder:           &builderTest.MockBuilderService{HasConfigured: true},
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, 1)
		proposerSlashing, err := util.GenerateProposerSlashingForValidator(
			bs,
			privKeys[0],
			0,
		)
		require.NoError(t, err)
		proposerSlashings[0] = proposerSlashing
		err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), bs, proposerSlashing)
		require.NoError(t, err)

		attSlashings := make([]*ethpbalpha.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
		attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
			bs,
			privKeys[1],
			1,
		)
		require.NoError(t, err)
		attSlashings[0] = attesterSlashing
		err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), bs, attesterSlashing)
		require.NoError(t, err)

		aggregationBits := bitfield.NewBitvector128()
		for i := range aggregationBits {
			aggregationBits[i] = 0xAA
		}

		syncCommitteeIndices, err := altair.NextSyncCommitteeIndices(context.Background(), bs)
		require.NoError(t, err)
		sigs := make([]bls.Signature, 0, len(syncCommitteeIndices))
		for i, indice := range syncCommitteeIndices {
			if aggregationBits.BitAt(uint64(i)) {
				b := p2pType.SSZBytes(parentRoot[:])
				sb, err := signing.ComputeDomainAndSign(bs, coreTime.CurrentEpoch(bs), &b, params.BeaconConfig().DomainSyncCommittee, privKeys[indice])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				sigs = append(sigs, sig)
			}
		}
		aggregatedSig := bls.AggregateSignatures(sigs).Marshal()
		contribution := &ethpbalpha.SyncCommitteeContribution{
			Slot:              params.BeaconConfig().SlotsPerEpoch,
			BlockRoot:         parentRoot[:],
			SubcommitteeIndex: 0,
			AggregationBits:   aggregationBits,
			Signature:         aggregatedSig,
		}
		require.NoError(t, v1Alpha1Server.SyncCommitteePool.SaveSyncCommitteeContribution(contribution))

		v1Server := &Server{
			V1Alpha1Server: v1Alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}
		randaoReveal, err := util.RandaoReveal(bs, 1, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))

		req := &ethpbv1.ProduceBlockRequest{
			Slot:         params.BeaconConfig().SlotsPerEpoch + 1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		v1Server.V1Alpha1Server.BeaconDB = db
		resp, err := v1Server.ProduceBlindedBlockSSZ(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)

		expectedBlock := &ethpbv2.BlindedBeaconBlockBellatrix{
			Slot:          33,
			ProposerIndex: 348,
			ParentRoot:    []byte{228, 15, 208, 120, 31, 194, 202, 144, 41, 107, 98, 126, 162, 234, 190, 94, 174, 176, 69, 177, 103, 82, 69, 254, 0, 230, 192, 67, 158, 29, 141, 85},
			StateRoot:     []byte{143, 107, 161, 135, 58, 60, 195, 107, 55, 142, 122, 111, 184, 1, 19, 233, 145, 204, 160, 226, 148, 67, 194, 102, 79, 196, 74, 242, 174, 108, 68, 82},
			Body: &ethpbv2.BlindedBeaconBlockBodyBellatrix{
				RandaoReveal: randaoReveal,
				Eth1Data: &ethpbv1.Eth1Data{
					DepositRoot:  []byte{40, 2, 99, 184, 81, 91, 153, 196, 115, 217, 104, 93, 31, 202, 27, 153, 42, 224, 148, 156, 116, 43, 161, 28, 155, 166, 37, 217, 205, 152, 69, 6},
					DepositCount: params.BeaconConfig().SyncCommitteeSize,
					BlockHash:    []byte{226, 231, 104, 45, 7, 68, 48, 54, 228, 109, 84, 245, 125, 45, 227, 127, 135, 155, 63, 38, 241, 251, 129, 192, 248, 49, 9, 120, 146, 18, 34, 228},
				},
				Graffiti: graffiti[:],
				ProposerSlashings: []*ethpbv1.ProposerSlashing{
					{
						SignedHeader_1: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_1.Header.Slot,
								ProposerIndex: proposerSlashing.Header_1.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_1.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_1.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_1.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_1.Signature,
						},
						SignedHeader_2: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_2.Header.Slot,
								ProposerIndex: proposerSlashing.Header_2.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_2.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_2.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_2.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_2.Signature,
						},
					},
				},
				AttesterSlashings: []*ethpbv1.AttesterSlashing{
					{
						Attestation_1: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_1.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_1.Data.Slot,
								Index:           attesterSlashing.Attestation_1.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_1.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_1.Signature,
						},
						Attestation_2: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_2.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_2.Data.Slot,
								Index:           attesterSlashing.Attestation_2.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_2.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_2.Signature,
						},
					},
				},
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits:      []byte{170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					SyncCommitteeSignature: []byte{153, 51, 238, 112, 158, 23, 41, 26, 18, 53, 3, 111, 57, 180, 45, 131, 90, 249, 28, 23, 153, 188, 171, 204, 45, 180, 133, 236, 47, 203, 119, 132, 162, 17, 61, 60, 122, 161, 45, 136, 130, 174, 120, 60, 64, 144, 6, 34, 24, 87, 41, 77, 16, 223, 36, 125, 80, 185, 178, 234, 74, 184, 196, 45, 242, 47, 124, 178, 83, 65, 106, 26, 179, 178, 27, 4, 72, 79, 191, 128, 114, 51, 246, 147, 3, 55, 210, 64, 148, 78, 144, 45, 97, 182, 157, 206},
				},
				ExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader{
					ParentHash:       make([]byte, 32),
					FeeRecipient:     make([]byte, 20),
					StateRoot:        make([]byte, 32),
					ReceiptsRoot:     make([]byte, 32),
					LogsBloom:        make([]byte, 256),
					PrevRandao:       make([]byte, 32),
					ExtraData:        nil,
					BaseFeePerGas:    make([]byte, 32),
					BlockHash:        make([]byte, 32),
					TransactionsRoot: []byte{127, 254, 36, 30, 166, 1, 135, 253, 176, 24, 123, 250, 34, 222, 53, 209, 249, 190, 215, 171, 6, 29, 148, 1, 253, 71, 227, 74, 84, 251, 237, 225},
				},
			},
		}
		expectedData, err := expectedBlock.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})

	t.Run("Capella", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		params.SetupTestConfigCleanup(t)
		bc := params.BeaconConfig()
		bc.AltairForkEpoch = primitives.Epoch(0)
		bc.BellatrixForkEpoch = primitives.Epoch(1)
		bc.CapellaForkEpoch = primitives.Epoch(2)
		params.OverrideBeaconConfig(bc)

		bs, privKeys := util.DeterministicGenesisStateCapella(t, params.BeaconConfig().SyncCommitteeSize)
		require.NoError(t, bs.SetSlot(params.BeaconConfig().SlotsPerEpoch*2))
		syncCommittee, err := altair.NextSyncCommittee(context.Background(), bs)
		require.NoError(t, err)
		require.NoError(t, bs.SetCurrentSyncCommittee(syncCommittee))
		require.NoError(t, bs.SetNextSyncCommittee(syncCommittee))

		stateRoot, err := bs.HashTreeRoot(ctx)
		require.NoError(t, err, "Could not hash genesis state")
		genesisBlock := util.NewBeaconBlockCapella()
		genesisBlock.Block.StateRoot = stateRoot[:]
		util.SaveBlock(t, ctx, db, genesisBlock)
		parentRoot, err := genesisBlock.Block.HashTreeRoot()
		require.NoError(t, err)

		require.NoError(t, db.SaveState(ctx, bs, parentRoot), "Could not save genesis state")
		require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

		prevRandao, err := helpers.RandaoMix(bs, 2)
		require.NoError(t, err)

		mockChainService := &mockChain.ChainService{State: bs, Root: parentRoot[:], ForkChoiceStore: doublylinkedtree.New()}
		v1Alpha1Server := &v1alpha1validator.Server{
			ExecutionEngineCaller: &mockExecution.EngineClient{
				ExecutionBlock: &enginev1.ExecutionBlock{
					TotalDifficulty: "0x1",
				},
				PayloadIDBytes: &enginev1.PayloadIDBytes{},
				ExecutionPayloadCapella: &enginev1.ExecutionPayloadCapella{
					ParentHash:   make([]byte, 32),
					FeeRecipient: make([]byte, 20),
					StateRoot:    make([]byte, 32),
					ReceiptsRoot: make([]byte, 32),
					LogsBloom:    make([]byte, 256),
					PrevRandao:   prevRandao,
					// State time at slot 65
					Timestamp:     780,
					ExtraData:     make([]byte, 32),
					BaseFeePerGas: make([]byte, 32),
					BlockHash:     make([]byte, 32),
				},
			},
			TimeFetcher:            &mockChain.ChainService{},
			HeadFetcher:            mockChainService,
			OptimisticModeFetcher:  &mockChain.ChainService{},
			SyncChecker:            &mockSync.Sync{IsSyncing: false},
			BlockReceiver:          mockChainService,
			ForkFetcher:            mockChainService,
			ForkchoiceFetcher:      mockChainService,
			FinalizationFetcher:    mockChainService,
			ChainStartFetcher:      &mockExecution.Chain{},
			Eth1InfoFetcher:        &mockExecution.Chain{},
			Eth1BlockFetcher:       &mockExecution.Chain{},
			MockEth1Votes:          true,
			AttPool:                attestations.NewPool(),
			SlashingsPool:          slashings.NewPool(),
			ExitPool:               voluntaryexits.NewPool(),
			BLSChangesPool:         blstoexec.NewPool(),
			StateGen:               stategen.New(db, doublylinkedtree.New()),
			SyncCommitteePool:      synccommittee.NewStore(),
			ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
			BlockBuilder: &builderTest.MockBuilderService{
				HasConfigured: true,
			},
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, 1)
		proposerSlashing, err := util.GenerateProposerSlashingForValidator(
			bs,
			privKeys[0],
			0,
		)
		require.NoError(t, err)
		proposerSlashings[0] = proposerSlashing
		err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), bs, proposerSlashing)
		require.NoError(t, err)

		attSlashings := make([]*ethpbalpha.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
		attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
			bs,
			privKeys[1],
			1,
		)
		require.NoError(t, err)
		attSlashings[0] = attesterSlashing
		err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), bs, attesterSlashing)
		require.NoError(t, err)

		aggregationBits := bitfield.NewBitvector128()
		for i := range aggregationBits {
			aggregationBits[i] = 0xAA
		}

		syncCommitteeIndices, err := altair.NextSyncCommitteeIndices(context.Background(), bs)
		require.NoError(t, err)
		sigs := make([]bls.Signature, 0, len(syncCommitteeIndices))
		for i, indice := range syncCommitteeIndices {
			if aggregationBits.BitAt(uint64(i)) {
				b := p2pType.SSZBytes(parentRoot[:])
				sb, err := signing.ComputeDomainAndSign(bs, coreTime.CurrentEpoch(bs), &b, params.BeaconConfig().DomainSyncCommittee, privKeys[indice])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				sigs = append(sigs, sig)
			}
		}
		aggregatedSig := bls.AggregateSignatures(sigs).Marshal()
		contribution := &ethpbalpha.SyncCommitteeContribution{
			Slot:              params.BeaconConfig().SlotsPerEpoch * 2,
			BlockRoot:         parentRoot[:],
			SubcommitteeIndex: 0,
			AggregationBits:   aggregationBits,
			Signature:         aggregatedSig,
		}
		require.NoError(t, v1Alpha1Server.SyncCommitteePool.SaveSyncCommitteeContribution(contribution))

		v1Server := &Server{
			V1Alpha1Server: v1Alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}
		randaoReveal, err := util.RandaoReveal(bs, 2, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))

		req := &ethpbv1.ProduceBlockRequest{
			Slot:         params.BeaconConfig().SlotsPerEpoch*2 + 1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		v1Server.V1Alpha1Server.BeaconDB = db
		require.NoError(t, v1Alpha1Server.BeaconDB.SaveRegistrationsByValidatorIDs(ctx, []primitives.ValidatorIndex{348},
			[]*ethpbalpha.ValidatorRegistrationV1{{FeeRecipient: bytesutil.PadTo([]byte{}, fieldparams.FeeRecipientLength), Pubkey: bytesutil.PadTo([]byte{}, fieldparams.BLSPubkeyLength)}}))

		resp, err := v1Server.ProduceBlindedBlockSSZ(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, ethpbv2.Version_CAPELLA, resp.Version)

		expectedBlock := &ethpbv2.BlindedBeaconBlockCapella{
			Slot:          65,
			ProposerIndex: 372,
			ParentRoot:    []byte{98, 110, 62, 255, 200, 27, 255, 78, 6, 11, 199, 119, 47, 57, 40, 250, 3, 255, 123, 149, 158, 235, 206, 73, 197, 53, 194, 112, 193, 78, 30, 209},
			StateRoot:     []byte{15, 201, 220, 7, 167, 148, 124, 27, 100, 119, 219, 2, 159, 243, 76, 2, 101, 209, 32, 213, 21, 158, 75, 203, 184, 93, 189, 243, 179, 78, 150, 171},
			Body: &ethpbv2.BlindedBeaconBlockBodyCapella{
				RandaoReveal: randaoReveal,
				Eth1Data: &ethpbv1.Eth1Data{
					DepositRoot:  []byte{209, 112, 83, 246, 47, 50, 100, 179, 169, 154, 93, 33, 128, 212, 195, 171, 204, 14, 180, 156, 134, 217, 151, 150, 60, 190, 195, 214, 0, 7, 248, 51},
					DepositCount: params.BeaconConfig().SyncCommitteeSize,
					BlockHash:    []byte{81, 55, 109, 120, 46, 24, 197, 204, 4, 239, 233, 63, 224, 130, 143, 78, 98, 6, 148, 78, 34, 173, 90, 38, 115, 7, 146, 56, 102, 19, 237, 43},
				},
				Graffiti: graffiti[:],
				ProposerSlashings: []*ethpbv1.ProposerSlashing{
					{
						SignedHeader_1: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_1.Header.Slot,
								ProposerIndex: proposerSlashing.Header_1.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_1.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_1.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_1.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_1.Signature,
						},
						SignedHeader_2: &ethpbv1.SignedBeaconBlockHeader{
							Message: &ethpbv1.BeaconBlockHeader{
								Slot:          proposerSlashing.Header_2.Header.Slot,
								ProposerIndex: proposerSlashing.Header_2.Header.ProposerIndex,
								ParentRoot:    proposerSlashing.Header_2.Header.ParentRoot,
								StateRoot:     proposerSlashing.Header_2.Header.StateRoot,
								BodyRoot:      proposerSlashing.Header_2.Header.BodyRoot,
							},
							Signature: proposerSlashing.Header_2.Signature,
						},
					},
				},
				AttesterSlashings: []*ethpbv1.AttesterSlashing{
					{
						Attestation_1: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_1.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_1.Data.Slot,
								Index:           attesterSlashing.Attestation_1.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_1.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_1.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_1.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_1.Signature,
						},
						Attestation_2: &ethpbv1.IndexedAttestation{
							AttestingIndices: attesterSlashing.Attestation_2.AttestingIndices,
							Data: &ethpbv1.AttestationData{
								Slot:            attesterSlashing.Attestation_2.Data.Slot,
								Index:           attesterSlashing.Attestation_2.Data.CommitteeIndex,
								BeaconBlockRoot: attesterSlashing.Attestation_2.Data.BeaconBlockRoot,
								Source: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Source.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Source.Root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: attesterSlashing.Attestation_2.Data.Target.Epoch,
									Root:  attesterSlashing.Attestation_2.Data.Target.Root,
								},
							},
							Signature: attesterSlashing.Attestation_2.Signature,
						},
					},
				},
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits:      []byte{170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					SyncCommitteeSignature: []byte{166, 223, 239, 55, 64, 151, 124, 78, 34, 159, 22, 51, 216, 224, 97, 5, 3, 97, 106, 183, 24, 71, 180, 209, 40, 202, 128, 169, 176, 187, 134, 85, 104, 150, 180, 125, 117, 192, 82, 189, 14, 15, 245, 196, 141, 54, 235, 23, 4, 41, 66, 171, 1, 35, 136, 193, 40, 242, 154, 63, 19, 13, 215, 141, 195, 110, 10, 201, 153, 85, 50, 0, 152, 237, 80, 68, 104, 181, 63, 237, 114, 83, 206, 252, 210, 155, 50, 136, 125, 141, 173, 158, 34, 165, 36, 95},
				},
				ExecutionPayloadHeader: &enginev1.ExecutionPayloadHeaderCapella{
					ParentHash:   make([]byte, 32),
					FeeRecipient: make([]byte, 20),
					StateRoot:    make([]byte, 32),
					ReceiptsRoot: make([]byte, 32),
					LogsBloom:    make([]byte, 256),
					PrevRandao:   prevRandao,
					// State time at slot 65
					Timestamp:        780,
					ExtraData:        make([]byte, 32),
					BaseFeePerGas:    make([]byte, 32),
					BlockHash:        make([]byte, 32),
					TransactionsRoot: []byte{127, 254, 36, 30, 166, 1, 135, 253, 176, 24, 123, 250, 34, 222, 53, 209, 249, 190, 215, 171, 6, 29, 148, 1, 253, 71, 227, 74, 84, 251, 237, 225},
					WithdrawalsRoot:  []byte{121, 41, 48, 187, 213, 186, 172, 67, 188, 199, 152, 238, 73, 170, 129, 133, 239, 118, 187, 59, 68, 186, 98, 185, 29, 134, 174, 86, 158, 75, 181, 53},
				},
			},
		}
		expectedData, err := expectedBlock.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
}

func TestProduceBlindedBlock_SyncNotReady(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	chainService := &mockChain.ChainService{State: st}
	vs := &Server{
		SyncChecker:           &mockSync.Sync{IsSyncing: true},
		HeadFetcher:           chainService,
		TimeFetcher:           chainService,
		OptimisticModeFetcher: chainService,
	}
	_, err = vs.ProduceBlindedBlock(context.Background(), &ethpbv1.ProduceBlockRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestProduceBlindedBlockSSZ_SyncNotReady(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	chainService := &mockChain.ChainService{State: st}
	vs := &Server{
		SyncChecker:           &mockSync.Sync{IsSyncing: true},
		HeadFetcher:           chainService,
		TimeFetcher:           chainService,
		OptimisticModeFetcher: chainService,
	}
	_, err = vs.ProduceBlindedBlockSSZ(context.Background(), &ethpbv1.ProduceBlockRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestProduceAttestationData(t *testing.T) {
	block := util.NewBeaconBlock()
	block.Block.Slot = 3*params.BeaconConfig().SlotsPerEpoch + 1
	targetBlock := util.NewBeaconBlock()
	targetBlock.Block.Slot = 1 * params.BeaconConfig().SlotsPerEpoch
	justifiedBlock := util.NewBeaconBlock()
	justifiedBlock.Block.Slot = 2 * params.BeaconConfig().SlotsPerEpoch
	blockRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not hash beacon block")
	justifiedRoot, err := justifiedBlock.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root for justified block")
	targetRoot, err := targetBlock.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root for target block")
	slot := 3*params.BeaconConfig().SlotsPerEpoch + 1
	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(slot))
	err = beaconState.SetCurrentJustifiedCheckpoint(&ethpbalpha.Checkpoint{
		Epoch: 2,
		Root:  justifiedRoot[:],
	})
	require.NoError(t, err)

	blockRoots := beaconState.BlockRoots()
	blockRoots[1] = blockRoot[:]
	blockRoots[1*params.BeaconConfig().SlotsPerEpoch] = targetRoot[:]
	blockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedRoot[:]
	require.NoError(t, beaconState.SetBlockRoots(blockRoots))
	chainService := &mockChain.ChainService{
		Genesis: time.Now(),
	}
	offset := int64(slot.Mul(params.BeaconConfig().SecondsPerSlot))
	v1Alpha1Server := &v1alpha1validator.Server{
		P2P:              &p2pmock.MockBroadcaster{},
		SyncChecker:      &mockSync.Sync{IsSyncing: false},
		AttestationCache: cache.NewAttestationCache(),
		HeadFetcher: &mockChain.ChainService{
			State: beaconState, Root: blockRoot[:],
		},
		FinalizationFetcher: &mockChain.ChainService{
			CurrentJustifiedCheckPoint: beaconState.CurrentJustifiedCheckpoint(),
		},
		TimeFetcher: &mockChain.ChainService{
			Genesis: time.Now().Add(time.Duration(-1*offset) * time.Second),
		},
		StateNotifier: chainService.StateNotifier(),
	}
	v1Server := &Server{
		V1Alpha1Server: v1Alpha1Server,
	}

	req := &ethpbv1.ProduceAttestationDataRequest{
		CommitteeIndex: 0,
		Slot:           3*params.BeaconConfig().SlotsPerEpoch + 1,
	}
	res, err := v1Server.ProduceAttestationData(context.Background(), req)
	require.NoError(t, err, "Could not get attestation info at slot")

	expectedInfo := &ethpbv1.AttestationData{
		Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
		BeaconBlockRoot: blockRoot[:],
		Source: &ethpbv1.Checkpoint{
			Epoch: 2,
			Root:  justifiedRoot[:],
		},
		Target: &ethpbv1.Checkpoint{
			Epoch: 3,
			Root:  blockRoot[:],
		},
	}

	if !proto.Equal(res.Data, expectedInfo) {
		t.Errorf("Expected attestation info to match, received %v, wanted %v", res, expectedInfo)
	}
}

func TestGetAggregateAttestation(t *testing.T) {
	ctx := context.Background()
	root1 := bytesutil.PadTo([]byte("root1"), 32)
	sig1 := bytesutil.PadTo([]byte("sig1"), fieldparams.BLSSignatureLength)
	attSlot1 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            1,
			CommitteeIndex:  1,
			BeaconBlockRoot: root1,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root1,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root1,
			},
		},
		Signature: sig1,
	}
	root21 := bytesutil.PadTo([]byte("root2_1"), 32)
	sig21 := bytesutil.PadTo([]byte("sig2_1"), fieldparams.BLSSignatureLength)
	attslot21 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 1, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            2,
			CommitteeIndex:  2,
			BeaconBlockRoot: root21,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root21,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root21,
			},
		},
		Signature: sig21,
	}
	root22 := bytesutil.PadTo([]byte("root2_2"), 32)
	sig22 := bytesutil.PadTo([]byte("sig2_2"), fieldparams.BLSSignatureLength)
	attslot22 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 1, 1, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            2,
			CommitteeIndex:  3,
			BeaconBlockRoot: root22,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root22,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root22,
			},
		},
		Signature: sig22,
	}
	root33 := bytesutil.PadTo([]byte("root3_3"), 32)
	sig33 := bls.NewAggregateSignature().Marshal()

	attslot33 := &ethpbalpha.Attestation{
		AggregationBits: []byte{1, 0, 0, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            2,
			CommitteeIndex:  3,
			BeaconBlockRoot: root33,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root33,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root33,
			},
		},
		Signature: sig33,
	}
	pool := attestations.NewPool()
	err := pool.SaveAggregatedAttestations([]*ethpbalpha.Attestation{attSlot1, attslot21, attslot22})
	assert.NoError(t, err)
	vs := &Server{
		AttestationsPool: pool,
	}

	t.Run("OK", func(t *testing.T) {
		reqRoot, err := attslot22.Data.HashTreeRoot()
		require.NoError(t, err)
		req := &ethpbv1.AggregateAttestationRequest{
			AttestationDataRoot: reqRoot[:],
			Slot:                2,
		}
		att, err := vs.GetAggregateAttestation(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, att)
		require.NotNil(t, att.Data)
		assert.DeepEqual(t, bitfield.Bitlist{0, 1, 1, 1}, att.Data.AggregationBits)
		assert.DeepEqual(t, sig22, att.Data.Signature)
		assert.Equal(t, primitives.Slot(2), att.Data.Data.Slot)
		assert.Equal(t, primitives.CommitteeIndex(3), att.Data.Data.Index)
		assert.DeepEqual(t, root22, att.Data.Data.BeaconBlockRoot)
		require.NotNil(t, att.Data.Data.Source)
		assert.Equal(t, primitives.Epoch(1), att.Data.Data.Source.Epoch)
		assert.DeepEqual(t, root22, att.Data.Data.Source.Root)
		require.NotNil(t, att.Data.Data.Target)
		assert.Equal(t, primitives.Epoch(1), att.Data.Data.Target.Epoch)
		assert.DeepEqual(t, root22, att.Data.Data.Target.Root)
	})

	t.Run("Aggregate Beforehand", func(t *testing.T) {
		reqRoot, err := attslot33.Data.HashTreeRoot()
		require.NoError(t, err)
		err = vs.AttestationsPool.SaveUnaggregatedAttestation(attslot33)
		require.NoError(t, err)
		newAtt := ethpbalpha.CopyAttestation(attslot33)
		newAtt.AggregationBits = []byte{0, 1, 0, 1}
		err = vs.AttestationsPool.SaveUnaggregatedAttestation(newAtt)
		require.NoError(t, err)
		req := &ethpbv1.AggregateAttestationRequest{
			AttestationDataRoot: reqRoot[:],
			Slot:                2,
		}
		aggAtt, err := vs.GetAggregateAttestation(ctx, req)
		require.NoError(t, err)
		assert.DeepEqual(t, bitfield.Bitlist{1, 1, 0, 1}, aggAtt.Data.AggregationBits)
	})
	t.Run("No matching attestation", func(t *testing.T) {
		req := &ethpbv1.AggregateAttestationRequest{
			AttestationDataRoot: bytesutil.PadTo([]byte("foo"), 32),
			Slot:                2,
		}
		_, err := vs.GetAggregateAttestation(ctx, req)
		assert.ErrorContains(t, "No matching attestation found", err)
	})
}

func TestGetAggregateAttestation_SameSlotAndRoot_ReturnMostAggregationBits(t *testing.T) {
	ctx := context.Background()
	root := bytesutil.PadTo([]byte("root"), 32)
	sig := bytesutil.PadTo([]byte("sig"), fieldparams.BLSSignatureLength)
	att1 := &ethpbalpha.Attestation{
		AggregationBits: []byte{3, 0, 0, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            1,
			CommitteeIndex:  1,
			BeaconBlockRoot: root,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
		},
		Signature: sig,
	}
	att2 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 3, 0, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            1,
			CommitteeIndex:  1,
			BeaconBlockRoot: root,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
		},
		Signature: sig,
	}
	pool := attestations.NewPool()
	err := pool.SaveAggregatedAttestations([]*ethpbalpha.Attestation{att1, att2})
	assert.NoError(t, err)
	vs := &Server{
		AttestationsPool: pool,
	}
	reqRoot, err := att1.Data.HashTreeRoot()
	require.NoError(t, err)
	req := &ethpbv1.AggregateAttestationRequest{
		AttestationDataRoot: reqRoot[:],
		Slot:                1,
	}
	att, err := vs.GetAggregateAttestation(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, att)
	require.NotNil(t, att.Data)
	assert.DeepEqual(t, bitfield.Bitlist{3, 0, 0, 1}, att.Data.AggregationBits)
}

func TestSubmitBeaconCommitteeSubscription(t *testing.T) {
	ctx := context.Background()
	genesis := util.NewBeaconBlock()
	depChainStart := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, err := util.DeterministicDepositsAndKeys(depChainStart)
	require.NoError(t, err)
	eth1Data, err := util.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bs, err := transition.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
	require.NoError(t, err, "Could not set up genesis state")
	// Set state to non-epoch start slot.
	require.NoError(t, bs.SetSlot(5))
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	roots := make([][]byte, fieldparams.BlockRootsLength)
	roots[0] = genesisRoot[:]
	require.NoError(t, bs.SetBlockRoots(roots))

	pubKeys := make([][]byte, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = deposits[i].Data.PublicKey
	}

	chainSlot := primitives.Slot(0)
	chain := &mockChain.ChainService{
		State: bs, Root: genesisRoot[:], Slot: &chainSlot,
	}
	vs := &Server{
		HeadFetcher:    chain,
		TimeFetcher:    chain,
		SyncChecker:    &mockSync.Sync{IsSyncing: false},
		V1Alpha1Server: &v1alpha1validator.Server{},
	}

	t.Run("Single subscription", func(t *testing.T) {
		cache.SubnetIDs.EmptyAllCaches()
		req := &ethpbv1.SubmitBeaconCommitteeSubscriptionsRequest{
			Data: []*ethpbv1.BeaconCommitteeSubscribe{
				{
					ValidatorIndex: 1,
					CommitteeIndex: 1,
					Slot:           1,
					IsAggregator:   false,
				},
			},
		}
		_, err = vs.SubmitBeaconCommitteeSubscription(ctx, req)
		require.NoError(t, err)
		subnets := cache.SubnetIDs.GetAttesterSubnetIDs(1)
		require.Equal(t, 1, len(subnets))
		assert.Equal(t, uint64(4), subnets[0])
	})

	t.Run("Multiple subscriptions", func(t *testing.T) {
		cache.SubnetIDs.EmptyAllCaches()
		req := &ethpbv1.SubmitBeaconCommitteeSubscriptionsRequest{
			Data: []*ethpbv1.BeaconCommitteeSubscribe{
				{
					ValidatorIndex: 1,
					CommitteeIndex: 1,
					Slot:           1,
					IsAggregator:   false,
				},
				{
					ValidatorIndex: 1000,
					CommitteeIndex: 16,
					Slot:           1,
					IsAggregator:   false,
				},
			},
		}
		_, err = vs.SubmitBeaconCommitteeSubscription(ctx, req)
		require.NoError(t, err)
		subnets := cache.SubnetIDs.GetAttesterSubnetIDs(1)
		require.Equal(t, 2, len(subnets))
	})

	t.Run("Is aggregator", func(t *testing.T) {
		cache.SubnetIDs.EmptyAllCaches()
		req := &ethpbv1.SubmitBeaconCommitteeSubscriptionsRequest{
			Data: []*ethpbv1.BeaconCommitteeSubscribe{
				{
					ValidatorIndex: 1,
					CommitteeIndex: 1,
					Slot:           1,
					IsAggregator:   true,
				},
			},
		}
		_, err = vs.SubmitBeaconCommitteeSubscription(ctx, req)
		require.NoError(t, err)
		ids := cache.SubnetIDs.GetAggregatorSubnetIDs(primitives.Slot(1))
		assert.Equal(t, 1, len(ids))
	})

	t.Run("Validators assigned to subnet", func(t *testing.T) {
		cache.SubnetIDs.EmptyAllCaches()
		req := &ethpbv1.SubmitBeaconCommitteeSubscriptionsRequest{
			Data: []*ethpbv1.BeaconCommitteeSubscribe{
				{
					ValidatorIndex: 1,
					CommitteeIndex: 1,
					Slot:           1,
					IsAggregator:   true,
				},
				{
					ValidatorIndex: 2,
					CommitteeIndex: 1,
					Slot:           1,
					IsAggregator:   false,
				},
			},
		}
		_, err = vs.SubmitBeaconCommitteeSubscription(ctx, req)
		require.NoError(t, err)
		ids, ok, _ := cache.SubnetIDs.GetPersistentSubnets(pubKeys[1])
		require.Equal(t, true, ok, "subnet for validator 1 not found")
		assert.Equal(t, 1, len(ids))
		ids, ok, _ = cache.SubnetIDs.GetPersistentSubnets(pubKeys[2])
		require.Equal(t, true, ok, "subnet for validator 2 not found")
		assert.Equal(t, 1, len(ids))
	})

	t.Run("No subscriptions", func(t *testing.T) {
		req := &ethpbv1.SubmitBeaconCommitteeSubscriptionsRequest{
			Data: make([]*ethpbv1.BeaconCommitteeSubscribe, 0),
		}
		_, err = vs.SubmitBeaconCommitteeSubscription(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "No subscriptions provided", err)
	})
}

func TestSubmitBeaconCommitteeSubscription_SyncNotReady(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	chainService := &mockChain.ChainService{State: st}
	vs := &Server{
		SyncChecker:           &mockSync.Sync{IsSyncing: true},
		HeadFetcher:           chainService,
		TimeFetcher:           chainService,
		OptimisticModeFetcher: chainService,
	}
	_, err = vs.SubmitBeaconCommitteeSubscription(context.Background(), &ethpbv1.SubmitBeaconCommitteeSubscriptionsRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestSubmitSyncCommitteeSubscription(t *testing.T) {
	ctx := context.Background()
	genesis := util.NewBeaconBlock()
	deposits, _, err := util.DeterministicDepositsAndKeys(64)
	require.NoError(t, err)
	eth1Data, err := util.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bs, err := util.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
	require.NoError(t, err, "Could not set up genesis state")
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	roots := make([][]byte, fieldparams.BlockRootsLength)
	roots[0] = genesisRoot[:]
	require.NoError(t, bs.SetBlockRoots(roots))

	pubkeys := make([][]byte, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubkeys[i] = deposits[i].Data.PublicKey
	}

	chainSlot := primitives.Slot(0)
	chain := &mockChain.ChainService{
		State: bs, Root: genesisRoot[:], Slot: &chainSlot,
	}
	vs := &Server{
		HeadFetcher:    chain,
		TimeFetcher:    chain,
		SyncChecker:    &mockSync.Sync{IsSyncing: false},
		V1Alpha1Server: &v1alpha1validator.Server{},
	}

	t.Run("Single subscription", func(t *testing.T) {
		cache.SyncSubnetIDs.EmptyAllCaches()
		req := &ethpbv2.SubmitSyncCommitteeSubscriptionsRequest{
			Data: []*ethpbv2.SyncCommitteeSubscription{
				{
					ValidatorIndex:       0,
					SyncCommitteeIndices: []uint64{0, 2},
					UntilEpoch:           1,
				},
			},
		}
		_, err = vs.SubmitSyncCommitteeSubscription(ctx, req)
		require.NoError(t, err)
		subnets, _, _, _ := cache.SyncSubnetIDs.GetSyncCommitteeSubnets(pubkeys[0], 0)
		require.Equal(t, 2, len(subnets))
		assert.Equal(t, uint64(0), subnets[0])
		assert.Equal(t, uint64(2), subnets[1])
	})

	t.Run("Multiple subscriptions", func(t *testing.T) {
		cache.SyncSubnetIDs.EmptyAllCaches()
		req := &ethpbv2.SubmitSyncCommitteeSubscriptionsRequest{
			Data: []*ethpbv2.SyncCommitteeSubscription{
				{
					ValidatorIndex:       0,
					SyncCommitteeIndices: []uint64{0},
					UntilEpoch:           1,
				},
				{
					ValidatorIndex:       1,
					SyncCommitteeIndices: []uint64{2},
					UntilEpoch:           1,
				},
			},
		}
		_, err = vs.SubmitSyncCommitteeSubscription(ctx, req)
		require.NoError(t, err)
		subnets, _, _, _ := cache.SyncSubnetIDs.GetSyncCommitteeSubnets(pubkeys[0], 0)
		require.Equal(t, 1, len(subnets))
		assert.Equal(t, uint64(0), subnets[0])
		subnets, _, _, _ = cache.SyncSubnetIDs.GetSyncCommitteeSubnets(pubkeys[1], 0)
		require.Equal(t, 1, len(subnets))
		assert.Equal(t, uint64(2), subnets[0])
	})

	t.Run("No subscriptions", func(t *testing.T) {
		req := &ethpbv2.SubmitSyncCommitteeSubscriptionsRequest{
			Data: make([]*ethpbv2.SyncCommitteeSubscription, 0),
		}
		_, err = vs.SubmitSyncCommitteeSubscription(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "No subscriptions provided", err)
	})

	t.Run("Invalid validator index", func(t *testing.T) {
		req := &ethpbv2.SubmitSyncCommitteeSubscriptionsRequest{
			Data: []*ethpbv2.SyncCommitteeSubscription{
				{
					ValidatorIndex:       99,
					SyncCommitteeIndices: []uint64{},
					UntilEpoch:           1,
				},
			},
		}
		_, err = vs.SubmitSyncCommitteeSubscription(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Could not get validator at index 99", err)
	})

	t.Run("Epoch in the past", func(t *testing.T) {
		req := &ethpbv2.SubmitSyncCommitteeSubscriptionsRequest{
			Data: []*ethpbv2.SyncCommitteeSubscription{
				{
					ValidatorIndex:       0,
					SyncCommitteeIndices: []uint64{},
					UntilEpoch:           0,
				},
			},
		}
		_, err = vs.SubmitSyncCommitteeSubscription(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Epoch for subscription at index 0 is in the past", err)
	})

	t.Run("First epoch after the next sync committee is valid", func(t *testing.T) {
		req := &ethpbv2.SubmitSyncCommitteeSubscriptionsRequest{
			Data: []*ethpbv2.SyncCommitteeSubscription{
				{
					ValidatorIndex:       0,
					SyncCommitteeIndices: []uint64{},
					UntilEpoch:           2 * params.BeaconConfig().EpochsPerSyncCommitteePeriod,
				},
			},
		}
		_, err = vs.SubmitSyncCommitteeSubscription(ctx, req)
		require.NoError(t, err)
	})

	t.Run("Epoch too far in the future", func(t *testing.T) {
		req := &ethpbv2.SubmitSyncCommitteeSubscriptionsRequest{
			Data: []*ethpbv2.SyncCommitteeSubscription{
				{
					ValidatorIndex:       0,
					SyncCommitteeIndices: []uint64{},
					UntilEpoch:           2*params.BeaconConfig().EpochsPerSyncCommitteePeriod + 1,
				},
			},
		}
		_, err = vs.SubmitSyncCommitteeSubscription(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Epoch for subscription at index 0 is too far in the future", err)
	})
}

func TestSubmitSyncCommitteeSubscription_SyncNotReady(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	chainService := &mockChain.ChainService{State: st}
	vs := &Server{
		SyncChecker:           &mockSync.Sync{IsSyncing: true},
		HeadFetcher:           chainService,
		TimeFetcher:           chainService,
		OptimisticModeFetcher: chainService,
	}
	_, err = vs.SubmitSyncCommitteeSubscription(context.Background(), &ethpbv2.SubmitSyncCommitteeSubscriptionsRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestSubmitAggregateAndProofs(t *testing.T) {
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	c := params.BeaconNetworkConfig()
	c.MaximumGossipClockDisparity = time.Hour
	params.OverrideBeaconNetworkConfig(c)
	root := bytesutil.PadTo([]byte("root"), 32)
	sig := bytesutil.PadTo([]byte("sig"), fieldparams.BLSSignatureLength)
	proof := bytesutil.PadTo([]byte("proof"), fieldparams.BLSSignatureLength)
	att := &ethpbv1.Attestation{
		AggregationBits: []byte{0, 1},
		Data: &ethpbv1.AttestationData{
			Slot:            1,
			Index:           1,
			BeaconBlockRoot: root,
			Source: &ethpbv1.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
			Target: &ethpbv1.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
		},
		Signature: sig,
	}

	t.Run("OK", func(t *testing.T) {
		chainSlot := primitives.Slot(0)
		chain := &mockChain.ChainService{
			Genesis: time.Now(), Slot: &chainSlot,
		}
		broadcaster := &p2pmock.MockBroadcaster{}
		vs := Server{
			TimeFetcher: chain,
			Broadcaster: broadcaster,
		}

		req := &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate:       att,
						SelectionProof:  proof,
					},
					Signature: sig,
				},
			},
		}

		_, err := vs.SubmitAggregateAndProofs(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, true, broadcaster.BroadcastCalled)
	})

	t.Run("nil aggregate", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		vs := Server{
			Broadcaster: broadcaster,
		}

		req := &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				nil,
			},
		}
		_, err := vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Signed aggregate request can't be nil", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)

		req = &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message:   nil,
					Signature: sig,
				},
			},
		}
		_, err = vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Signed aggregate request can't be nil", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)

		req = &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate: &ethpbv1.Attestation{
							AggregationBits: []byte{0, 1},
							Data:            nil,
							Signature:       sig,
						},
						SelectionProof: proof,
					},
					Signature: sig,
				},
			},
		}
		_, err = vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Signed aggregate request can't be nil", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)
	})

	t.Run("zero signature", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		vs := Server{
			Broadcaster: broadcaster,
		}
		req := &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate:       att,
						SelectionProof:  proof,
					},
					Signature: make([]byte, 96),
				},
			},
		}
		_, err := vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Signed signatures can't be zero hashes", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)
	})

	t.Run("zero proof", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		vs := Server{
			Broadcaster: broadcaster,
		}
		req := &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate:       att,
						SelectionProof:  make([]byte, 96),
					},
					Signature: sig,
				},
			},
		}
		_, err := vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Signed signatures can't be zero hashes", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)
	})

	t.Run("zero message signature", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		vs := Server{
			Broadcaster: broadcaster,
		}
		req := &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate: &ethpbv1.Attestation{
							AggregationBits: []byte{0, 1},
							Data: &ethpbv1.AttestationData{
								Slot:            1,
								Index:           1,
								BeaconBlockRoot: root,
								Source: &ethpbv1.Checkpoint{
									Epoch: 1,
									Root:  root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: 1,
									Root:  root,
								},
							},
							Signature: make([]byte, 96),
						},
						SelectionProof: proof,
					},
					Signature: sig,
				},
			},
		}
		_, err := vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Signed signatures can't be zero hashes", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)
	})

	t.Run("wrong signature length", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		vs := Server{
			Broadcaster: broadcaster,
		}

		req := &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate:       att,
						SelectionProof:  proof,
					},
					Signature: make([]byte, 99),
				},
			},
		}
		_, err := vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Incorrect signature length. Expected "+strconv.Itoa(96)+" bytes", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)

		req = &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate: &ethpbv1.Attestation{
							AggregationBits: []byte{0, 1},
							Data: &ethpbv1.AttestationData{
								Slot:            1,
								Index:           1,
								BeaconBlockRoot: root,
								Source: &ethpbv1.Checkpoint{
									Epoch: 1,
									Root:  root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: 1,
									Root:  root,
								},
							},
							Signature: make([]byte, 99),
						},
						SelectionProof: proof,
					},
					Signature: sig,
				},
			},
		}
		_, err = vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Incorrect signature length. Expected "+strconv.Itoa(96)+" bytes", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)
	})

	t.Run("invalid attestation time", func(t *testing.T) {
		chainSlot := primitives.Slot(0)
		chain := &mockChain.ChainService{
			Genesis: time.Now().Add(time.Hour * 2), Slot: &chainSlot,
		}
		broadcaster := &p2pmock.MockBroadcaster{}
		vs := Server{
			TimeFetcher: chain,
			Broadcaster: broadcaster,
		}

		req := &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate:       att,
						SelectionProof:  proof,
					},
					Signature: sig,
				},
			},
		}

		_, err := vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Attestation slot is no longer valid from current time", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)
	})
}

func TestProduceSyncCommitteeContribution(t *testing.T) {
	ctx := context.Background()
	root := bytesutil.PadTo([]byte("root"), 32)
	sig := bls.NewAggregateSignature().Marshal()
	messsage := &ethpbalpha.SyncCommitteeMessage{
		Slot:           0,
		BlockRoot:      root,
		ValidatorIndex: 0,
		Signature:      sig,
	}
	syncCommitteePool := synccommittee.NewStore()
	require.NoError(t, syncCommitteePool.SaveSyncCommitteeMessage(messsage))
	v1Server := &v1alpha1validator.Server{
		SyncCommitteePool: syncCommitteePool,
		HeadFetcher: &mockChain.ChainService{
			SyncCommitteeIndices: []primitives.CommitteeIndex{0},
		},
	}
	server := Server{
		V1Alpha1Server:    v1Server,
		SyncCommitteePool: syncCommitteePool,
	}

	req := &ethpbv2.ProduceSyncCommitteeContributionRequest{
		Slot:              0,
		SubcommitteeIndex: 0,
		BeaconBlockRoot:   root,
	}
	resp, err := server.ProduceSyncCommitteeContribution(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, primitives.Slot(0), resp.Data.Slot)
	assert.Equal(t, uint64(0), resp.Data.SubcommitteeIndex)
	assert.DeepEqual(t, root, resp.Data.BeaconBlockRoot)
	aggregationBits := resp.Data.AggregationBits
	assert.Equal(t, true, aggregationBits.BitAt(0))
	assert.DeepEqual(t, sig, resp.Data.Signature)

	syncCommitteePool = synccommittee.NewStore()
	v1Server = &v1alpha1validator.Server{
		SyncCommitteePool: syncCommitteePool,
		HeadFetcher: &mockChain.ChainService{
			SyncCommitteeIndices: []primitives.CommitteeIndex{0},
		},
	}
	server = Server{
		V1Alpha1Server:    v1Server,
		SyncCommitteePool: syncCommitteePool,
	}
	req = &ethpbv2.ProduceSyncCommitteeContributionRequest{
		Slot:              0,
		SubcommitteeIndex: 0,
		BeaconBlockRoot:   root,
	}
	_, err = server.ProduceSyncCommitteeContribution(ctx, req)
	assert.ErrorContains(t, "No subcommittee messages found", err)
}

func TestSubmitContributionAndProofs(t *testing.T) {
	ctx := context.Background()
	sig := bls.NewAggregateSignature().Marshal()
	root := bytesutil.PadTo([]byte("root"), 32)
	proof := bytesutil.PadTo([]byte("proof"), 96)
	aggBits := bitfield.NewBitvector128()
	aggBits.SetBitAt(0, true)
	v1Server := &v1alpha1validator.Server{
		P2P:               &p2pmock.MockBroadcaster{},
		OperationNotifier: (&mockChain.ChainService{}).OperationNotifier(),
	}
	server := &Server{
		V1Alpha1Server: v1Server,
	}

	t.Run("Single contribution", func(t *testing.T) {
		v1Server.SyncCommitteePool = synccommittee.NewStore()
		req := &ethpbv2.SubmitContributionAndProofsRequest{
			Data: []*ethpbv2.SignedContributionAndProof{
				{
					Message: &ethpbv2.ContributionAndProof{
						AggregatorIndex: 0,
						Contribution: &ethpbv2.SyncCommitteeContribution{
							Slot:              0,
							BeaconBlockRoot:   root,
							SubcommitteeIndex: 0,
							AggregationBits:   aggBits,
							Signature:         sig,
						},
						SelectionProof: proof,
					},
					Signature: sig,
				},
			},
		}

		_, err := server.SubmitContributionAndProofs(ctx, req)
		require.NoError(t, err)
		savedMsgs, err := v1Server.SyncCommitteePool.SyncCommitteeContributions(0)
		require.NoError(t, err)
		expectedContribution := &ethpbalpha.SyncCommitteeContribution{
			Slot:              req.Data[0].Message.Contribution.Slot,
			BlockRoot:         req.Data[0].Message.Contribution.BeaconBlockRoot,
			SubcommitteeIndex: req.Data[0].Message.Contribution.SubcommitteeIndex,
			AggregationBits:   req.Data[0].Message.Contribution.AggregationBits,
			Signature:         req.Data[0].Message.Contribution.Signature,
		}
		require.DeepEqual(t, []*ethpbalpha.SyncCommitteeContribution{expectedContribution}, savedMsgs)
	})

	t.Run("Multiple contributions", func(t *testing.T) {
		v1Server.SyncCommitteePool = synccommittee.NewStore()
		req := &ethpbv2.SubmitContributionAndProofsRequest{
			Data: []*ethpbv2.SignedContributionAndProof{
				{
					Message: &ethpbv2.ContributionAndProof{
						AggregatorIndex: 0,
						Contribution: &ethpbv2.SyncCommitteeContribution{
							Slot:              0,
							BeaconBlockRoot:   root,
							SubcommitteeIndex: 0,
							AggregationBits:   aggBits,
							Signature:         sig,
						},
						SelectionProof: proof,
					},
					Signature: sig,
				},
				{
					Message: &ethpbv2.ContributionAndProof{
						AggregatorIndex: 1,
						Contribution: &ethpbv2.SyncCommitteeContribution{
							Slot:              1,
							BeaconBlockRoot:   root,
							SubcommitteeIndex: 1,
							AggregationBits:   aggBits,
							Signature:         sig,
						},
						SelectionProof: proof,
					},
					Signature: sig,
				},
			},
		}

		_, err := server.SubmitContributionAndProofs(ctx, req)
		require.NoError(t, err)
		savedMsgs, err := v1Server.SyncCommitteePool.SyncCommitteeContributions(0)
		require.NoError(t, err)
		expectedContributions := []*ethpbalpha.SyncCommitteeContribution{
			{
				Slot:              req.Data[0].Message.Contribution.Slot,
				BlockRoot:         req.Data[0].Message.Contribution.BeaconBlockRoot,
				SubcommitteeIndex: req.Data[0].Message.Contribution.SubcommitteeIndex,
				AggregationBits:   req.Data[0].Message.Contribution.AggregationBits,
				Signature:         req.Data[0].Message.Contribution.Signature,
			},
		}
		require.DeepEqual(t, expectedContributions, savedMsgs)
		savedMsgs, err = v1Server.SyncCommitteePool.SyncCommitteeContributions(1)
		require.NoError(t, err)
		expectedContributions = []*ethpbalpha.SyncCommitteeContribution{
			{
				Slot:              req.Data[1].Message.Contribution.Slot,
				BlockRoot:         req.Data[1].Message.Contribution.BeaconBlockRoot,
				SubcommitteeIndex: req.Data[1].Message.Contribution.SubcommitteeIndex,
				AggregationBits:   req.Data[1].Message.Contribution.AggregationBits,
				Signature:         req.Data[1].Message.Contribution.Signature,
			},
		}
		require.DeepEqual(t, expectedContributions, savedMsgs)
	})
}

func TestPrepareBeaconProposer(t *testing.T) {
	type args struct {
		request *ethpbv1.PrepareBeaconProposerRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr string
	}{
		{
			name: "Happy Path",
			args: args{
				request: &ethpbv1.PrepareBeaconProposerRequest{
					Recipients: []*ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer{
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
				request: &ethpbv1.PrepareBeaconProposerRequest{
					Recipients: []*ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer{
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
			hook := logTest.NewGlobal()
			v1Server := &v1alpha1validator.Server{
				BeaconDB: db,
			}
			server := &Server{
				V1Alpha1Server: v1Server,
			}
			_, err := server.PrepareBeaconProposer(ctx, tt.args.request)
			if tt.wantErr != "" {
				require.ErrorContains(t, tt.wantErr, err)
				return
			}
			require.NoError(t, err)
			address, err := server.V1Alpha1Server.BeaconDB.FeeRecipientByValidatorID(ctx, 1)
			require.NoError(t, err)
			require.Equal(t, common.BytesToAddress(tt.args.request.Recipients[0].FeeRecipient), address)
			indexs := make([]primitives.ValidatorIndex, len(tt.args.request.Recipients))
			for i, recipient := range tt.args.request.Recipients {
				indexs[i] = recipient.ValidatorIndex
			}
			require.LogsContain(t, hook, fmt.Sprintf(`validatorIndices="%v"`, indexs))
		})
	}
}
func TestProposer_PrepareBeaconProposerOverlapping(t *testing.T) {
	hook := logTest.NewGlobal()
	db := dbutil.SetupDB(t)
	ctx := context.Background()
	v1Server := &v1alpha1validator.Server{
		BeaconDB: db,
	}
	proposerServer := &Server{V1Alpha1Server: v1Server}

	// New validator
	f := bytesutil.PadTo([]byte{0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}, fieldparams.FeeRecipientLength)
	req := &ethpbv1.PrepareBeaconProposerRequest{
		Recipients: []*ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer{
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
	req = &ethpbv1.PrepareBeaconProposerRequest{
		Recipients: []*ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer{
			{FeeRecipient: f, ValidatorIndex: 1},
		},
	}
	_, err = proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsContain(t, hook, "Updated fee recipient addresses for validator indices")

	// More than one validator
	hook.Reset()
	f = bytesutil.PadTo([]byte{0x01, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}, fieldparams.FeeRecipientLength)
	req = &ethpbv1.PrepareBeaconProposerRequest{
		Recipients: []*ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer{
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
	v1Server := &v1alpha1validator.Server{
		BeaconDB: db,
	}
	proposerServer := &Server{V1Alpha1Server: v1Server}

	f := bytesutil.PadTo([]byte{0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}, fieldparams.FeeRecipientLength)
	recipients := make([]*ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer, 0)
	for i := 0; i < 10000; i++ {
		recipients = append(recipients, &ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer{FeeRecipient: f, ValidatorIndex: primitives.ValidatorIndex(i)})
	}

	req := &ethpbv1.PrepareBeaconProposerRequest{
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

func TestServer_SubmitValidatorRegistrations(t *testing.T) {
	type args struct {
		request *ethpbv1.SubmitValidatorRegistrationsRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr string
	}{
		{
			name: "Happy Path",
			args: args{
				request: &ethpbv1.SubmitValidatorRegistrationsRequest{
					Registrations: []*ethpbv1.SubmitValidatorRegistrationsRequest_SignedValidatorRegistration{
						{
							Message: &ethpbv1.SubmitValidatorRegistrationsRequest_ValidatorRegistration{
								FeeRecipient: make([]byte, fieldparams.BLSPubkeyLength),
								GasLimit:     30000000,
								Timestamp:    uint64(time.Now().Unix()),
								Pubkey:       make([]byte, fieldparams.BLSPubkeyLength),
							},
							Signature: make([]byte, fieldparams.BLSSignatureLength),
						},
					},
				},
			},
			wantErr: "",
		},
		{
			name: "Empty Request",
			args: args{
				request: &ethpbv1.SubmitValidatorRegistrationsRequest{
					Registrations: []*ethpbv1.SubmitValidatorRegistrationsRequest_SignedValidatorRegistration{},
				},
			},
			wantErr: "Validator registration request is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := dbutil.SetupDB(t)
			ctx := context.Background()
			v1Server := &v1alpha1validator.Server{
				BlockBuilder: &builderTest.MockBuilderService{
					HasConfigured: true,
				},
				BeaconDB: db,
			}
			server := &Server{
				V1Alpha1Server: v1Server,
			}
			_, err := server.SubmitValidatorRegistration(ctx, tt.args.request)
			if tt.wantErr != "" {
				require.ErrorContains(t, tt.wantErr, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGetLiveness(t *testing.T) {
	ctx := context.Background()

	// Setup:
	// Epoch 0 - both validators not live
	// Epoch 1 - validator with index 1 is live
	// Epoch 2 - validator with index 0 is live
	oldSt, err := util.NewBeaconStateBellatrix()
	require.NoError(t, err)
	require.NoError(t, oldSt.AppendCurrentParticipationBits(0))
	require.NoError(t, oldSt.AppendCurrentParticipationBits(0))
	headSt, err := util.NewBeaconStateBellatrix()
	require.NoError(t, err)
	require.NoError(t, headSt.SetSlot(params.BeaconConfig().SlotsPerEpoch*2))
	require.NoError(t, headSt.AppendPreviousParticipationBits(0))
	require.NoError(t, headSt.AppendPreviousParticipationBits(1))
	require.NoError(t, headSt.AppendCurrentParticipationBits(1))
	require.NoError(t, headSt.AppendCurrentParticipationBits(0))

	server := &Server{
		HeadFetcher: &mockChain.ChainService{State: headSt},
		Stater: &testutil.MockStater{
			// We configure states for last slots of an epoch
			StatesBySlot: map[primitives.Slot]state.BeaconState{
				params.BeaconConfig().SlotsPerEpoch - 1:   oldSt,
				params.BeaconConfig().SlotsPerEpoch*3 - 1: headSt,
			},
		},
	}

	t.Run("old epoch", func(t *testing.T) {
		resp, err := server.GetLiveness(ctx, &ethpbv2.GetLivenessRequest{
			Epoch: 0,
			Index: []primitives.ValidatorIndex{0, 1},
		})
		require.NoError(t, err)
		data0 := resp.Data[0]
		data1 := resp.Data[1]
		assert.Equal(t, true, (data0.Index == 0 && !data0.IsLive) || (data0.Index == 1 && !data0.IsLive))
		assert.Equal(t, true, (data1.Index == 0 && !data1.IsLive) || (data1.Index == 1 && !data1.IsLive))
	})
	t.Run("previous epoch", func(t *testing.T) {
		resp, err := server.GetLiveness(ctx, &ethpbv2.GetLivenessRequest{
			Epoch: 1,
			Index: []primitives.ValidatorIndex{0, 1},
		})
		require.NoError(t, err)
		data0 := resp.Data[0]
		data1 := resp.Data[1]
		assert.Equal(t, true, (data0.Index == 0 && !data0.IsLive) || (data0.Index == 1 && data0.IsLive))
		assert.Equal(t, true, (data1.Index == 0 && !data1.IsLive) || (data1.Index == 1 && data1.IsLive))
	})
	t.Run("current epoch", func(t *testing.T) {
		resp, err := server.GetLiveness(ctx, &ethpbv2.GetLivenessRequest{
			Epoch: 2,
			Index: []primitives.ValidatorIndex{0, 1},
		})
		require.NoError(t, err)
		data0 := resp.Data[0]
		data1 := resp.Data[1]
		assert.Equal(t, true, (data0.Index == 0 && data0.IsLive) || (data0.Index == 1 && !data0.IsLive))
		assert.Equal(t, true, (data1.Index == 0 && data1.IsLive) || (data1.Index == 1 && !data1.IsLive))
	})
	t.Run("future epoch", func(t *testing.T) {
		_, err := server.GetLiveness(ctx, &ethpbv2.GetLivenessRequest{
			Epoch: 3,
			Index: []primitives.ValidatorIndex{0, 1},
		})
		require.ErrorContains(t, "Requested epoch cannot be in the future", err)
	})
	t.Run("unknown validator index", func(t *testing.T) {
		_, err := server.GetLiveness(ctx, &ethpbv2.GetLivenessRequest{
			Epoch: 0,
			Index: []primitives.ValidatorIndex{0, 1, 2},
		})
		require.ErrorContains(t, "Validator index 2 is invalid", err)
	})
}
