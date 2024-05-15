package electra_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls/blst"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls/common"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/interop"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
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

func TestProcessConsolidations(t *testing.T) {
	secretKeys, publicKeys, err := interop.DeterministicallyGenerateKeys(0, 2)
	require.NoError(t, err)

	genesisValidatorRoot := bytesutil.PadTo([]byte("genesisValidatorRoot"), fieldparams.RootLength)

	_ = secretKeys

	tests := []struct {
		name    string
		state   state.BeaconState
		scs     []*eth.SignedConsolidation
		check   func(*testing.T, state.BeaconState)
		wantErr string
	}{
		{
			name:    "nil state",
			scs:     make([]*eth.SignedConsolidation, 10),
			wantErr: "nil state",
		},
		{
			name:    "nil consolidation in slice",
			state:   stateWithActiveBalanceETH(t, 19_000_000),
			scs:     []*eth.SignedConsolidation{nil, nil},
			wantErr: "nil consolidation",
		},
		{
			name: "state is 100% full of pending consolidations",
			state: func() state.BeaconState {
				st := stateWithActiveBalanceETH(t, 19_000_000)
				pc := make([]*eth.PendingConsolidation, params.BeaconConfig().PendingConsolidationsLimit)
				require.NoError(t, st.SetPendingConsolidations(pc))
				return st
			}(),
			scs:     []*eth.SignedConsolidation{{Message: &eth.Consolidation{}}},
			wantErr: "pending consolidations queue is full",
		},
		{
			name: "state has too little consolidation churn limit available to process a consolidation",
			state: func() state.BeaconState {
				st, _ := util.DeterministicGenesisStateElectra(t, 1)
				return st
			}(),
			scs:     []*eth.SignedConsolidation{{Message: &eth.Consolidation{}}},
			wantErr: "too little available consolidation churn limit",
		},
		{
			name:    "consolidation with source and target as the same index is rejected",
			state:   stateWithActiveBalanceETH(t, 19_000_000),
			scs:     []*eth.SignedConsolidation{{Message: &eth.Consolidation{SourceIndex: 100, TargetIndex: 100}}},
			wantErr: "source and target index are the same",
		},
		{
			name: "consolidation with inactive source is rejected",
			state: func() state.BeaconState {
				st := stateWithActiveBalanceETH(t, 19_000_000)
				val, err := st.ValidatorAtIndex(25)
				require.NoError(t, err)
				val.ActivationEpoch = params.BeaconConfig().FarFutureEpoch
				require.NoError(t, st.UpdateValidatorAtIndex(25, val))
				return st
			}(),
			scs:     []*eth.SignedConsolidation{{Message: &eth.Consolidation{SourceIndex: 25, TargetIndex: 100}}},
			wantErr: "source is not active",
		},
		{
			name: "consolidation with inactive target is rejected",
			state: func() state.BeaconState {
				st := stateWithActiveBalanceETH(t, 19_000_000)
				val, err := st.ValidatorAtIndex(25)
				require.NoError(t, err)
				val.ActivationEpoch = params.BeaconConfig().FarFutureEpoch
				require.NoError(t, st.UpdateValidatorAtIndex(25, val))
				return st
			}(),
			scs:     []*eth.SignedConsolidation{{Message: &eth.Consolidation{SourceIndex: 100, TargetIndex: 25}}},
			wantErr: "target is not active",
		},
		{
			name: "consolidation with exiting source is rejected",
			state: func() state.BeaconState {
				st := stateWithActiveBalanceETH(t, 19_000_000)
				val, err := st.ValidatorAtIndex(25)
				require.NoError(t, err)
				val.ExitEpoch = 256
				require.NoError(t, st.UpdateValidatorAtIndex(25, val))
				return st
			}(),
			scs:     []*eth.SignedConsolidation{{Message: &eth.Consolidation{SourceIndex: 25, TargetIndex: 100}}},
			wantErr: "source exit epoch has been initiated",
		},
		{
			name: "consolidation with exiting target is rejected",
			state: func() state.BeaconState {
				st := stateWithActiveBalanceETH(t, 19_000_000)
				val, err := st.ValidatorAtIndex(25)
				require.NoError(t, err)
				val.ExitEpoch = 256
				require.NoError(t, st.UpdateValidatorAtIndex(25, val))
				return st
			}(),
			scs:     []*eth.SignedConsolidation{{Message: &eth.Consolidation{SourceIndex: 100, TargetIndex: 25}}},
			wantErr: "target exit epoch has been initiated",
		},
		{
			name:    "consolidation with future epoch is rejected",
			state:   stateWithActiveBalanceETH(t, 19_000_000),
			scs:     []*eth.SignedConsolidation{{Message: &eth.Consolidation{SourceIndex: 100, TargetIndex: 25, Epoch: 55}}},
			wantErr: "consolidation is not valid yet",
		},
		{
			name: "source validator without withdrawal credentials is rejected",
			state: func() state.BeaconState {
				st := stateWithActiveBalanceETH(t, 19_000_000)
				val, err := st.ValidatorAtIndex(25)
				require.NoError(t, err)
				val.WithdrawalCredentials = []byte{}
				require.NoError(t, st.UpdateValidatorAtIndex(25, val))
				return st
			}(),
			scs:     []*eth.SignedConsolidation{{Message: &eth.Consolidation{SourceIndex: 25, TargetIndex: 100}}},
			wantErr: "source does not have execution withdrawal credentials",
		},
		{
			name: "target validator without withdrawal credentials is rejected",
			state: func() state.BeaconState {
				st := stateWithActiveBalanceETH(t, 19_000_000)
				val, err := st.ValidatorAtIndex(25)
				require.NoError(t, err)
				val.WithdrawalCredentials = []byte{}
				require.NoError(t, st.UpdateValidatorAtIndex(25, val))
				return st
			}(),
			scs:     []*eth.SignedConsolidation{{Message: &eth.Consolidation{SourceIndex: 100, TargetIndex: 25}}},
			wantErr: "target does not have execution withdrawal credentials",
		},
		{
			name:    "source and target with different withdrawal credentials is rejected",
			state:   stateWithActiveBalanceETH(t, 19_000_000),
			scs:     []*eth.SignedConsolidation{{Message: &eth.Consolidation{SourceIndex: 100, TargetIndex: 25}}},
			wantErr: "source and target have different withdrawal credentials",
		},
		{
			name: "consolidation with valid signatures is OK",
			state: func() state.BeaconState {
				st := stateWithActiveBalanceETH(t, 19_000_000)
				require.NoError(t, st.SetGenesisValidatorsRoot(genesisValidatorRoot))
				source, err := st.ValidatorAtIndex(100)
				require.NoError(t, err)
				target, err := st.ValidatorAtIndex(25)
				require.NoError(t, err)
				source.PublicKey = publicKeys[0].Marshal()
				source.WithdrawalCredentials = target.WithdrawalCredentials
				require.NoError(t, st.UpdateValidatorAtIndex(100, source))
				target.PublicKey = publicKeys[1].Marshal()
				require.NoError(t, st.UpdateValidatorAtIndex(25, target))
				return st
			}(),
			scs: func() []*eth.SignedConsolidation {
				sc := &eth.SignedConsolidation{Message: &eth.Consolidation{SourceIndex: 100, TargetIndex: 25, Epoch: 8}}

				domain, err := signing.ComputeDomain(
					params.BeaconConfig().DomainConsolidation,
					nil,
					genesisValidatorRoot,
				)
				require.NoError(t, err)
				sr, err := signing.ComputeSigningRoot(sc.Message, domain)
				require.NoError(t, err)

				sig0 := secretKeys[0].Sign(sr[:])
				sig1 := secretKeys[1].Sign(sr[:])

				sc.Signature = blst.AggregateSignatures([]common.Signature{sig0, sig1}).Marshal()

				return []*eth.SignedConsolidation{sc}
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				source, err := st.ValidatorAtIndex(100)
				require.NoError(t, err)
				// The consolidated validator is exiting.
				require.Equal(t, primitives.Epoch(15), source.ExitEpoch) // 15 = state.Epoch(10) + MIN_SEED_LOOKAHEAD(4) + 1
				require.Equal(t, primitives.Epoch(15+params.BeaconConfig().MinValidatorWithdrawabilityDelay), source.WithdrawableEpoch)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := electra.ProcessConsolidations(context.TODO(), tt.state, tt.scs)
			if len(tt.wantErr) > 0 {
				require.ErrorContains(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
			if tt.check != nil {
				tt.check(t, tt.state)
			}
		})
	}
}
