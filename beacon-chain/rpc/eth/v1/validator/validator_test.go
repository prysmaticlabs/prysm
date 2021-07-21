package validator

import (
	"context"
	"fmt"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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
		AggregationBits: []byte{0, 2},
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
		AggregationBits: []byte{0, 3},
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
		assert.DeepEqual(t, bitfield.Bitlist{0, 3}, att.Data.AggregationBits)
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
