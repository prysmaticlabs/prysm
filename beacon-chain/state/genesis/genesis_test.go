package genesis_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state/genesis"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestGenesisState(t *testing.T) {
	tests := []struct {
		name params.ConfigName
	}{
		{
			name: params.Mainnet,
		},
	}
	for _, tt := range tests {
		t.Run(params.ConfigNames[tt.name], func(t *testing.T) {
			st, err := genesis.State(params.ConfigNames[tt.name])
			if err != nil {
				t.Fatal(err)
			}
			if st == nil {
				t.Fatal("nil state")
			}
			if st.NumValidators() <= 0 {
				t.Error("No validators present in state")
			}
		})
	}
}

func TestGenesisStateDifferentDepositContract(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.DepositContractAddress = "0x00000000219ab540356cBB839Cbe05303d770000"
	params.OverrideBeaconConfig(config)

	tests := []struct {
		name params.ConfigName
	}{
		{
			name: params.Mainnet,
		},
	}
	for _, tt := range tests {
		t.Run(params.ConfigNames[tt.name], func(t *testing.T) {
			st, err := genesis.State(params.ConfigNames[tt.name])
			require.NoError(t, err)
			require.DeepEqual(t, (*v1.BeaconState)(nil), st)
		})
	}
}
