package validator

import (
	"context"
	"fmt"
	"testing"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	v1alpha1validator "github.com/prysmaticlabs/prysm/beacon-chain/rpc/prysm/v1alpha1/validator"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/proto/migration"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"google.golang.org/protobuf/proto"
)

func TestGetAttesterDuties(t *testing.T) {
	ctx := context.Background()
	genesis := testutil.NewBeaconBlock()
	depChainStart := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, err := testutil.DeterministicDepositsAndKeys(depChainStart)
	require.NoError(t, err)
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bs, err := state.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
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
		req := &v1.AttesterDutiesRequest{
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
		req := &v1.AttesterDutiesRequest{
			Epoch: 0,
			Index: []types.ValidatorIndex{0, 1},
		}
		resp, err := vs.GetAttesterDuties(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 2, len(resp.Data))
	})

	t.Run("Next epoch", func(t *testing.T) {
		req := &v1.AttesterDutiesRequest{
			Epoch: helpers.SlotToEpoch(bs.Slot()) + 1,
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

		bs, err := state.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
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

		req := &v1.AttesterDutiesRequest{
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
		currentEpoch := helpers.SlotToEpoch(bs.Slot())
		req := &v1.AttesterDutiesRequest{
			Epoch: currentEpoch + 2,
			Index: []types.ValidatorIndex{0},
		}
		_, err := vs.GetAttesterDuties(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, fmt.Sprintf("Request epoch %d can not be greater than next epoch %d", currentEpoch+2, currentEpoch+1), err)
	})

	t.Run("Validator index out of bound", func(t *testing.T) {
		req := &v1.AttesterDutiesRequest{
			Epoch: 0,
			Index: []types.ValidatorIndex{types.ValidatorIndex(len(pubKeys))},
		}
		_, err := vs.GetAttesterDuties(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Invalid index", err)
	})
}

func TestGetAttesterDuties_SyncNotReady(t *testing.T) {
	vs := &Server{
		SyncChecker: &mockSync.Sync{IsSyncing: true},
	}
	_, err := vs.GetAttesterDuties(context.Background(), &v1.AttesterDutiesRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestGetProposerDuties(t *testing.T) {
	ctx := context.Background()
	genesis := testutil.NewBeaconBlock()
	depChainStart := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, err := testutil.DeterministicDepositsAndKeys(depChainStart)
	require.NoError(t, err)
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bs, err := state.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
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
		req := &v1.ProposerDutiesRequest{
			Epoch: 0,
		}
		resp, err := vs.GetProposerDuties(ctx, req)
		require.NoError(t, err)
		assert.DeepEqual(t, genesisRoot[:], resp.DependentRoot)
		assert.Equal(t, 31, len(resp.Data))
		// We expect a proposer duty for slot 11.
		var expectedDuty *v1.ProposerDuty
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

		bs, err := state.GenesisBeaconState(context.Background(), deposits, 0, eth1Data)
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

		req := &v1.ProposerDutiesRequest{
			Epoch: 2,
		}
		resp, err := vs.GetProposerDuties(ctx, req)
		require.NoError(t, err)
		assert.DeepEqual(t, bs.BlockRoots()[31], resp.DependentRoot)
		assert.Equal(t, 32, len(resp.Data))
		// We expect a proposer duty for slot 74.
		var expectedDuty *v1.ProposerDuty
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
		currentEpoch := helpers.SlotToEpoch(bs.Slot())
		req := &v1.ProposerDutiesRequest{
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
	_, err := vs.GetProposerDuties(context.Background(), &v1.ProposerDutiesRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestProduceBlock(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 64)

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

	proposerSlashings := make([]*ethpb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
	for i := types.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
		proposerSlashing, err := testutil.GenerateProposerSlashingForValidator(
			beaconState,
			privKeys[i],
			i, /* validator index */
		)
		require.NoError(t, err)
		proposerSlashings[i] = proposerSlashing
		err = v1Alpha1Server.SlashingsPool.InsertProposerSlashing(context.Background(), beaconState, proposerSlashing)
		require.NoError(t, err)
	}

	attSlashings := make([]*ethpb.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
	for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings; i++ {
		attesterSlashing, err := testutil.GenerateAttesterSlashingForValidator(
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
	randaoReveal, err := testutil.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)
	graffiti := bytesutil.ToBytes32([]byte("eth2"))
	req := &v1.ProduceBlockRequest{
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
	expectedPropSlashings := make([]*v1.ProposerSlashing, len(proposerSlashings))
	for i, slash := range proposerSlashings {
		expectedPropSlashings[i] = migration.V1Alpha1ProposerSlashingToV1(slash)
	}
	assert.DeepEqual(t, expectedPropSlashings, resp.Data.Body.ProposerSlashings)
	assert.Equal(t, params.BeaconConfig().MaxAttesterSlashings, uint64(len(resp.Data.Body.AttesterSlashings)))
	expectedAttSlashings := make([]*v1.AttesterSlashing, len(attSlashings))
	for i, slash := range attSlashings {
		expectedAttSlashings[i] = migration.V1Alpha1AttSlashingToV1(slash)
	}
	assert.DeepEqual(t, expectedAttSlashings, resp.Data.Body.AttesterSlashings)
}

func TestProduceAttestationData(t *testing.T) {
	block := testutil.NewBeaconBlock()
	block.Block.Slot = 3*params.BeaconConfig().SlotsPerEpoch + 1
	targetBlock := testutil.NewBeaconBlock()
	targetBlock.Block.Slot = 1 * params.BeaconConfig().SlotsPerEpoch
	justifiedBlock := testutil.NewBeaconBlock()
	justifiedBlock.Block.Slot = 2 * params.BeaconConfig().SlotsPerEpoch
	blockRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not hash beacon block")
	justifiedRoot, err := justifiedBlock.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root for justified block")
	targetRoot, err := targetBlock.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root for target block")
	slot := 3*params.BeaconConfig().SlotsPerEpoch + 1
	beaconState, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(slot))
	err = beaconState.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{
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
		P2P:              &mockp2p.MockBroadcaster{},
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

	req := &v1.ProduceAttestationDataRequest{
		CommitteeIndex: 0,
		Slot:           3*params.BeaconConfig().SlotsPerEpoch + 1,
	}
	res, err := v1Server.ProduceAttestationData(context.Background(), req)
	require.NoError(t, err, "Could not get attestation info at slot")

	expectedInfo := &v1.AttestationData{
		Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
		BeaconBlockRoot: blockRoot[:],
		Source: &v1.Checkpoint{
			Epoch: 2,
			Root:  justifiedRoot[:],
		},
		Target: &v1.Checkpoint{
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
	attSlot1 := &ethpb.Attestation{
		AggregationBits: []byte{0, 1},
		Data: &ethpb.AttestationData{
			Slot:            1,
			CommitteeIndex:  1,
			BeaconBlockRoot: root1,
			Source: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  root1,
			},
			Target: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  root1,
			},
		},
		Signature: sig1,
	}
	root2_1 := bytesutil.PadTo([]byte("root2_1"), 32)
	sig2_1 := bytesutil.PadTo([]byte("sig2_1"), 96)
	attSlot2_1 := &ethpb.Attestation{
		AggregationBits: []byte{0, 1, 1},
		Data: &ethpb.AttestationData{
			Slot:            2,
			CommitteeIndex:  2,
			BeaconBlockRoot: root2_1,
			Source: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  root2_1,
			},
			Target: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  root2_1,
			},
		},
		Signature: sig2_1,
	}
	root2_2 := bytesutil.PadTo([]byte("root2_2"), 32)
	sig2_2 := bytesutil.PadTo([]byte("sig2_2"), 96)
	attSlot2_2 := &ethpb.Attestation{
		AggregationBits: []byte{0, 1, 1, 1},
		Data: &ethpb.AttestationData{
			Slot:            2,
			CommitteeIndex:  3,
			BeaconBlockRoot: root2_2,
			Source: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  root2_2,
			},
			Target: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  root2_2,
			},
		},
		Signature: sig2_2,
	}
	vs := &Server{
		AttestationsPool: &attestations.PoolMock{AggregatedAtts: []*ethpb.Attestation{attSlot1, attSlot2_1, attSlot2_2}},
	}

	t.Run("OK", func(t *testing.T) {
		reqRoot, err := attSlot2_2.Data.HashTreeRoot()
		require.NoError(t, err)
		req := &v1.AggregateAttestationRequest{
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
		req := &v1.AggregateAttestationRequest{
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
	att1 := &ethpb.Attestation{
		AggregationBits: []byte{0, 1},
		Data: &ethpb.AttestationData{
			Slot:            1,
			CommitteeIndex:  1,
			BeaconBlockRoot: root,
			Source: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
			Target: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
		},
		Signature: sig,
	}
	att2 := &ethpb.Attestation{
		AggregationBits: []byte{0, 1, 1},
		Data: &ethpb.AttestationData{
			Slot:            1,
			CommitteeIndex:  1,
			BeaconBlockRoot: root,
			Source: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
			Target: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
		},
		Signature: sig,
	}
	vs := &Server{
		AttestationsPool: &attestations.PoolMock{AggregatedAtts: []*ethpb.Attestation{att1, att2}},
	}

	reqRoot, err := att1.Data.HashTreeRoot()
	require.NoError(t, err)
	req := &v1.AggregateAttestationRequest{
		AttestationDataRoot: reqRoot[:],
		Slot:                1,
	}
	att, err := vs.GetAggregateAttestation(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, att)
	require.NotNil(t, att.Data)
	assert.DeepEqual(t, bitfield.Bitlist{0, 1, 1}, att.Data.AggregationBits)
}
