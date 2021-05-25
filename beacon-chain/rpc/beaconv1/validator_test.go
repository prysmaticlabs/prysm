package beaconv1

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/statefetcher"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestGetValidator(t *testing.T) {
	ctx := context.Background()

	var state iface.BeaconState
	state, _ = testutil.DeterministicGenesisState(t, 8192)

	t.Run("Head Get Validator by index", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		resp, err := s.GetValidator(ctx, &ethpb.StateValidatorRequest{
			StateId:     []byte("head"),
			ValidatorId: []byte("15"),
		})
		require.NoError(t, err)
		assert.Equal(t, types.ValidatorIndex(15), resp.Data.Index)
	})

	t.Run("Head Get Validator by pubkey", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		pubKey := state.PubkeyAtIndex(types.ValidatorIndex(20))
		resp, err := s.GetValidator(ctx, &ethpb.StateValidatorRequest{
			StateId:     []byte("head"),
			ValidatorId: pubKey[:],
		})
		require.NoError(t, err)
		assert.Equal(t, types.ValidatorIndex(20), resp.Data.Index)
		assert.Equal(t, true, bytes.Equal(pubKey[:], resp.Data.Validator.Pubkey))
	})

	t.Run("Hex root not found", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}
		stateId, err := hexutil.Decode("0x" + strings.Repeat("f", 64))
		require.NoError(t, err)
		_, err = s.GetValidator(ctx, &ethpb.StateValidatorRequest{
			StateId: stateId,
		})
		require.ErrorContains(t, "state not found in the last 8192 state roots in head state", err)
	})

	t.Run("Invalid state ID", func(t *testing.T) {
		s := Server{}
		pubKey := state.PubkeyAtIndex(types.ValidatorIndex(20))
		_, err := s.GetValidator(ctx, &ethpb.StateValidatorRequest{
			StateId:     []byte("foo"),
			ValidatorId: pubKey[:],
		})
		require.ErrorContains(t, "invalid state ID: foo", err)
	})

	t.Run("Validator ID required", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}
		_, err := s.GetValidator(ctx, &ethpb.StateValidatorRequest{
			StateId: []byte("head"),
		})
		require.ErrorContains(t, "Must request a validator id", err)
	})
}

func TestListValidators(t *testing.T) {
	ctx := context.Background()

	var state iface.BeaconState
	state, _ = testutil.DeterministicGenesisState(t, 8192)

	t.Run("Head List All Validators", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
		})
		require.NoError(t, err)
		assert.Equal(t, len(resp.Data), 8192)
	})

	t.Run("Head List Validators by index", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		ids := [][]byte{[]byte("15"), []byte("26"), []byte("400")}
		idNums := []types.ValidatorIndex{15, 26, 400}
		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Id:      ids,
		})
		require.NoError(t, err)
		for i, val := range resp.Data {
			assert.Equal(t, idNums[i], val.Index)
		}
	})

	t.Run("Head List Validators by pubkey", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}
		idNums := []types.ValidatorIndex{20, 66, 90, 100}
		pubkey1 := state.PubkeyAtIndex(types.ValidatorIndex(20))
		pubkey2 := state.PubkeyAtIndex(types.ValidatorIndex(66))
		pubkey3 := state.PubkeyAtIndex(types.ValidatorIndex(90))
		pubkey4 := state.PubkeyAtIndex(types.ValidatorIndex(100))
		pubKeys := [][]byte{pubkey1[:], pubkey2[:], pubkey3[:], pubkey4[:]}
		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Id:      pubKeys,
		})
		require.NoError(t, err)
		for i, val := range resp.Data {
			assert.Equal(t, idNums[i], val.Index)
			assert.Equal(t, true, bytes.Equal(pubKeys[i], val.Validator.Pubkey))
		}
	})

	t.Run("Head List Validators by both index and pubkey", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		idNums := []types.ValidatorIndex{20, 90, 170, 129}
		pubkey1 := state.PubkeyAtIndex(types.ValidatorIndex(20))
		pubkey2 := state.PubkeyAtIndex(types.ValidatorIndex(90))
		pubkey3 := state.PubkeyAtIndex(types.ValidatorIndex(170))
		pubkey4 := state.PubkeyAtIndex(types.ValidatorIndex(129))
		pubkeys := [][]byte{pubkey1[:], pubkey2[:], pubkey3[:], pubkey4[:]}
		ids := [][]byte{pubkey1[:], []byte("90"), pubkey3[:], []byte("129")}
		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Id:      ids,
		})
		require.NoError(t, err)
		for i, val := range resp.Data {
			assert.Equal(t, idNums[i], val.Index)
			assert.Equal(t, true, bytes.Equal(pubkeys[i], val.Validator.Pubkey))
		}
	})

	t.Run("Hex root not found", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}
		stateId, err := hexutil.Decode("0x" + strings.Repeat("f", 64))
		require.NoError(t, err)
		_, err = s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: stateId,
		})
		require.ErrorContains(t, "state not found in the last 8192 state roots in head state", err)
	})

	t.Run("Invalid state ID", func(t *testing.T) {
		s := Server{}
		_, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("foo"),
		})
		require.ErrorContains(t, "invalid state ID: foo", err)
	})
}

func TestListValidatorBalances(t *testing.T) {
	ctx := context.Background()

	var state iface.BeaconState
	state, _ = testutil.DeterministicGenesisState(t, 8192)

	t.Run("Head List Validators Balance by index", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		ids := [][]byte{[]byte("15"), []byte("26"), []byte("400")}
		idNums := []types.ValidatorIndex{15, 26, 400}
		resp, err := s.ListValidatorBalances(ctx, &ethpb.ValidatorBalancesRequest{
			StateId: []byte("head"),
			Id:      ids,
		})
		require.NoError(t, err)
		for i, val := range resp.Data {
			assert.Equal(t, idNums[i], val.Index)
			assert.Equal(t, params.BeaconConfig().MaxEffectiveBalance, val.Balance)
		}
	})

	t.Run("Head List Validators Balance by pubkey", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}
		idNums := []types.ValidatorIndex{20, 66, 90, 100}
		pubkey1 := state.PubkeyAtIndex(types.ValidatorIndex(20))
		pubkey2 := state.PubkeyAtIndex(types.ValidatorIndex(66))
		pubkey3 := state.PubkeyAtIndex(types.ValidatorIndex(90))
		pubkey4 := state.PubkeyAtIndex(types.ValidatorIndex(100))
		pubKeys := [][]byte{pubkey1[:], pubkey2[:], pubkey3[:], pubkey4[:]}
		resp, err := s.ListValidatorBalances(ctx, &ethpb.ValidatorBalancesRequest{
			StateId: []byte("head"),
			Id:      pubKeys,
		})
		require.NoError(t, err)
		for i, val := range resp.Data {
			assert.Equal(t, idNums[i], val.Index)
			assert.Equal(t, params.BeaconConfig().MaxEffectiveBalance, val.Balance)
		}
	})

	t.Run("Head List Validators Balance by both index and pubkey", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		idNums := []types.ValidatorIndex{20, 90, 170, 129}
		pubkey1 := state.PubkeyAtIndex(types.ValidatorIndex(20))
		pubkey3 := state.PubkeyAtIndex(types.ValidatorIndex(170))
		ids := [][]byte{pubkey1[:], []byte("90"), pubkey3[:], []byte("129")}
		resp, err := s.ListValidatorBalances(ctx, &ethpb.ValidatorBalancesRequest{
			StateId: []byte("head"),
			Id:      ids,
		})
		require.NoError(t, err)
		for i, val := range resp.Data {
			assert.Equal(t, idNums[i], val.Index)
			assert.Equal(t, params.BeaconConfig().MaxEffectiveBalance, val.Balance)
		}
	})

	t.Run("Hex root not found", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}
		stateId, err := hexutil.Decode("0x" + strings.Repeat("f", 64))
		require.NoError(t, err)
		_, err = s.ListValidatorBalances(ctx, &ethpb.ValidatorBalancesRequest{
			StateId: stateId,
		})
		require.ErrorContains(t, "state not found in the last 8192 state roots in head state", err)
	})

	t.Run("Invalid state ID", func(t *testing.T) {
		s := Server{}
		_, err := s.ListValidatorBalances(ctx, &ethpb.ValidatorBalancesRequest{
			StateId: []byte("foo"),
		})
		require.ErrorContains(t, "invalid state ID: foo", err)
	})
}

func TestListCommittees(t *testing.T) {
	ctx := context.Background()

	var state iface.BeaconState
	state, _ = testutil.DeterministicGenesisState(t, 8192)
	epoch := helpers.SlotToEpoch(state.Slot())

	t.Run("Head All Committees", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
		})
		require.NoError(t, err)
		assert.Equal(t, int(params.BeaconConfig().SlotsPerEpoch)*2, len(resp.Data))
		for _, datum := range resp.Data {
			assert.Equal(t, true, datum.Index == types.CommitteeIndex(0) || datum.Index == types.CommitteeIndex(1))
			assert.Equal(t, epoch, helpers.SlotToEpoch(datum.Slot))
		}
	})

	t.Run("Head All Committees of Epoch 10", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}
		epoch := types.Epoch(10)
		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
			Epoch:   &epoch,
		})
		require.NoError(t, err)
		for _, datum := range resp.Data {
			assert.Equal(t, true, datum.Slot >= 320 && datum.Slot <= 351)
		}
	})

	t.Run("Head All Committees of Slot 4", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		slot := types.Slot(4)
		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
			Slot:    &slot,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(resp.Data))
		index := types.CommitteeIndex(0)
		for _, datum := range resp.Data {
			assert.Equal(t, epoch, helpers.SlotToEpoch(datum.Slot))
			assert.Equal(t, slot, datum.Slot)
			assert.Equal(t, index, datum.Index)
			index++
		}
	})

	t.Run("Head All Committees of Index 1", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		index := types.CommitteeIndex(1)
		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
			Index:   &index,
		})
		require.NoError(t, err)
		assert.Equal(t, int(params.BeaconConfig().SlotsPerEpoch), len(resp.Data))
		slot := types.Slot(0)
		for _, datum := range resp.Data {
			assert.Equal(t, epoch, helpers.SlotToEpoch(datum.Slot))
			assert.Equal(t, slot, datum.Slot)
			assert.Equal(t, index, datum.Index)
			slot++
		}
	})

	t.Run("Head All Committees of Slot 2, Index 1", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		index := types.CommitteeIndex(1)
		slot := types.Slot(2)
		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
			Slot:    &slot,
			Index:   &index,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(resp.Data))
		for _, datum := range resp.Data {
			assert.Equal(t, epoch, helpers.SlotToEpoch(datum.Slot))
			assert.Equal(t, slot, datum.Slot)
			assert.Equal(t, index, datum.Index)
		}
	})

	t.Run("Hex root not found", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}
		stateId, err := hexutil.Decode("0x" + strings.Repeat("f", 64))
		require.NoError(t, err)
		_, err = s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: stateId,
		})
		require.ErrorContains(t, "state not found in the last 8192 state roots in head state", err)
	})

	t.Run("Invalid state ID", func(t *testing.T) {
		s := Server{}
		_, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("foo"),
		})
		require.ErrorContains(t, "invalid state ID: foo", err)
	})
}
