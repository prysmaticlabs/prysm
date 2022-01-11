package genesis_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state/genesis"
	"github.com/prysmaticlabs/prysm/config/params"
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
