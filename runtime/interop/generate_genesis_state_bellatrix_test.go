package interop

import (
	"context"
	"testing"

	state_native "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/container/trie"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestGenerateGenesisStateBellatrix(t *testing.T) {
	g, _, err := GenerateGenesisStateBellatrix(context.Background(), 0, params.BeaconConfig().MinGenesisActiveValidatorCount)
	require.NoError(t, err)

	tr, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err)
	dr, err := tr.HashTreeRoot()
	require.NoError(t, err)
	g.Eth1Data.DepositRoot = dr[:]
	g.Eth1Data.BlockHash = make([]byte, 32)
	st, err := state_native.InitializeFromProtoUnsafeBellatrix(g)
	require.NoError(t, err)
	_, err = st.MarshalSSZ()
	require.NoError(t, err)
}
