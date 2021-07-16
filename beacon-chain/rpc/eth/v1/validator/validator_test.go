package validator

import (
	"context"
	"fmt"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestGetDuties(t *testing.T) {
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

func TestGetDuties_SyncNotReady(t *testing.T) {
	vs := &Server{
		SyncChecker: &mockSync.Sync{IsSyncing: true},
	}
	_, err := vs.GetAttesterDuties(context.Background(), &v1.AttesterDutiesRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}
