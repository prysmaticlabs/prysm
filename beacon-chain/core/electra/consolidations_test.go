package electra_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestProcessPendingConsolidations(t *testing.T) {
	tests := []struct {
		name    string
		state   state.BeaconState
		check   func(*testing.T, state.BeaconState)
		wantErr bool
	}{
		{
			name:    "nil state",
			state:   nil,
			wantErr: true,
		},
		{
			name: "no pending consolidations",
			state: func() state.BeaconState {
				pb := &eth.BeaconStateElectra{}

				st, err := state_native.InitializeFromProtoUnsafeElectra(pb)
				require.NoError(t, err)
				return st
			}(),
			wantErr: false,
		},
		{
			name: "processes pending consolidation successfully",
			state: func() state.BeaconState {
				pb := &eth.BeaconStateElectra{
					Validators: []*eth.Validator{
						{
							WithdrawalCredentials: []byte{0x01, 0xFF},
						},
						{
							WithdrawalCredentials: []byte{0x01, 0xAB},
						},
					},
					Balances: []uint64{
						params.BeaconConfig().MinActivationBalance,
						params.BeaconConfig().MinActivationBalance,
					},
					PendingConsolidations: []*eth.PendingConsolidation{
						{
							SourceIndex: 0,
							TargetIndex: 1,
						},
					},
				}

				st, err := state_native.InitializeFromProtoUnsafeElectra(pb)
				require.NoError(t, err)
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				// Balances are transferred from v0 to v1.
				bal0, err := st.BalanceAtIndex(0)
				require.NoError(t, err)
				require.Equal(t, uint64(0), bal0)
				bal1, err := st.BalanceAtIndex(1)
				require.NoError(t, err)
				require.Equal(t, 2*params.BeaconConfig().MinActivationBalance, bal1)

				// The pending consolidation is removed from the list.
				num, err := st.NumPendingConsolidations()
				require.NoError(t, err)
				require.Equal(t, uint64(0), num)

				// v1 is switched to compounding validator.
				v1, err := st.ValidatorAtIndex(1)
				require.NoError(t, err)
				require.Equal(t, params.BeaconConfig().CompoundingWithdrawalPrefixByte, v1.WithdrawalCredentials[0])
			},
			wantErr: false,
		},
		{
			name: "stop processing when a source val withdrawable epoch is in the future",
			state: func() state.BeaconState {
				pb := &eth.BeaconStateElectra{
					Validators: []*eth.Validator{
						{
							WithdrawalCredentials: []byte{0x01, 0xFF},
							WithdrawableEpoch:     100,
						},
						{
							WithdrawalCredentials: []byte{0x01, 0xAB},
						},
					},
					Balances: []uint64{
						params.BeaconConfig().MinActivationBalance,
						params.BeaconConfig().MinActivationBalance,
					},
					PendingConsolidations: []*eth.PendingConsolidation{
						{
							SourceIndex: 0,
							TargetIndex: 1,
						},
					},
				}

				st, err := state_native.InitializeFromProtoUnsafeElectra(pb)
				require.NoError(t, err)
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				// No balances are transferred from v0 to v1.
				bal0, err := st.BalanceAtIndex(0)
				require.NoError(t, err)
				require.Equal(t, params.BeaconConfig().MinActivationBalance, bal0)
				bal1, err := st.BalanceAtIndex(1)
				require.NoError(t, err)
				require.Equal(t, params.BeaconConfig().MinActivationBalance, bal1)

				// The pending consolidation is still in the list.
				num, err := st.NumPendingConsolidations()
				require.NoError(t, err)
				require.Equal(t, uint64(1), num)
			},
			wantErr: false,
		},
		{
			name: "slashed validator is not consolidated",
			state: func() state.BeaconState {
				pb := &eth.BeaconStateElectra{
					Validators: []*eth.Validator{
						{
							WithdrawalCredentials: []byte{0x01, 0xFF},
						},
						{
							WithdrawalCredentials: []byte{0x01, 0xAB},
						},
						{
							Slashed: true,
						},
						{
							WithdrawalCredentials: []byte{0x01, 0xCC},
						},
					},
					Balances: []uint64{
						params.BeaconConfig().MinActivationBalance,
						params.BeaconConfig().MinActivationBalance,
						params.BeaconConfig().MinActivationBalance,
						params.BeaconConfig().MinActivationBalance,
					},
					PendingConsolidations: []*eth.PendingConsolidation{
						{
							SourceIndex: 2,
							TargetIndex: 3,
						},
						{
							SourceIndex: 0,
							TargetIndex: 1,
						},
					},
				}

				st, err := state_native.InitializeFromProtoUnsafeElectra(pb)
				require.NoError(t, err)
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				// No balances are transferred from v2 to v3.
				bal0, err := st.BalanceAtIndex(2)
				require.NoError(t, err)
				require.Equal(t, params.BeaconConfig().MinActivationBalance, bal0)
				bal1, err := st.BalanceAtIndex(3)
				require.NoError(t, err)
				require.Equal(t, params.BeaconConfig().MinActivationBalance, bal1)

				// No pending consolidation remaining.
				num, err := st.NumPendingConsolidations()
				require.NoError(t, err)
				require.Equal(t, uint64(0), num)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := electra.ProcessPendingConsolidations(context.TODO(), tt.state)
			require.Equal(t, tt.wantErr, err != nil)
			if tt.check != nil {
				tt.check(t, tt.state)
			}
		})
	}
}

func stateWithActiveBalanceETH(t *testing.T, balETH uint64) state.BeaconState {
	gwei := balETH * 1_000_000_000
	balPerVal := params.BeaconConfig().MinActivationBalance
	numVals := gwei / balPerVal

	vals := make([]*eth.Validator, numVals)
	bals := make([]uint64, numVals)
	for i := uint64(0); i < numVals; i++ {
		wc := make([]byte, 32)
		wc[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
		wc[31] = byte(i)
		vals[i] = &eth.Validator{
			ActivationEpoch:       0,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance:      balPerVal,
			WithdrawalCredentials: wc,
		}
		bals[i] = balPerVal
	}
	st, err := state_native.InitializeFromProtoUnsafeElectra(&eth.BeaconStateElectra{
		Slot:       10 * params.BeaconConfig().SlotsPerEpoch,
		Validators: vals,
		Balances:   bals,
		Fork: &eth.Fork{
			CurrentVersion: params.BeaconConfig().ElectraForkVersion,
		},
	})
	require.NoError(t, err)

	return st
}
