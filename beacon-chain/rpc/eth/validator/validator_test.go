package validator

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/synccommittee"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	p2pmock "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	p2pType "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	v1alpha1validator "github.com/prysmaticlabs/prysm/beacon-chain/rpc/prysm/v1alpha1/validator"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/proto/migration"
	ethpbalpha "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	sharedtestutil "github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"google.golang.org/protobuf/proto"
)

func TestGetAttesterDuties(t *testing.T) {
	ctx := context.Background()
	genesis := sharedtestutil.NewBeaconBlock()
	depChainStart := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, err := sharedtestutil.DeterministicDepositsAndKeys(depChainStart)
	require.NoError(t, err)
	eth1Data, err := sharedtestutil.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bs, err := transition.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
	require.NoError(t, err, "Could not set up genesis state")
	// Set state to non-epoch start slot.
	require.NoError(t, bs.SetSlot(5))
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	roots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	roots[0] = genesisRoot[:]
	require.NoError(t, bs.SetBlockRoots(roots))

	pubKeys := make([][]byte, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = deposits[i].Data.PublicKey
	}

	chainSlot := types.Slot(0)
	chain := &mockChain.ChainService{
		State: bs, Root: genesisRoot[:], Slot: &chainSlot,
	}
	vs := &Server{
		HeadFetcher: chain,
		TimeFetcher: chain,
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}

	t.Run("Single validator", func(t *testing.T) {
		req := &ethpbv1.AttesterDutiesRequest{
			Epoch: 0,
			Index: []types.ValidatorIndex{0},
		}
		resp, err := vs.GetAttesterDuties(ctx, req)
		require.NoError(t, err)
		assert.DeepEqual(t, genesisRoot[:], resp.DependentRoot)
		require.Equal(t, 1, len(resp.Data))
		duty := resp.Data[0]
		assert.Equal(t, types.CommitteeIndex(2), duty.CommitteeIndex)
		assert.Equal(t, types.Slot(7), duty.Slot)
		assert.Equal(t, types.ValidatorIndex(0), duty.ValidatorIndex)
		assert.DeepEqual(t, pubKeys[0], duty.Pubkey)
		assert.Equal(t, uint64(128), duty.CommitteeLength)
		assert.Equal(t, uint64(4), duty.CommitteesAtSlot)
		assert.Equal(t, types.CommitteeIndex(123), duty.ValidatorCommitteeIndex)
	})

	t.Run("Multiple validators", func(t *testing.T) {
		req := &ethpbv1.AttesterDutiesRequest{
			Epoch: 0,
			Index: []types.ValidatorIndex{0, 1},
		}
		resp, err := vs.GetAttesterDuties(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 2, len(resp.Data))
	})

	t.Run("Next epoch", func(t *testing.T) {
		req := &ethpbv1.AttesterDutiesRequest{
			Epoch: core.SlotToEpoch(bs.Slot()) + 1,
			Index: []types.ValidatorIndex{0},
		}
		resp, err := vs.GetAttesterDuties(ctx, req)
		require.NoError(t, err)
		assert.DeepEqual(t, genesisRoot[:], resp.DependentRoot)
		require.Equal(t, 1, len(resp.Data))
		duty := resp.Data[0]
		assert.Equal(t, types.CommitteeIndex(1), duty.CommitteeIndex)
		assert.Equal(t, types.Slot(38), duty.Slot)
		assert.Equal(t, types.ValidatorIndex(0), duty.ValidatorIndex)
		assert.DeepEqual(t, pubKeys[0], duty.Pubkey)
		assert.Equal(t, uint64(128), duty.CommitteeLength)
		assert.Equal(t, uint64(4), duty.CommitteesAtSlot)
		assert.Equal(t, types.CommitteeIndex(27), duty.ValidatorCommitteeIndex)
	})

	t.Run("Require slot processing", func(t *testing.T) {
		// We create local variables to not interfere with other tests.
		// Slot processing might have unexpected side-effects.

		bs, err := transition.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
		require.NoError(t, err, "Could not set up genesis state")
		// Set state to non-epoch start slot.
		require.NoError(t, bs.SetSlot(5))
		genesisRoot, err := genesis.Block.HashTreeRoot()
		require.NoError(t, err, "Could not get signing root")
		roots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
		roots[0] = genesisRoot[:]
		require.NoError(t, bs.SetBlockRoots(roots))

		pubKeys := make([][]byte, len(deposits))
		indices := make([]uint64, len(deposits))
		for i := 0; i < len(deposits); i++ {
			pubKeys[i] = deposits[i].Data.PublicKey
			indices[i] = uint64(i)
		}
		chainSlot := params.BeaconConfig().SlotsPerEpoch.Mul(2)
		chain := &mockChain.ChainService{
			State: bs, Root: genesisRoot[:], Slot: &chainSlot,
		}
		vs := &Server{
			HeadFetcher: chain,
			TimeFetcher: chain,
			SyncChecker: &mockSync.Sync{IsSyncing: false},
		}

		req := &ethpbv1.AttesterDutiesRequest{
			Epoch: 2,
			Index: []types.ValidatorIndex{0},
		}
		resp, err := vs.GetAttesterDuties(ctx, req)
		require.NoError(t, err)
		assert.DeepEqual(t, bs.BlockRoots()[31], resp.DependentRoot)
		require.Equal(t, 1, len(resp.Data))
		duty := resp.Data[0]
		assert.Equal(t, types.CommitteeIndex(1), duty.CommitteeIndex)
		assert.Equal(t, types.Slot(86), duty.Slot)
		assert.Equal(t, types.ValidatorIndex(0), duty.ValidatorIndex)
		assert.DeepEqual(t, pubKeys[0], duty.Pubkey)
		assert.Equal(t, uint64(128), duty.CommitteeLength)
		assert.Equal(t, uint64(4), duty.CommitteesAtSlot)
		assert.Equal(t, types.CommitteeIndex(44), duty.ValidatorCommitteeIndex)
	})

	t.Run("Epoch out of bound", func(t *testing.T) {
		currentEpoch := core.SlotToEpoch(bs.Slot())
		req := &ethpbv1.AttesterDutiesRequest{
			Epoch: currentEpoch + 2,
			Index: []types.ValidatorIndex{0},
		}
		_, err := vs.GetAttesterDuties(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, fmt.Sprintf("Request epoch %d can not be greater than next epoch %d", currentEpoch+2, currentEpoch+1), err)
	})

	t.Run("Validator index out of bound", func(t *testing.T) {
		req := &ethpbv1.AttesterDutiesRequest{
			Epoch: 0,
			Index: []types.ValidatorIndex{types.ValidatorIndex(len(pubKeys))},
		}
		_, err := vs.GetAttesterDuties(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Invalid validator index", err)
	})
}

func TestGetAttesterDuties_SyncNotReady(t *testing.T) {
	vs := &Server{
		SyncChecker: &mockSync.Sync{IsSyncing: true},
	}
	_, err := vs.GetAttesterDuties(context.Background(), &ethpbv1.AttesterDutiesRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestGetProposerDuties(t *testing.T) {
	ctx := context.Background()
	genesis := sharedtestutil.NewBeaconBlock()
	depChainStart := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, err := sharedtestutil.DeterministicDepositsAndKeys(depChainStart)
	require.NoError(t, err)
	eth1Data, err := sharedtestutil.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bs, err := transition.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
	require.NoError(t, err, "Could not set up genesis state")
	// Set state to non-epoch start slot.
	require.NoError(t, bs.SetSlot(5))
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	roots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	roots[0] = genesisRoot[:]
	require.NoError(t, bs.SetBlockRoots(roots))

	pubKeys := make([][]byte, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = deposits[i].Data.PublicKey
	}

	chainSlot := types.Slot(0)
	chain := &mockChain.ChainService{
		State: bs, Root: genesisRoot[:], Slot: &chainSlot,
	}
	vs := &Server{
		HeadFetcher: chain,
		TimeFetcher: chain,
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}

	t.Run("Ok", func(t *testing.T) {
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
		require.NotNil(t, expectedDuty, "Expected duty for slot 11 not found")
		assert.Equal(t, types.ValidatorIndex(12289), expectedDuty.ValidatorIndex)
		assert.DeepEqual(t, pubKeys[12289], expectedDuty.Pubkey)
	})

	t.Run("Require slot processing", func(t *testing.T) {
		// We create local variables to not interfere with other tests.
		// Slot processing might have unexpected side-effects.

		bs, err := transition.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
		require.NoError(t, err, "Could not set up genesis state")
		// Set state to non-epoch start slot.
		require.NoError(t, bs.SetSlot(5))
		genesisRoot, err := genesis.Block.HashTreeRoot()
		require.NoError(t, err, "Could not get signing root")
		roots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
		roots[0] = genesisRoot[:]
		require.NoError(t, bs.SetBlockRoots(roots))

		pubKeys := make([][]byte, len(deposits))
		indices := make([]uint64, len(deposits))
		for i := 0; i < len(deposits); i++ {
			pubKeys[i] = deposits[i].Data.PublicKey
			indices[i] = uint64(i)
		}
		chainSlot := params.BeaconConfig().SlotsPerEpoch.Mul(2)
		chain := &mockChain.ChainService{
			State: bs, Root: genesisRoot[:], Slot: &chainSlot,
		}
		vs := &Server{
			HeadFetcher: chain,
			TimeFetcher: chain,
			SyncChecker: &mockSync.Sync{IsSyncing: false},
		}

		req := &ethpbv1.ProposerDutiesRequest{
			Epoch: 2,
		}
		resp, err := vs.GetProposerDuties(ctx, req)
		require.NoError(t, err)
		assert.DeepEqual(t, bs.BlockRoots()[31], resp.DependentRoot)
		assert.Equal(t, 32, len(resp.Data))
		// We expect a proposer duty for slot 74.
		var expectedDuty *ethpbv1.ProposerDuty
		for _, duty := range resp.Data {
			if duty.Slot == 74 {
				expectedDuty = duty
			}
		}
		require.NotNil(t, expectedDuty, "Expected duty for slot 74 not found")
		assert.Equal(t, types.ValidatorIndex(11741), expectedDuty.ValidatorIndex)
		assert.DeepEqual(t, pubKeys[11741], expectedDuty.Pubkey)
	})

	t.Run("Epoch out of bound", func(t *testing.T) {
		currentEpoch := core.SlotToEpoch(bs.Slot())
		req := &ethpbv1.ProposerDutiesRequest{
			Epoch: currentEpoch + 1,
		}
		_, err := vs.GetProposerDuties(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, fmt.Sprintf("Request epoch %d can not be greater than current epoch %d", currentEpoch+1, currentEpoch), err)
	})
}

func TestGetProposerDuties_SyncNotReady(t *testing.T) {
	vs := &Server{
		SyncChecker: &mockSync.Sync{IsSyncing: true},
	}
	_, err := vs.GetProposerDuties(context.Background(), &ethpbv1.ProposerDutiesRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestGetSyncCommitteeDuties(t *testing.T) {
	ctx := context.Background()
	numVals := uint64(10)
	st, _ := sharedtestutil.DeterministicGenesisStateAltair(t, numVals)
	vals := st.Validators()
	committee := &ethpbalpha.SyncCommittee{}
	for _, v := range vals {
		committee.Pubkeys = append(committee.Pubkeys, v.PublicKey)
	}
	// add one public key twice - this is needed for one of the test cases
	committee.Pubkeys = append(committee.Pubkeys, vals[0].PublicKey)
	require.NoError(t, st.SetCurrentSyncCommittee(committee))
	vs := &Server{
		StateFetcher: &testutil.MockFetcher{BeaconState: st},
		SyncChecker:  &mockSync.Sync{IsSyncing: false},
	}

	t.Run("Single validator", func(t *testing.T) {
		req := &ethpbv2.SyncCommitteeDutiesRequest{
			Epoch: 0,
			Index: []types.ValidatorIndex{1},
		}
		resp, err := vs.GetSyncCommitteeDuties(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		require.Equal(t, 1, len(resp.Data))
		duty := resp.Data[0]
		assert.DeepEqual(t, vals[1].PublicKey, duty.Pubkey)
		assert.Equal(t, types.ValidatorIndex(1), duty.ValidatorIndex)
		require.Equal(t, 1, len(duty.ValidatorSyncCommitteeIndices))
		assert.Equal(t, uint64(1), duty.ValidatorSyncCommitteeIndices[0])
	})

	t.Run("Multiple validators", func(t *testing.T) {
		req := &ethpbv2.SyncCommitteeDutiesRequest{
			Epoch: 0,
			Index: []types.ValidatorIndex{1, 2},
		}
		resp, err := vs.GetSyncCommitteeDuties(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 2, len(resp.Data))
	})

	t.Run("Multiple indices for validator", func(t *testing.T) {
		req := &ethpbv2.SyncCommitteeDutiesRequest{
			Epoch: 0,
			Index: []types.ValidatorIndex{0},
		}
		resp, err := vs.GetSyncCommitteeDuties(ctx, req)
		require.NoError(t, err)
		duty := resp.Data[0]
		require.Equal(t, 2, len(duty.ValidatorSyncCommitteeIndices))
		assert.DeepEqual(t, []uint64{0, 10}, duty.ValidatorSyncCommitteeIndices)
	})

	t.Run("Validator index out of bound", func(t *testing.T) {
		req := &ethpbv2.SyncCommitteeDutiesRequest{
			Epoch: 0,
			Index: []types.ValidatorIndex{types.ValidatorIndex(numVals)},
		}
		_, err := vs.GetSyncCommitteeDuties(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Invalid validator index", err)
	})
}

func TestGetSyncCommitteeDuties_SyncNotReady(t *testing.T) {
	vs := &Server{
		SyncChecker: &mockSync.Sync{IsSyncing: true},
	}
	_, err := vs.GetSyncCommitteeDuties(context.Background(), &ethpbv2.SyncCommitteeDutiesRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestProduceBlock(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privKeys := sharedtestutil.DeterministicGenesisState(t, 64)

	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	genesis := b.NewGenesisBlock(stateRoot[:])
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)), "Could not save genesis block")

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	v1Alpha1Server := &v1alpha1validator.Server{
		HeadFetcher:       &mockChain.ChainService{State: beaconState, Root: parentRoot[:]},
		SyncChecker:       &mockSync.Sync{IsSyncing: false},
		BlockReceiver:     &mockChain.ChainService{},
		ChainStartFetcher: &mockPOW.POWChain{},
		Eth1InfoFetcher:   &mockPOW.POWChain{},
		Eth1BlockFetcher:  &mockPOW.POWChain{},
		MockEth1Votes:     true,
		AttPool:           attestations.NewPool(),
		SlashingsPool:     slashings.NewPool(),
		ExitPool:          voluntaryexits.NewPool(),
		StateGen:          stategen.New(db),
	}

	proposerSlashings := make([]*ethpbalpha.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
	for i := types.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
		proposerSlashing, err := sharedtestutil.GenerateProposerSlashingForValidator(
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
		attesterSlashing, err := sharedtestutil.GenerateAttesterSlashingForValidator(
			beaconState,
			privKeys[i+params.BeaconConfig().MaxProposerSlashings],
			types.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings), /* validator index */
		)
		require.NoError(t, err)
		attSlashings[i] = attesterSlashing
		err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), beaconState, attesterSlashing)
		require.NoError(t, err)
	}

	v1Server := &Server{
		V1Alpha1Server: v1Alpha1Server,
	}
	randaoReveal, err := sharedtestutil.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)
	graffiti := bytesutil.ToBytes32([]byte("eth2"))
	req := &ethpbv1.ProduceBlockRequest{
		Slot:         1,
		RandaoReveal: randaoReveal,
		Graffiti:     graffiti[:],
	}
	resp, err := v1Server.ProduceBlock(ctx, req)
	require.NoError(t, err)

	assert.Equal(t, req.Slot, resp.Data.Slot, "Expected block to have slot of 1")
	assert.DeepEqual(t, parentRoot[:], resp.Data.ParentRoot, "Expected block to have correct parent root")
	assert.DeepEqual(t, randaoReveal, resp.Data.Body.RandaoReveal, "Expected block to have correct randao reveal")
	assert.DeepEqual(t, req.Graffiti, resp.Data.Body.Graffiti, "Expected block to have correct graffiti")
	assert.Equal(t, params.BeaconConfig().MaxProposerSlashings, uint64(len(resp.Data.Body.ProposerSlashings)))
	expectedPropSlashings := make([]*ethpbv1.ProposerSlashing, len(proposerSlashings))
	for i, slash := range proposerSlashings {
		expectedPropSlashings[i] = migration.V1Alpha1ProposerSlashingToV1(slash)
	}
	assert.DeepEqual(t, expectedPropSlashings, resp.Data.Body.ProposerSlashings)
	assert.Equal(t, params.BeaconConfig().MaxAttesterSlashings, uint64(len(resp.Data.Body.AttesterSlashings)))
	expectedAttSlashings := make([]*ethpbv1.AttesterSlashing, len(attSlashings))
	for i, slash := range attSlashings {
		expectedAttSlashings[i] = migration.V1Alpha1AttSlashingToV1(slash)
	}
	assert.DeepEqual(t, expectedAttSlashings, resp.Data.Body.AttesterSlashings)
}

func TestProduceBlockV2(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		db := dbutil.SetupDB(t)
		ctx := context.Background()

		beaconState, privKeys := sharedtestutil.DeterministicGenesisState(t, 64)

		stateRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err, "Could not hash genesis state")

		genesis := b.NewGenesisBlock(stateRoot[:])
		require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)), "Could not save genesis block")

		parentRoot, err := genesis.Block.HashTreeRoot()
		require.NoError(t, err, "Could not get signing root")
		require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
		require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

		v1Alpha1Server := &v1alpha1validator.Server{
			HeadFetcher:       &mockChain.ChainService{State: beaconState, Root: parentRoot[:]},
			SyncChecker:       &mockSync.Sync{IsSyncing: false},
			BlockReceiver:     &mockChain.ChainService{},
			ChainStartFetcher: &mockPOW.POWChain{},
			Eth1InfoFetcher:   &mockPOW.POWChain{},
			Eth1BlockFetcher:  &mockPOW.POWChain{},
			MockEth1Votes:     true,
			AttPool:           attestations.NewPool(),
			SlashingsPool:     slashings.NewPool(),
			ExitPool:          voluntaryexits.NewPool(),
			StateGen:          stategen.New(db),
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
		for i := types.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
			proposerSlashing, err := sharedtestutil.GenerateProposerSlashingForValidator(
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
			attesterSlashing, err := sharedtestutil.GenerateAttesterSlashingForValidator(
				beaconState,
				privKeys[i+params.BeaconConfig().MaxProposerSlashings],
				types.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings), /* validator index */
			)
			require.NoError(t, err)
			attSlashings[i] = attesterSlashing
			err = v1Alpha1Server.SlashingsPool.InsertAttesterSlashing(context.Background(), beaconState, attesterSlashing)
			require.NoError(t, err)
		}

		v1Server := &Server{
			V1Alpha1Server: v1Alpha1Server,
		}
		randaoReveal, err := sharedtestutil.RandaoReveal(beaconState, 0, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))
		req := &ethpbv1.ProduceBlockRequest{
			Slot:         1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		resp, err := v1Server.ProduceBlockV2(ctx, req)
		require.NoError(t, err)

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
		bc := params.BeaconConfig()
		bc.AltairForkEpoch = types.Epoch(0)
		params.OverrideBeaconConfig(bc)

		beaconState, privKeys := sharedtestutil.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
		syncCommittee, err := altair.NextSyncCommittee(context.Background(), beaconState)
		require.NoError(t, err)
		require.NoError(t, beaconState.SetCurrentSyncCommittee(syncCommittee))
		require.NoError(t, beaconState.SetNextSyncCommittee(syncCommittee))

		stateRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err, "Could not hash genesis state")
		genesisBlock := sharedtestutil.NewBeaconBlockAltair()
		genesisBlock.Block.StateRoot = stateRoot[:]
		wrappedAltairBlock, err := wrapper.WrappedAltairSignedBeaconBlock(genesisBlock)
		require.NoError(t, err)
		require.NoError(t, db.SaveBlock(ctx, wrappedAltairBlock))
		parentRoot, err := genesisBlock.Block.HashTreeRoot()

		require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
		require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

		v1Alpha1Server := &v1alpha1validator.Server{
			HeadFetcher:       &mockChain.ChainService{State: beaconState, Root: parentRoot[:]},
			SyncChecker:       &mockSync.Sync{IsSyncing: false},
			BlockReceiver:     &mockChain.ChainService{},
			ChainStartFetcher: &mockPOW.POWChain{},
			Eth1InfoFetcher:   &mockPOW.POWChain{},
			Eth1BlockFetcher:  &mockPOW.POWChain{},
			MockEth1Votes:     true,
			AttPool:           attestations.NewPool(),
			SlashingsPool:     slashings.NewPool(),
			ExitPool:          voluntaryexits.NewPool(),
			StateGen:          stategen.New(db),
			SyncCommitteePool: synccommittee.NewStore(),
		}

		proposerSlashings := make([]*ethpbalpha.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
		for i := types.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
			proposerSlashing, err := sharedtestutil.GenerateProposerSlashingForValidator(
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
			attesterSlashing, err := sharedtestutil.GenerateAttesterSlashingForValidator(
				beaconState,
				privKeys[i+params.BeaconConfig().MaxProposerSlashings],
				types.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings), /* validator index */
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
		//ps := helpers.PrevSlot(beaconState.Slot())
		//pbr, err := helpers.BlockRootAtSlot(beaconState, ps)
		//require.NoError(t, err)

		syncCommitteeIndices, err := altair.NextSyncCommitteeIndices(context.Background(), beaconState)
		require.NoError(t, err)
		sigs := make([]bls.Signature, 0, len(syncCommitteeIndices))
		for i, indice := range syncCommitteeIndices {
			if aggregationBits.BitAt(uint64(i)) {
				b := p2pType.SSZBytes(parentRoot[:])
				sb, err := helpers.ComputeDomainAndSign(beaconState, core.CurrentEpoch(beaconState), &b, params.BeaconConfig().DomainSyncCommittee, privKeys[indice])
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
		}
		randaoReveal, err := sharedtestutil.RandaoReveal(beaconState, 0, privKeys)
		require.NoError(t, err)
		graffiti := bytesutil.ToBytes32([]byte("eth2"))

		req := &ethpbv1.ProduceBlockRequest{
			Slot:         1,
			RandaoReveal: randaoReveal,
			Graffiti:     graffiti[:],
		}
		resp, err := v1Server.ProduceBlockV2(ctx, req)
		require.NoError(t, err)

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
		// TODO sync aggregate
	})
}

func TestProduceAttestationData(t *testing.T) {
	block := sharedtestutil.NewBeaconBlock()
	block.Block.Slot = 3*params.BeaconConfig().SlotsPerEpoch + 1
	targetBlock := sharedtestutil.NewBeaconBlock()
	targetBlock.Block.Slot = 1 * params.BeaconConfig().SlotsPerEpoch
	justifiedBlock := sharedtestutil.NewBeaconBlock()
	justifiedBlock.Block.Slot = 2 * params.BeaconConfig().SlotsPerEpoch
	blockRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not hash beacon block")
	justifiedRoot, err := justifiedBlock.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root for justified block")
	targetRoot, err := targetBlock.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root for target block")
	slot := 3*params.BeaconConfig().SlotsPerEpoch + 1
	beaconState, err := sharedtestutil.NewBeaconState()
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
	sig1 := bytesutil.PadTo([]byte("sig1"), 96)
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
	root2_1 := bytesutil.PadTo([]byte("root2_1"), 32)
	sig2_1 := bytesutil.PadTo([]byte("sig2_1"), 96)
	attSlot2_1 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 1, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            2,
			CommitteeIndex:  2,
			BeaconBlockRoot: root2_1,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root2_1,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root2_1,
			},
		},
		Signature: sig2_1,
	}
	root2_2 := bytesutil.PadTo([]byte("root2_2"), 32)
	sig2_2 := bytesutil.PadTo([]byte("sig2_2"), 96)
	attSlot2_2 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 1, 1, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            2,
			CommitteeIndex:  3,
			BeaconBlockRoot: root2_2,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root2_2,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root2_2,
			},
		},
		Signature: sig2_2,
	}
	vs := &Server{
		AttestationsPool: &attestations.PoolMock{AggregatedAtts: []*ethpbalpha.Attestation{attSlot1, attSlot2_1, attSlot2_2}},
	}

	t.Run("OK", func(t *testing.T) {
		reqRoot, err := attSlot2_2.Data.HashTreeRoot()
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
		assert.DeepEqual(t, sig2_2, att.Data.Signature)
		assert.Equal(t, types.Slot(2), att.Data.Data.Slot)
		assert.Equal(t, types.CommitteeIndex(3), att.Data.Data.Index)
		assert.DeepEqual(t, root2_2, att.Data.Data.BeaconBlockRoot)
		require.NotNil(t, att.Data.Data.Source)
		assert.Equal(t, types.Epoch(1), att.Data.Data.Source.Epoch)
		assert.DeepEqual(t, root2_2, att.Data.Data.Source.Root)
		require.NotNil(t, att.Data.Data.Target)
		assert.Equal(t, types.Epoch(1), att.Data.Data.Target.Epoch)
		assert.DeepEqual(t, root2_2, att.Data.Data.Target.Root)
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
	sig := bytesutil.PadTo([]byte("sig"), 96)
	att1 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 1},
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
		AggregationBits: []byte{0, 1, 1},
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
	vs := &Server{
		AttestationsPool: &attestations.PoolMock{AggregatedAtts: []*ethpbalpha.Attestation{att1, att2}},
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
	assert.DeepEqual(t, bitfield.Bitlist{0, 1, 1}, att.Data.AggregationBits)
}

func TestSubmitBeaconCommitteeSubscription(t *testing.T) {
	ctx := context.Background()
	genesis := sharedtestutil.NewBeaconBlock()
	depChainStart := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, err := sharedtestutil.DeterministicDepositsAndKeys(depChainStart)
	require.NoError(t, err)
	eth1Data, err := sharedtestutil.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bs, err := transition.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
	require.NoError(t, err, "Could not set up genesis state")
	// Set state to non-epoch start slot.
	require.NoError(t, bs.SetSlot(5))
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	roots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	roots[0] = genesisRoot[:]
	require.NoError(t, bs.SetBlockRoots(roots))

	pubKeys := make([][]byte, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = deposits[i].Data.PublicKey
	}

	chainSlot := types.Slot(0)
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
		assert.Equal(t, uint64(5), subnets[0])
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
		ids := cache.SubnetIDs.GetAggregatorSubnetIDs(types.Slot(1))
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
	vs := &Server{
		SyncChecker: &mockSync.Sync{IsSyncing: true},
	}
	_, err := vs.SubmitBeaconCommitteeSubscription(context.Background(), &ethpbv1.SubmitBeaconCommitteeSubscriptionsRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestSubmitSyncCommitteeSubscription(t *testing.T) {
	ctx := context.Background()
	genesis := sharedtestutil.NewBeaconBlock()
	deposits, _, err := sharedtestutil.DeterministicDepositsAndKeys(64)
	require.NoError(t, err)
	eth1Data, err := sharedtestutil.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bs, err := sharedtestutil.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
	require.NoError(t, err, "Could not set up genesis state")
	// Set state to epoch 1.
	require.NoError(t, bs.SetSlot(params.BeaconConfig().SlotsPerEpoch))
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	roots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	roots[0] = genesisRoot[:]
	require.NoError(t, bs.SetBlockRoots(roots))

	pubkeys := make([][]byte, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubkeys[i] = deposits[i].Data.PublicKey
	}

	chainSlot := types.Slot(0)
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

	t.Run("Epoch too far in the future", func(t *testing.T) {
		req := &ethpbv2.SubmitSyncCommitteeSubscriptionsRequest{
			Data: []*ethpbv2.SyncCommitteeSubscription{
				{
					ValidatorIndex:       0,
					SyncCommitteeIndices: []uint64{},
					UntilEpoch:           params.BeaconConfig().EpochsPerSyncCommitteePeriod + 1,
				},
			},
		}
		_, err = vs.SubmitSyncCommitteeSubscription(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Epoch for subscription at index 0 is too far in the future", err)
	})
}

func TestSubmitSyncCommitteeSubscription_SyncNotReady(t *testing.T) {
	vs := &Server{
		SyncChecker: &mockSync.Sync{IsSyncing: true},
	}
	_, err := vs.SubmitSyncCommitteeSubscription(context.Background(), &ethpbv2.SubmitSyncCommitteeSubscriptionsRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestSubmitAggregateAndProofs(t *testing.T) {
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	c := params.BeaconNetworkConfig()
	c.MaximumGossipClockDisparity = time.Hour
	params.OverrideBeaconNetworkConfig(c)
	root := bytesutil.PadTo([]byte("root"), 32)
	sig := bytesutil.PadTo([]byte("sig"), 96)
	proof := bytesutil.PadTo([]byte("proof"), 96)
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
		chainSlot := types.Slot(0)
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
		chainSlot := types.Slot(0)
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
			CurrentSyncCommitteeIndices: []types.CommitteeIndex{0},
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
	assert.Equal(t, types.Slot(0), resp.Data.Slot)
	assert.Equal(t, uint64(0), resp.Data.SubcommitteeIndex)
	assert.DeepEqual(t, root, resp.Data.BeaconBlockRoot)
	aggregationBits := resp.Data.AggregationBits
	assert.Equal(t, true, aggregationBits.BitAt(0))
	assert.DeepEqual(t, sig, resp.Data.Signature)
}

func TestSubmitContributionAndProofs(t *testing.T) {
	ctx := context.Background()
	sig := bls.NewAggregateSignature().Marshal()
	root := bytesutil.PadTo([]byte("root"), 32)
	proof := bytesutil.PadTo([]byte("proof"), 96)
	aggBits := bitfield.NewBitvector128()
	aggBits.SetBitAt(0, true)
	v1Server := &v1alpha1validator.Server{
		P2P: &p2pmock.MockBroadcaster{},
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
