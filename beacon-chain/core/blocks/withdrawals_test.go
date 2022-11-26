package blocks_test

import (
	"context"
	"math/rand"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	consensusblocks "github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls/common"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

func TestProcessBLSToExecutionChange(t *testing.T) {
	t.Run("happy case", func(t *testing.T) {
		priv, err := bls.RandKey()
		require.NoError(t, err)
		pubkey := priv.PublicKey().Marshal()

		message := &ethpb.BLSToExecutionChange{
			ToExecutionAddress: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13},
			ValidatorIndex:     0,
			FromBlsPubkey:      pubkey,
		}

		hashFn := ssz.NewHasherFunc(hash.CustomSHA256Hasher())
		digest := hashFn.Hash(pubkey)
		digest[0] = params.BeaconConfig().BLSWithdrawalPrefixByte

		registry := []*ethpb.Validator{
			{
				WithdrawalCredentials: digest[:],
			},
		}
		st, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Validators: registry,
			Fork: &ethpb.Fork{
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			},
			Slot: params.BeaconConfig().SlotsPerEpoch * 5,
		})
		require.NoError(t, err)

		signature, err := signing.ComputeDomainAndSign(st, time.CurrentEpoch(st), message, params.BeaconConfig().DomainBLSToExecutionChange, priv)
		require.NoError(t, err)

		signed := &ethpb.SignedBLSToExecutionChange{
			Message:   message,
			Signature: signature,
		}

		st, err = blocks.ProcessBLSToExecutionChange(st, signed)
		require.NoError(t, err)

		val, err := st.ValidatorAtIndex(0)
		require.NoError(t, err)

		require.DeepEqual(t, message.ToExecutionAddress, val.WithdrawalCredentials[12:])
	})
	t.Run("happy case only validation", func(t *testing.T) {
		priv, err := bls.RandKey()
		require.NoError(t, err)
		pubkey := priv.PublicKey().Marshal()

		message := &ethpb.BLSToExecutionChange{
			ToExecutionAddress: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13},
			ValidatorIndex:     0,
			FromBlsPubkey:      pubkey,
		}

		hashFn := ssz.NewHasherFunc(hash.CustomSHA256Hasher())
		digest := hashFn.Hash(pubkey)
		digest[0] = params.BeaconConfig().BLSWithdrawalPrefixByte

		registry := []*ethpb.Validator{
			{
				WithdrawalCredentials: digest[:],
			},
		}
		st, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Validators: registry,
			Fork: &ethpb.Fork{
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			},
			Slot: params.BeaconConfig().SlotsPerEpoch * 5,
		})
		require.NoError(t, err)

		signature, err := signing.ComputeDomainAndSign(st, time.CurrentEpoch(st), message, params.BeaconConfig().DomainBLSToExecutionChange, priv)
		require.NoError(t, err)

		signed := &ethpb.SignedBLSToExecutionChange{
			Message:   message,
			Signature: signature,
		}
		val, err := blocks.ValidateBLSToExecutionChange(st, signed)
		require.NoError(t, err)
		require.DeepEqual(t, digest[:], val.WithdrawalCredentials)
	})

	t.Run("non-existent validator", func(t *testing.T) {
		priv, err := bls.RandKey()
		require.NoError(t, err)
		pubkey := priv.PublicKey().Marshal()

		message := &ethpb.BLSToExecutionChange{
			ToExecutionAddress: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13},
			ValidatorIndex:     1,
			FromBlsPubkey:      pubkey,
		}

		hashFn := ssz.NewHasherFunc(hash.CustomSHA256Hasher())
		digest := hashFn.Hash(pubkey)
		digest[0] = params.BeaconConfig().BLSWithdrawalPrefixByte

		registry := []*ethpb.Validator{
			{
				WithdrawalCredentials: digest[:],
			},
		}
		st, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Validators: registry,
			Fork: &ethpb.Fork{
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			},
			Slot: params.BeaconConfig().SlotsPerEpoch * 5,
		})
		require.NoError(t, err)

		signature, err := signing.ComputeDomainAndSign(st, time.CurrentEpoch(st), message, params.BeaconConfig().DomainBLSToExecutionChange, priv)
		require.NoError(t, err)

		signed := &ethpb.SignedBLSToExecutionChange{
			Message:   message,
			Signature: signature,
		}

		_, err = blocks.ProcessBLSToExecutionChange(st, signed)
		require.ErrorContains(t, "out of range", err)
	})

	t.Run("signature does not verify", func(t *testing.T) {
		priv, err := bls.RandKey()
		require.NoError(t, err)
		pubkey := priv.PublicKey().Marshal()

		message := &ethpb.BLSToExecutionChange{
			ToExecutionAddress: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13},
			ValidatorIndex:     0,
			FromBlsPubkey:      pubkey,
		}

		registry := []*ethpb.Validator{
			{
				WithdrawalCredentials: params.BeaconConfig().ZeroHash[:],
			},
		}
		st, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Validators: registry,
			Fork: &ethpb.Fork{
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			},
			Slot: params.BeaconConfig().SlotsPerEpoch * 5,
		})
		require.NoError(t, err)

		signature, err := signing.ComputeDomainAndSign(st, time.CurrentEpoch(st), message, params.BeaconConfig().DomainBLSToExecutionChange, priv)
		require.NoError(t, err)

		signed := &ethpb.SignedBLSToExecutionChange{
			Message:   message,
			Signature: signature,
		}

		_, err = blocks.ProcessBLSToExecutionChange(st, signed)
		require.ErrorContains(t, "withdrawal credentials do not match", err)
	})

	t.Run("invalid BLS prefix", func(t *testing.T) {
		priv, err := bls.RandKey()
		require.NoError(t, err)
		pubkey := priv.PublicKey().Marshal()

		message := &ethpb.BLSToExecutionChange{
			ToExecutionAddress: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13},
			ValidatorIndex:     0,
			FromBlsPubkey:      pubkey,
		}
		hashFn := ssz.NewHasherFunc(hash.CustomSHA256Hasher())
		digest := hashFn.Hash(pubkey)
		digest[0] = params.BeaconConfig().BLSWithdrawalPrefixByte

		registry := []*ethpb.Validator{
			{
				WithdrawalCredentials: digest[:],
			},
		}
		registry[0].WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte

		st, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{
			Validators: registry,
			Fork: &ethpb.Fork{
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			},
			Slot: params.BeaconConfig().SlotsPerEpoch * 5,
		})
		require.NoError(t, err)

		signature, err := signing.ComputeDomainAndSign(st, time.CurrentEpoch(st), message, params.BeaconConfig().DomainBLSToExecutionChange, priv)
		require.NoError(t, err)

		signed := &ethpb.SignedBLSToExecutionChange{
			Message:   message,
			Signature: signature,
		}

		_, err = blocks.ProcessBLSToExecutionChange(st, signed)
		require.ErrorContains(t, "withdrawal credential prefix is not a BLS prefix", err)

	})
}

func TestProcessWithdrawals(t *testing.T) {
	const (
		currentEpoch             = types.Epoch(10)
		epochInFuture            = types.Epoch(12)
		epochInPast              = types.Epoch(8)
		numValidators            = 128
		notWithdrawableIndex     = 127
		notPartiallyWithdrawable = 126
	)
	maxEffectiveBalance := params.BeaconConfig().MaxEffectiveBalance

	type args struct {
		Name                         string
		NextWithdrawalValidatorIndex types.ValidatorIndex
		NextWithdrawalIndex          uint64
		FullWithdrawalIndices        []types.ValidatorIndex
		PartialWithdrawalIndices     []types.ValidatorIndex
		Withdrawals                  []*enginev1.Withdrawal
	}
	type control struct {
		NextWithdrawalValidatorIndex types.ValidatorIndex
		NextWithdrawalIndex          uint64
		ExpectedError                bool
		Balances                     map[uint64]uint64
	}
	type Test struct {
		Args    args
		Control control
	}
	executionAddress := func(i types.ValidatorIndex) []byte {
		wc := make([]byte, 20)
		wc[19] = byte(i)
		return wc
	}
	withdrawalAmount := func(i types.ValidatorIndex) uint64 {
		return maxEffectiveBalance + uint64(i)*100000
	}
	fullWithdrawal := func(i types.ValidatorIndex, idx uint64) *enginev1.Withdrawal {
		return &enginev1.Withdrawal{
			WithdrawalIndex:  idx,
			ValidatorIndex:   i,
			ExecutionAddress: executionAddress(i),
			Amount:           withdrawalAmount(i),
		}
	}
	partialWithdrawal := func(i types.ValidatorIndex, idx uint64) *enginev1.Withdrawal {
		return &enginev1.Withdrawal{
			WithdrawalIndex:  idx,
			ValidatorIndex:   i,
			ExecutionAddress: executionAddress(i),
			Amount:           withdrawalAmount(i) - maxEffectiveBalance,
		}
	}
	tests := []Test{
		{
			Args: args{
				Name:                         "success no withdrawals",
				NextWithdrawalValidatorIndex: 10,
				NextWithdrawalIndex:          3,
			},
			Control: control{
				NextWithdrawalValidatorIndex: 10,
				NextWithdrawalIndex:          3,
			},
		},
		{
			Args: args{
				Name:                         "success one full withdrawal",
				NextWithdrawalIndex:          3,
				NextWithdrawalValidatorIndex: 5,
				FullWithdrawalIndices:        []types.ValidatorIndex{1},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(1, 3),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 2,
				NextWithdrawalIndex:          4,
				Balances:                     map[uint64]uint64{1: 0},
			},
		},
		{
			Args: args{
				Name:                         "success one partial withdrawal",
				NextWithdrawalIndex:          21,
				NextWithdrawalValidatorIndex: 37,
				PartialWithdrawalIndices:     []types.ValidatorIndex{7},
				Withdrawals: []*enginev1.Withdrawal{
					partialWithdrawal(7, 21),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 8,
				NextWithdrawalIndex:          22,
				Balances:                     map[uint64]uint64{7: maxEffectiveBalance},
			},
		},
		{
			Args: args{
				Name:                         "success many full withdrawals",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []types.ValidatorIndex{7, 19, 28, 1},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(7, 22), fullWithdrawal(19, 23), fullWithdrawal(28, 24),
					fullWithdrawal(1, 25),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 2,
				NextWithdrawalIndex:          26,
				Balances:                     map[uint64]uint64{7: 0, 19: 0, 28: 0, 1: 0},
			},
		},
		{
			Args: args{
				Name:                         "success many partial withdrawals",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				PartialWithdrawalIndices:     []types.ValidatorIndex{7, 19, 28, 1},
				Withdrawals: []*enginev1.Withdrawal{
					partialWithdrawal(7, 22), partialWithdrawal(19, 23), partialWithdrawal(28, 24),
					partialWithdrawal(1, 25),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 2,
				NextWithdrawalIndex:          26,
				Balances: map[uint64]uint64{
					7:  maxEffectiveBalance,
					19: maxEffectiveBalance,
					28: maxEffectiveBalance,
					1:  maxEffectiveBalance,
				},
			},
		},
		{
			Args: args{
				Name:                         "success many withdrawals",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 12,
				FullWithdrawalIndices:        []types.ValidatorIndex{7, 19, 28},
				PartialWithdrawalIndices:     []types.ValidatorIndex{2, 1, 89, 15},
				Withdrawals: []*enginev1.Withdrawal{
					partialWithdrawal(15, 22), fullWithdrawal(19, 23), fullWithdrawal(28, 24),
					partialWithdrawal(89, 25), partialWithdrawal(1, 26), partialWithdrawal(2, 27),
					fullWithdrawal(7, 28),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 8,
				NextWithdrawalIndex:          29,
				Balances: map[uint64]uint64{
					7: 0, 19: 0, 28: 0,
					2: maxEffectiveBalance, 1: maxEffectiveBalance, 89: maxEffectiveBalance,
					15: maxEffectiveBalance,
				},
			},
		},
		{
			Args: args{
				Name:                         "success more than max fully withdrawals",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 0,
				FullWithdrawalIndices:        []types.ValidatorIndex{1, 2, 3, 4, 5, 6, 7, 8, 9, 21, 22, 23, 24, 25, 26, 27, 29, 35, 89},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(1, 22), fullWithdrawal(2, 23), fullWithdrawal(3, 24),
					fullWithdrawal(4, 25), fullWithdrawal(5, 26), fullWithdrawal(6, 27),
					fullWithdrawal(7, 28), fullWithdrawal(8, 29), fullWithdrawal(9, 30),
					fullWithdrawal(21, 31), fullWithdrawal(22, 32), fullWithdrawal(23, 33),
					fullWithdrawal(24, 34), fullWithdrawal(25, 35), fullWithdrawal(26, 36),
					fullWithdrawal(27, 37),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 28,
				NextWithdrawalIndex:          38,
				Balances: map[uint64]uint64{
					1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0, 7: 0, 8: 0, 9: 0,
					21: 0, 22: 0, 23: 0, 24: 0, 25: 0, 26: 0, 27: 0,
				},
			},
		},
		{
			Args: args{
				Name:                         "success more than max partially withdrawals",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 0,
				PartialWithdrawalIndices:     []types.ValidatorIndex{1, 2, 3, 4, 5, 6, 7, 8, 9, 21, 22, 23, 24, 25, 26, 27, 29, 35, 89},
				Withdrawals: []*enginev1.Withdrawal{
					partialWithdrawal(1, 22), partialWithdrawal(2, 23), partialWithdrawal(3, 24),
					partialWithdrawal(4, 25), partialWithdrawal(5, 26), partialWithdrawal(6, 27),
					partialWithdrawal(7, 28), partialWithdrawal(8, 29), partialWithdrawal(9, 30),
					partialWithdrawal(21, 31), partialWithdrawal(22, 32), partialWithdrawal(23, 33),
					partialWithdrawal(24, 34), partialWithdrawal(25, 35), partialWithdrawal(26, 36),
					partialWithdrawal(27, 37),
				},
			},
			Control: control{
				NextWithdrawalValidatorIndex: 28,
				NextWithdrawalIndex:          38,
				Balances: map[uint64]uint64{
					1:  maxEffectiveBalance,
					2:  maxEffectiveBalance,
					3:  maxEffectiveBalance,
					4:  maxEffectiveBalance,
					5:  maxEffectiveBalance,
					6:  maxEffectiveBalance,
					7:  maxEffectiveBalance,
					8:  maxEffectiveBalance,
					9:  maxEffectiveBalance,
					21: maxEffectiveBalance,
					22: maxEffectiveBalance,
					23: maxEffectiveBalance,
					24: maxEffectiveBalance,
					25: maxEffectiveBalance,
					26: maxEffectiveBalance,
					27: maxEffectiveBalance,
				},
			},
		},
		{
			Args: args{
				Name:                         "failure wrong number of partial withdrawal",
				NextWithdrawalIndex:          21,
				NextWithdrawalValidatorIndex: 37,
				PartialWithdrawalIndices:     []types.ValidatorIndex{7},
				Withdrawals: []*enginev1.Withdrawal{
					partialWithdrawal(7, 21), partialWithdrawal(9, 22),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
		{
			Args: args{
				Name:                         "failure invalid withdrawal index",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []types.ValidatorIndex{7, 19, 28, 1},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(7, 22), fullWithdrawal(19, 23), fullWithdrawal(28, 25),
					fullWithdrawal(1, 25),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
		{
			Args: args{
				Name:                         "failure invalid validator index",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []types.ValidatorIndex{7, 19, 28, 1},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(7, 22), fullWithdrawal(19, 23), fullWithdrawal(27, 24),
					fullWithdrawal(1, 25),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
		{
			Args: args{
				Name:                         "failure invalid withdrawal amount",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []types.ValidatorIndex{7, 19, 28, 1},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(7, 22), fullWithdrawal(19, 23), partialWithdrawal(28, 24),
					fullWithdrawal(1, 25),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
		{
			Args: args{
				Name:                         "failure validator not fully withdrawable",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				FullWithdrawalIndices:        []types.ValidatorIndex{notWithdrawableIndex},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(notWithdrawableIndex, 22),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
		{
			Args: args{
				Name:                         "failure validator not partially withdrawable",
				NextWithdrawalIndex:          22,
				NextWithdrawalValidatorIndex: 4,
				PartialWithdrawalIndices:     []types.ValidatorIndex{notPartiallyWithdrawable},
				Withdrawals: []*enginev1.Withdrawal{
					fullWithdrawal(notPartiallyWithdrawable, 22),
				},
			},
			Control: control{
				ExpectedError: true,
			},
		},
	}

	checkPostState := func(t *testing.T, expected control, st state.BeaconState) {
		l, err := st.LastWithdrawalValidatorIndex()
		require.NoError(t, err)
		require.Equal(t, expected.NextWithdrawalValidatorIndex, l)

		n, err := st.NextWithdrawalIndex()
		require.NoError(t, err)
		require.Equal(t, expected.NextWithdrawalIndex, n)
		balances := st.Balances()
		for idx, bal := range expected.Balances {
			require.Equal(t, bal, balances[idx])
		}
	}

	prepareValidators := func(st *ethpb.BeaconStateCapella, arguments args) (state.BeaconState, error) {
		validators := make([]*ethpb.Validator, numValidators)
		st.Balances = make([]uint64, numValidators)
		for i := range validators {
			v := &ethpb.Validator{}
			v.EffectiveBalance = maxEffectiveBalance
			v.WithdrawableEpoch = epochInFuture
			v.WithdrawalCredentials = make([]byte, 32)
			v.WithdrawalCredentials[31] = byte(i)
			st.Balances[i] = v.EffectiveBalance - uint64(rand.Intn(1000))
			validators[i] = v
		}
		for _, idx := range arguments.FullWithdrawalIndices {
			if idx != notWithdrawableIndex {
				validators[idx].WithdrawableEpoch = epochInPast
			}
			st.Balances[idx] = withdrawalAmount(idx)
			validators[idx].WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
		}
		for _, idx := range arguments.PartialWithdrawalIndices {
			validators[idx].WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
			st.Balances[idx] = withdrawalAmount(idx)
		}
		st.Validators = validators
		return state_native.InitializeFromProtoCapella(st)
	}

	for _, test := range tests {
		t.Run(test.Args.Name, func(t *testing.T) {
			if test.Args.Withdrawals == nil {
				test.Args.Withdrawals = make([]*enginev1.Withdrawal, 0)
			}
			if test.Args.FullWithdrawalIndices == nil {
				test.Args.FullWithdrawalIndices = make([]types.ValidatorIndex, 0)
			}
			if test.Args.PartialWithdrawalIndices == nil {
				test.Args.PartialWithdrawalIndices = make([]types.ValidatorIndex, 0)
			}
			slot, err := slots.EpochStart(currentEpoch)
			require.NoError(t, err)
			spb := &ethpb.BeaconStateCapella{
				Slot:                         slot,
				NextWithdrawalValidatorIndex: test.Args.NextWithdrawalValidatorIndex,
				NextWithdrawalIndex:          test.Args.NextWithdrawalIndex,
			}
			st, err := prepareValidators(spb, test.Args)
			require.NoError(t, err)
			post, err := blocks.ProcessWithdrawals(st, test.Args.Withdrawals)
			if test.Control.ExpectedError {
				require.NotNil(t, err)
			} else {
				require.NoError(t, err)
				checkPostState(t, test.Control, post)
			}
		})
	}
}

func TestProcessBLSToExecutionChanges(t *testing.T) {
	spb := &ethpb.BeaconStateCapella{
		Fork: &ethpb.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
	}
	numValidators := 10
	validators := make([]*ethpb.Validator, numValidators)
	blsChanges := make([]*ethpb.BLSToExecutionChange, numValidators)
	spb.Balances = make([]uint64, numValidators)
	privKeys := make([]common.SecretKey, numValidators)
	maxEffectiveBalance := params.BeaconConfig().MaxEffectiveBalance
	executionAddress := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13}

	for i := range validators {
		v := &ethpb.Validator{}
		v.EffectiveBalance = maxEffectiveBalance
		v.WithdrawableEpoch = params.BeaconConfig().FarFutureEpoch
		v.WithdrawalCredentials = make([]byte, 32)
		priv, err := bls.RandKey()
		require.NoError(t, err)
		privKeys[i] = priv
		pubkey := priv.PublicKey().Marshal()

		message := &ethpb.BLSToExecutionChange{
			ToExecutionAddress: executionAddress,
			ValidatorIndex:     types.ValidatorIndex(i),
			FromBlsPubkey:      pubkey,
		}

		hashFn := ssz.NewHasherFunc(hash.CustomSHA256Hasher())
		digest := hashFn.Hash(pubkey)
		digest[0] = params.BeaconConfig().BLSWithdrawalPrefixByte
		copy(v.WithdrawalCredentials, digest[:])
		validators[i] = v
		blsChanges[i] = message
	}
	spb.Validators = validators
	st, err := state_native.InitializeFromProtoCapella(spb)
	require.NoError(t, err)

	signedChanges := make([]*ethpb.SignedBLSToExecutionChange, numValidators)
	for i, message := range blsChanges {
		signature, err := signing.ComputeDomainAndSign(st, time.CurrentEpoch(st), message, params.BeaconConfig().DomainBLSToExecutionChange, privKeys[i])
		require.NoError(t, err)

		signed := &ethpb.SignedBLSToExecutionChange{
			Message:   message,
			Signature: signature,
		}
		signedChanges[i] = signed
	}

	body := &ethpb.BeaconBlockBodyCapella{
		BlsToExecutionChanges: signedChanges,
	}
	bpb := &ethpb.BeaconBlockCapella{
		Body: body,
	}
	sbpb := &ethpb.SignedBeaconBlockCapella{
		Block: bpb,
	}
	signed, err := consensusblocks.NewSignedBeaconBlock(sbpb)
	require.NoError(t, err)
	st, err = blocks.ProcessBLSToExecutionChanges(st, signed)
	require.NoError(t, err)
	vals := st.Validators()
	for _, val := range vals {
		require.DeepEqual(t, executionAddress, val.WithdrawalCredentials[12:])
		require.Equal(t, params.BeaconConfig().ETH1AddressWithdrawalPrefixByte, val.WithdrawalCredentials[0])
	}
}

func TestBLSChangesSignatureBatch(t *testing.T) {
	spb := &ethpb.BeaconStateCapella{
		Fork: &ethpb.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
	}
	numValidators := 10
	validators := make([]*ethpb.Validator, numValidators)
	blsChanges := make([]*ethpb.BLSToExecutionChange, numValidators)
	spb.Balances = make([]uint64, numValidators)
	privKeys := make([]common.SecretKey, numValidators)
	maxEffectiveBalance := params.BeaconConfig().MaxEffectiveBalance
	executionAddress := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13}

	for i := range validators {
		v := &ethpb.Validator{}
		v.EffectiveBalance = maxEffectiveBalance
		v.WithdrawableEpoch = params.BeaconConfig().FarFutureEpoch
		v.WithdrawalCredentials = make([]byte, 32)
		priv, err := bls.RandKey()
		require.NoError(t, err)
		privKeys[i] = priv
		pubkey := priv.PublicKey().Marshal()

		message := &ethpb.BLSToExecutionChange{
			ToExecutionAddress: executionAddress,
			ValidatorIndex:     types.ValidatorIndex(i),
			FromBlsPubkey:      pubkey,
		}

		hashFn := ssz.NewHasherFunc(hash.CustomSHA256Hasher())
		digest := hashFn.Hash(pubkey)
		digest[0] = params.BeaconConfig().BLSWithdrawalPrefixByte
		copy(v.WithdrawalCredentials, digest[:])
		validators[i] = v
		blsChanges[i] = message
	}
	spb.Validators = validators
	st, err := state_native.InitializeFromProtoCapella(spb)
	require.NoError(t, err)

	signedChanges := make([]*ethpb.SignedBLSToExecutionChange, numValidators)
	for i, message := range blsChanges {
		signature, err := signing.ComputeDomainAndSign(st, time.CurrentEpoch(st), message, params.BeaconConfig().DomainBLSToExecutionChange, privKeys[i])
		require.NoError(t, err)

		signed := &ethpb.SignedBLSToExecutionChange{
			Message:   message,
			Signature: signature,
		}
		signedChanges[i] = signed
	}
	batch, err := blocks.BLSChangesSignatureBatch(context.Background(), st, signedChanges)
	require.NoError(t, err)
	verify, err := batch.Verify()
	require.NoError(t, err)
	require.Equal(t, true, verify)
}
