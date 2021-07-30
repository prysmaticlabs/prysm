package beacon

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/statefetcher"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpb_alpha "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	sharedtestutil "github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestGetValidator(t *testing.T) {
	ctx := context.Background()

	var state state.BeaconState
	state, _ = sharedtestutil.DeterministicGenesisState(t, 8192)

	t.Run("Head Get Validator by index", func(t *testing.T) {
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
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
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
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

	t.Run("Validator ID required", func(t *testing.T) {
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
			},
		}
		_, err := s.GetValidator(ctx, &ethpb.StateValidatorRequest{
			StateId: []byte("head"),
		})
		require.ErrorContains(t, "Validator ID is required", err)
	})
}

func TestListValidators(t *testing.T) {
	ctx := context.Background()

	var state state.BeaconState
	state, _ = sharedtestutil.DeterministicGenesisState(t, 8192)

	t.Run("Head List All Validators", func(t *testing.T) {
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
			},
		}

		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
		})
		require.NoError(t, err)
		assert.Equal(t, len(resp.Data), 8192)
		for _, val := range resp.Data {
			assert.Equal(t, ethpb.ValidatorStatus_ACTIVE_ONGOING, val.Status)
		}
	})

	t.Run("Head List Validators by index", func(t *testing.T) {
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
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
			assert.Equal(t, ethpb.ValidatorStatus_ACTIVE_ONGOING, val.Status)
		}
	})

	t.Run("Head List Validators by pubkey", func(t *testing.T) {
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
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
			assert.Equal(t, ethpb.ValidatorStatus_ACTIVE_ONGOING, val.Status)
		}
	})

	t.Run("Head List Validators by both index and pubkey", func(t *testing.T) {
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
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
			assert.Equal(t, ethpb.ValidatorStatus_ACTIVE_ONGOING, val.Status)
		}
	})

	t.Run("Unknown public key is ignored", func(t *testing.T) {
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
			},
		}

		existingKey := state.PubkeyAtIndex(types.ValidatorIndex(1))
		pubkeys := [][]byte{existingKey[:], []byte(strings.Repeat("f", 48))}
		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Id:      pubkeys,
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Data))
		assert.Equal(t, types.ValidatorIndex(1), resp.Data[0].Index)
	})

	t.Run("Unknown index is ignored", func(t *testing.T) {
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
			},
		}

		ids := [][]byte{[]byte("1"), []byte("99999")}
		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Id:      ids,
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Data))
		assert.Equal(t, types.ValidatorIndex(1), resp.Data[0].Index)
	})
}

func TestListValidators_Status(t *testing.T) {
	ctx := context.Background()

	var state state.BeaconState
	state, _ = sharedtestutil.DeterministicGenesisState(t, 8192)

	farFutureEpoch := params.BeaconConfig().FarFutureEpoch
	validators := []*ethpb_alpha.Validator{
		// Pending initialized.
		{
			ActivationEpoch:            farFutureEpoch,
			ActivationEligibilityEpoch: farFutureEpoch,
		},
		// Pending queued.
		{
			ActivationEpoch:            10,
			ActivationEligibilityEpoch: 4,
		},
		// Active ongoing.
		{
			ActivationEpoch: 0,
			ExitEpoch:       farFutureEpoch,
		},
		// Active slashed.
		{
			ActivationEpoch: 0,
			ExitEpoch:       30,
			Slashed:         true,
		},
		// Active exiting.
		{
			ActivationEpoch: 3,
			ExitEpoch:       30,
			Slashed:         false,
		},
		// Exit slashed (at epoch 35).
		{
			ActivationEpoch:   3,
			ExitEpoch:         30,
			WithdrawableEpoch: 40,
			Slashed:           true,
		},
		// Exit unslashed (at epoch 35).
		{
			ActivationEpoch:   3,
			ExitEpoch:         30,
			WithdrawableEpoch: 40,
			Slashed:           false,
		},
		// Withdrawable (at epoch 45).
		{
			ActivationEpoch:   3,
			ExitEpoch:         30,
			WithdrawableEpoch: 40,
			EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
			Slashed:           false,
		},
		// Withdrawal done (at epoch 45).
		{
			ActivationEpoch:   3,
			ExitEpoch:         30,
			WithdrawableEpoch: 40,
			EffectiveBalance:  0,
			Slashed:           false,
		},
	}
	for _, validator := range validators {
		require.NoError(t, state.AppendValidator(validator))
		require.NoError(t, state.AppendBalance(params.BeaconConfig().MaxEffectiveBalance))
	}

	t.Run("Head List All ACTIVE Validators", func(t *testing.T) {
		s := Server{
			StateFetcher: &statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Status:  []ethpb.ValidatorStatus{ethpb.ValidatorStatus_ACTIVE},
		})
		require.NoError(t, err)
		assert.Equal(t, len(resp.Data), 8192+2 /* 2 active */)
		for _, datum := range resp.Data {
			status, err := validatorStatus(datum.Validator, 0)
			require.NoError(t, err)
			require.Equal(
				t,
				true,
				status == ethpb.ValidatorStatus_ACTIVE,
			)
			require.Equal(
				t,
				true,
				datum.Status == ethpb.ValidatorStatus_ACTIVE_ONGOING ||
					datum.Status == ethpb.ValidatorStatus_ACTIVE_EXITING ||
					datum.Status == ethpb.ValidatorStatus_ACTIVE_SLASHED,
			)
		}
	})

	t.Run("Head List All ACTIVE_ONGOING Validators", func(t *testing.T) {
		s := Server{
			StateFetcher: &statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Status:  []ethpb.ValidatorStatus{ethpb.ValidatorStatus_ACTIVE_ONGOING},
		})
		require.NoError(t, err)
		assert.Equal(t, len(resp.Data), 8192+1 /* 1 active_ongoing */)
		for _, datum := range resp.Data {
			status, err := validatorSubStatus(datum.Validator, 0)
			require.NoError(t, err)
			require.Equal(
				t,
				true,
				status == ethpb.ValidatorStatus_ACTIVE_ONGOING,
			)
			require.Equal(
				t,
				true,
				datum.Status == ethpb.ValidatorStatus_ACTIVE_ONGOING,
			)
		}
	})

	require.NoError(t, state.SetSlot(params.BeaconConfig().SlotsPerEpoch*35))
	t.Run("Head List All EXITED Validators", func(t *testing.T) {
		s := Server{
			StateFetcher: &statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Status:  []ethpb.ValidatorStatus{ethpb.ValidatorStatus_EXITED},
		})
		require.NoError(t, err)
		assert.Equal(t, 4 /* 4 exited */, len(resp.Data))
		for _, datum := range resp.Data {
			status, err := validatorStatus(datum.Validator, 35)
			require.NoError(t, err)
			require.Equal(
				t,
				true,
				status == ethpb.ValidatorStatus_EXITED,
			)
			require.Equal(
				t,
				true,
				datum.Status == ethpb.ValidatorStatus_EXITED_UNSLASHED || datum.Status == ethpb.ValidatorStatus_EXITED_SLASHED,
			)
		}
	})

	t.Run("Head List All PENDING_INITIALIZED and EXITED_UNSLASHED Validators", func(t *testing.T) {
		s := Server{
			StateFetcher: &statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Status:  []ethpb.ValidatorStatus{ethpb.ValidatorStatus_PENDING_INITIALIZED, ethpb.ValidatorStatus_EXITED_UNSLASHED},
		})
		require.NoError(t, err)
		assert.Equal(t, 4 /* 4 exited */, len(resp.Data))
		for _, datum := range resp.Data {
			status, err := validatorSubStatus(datum.Validator, 35)
			require.NoError(t, err)
			require.Equal(
				t,
				true,
				status == ethpb.ValidatorStatus_PENDING_INITIALIZED || status == ethpb.ValidatorStatus_EXITED_UNSLASHED,
			)
			require.Equal(
				t,
				true,
				datum.Status == ethpb.ValidatorStatus_PENDING_INITIALIZED || datum.Status == ethpb.ValidatorStatus_EXITED_UNSLASHED,
			)
		}
	})

	t.Run("Head List All PENDING and EXITED Validators", func(t *testing.T) {
		s := Server{
			StateFetcher: &statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Status:  []ethpb.ValidatorStatus{ethpb.ValidatorStatus_PENDING, ethpb.ValidatorStatus_EXITED_SLASHED},
		})
		require.NoError(t, err)
		assert.Equal(t, 2 /* 1 pending, 1 exited */, len(resp.Data))
		for _, datum := range resp.Data {
			status, err := validatorStatus(datum.Validator, 35)
			require.NoError(t, err)
			subStatus, err := validatorSubStatus(datum.Validator, 35)
			require.NoError(t, err)
			require.Equal(
				t,
				true,
				status == ethpb.ValidatorStatus_PENDING || subStatus == ethpb.ValidatorStatus_EXITED_SLASHED,
			)
			require.Equal(
				t,
				true,
				datum.Status == ethpb.ValidatorStatus_PENDING_INITIALIZED || datum.Status == ethpb.ValidatorStatus_EXITED_SLASHED,
			)
		}
	})
}
func TestListValidatorBalances(t *testing.T) {
	ctx := context.Background()

	var state state.BeaconState
	state, _ = sharedtestutil.DeterministicGenesisState(t, 8192)

	t.Run("Head List Validators Balance by index", func(t *testing.T) {
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
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
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
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
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
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
}

func TestListCommittees(t *testing.T) {
	ctx := context.Background()

	var state state.BeaconState
	state, _ = sharedtestutil.DeterministicGenesisState(t, 8192)
	epoch := helpers.SlotToEpoch(state.Slot())

	t.Run("Head All Committees", func(t *testing.T) {
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
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
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
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
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
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
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
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
			StateFetcher: &testutil.MockFetcher{
				BeaconState: state,
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
}

func Test_validatorStatus(t *testing.T) {
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch

	type args struct {
		validator *ethpb.Validator
		epoch     types.Epoch
	}
	tests := []struct {
		name    string
		args    args
		want    ethpb.ValidatorStatus
		wantErr bool
	}{
		{
			name: "pending initialized",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:            farFutureEpoch,
					ActivationEligibilityEpoch: farFutureEpoch,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_PENDING,
		},
		{
			name: "pending queued",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:            10,
					ActivationEligibilityEpoch: 2,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_PENDING,
		},
		{
			name: "active ongoing",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       farFutureEpoch,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_ACTIVE,
		},
		{
			name: "active slashed",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       30,
					Slashed:         true,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_ACTIVE,
		},
		{
			name: "active exiting",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       30,
					Slashed:         false,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_ACTIVE,
		},
		{
			name: "exited slashed",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					Slashed:           true,
				},
				epoch: types.Epoch(35),
			},
			want: ethpb.ValidatorStatus_EXITED,
		},
		{
			name: "exited unslashed",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					Slashed:           false,
				},
				epoch: types.Epoch(35),
			},
			want: ethpb.ValidatorStatus_EXITED,
		},
		{
			name: "withdrawal possible",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
					Slashed:           false,
				},
				epoch: types.Epoch(45),
			},
			want: ethpb.ValidatorStatus_WITHDRAWAL,
		},
		{
			name: "withdrawal done",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					EffectiveBalance:  0,
					Slashed:           false,
				},
				epoch: types.Epoch(45),
			},
			want: ethpb.ValidatorStatus_WITHDRAWAL,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validatorStatus(tt.args.validator, tt.args.epoch)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("validatorStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validatorSubStatus(t *testing.T) {
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch

	type args struct {
		validator *ethpb.Validator
		epoch     types.Epoch
	}
	tests := []struct {
		name    string
		args    args
		want    ethpb.ValidatorStatus
		wantErr bool
	}{
		{
			name: "pending initialized",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:            farFutureEpoch,
					ActivationEligibilityEpoch: farFutureEpoch,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_PENDING_INITIALIZED,
		},
		{
			name: "pending queued",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:            10,
					ActivationEligibilityEpoch: 2,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_PENDING_QUEUED,
		},
		{
			name: "active ongoing",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       farFutureEpoch,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_ACTIVE_ONGOING,
		},
		{
			name: "active slashed",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       30,
					Slashed:         true,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_ACTIVE_SLASHED,
		},
		{
			name: "active exiting",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       30,
					Slashed:         false,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_ACTIVE_EXITING,
		},
		{
			name: "exited slashed",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					Slashed:           true,
				},
				epoch: types.Epoch(35),
			},
			want: ethpb.ValidatorStatus_EXITED_SLASHED,
		},
		{
			name: "exited unslashed",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					Slashed:           false,
				},
				epoch: types.Epoch(35),
			},
			want: ethpb.ValidatorStatus_EXITED_UNSLASHED,
		},
		{
			name: "withdrawal possible",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
					Slashed:           false,
				},
				epoch: types.Epoch(45),
			},
			want: ethpb.ValidatorStatus_WITHDRAWAL_POSSIBLE,
		},
		{
			name: "withdrawal done",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					EffectiveBalance:  0,
					Slashed:           false,
				},
				epoch: types.Epoch(45),
			},
			want: ethpb.ValidatorStatus_WITHDRAWAL_DONE,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validatorSubStatus(tt.args.validator, tt.args.epoch)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("validatorSubStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// This test verifies how many validator statuses have meaningful values.
// The first expected non-meaningful value will have x.String() equal to its numeric representation.
// This test assumes we start numbering from 0 and do not skip any values.
// Having a test like this allows us to use e.g. `if value < 10` for validity checks.
func TestNumberOfStatuses(t *testing.T) {
	lastValidEnumValue := 12
	x := ethpb.ValidatorStatus(lastValidEnumValue)
	assert.NotEqual(t, strconv.Itoa(lastValidEnumValue), x.String())
	x = ethpb.ValidatorStatus(lastValidEnumValue + 1)
	assert.Equal(t, strconv.Itoa(lastValidEnumValue+1), x.String())
}
