package electra_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
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
