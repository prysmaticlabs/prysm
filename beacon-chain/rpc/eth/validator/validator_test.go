package validator

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/go-bitfield"
	mockChain "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	builderTest "github.com/prysmaticlabs/prysm/v4/beacon-chain/builder/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	dbutil "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/synccommittee"
	p2pmock "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/testing"
	v1alpha1validator "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/prysm/v1alpha1/validator"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	mockSync "github.com/prysmaticlabs/prysm/v4/beacon-chain/sync/initial-sync/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/mock"
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
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	t.Run("Phase 0", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Phase0{Phase0: &ethpbalpha.BeaconBlock{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}

		resp, err := server.ProduceBlockV2(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_PHASE0, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BeaconBlockContainerV2_Phase0Block)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.Phase0Block.Slot)
	})
	t.Run("Altair", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Altair{Altair: &ethpbalpha.BeaconBlockAltair{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}

		resp, err := server.ProduceBlockV2(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BeaconBlockContainerV2_AltairBlock)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.AltairBlock.Slot)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Bellatrix{Bellatrix: &ethpbalpha.BeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlockV2(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BeaconBlockContainerV2_BellatrixBlock)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.BellatrixBlock.Slot)
	})
	t.Run("Bellatrix blinded", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: &ethpbalpha.BlindedBeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlockV2(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared Bellatrix beacon block is blinded", err)
	})
	t.Run("Capella", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Capella{Capella: &ethpbalpha.BeaconBlockCapella{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlockV2(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		assert.Equal(t, ethpbv2.Version_CAPELLA, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BeaconBlockContainerV2_CapellaBlock)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.CapellaBlock.Slot)
	})
	t.Run("Capella blinded", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedCapella{BlindedCapella: &ethpbalpha.BlindedBeaconBlockCapella{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlockV2(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared Capella beacon block is blinded", err)
	})
	t.Run("optimistic", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Bellatrix{Bellatrix: &ethpbalpha.BeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: true},
		}

		_, err := server.ProduceBlockV2(ctx, &ethpbv1.ProduceBlockRequest{})
		require.ErrorContains(t, "The node is currently optimistic and cannot serve validators", err)
	})
	t.Run("sync not ready", func(t *testing.T) {
		st, err := util.NewBeaconState()
		require.NoError(t, err)
		chainService := &mockChain.ChainService{State: st}
		v1Server := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: true},
			HeadFetcher:           chainService,
			TimeFetcher:           chainService,
			OptimisticModeFetcher: chainService,
		}
		_, err = v1Server.ProduceBlockV2(context.Background(), nil)
		require.ErrorContains(t, "Syncing to latest head", err)
	})
}

func TestProduceBlockV2SSZ(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	t.Run("Phase 0", func(t *testing.T) {
		b := util.HydrateBeaconBlock(&ethpbalpha.BeaconBlock{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Phase0{Phase0: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}

		resp, err := server.ProduceBlockV2SSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Altair", func(t *testing.T) {
		b := util.HydrateBeaconBlockAltair(&ethpbalpha.BeaconBlockAltair{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Altair{Altair: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}

		resp, err := server.ProduceBlockV2SSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		b := util.HydrateBeaconBlockBellatrix(&ethpbalpha.BeaconBlockBellatrix{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Bellatrix{Bellatrix: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlockV2SSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Bellatrix blinded", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: &ethpbalpha.BlindedBeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlockV2SSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared Bellatrix beacon block is blinded", err)
	})
	t.Run("Capella", func(t *testing.T) {
		b := util.HydrateBeaconBlockCapella(&ethpbalpha.BeaconBlockCapella{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Capella{Capella: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlockV2SSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Capella blinded", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedCapella{BlindedCapella: &ethpbalpha.BlindedBeaconBlockCapella{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlockV2SSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared Capella beacon block is blinded", err)
	})
	t.Run("optimistic", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Bellatrix{Bellatrix: &ethpbalpha.BeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: true},
		}

		_, err := server.ProduceBlockV2SSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.ErrorContains(t, "The node is currently optimistic and cannot serve validators", err)
	})
	t.Run("sync not ready", func(t *testing.T) {
		st, err := util.NewBeaconState()
		require.NoError(t, err)
		chainService := &mockChain.ChainService{State: st}
		v1Server := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: true},
			HeadFetcher:           chainService,
			TimeFetcher:           chainService,
			OptimisticModeFetcher: chainService,
		}
		_, err = v1Server.ProduceBlockV2SSZ(context.Background(), nil)
		require.ErrorContains(t, "Syncing to latest head", err)
	})
}

func TestProduceBlindedBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	t.Run("Phase 0", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Phase0{Phase0: &ethpbalpha.BeaconBlock{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
			BlockBuilder:   &builderTest.MockBuilderService{HasConfigured: true},
		}

		resp, err := server.ProduceBlindedBlock(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_PHASE0, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BlindedBeaconBlockContainer_Phase0Block)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.Phase0Block.Slot)
	})
	t.Run("Altair", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Altair{Altair: &ethpbalpha.BeaconBlockAltair{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
			BlockBuilder:   &builderTest.MockBuilderService{HasConfigured: true},
		}

		resp, err := server.ProduceBlindedBlock(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BlindedBeaconBlockContainer_AltairBlock)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.AltairBlock.Slot)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: &ethpbalpha.BlindedBeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlindedBlock(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BlindedBeaconBlockContainer_BellatrixBlock)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.BellatrixBlock.Slot)
	})
	t.Run("Bellatrix full", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Bellatrix{Bellatrix: &ethpbalpha.BeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlindedBlock(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared beacon block is not blinded", err)
	})
	t.Run("Capella", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedCapella{BlindedCapella: &ethpbalpha.BlindedBeaconBlockCapella{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlindedBlock(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_CAPELLA, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BlindedBeaconBlockContainer_CapellaBlock)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.CapellaBlock.Slot)
	})
	t.Run("Capella full", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Capella{Capella: &ethpbalpha.BeaconBlockCapella{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlindedBlock(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared beacon block is not blinded", err)
	})
	t.Run("optimistic", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: &ethpbalpha.BlindedBeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: true},
		}

		_, err := server.ProduceBlindedBlock(ctx, &ethpbv1.ProduceBlockRequest{})
		require.ErrorContains(t, "The node is currently optimistic and cannot serve validators", err)
	})
	t.Run("builder not configured", func(t *testing.T) {
		v1Server := &Server{
			BlockBuilder: &builderTest.MockBuilderService{HasConfigured: false},
		}
		_, err := v1Server.ProduceBlindedBlock(context.Background(), nil)
		require.ErrorContains(t, "Block builder not configured", err)
	})
	t.Run("sync not ready", func(t *testing.T) {
		st, err := util.NewBeaconState()
		require.NoError(t, err)
		chainService := &mockChain.ChainService{State: st}
		v1Server := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: true},
			HeadFetcher:           chainService,
			TimeFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
		}
		_, err = v1Server.ProduceBlindedBlock(context.Background(), nil)
		require.ErrorContains(t, "Syncing to latest head", err)
	})
}

func TestProduceBlindedBlockSSZ(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	t.Run("Phase 0", func(t *testing.T) {
		b := util.HydrateBeaconBlock(&ethpbalpha.BeaconBlock{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Phase0{Phase0: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
			BlockBuilder:   &builderTest.MockBuilderService{HasConfigured: true},
		}

		resp, err := server.ProduceBlindedBlockSSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Altair", func(t *testing.T) {
		b := util.HydrateBeaconBlockAltair(&ethpbalpha.BeaconBlockAltair{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Altair{Altair: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
			BlockBuilder:   &builderTest.MockBuilderService{HasConfigured: true},
		}

		resp, err := server.ProduceBlindedBlockSSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		b := util.HydrateBlindedBeaconBlockBellatrix(&ethpbalpha.BlindedBeaconBlockBellatrix{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlindedBlockSSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Bellatrix full", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Bellatrix{Bellatrix: &ethpbalpha.BeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlindedBlockSSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared Bellatrix beacon block is not blinded", err)
	})
	t.Run("Capella", func(t *testing.T) {
		b := util.HydrateBlindedBeaconBlockCapella(&ethpbalpha.BlindedBeaconBlockCapella{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedCapella{BlindedCapella: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlindedBlockSSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Capella full", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Capella{Capella: &ethpbalpha.BeaconBlockCapella{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlindedBlockSSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared Capella beacon block is not blinded", err)
	})
	t.Run("optimistic", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: &ethpbalpha.BlindedBeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: true},
		}

		_, err := server.ProduceBlindedBlockSSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.ErrorContains(t, "The node is currently optimistic and cannot serve validators", err)
	})
	t.Run("builder not configured", func(t *testing.T) {
		v1Server := &Server{
			BlockBuilder: &builderTest.MockBuilderService{HasConfigured: false},
		}
		_, err := v1Server.ProduceBlindedBlockSSZ(context.Background(), nil)
		require.ErrorContains(t, "Block builder not configured", err)
	})
	t.Run("sync not ready", func(t *testing.T) {
		st, err := util.NewBeaconState()
		require.NoError(t, err)
		chainService := &mockChain.ChainService{State: st}
		v1Server := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: true},
			HeadFetcher:           chainService,
			TimeFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
		}
		_, err = v1Server.ProduceBlindedBlockSSZ(context.Background(), nil)
		require.ErrorContains(t, "Syncing to latest head", err)
	})
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
			server := &Server{
				BeaconDB: db,
			}
			_, err := server.PrepareBeaconProposer(ctx, tt.args.request)
			if tt.wantErr != "" {
				require.ErrorContains(t, tt.wantErr, err)
				return
			}
			require.NoError(t, err)
			address, err := server.BeaconDB.FeeRecipientByValidatorID(ctx, 1)
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
	proposerServer := &Server{BeaconDB: db}

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
	proposerServer := &Server{BeaconDB: db}

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
			server := &Server{
				BlockBuilder: &builderTest.MockBuilderService{
					HasConfigured: true,
				},
				BeaconDB: db,
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
