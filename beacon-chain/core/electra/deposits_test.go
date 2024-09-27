package electra_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestProcessPendingBalanceDeposits(t *testing.T) {
	tests := []struct {
		name    string
		state   state.BeaconState
		wantErr bool
		check   func(*testing.T, state.BeaconState)
	}{
		{
			name:    "nil state fails",
			state:   nil,
			wantErr: true,
		},
		{
			name: "no deposits resets balance to consume",
			state: func() state.BeaconState {
				st := stateWithActiveBalanceETH(t, 1_000)
				require.NoError(t, st.SetDepositBalanceToConsume(100))
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				res, err := st.DepositBalanceToConsume()
				require.NoError(t, err)
				require.Equal(t, primitives.Gwei(0), res)
			},
		},
		{
			name: "more deposits than balance to consume processes partial deposits",
			state: func() state.BeaconState {
				st := stateWithActiveBalanceETH(t, 1_000)
				require.NoError(t, st.SetDepositBalanceToConsume(100))
				amountAvailForProcessing := helpers.ActivationExitChurnLimit(1_000 * 1e9)
				deps := make([]*eth.PendingBalanceDeposit, 20)
				for i := 0; i < len(deps); i += 1 {
					deps[i] = &eth.PendingBalanceDeposit{
						Amount: uint64(amountAvailForProcessing) / 10,
						Index:  primitives.ValidatorIndex(i),
					}
				}
				require.NoError(t, st.SetPendingBalanceDeposits(deps))
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				amountAvailForProcessing := helpers.ActivationExitChurnLimit(1_000 * 1e9)
				res, err := st.DepositBalanceToConsume()
				require.NoError(t, err)
				require.Equal(t, primitives.Gwei(100), res)
				// Validators 0..9 should have their balance increased
				for i := primitives.ValidatorIndex(0); i < 10; i++ {
					b, err := st.BalanceAtIndex(i)
					require.NoError(t, err)
					require.Equal(t, params.BeaconConfig().MinActivationBalance+uint64(amountAvailForProcessing)/10, b)
				}

				// Half of the balance deposits should have been processed.
				remaining, err := st.PendingBalanceDeposits()
				require.NoError(t, err)
				require.Equal(t, 10, len(remaining))
			},
		},
		{
			name: "less deposits than balance to consume processes all deposits",
			state: func() state.BeaconState {
				st := stateWithActiveBalanceETH(t, 1_000)
				require.NoError(t, st.SetDepositBalanceToConsume(0))
				amountAvailForProcessing := helpers.ActivationExitChurnLimit(1_000 * 1e9)
				deps := make([]*eth.PendingBalanceDeposit, 5)
				for i := 0; i < len(deps); i += 1 {
					deps[i] = &eth.PendingBalanceDeposit{
						Amount: uint64(amountAvailForProcessing) / 5,
						Index:  primitives.ValidatorIndex(i),
					}
				}
				require.NoError(t, st.SetPendingBalanceDeposits(deps))
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				amountAvailForProcessing := helpers.ActivationExitChurnLimit(1_000 * 1e9)
				res, err := st.DepositBalanceToConsume()
				require.NoError(t, err)
				require.Equal(t, primitives.Gwei(0), res)
				// Validators 0..4 should have their balance increased
				for i := primitives.ValidatorIndex(0); i < 4; i++ {
					b, err := st.BalanceAtIndex(i)
					require.NoError(t, err)
					require.Equal(t, params.BeaconConfig().MinActivationBalance+uint64(amountAvailForProcessing)/5, b)
				}

				// All of the balance deposits should have been processed.
				remaining, err := st.PendingBalanceDeposits()
				require.NoError(t, err)
				require.Equal(t, 0, len(remaining))
			},
		},
		{
			name: "exiting validator deposit postponed",
			state: func() state.BeaconState {
				st := stateWithActiveBalanceETH(t, 1_000)
				require.NoError(t, st.SetDepositBalanceToConsume(0))
				amountAvailForProcessing := helpers.ActivationExitChurnLimit(1_000 * 1e9)
				deps := make([]*eth.PendingBalanceDeposit, 5)
				for i := 0; i < len(deps); i += 1 {
					deps[i] = &eth.PendingBalanceDeposit{
						Amount: uint64(amountAvailForProcessing) / 5,
						Index:  primitives.ValidatorIndex(i),
					}
				}
				require.NoError(t, st.SetPendingBalanceDeposits(deps))
				v, err := st.ValidatorAtIndex(0)
				require.NoError(t, err)
				v.ExitEpoch = 10
				v.WithdrawableEpoch = 20
				require.NoError(t, st.UpdateValidatorAtIndex(0, v))
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				amountAvailForProcessing := helpers.ActivationExitChurnLimit(1_000 * 1e9)
				res, err := st.DepositBalanceToConsume()
				require.NoError(t, err)
				require.Equal(t, primitives.Gwei(0), res)
				// Validators 1..4 should have their balance increased
				for i := primitives.ValidatorIndex(1); i < 4; i++ {
					b, err := st.BalanceAtIndex(i)
					require.NoError(t, err)
					require.Equal(t, params.BeaconConfig().MinActivationBalance+uint64(amountAvailForProcessing)/5, b)
				}

				// All of the balance deposits should have been processed, except validator index 0 was
				// added back to the pending deposits queue.
				remaining, err := st.PendingBalanceDeposits()
				require.NoError(t, err)
				require.Equal(t, 1, len(remaining))
			},
		},
		{
			name: "exited validator balance increased",
			state: func() state.BeaconState {
				st := stateWithActiveBalanceETH(t, 1_000)
				deps := make([]*eth.PendingBalanceDeposit, 1)
				for i := 0; i < len(deps); i += 1 {
					deps[i] = &eth.PendingBalanceDeposit{
						Amount: 1_000_000,
						Index:  primitives.ValidatorIndex(i),
					}
				}
				require.NoError(t, st.SetPendingBalanceDeposits(deps))
				v, err := st.ValidatorAtIndex(0)
				require.NoError(t, err)
				v.ExitEpoch = 2
				v.WithdrawableEpoch = 8
				require.NoError(t, st.UpdateValidatorAtIndex(0, v))
				require.NoError(t, st.UpdateBalancesAtIndex(0, 100_000))
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				res, err := st.DepositBalanceToConsume()
				require.NoError(t, err)
				require.Equal(t, primitives.Gwei(0), res)
				b, err := st.BalanceAtIndex(0)
				require.NoError(t, err)
				require.Equal(t, uint64(1_100_000), b)

				// All of the balance deposits should have been processed.
				remaining, err := st.PendingBalanceDeposits()
				require.NoError(t, err)
				require.Equal(t, 0, len(remaining))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tab uint64
			var err error
			if tt.state != nil {
				// The caller of this method would normally have the precompute balance values for total
				// active balance for this epoch. For ease of test setup, we will compute total active
				// balance from the given state.
				tab, err = helpers.TotalActiveBalance(tt.state)
			}
			require.NoError(t, err)
			err = electra.ProcessPendingBalanceDeposits(context.TODO(), tt.state, primitives.Gwei(tab))
			require.Equal(t, tt.wantErr, err != nil, "wantErr=%v, got err=%s", tt.wantErr, err)
			if tt.check != nil {
				tt.check(t, tt.state)
			}
		})
	}
}

func TestProcessDepositRequests(t *testing.T) {
	st, _ := util.DeterministicGenesisStateElectra(t, 1)
	sk, err := bls.RandKey()
	require.NoError(t, err)

	t.Run("empty requests continues", func(t *testing.T) {
		newSt, err := electra.ProcessDepositRequests(context.Background(), st, []*enginev1.DepositRequest{})
		require.NoError(t, err)
		require.DeepEqual(t, newSt, st)
	})
	t.Run("nil request errors", func(t *testing.T) {
		_, err = electra.ProcessDepositRequests(context.Background(), st, []*enginev1.DepositRequest{nil})
		require.ErrorContains(t, "got a nil DepositRequest", err)
	})

	vals := st.Validators()
	vals[0].PublicKey = sk.PublicKey().Marshal()
	vals[0].WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
	require.NoError(t, st.SetValidators(vals))
	bals := st.Balances()
	bals[0] = params.BeaconConfig().MinActivationBalance + 2000
	require.NoError(t, st.SetBalances(bals))
	require.NoError(t, st.SetPendingBalanceDeposits(make([]*eth.PendingBalanceDeposit, 0))) // reset pbd as the determinitstic state populates this already
	withdrawalCred := make([]byte, 32)
	withdrawalCred[0] = params.BeaconConfig().CompoundingWithdrawalPrefixByte
	depositMessage := &eth.DepositMessage{
		PublicKey:             sk.PublicKey().Marshal(),
		Amount:                1000,
		WithdrawalCredentials: withdrawalCred,
	}
	domain, err := signing.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
	require.NoError(t, err)
	sr, err := signing.ComputeSigningRoot(depositMessage, domain)
	require.NoError(t, err)
	sig := sk.Sign(sr[:])
	requests := []*enginev1.DepositRequest{
		{
			Pubkey:                depositMessage.PublicKey,
			Index:                 0,
			WithdrawalCredentials: depositMessage.WithdrawalCredentials,
			Amount:                depositMessage.Amount,
			Signature:             sig.Marshal(),
		},
	}
	st, err = electra.ProcessDepositRequests(context.Background(), st, requests)
	require.NoError(t, err)

	pbd, err := st.PendingBalanceDeposits()
	require.NoError(t, err)
	require.Equal(t, 2, len(pbd))
	require.Equal(t, uint64(1000), pbd[0].Amount)
	require.Equal(t, uint64(2000), pbd[1].Amount)
}

func TestProcessDeposit_Electra_Simple(t *testing.T) {
	deps, _, err := util.DeterministicDepositsAndKeysSameValidator(3)
	require.NoError(t, err)
	eth1Data, err := util.DeterministicEth1Data(len(deps))
	require.NoError(t, err)
	registry := []*eth.Validator{
		{
			PublicKey:             []byte{1},
			WithdrawalCredentials: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	st, err := state_native.InitializeFromProtoElectra(&eth.BeaconStateElectra{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &eth.Fork{
			PreviousVersion: params.BeaconConfig().ElectraForkVersion,
			CurrentVersion:  params.BeaconConfig().ElectraForkVersion,
		},
	})
	require.NoError(t, err)
	pdSt, err := electra.ProcessDeposits(context.Background(), st, deps)
	require.NoError(t, err)
	pbd, err := pdSt.PendingBalanceDeposits()
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().MinActivationBalance, pbd[2].Amount)
	require.Equal(t, 3, len(pbd))
}

func TestProcessDeposit_SkipsInvalidDeposit(t *testing.T) {
	// Same test settings as in TestProcessDeposit_AddsNewValidatorDeposit, except that we use an invalid signature
	dep, _, err := util.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	dep[0].Data.Signature = make([]byte, 96)
	dt, _, err := util.DepositTrieFromDeposits(dep)
	require.NoError(t, err)
	root, err := dt.HashTreeRoot()
	require.NoError(t, err)
	eth1Data := &eth.Eth1Data{
		DepositRoot:  root[:],
		DepositCount: 1,
	}
	registry := []*eth.Validator{
		{
			PublicKey:             []byte{1},
			WithdrawalCredentials: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	beaconState, err := state_native.InitializeFromProtoElectra(&eth.BeaconStateElectra{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &eth.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	require.NoError(t, err)
	newState, err := electra.ProcessDeposit(beaconState, dep[0], true)
	require.NoError(t, err, "Expected invalid block deposit to be ignored without error")

	if newState.Eth1DepositIndex() != 1 {
		t.Errorf(
			"Expected Eth1DepositIndex to be increased by 1 after processing an invalid deposit, received change: %v",
			newState.Eth1DepositIndex(),
		)
	}
	if len(newState.Validators()) != 1 {
		t.Errorf("Expected validator list to have length 1, received: %v", len(newState.Validators()))
	}
	if len(newState.Balances()) != 1 {
		t.Errorf("Expected validator balances list to have length 1, received: %v", len(newState.Balances()))
	}
	if newState.Balances()[0] != 0 {
		t.Errorf("Expected validator balance at index 0 to stay 0, received: %v", newState.Balances()[0])
	}
}

func TestApplyDeposit_TopUps_WithBadSignature(t *testing.T) {
	st, _ := util.DeterministicGenesisStateElectra(t, 3)
	sk, err := bls.RandKey()
	require.NoError(t, err)
	withdrawalCred := make([]byte, 32)
	withdrawalCred[0] = params.BeaconConfig().CompoundingWithdrawalPrefixByte
	topUpAmount := uint64(1234)
	depositData := &eth.Deposit_Data{
		PublicKey:             sk.PublicKey().Marshal(),
		Amount:                topUpAmount,
		WithdrawalCredentials: withdrawalCred,
		Signature:             make([]byte, fieldparams.BLSSignatureLength),
	}
	vals := st.Validators()
	vals[0].PublicKey = sk.PublicKey().Marshal()
	vals[0].WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
	require.NoError(t, st.SetValidators(vals))
	adSt, err := electra.ApplyDeposit(st, depositData, true)
	require.NoError(t, err)
	pbd, err := adSt.PendingBalanceDeposits()
	require.NoError(t, err)
	require.Equal(t, 1, len(pbd))
	require.Equal(t, topUpAmount, pbd[0].Amount)
}

func TestApplyDeposit_Electra_SwitchToCompoundingValidator(t *testing.T) {
	st, _ := util.DeterministicGenesisStateElectra(t, 3)
	sk, err := bls.RandKey()
	require.NoError(t, err)
	withdrawalCred := make([]byte, 32)
	withdrawalCred[0] = params.BeaconConfig().CompoundingWithdrawalPrefixByte
	depositData := &eth.Deposit_Data{
		PublicKey:             sk.PublicKey().Marshal(),
		Amount:                1000,
		WithdrawalCredentials: withdrawalCred,
		Signature:             make([]byte, fieldparams.BLSSignatureLength),
	}
	vals := st.Validators()
	vals[0].PublicKey = sk.PublicKey().Marshal()
	vals[0].WithdrawalCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
	require.NoError(t, st.SetValidators(vals))
	bals := st.Balances()
	bals[0] = params.BeaconConfig().MinActivationBalance + 2000
	require.NoError(t, st.SetBalances(bals))
	sr, err := signing.ComputeSigningRoot(depositData, bytesutil.ToBytes(3, 32))
	require.NoError(t, err)
	sig := sk.Sign(sr[:])
	depositData.Signature = sig.Marshal()
	adSt, err := electra.ApplyDeposit(st, depositData, false)
	require.NoError(t, err)
	pbd, err := adSt.PendingBalanceDeposits()
	require.NoError(t, err)
	require.Equal(t, 2, len(pbd))
	require.Equal(t, uint64(1000), pbd[0].Amount)
	require.Equal(t, uint64(2000), pbd[1].Amount)
}
