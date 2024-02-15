package structs

import (
	"testing"

	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestDepositSnapshotFromConsensus(t *testing.T) {
	ds := &eth.DepositSnapshot{
		Finalized:      [][]byte{{0xde, 0xad, 0xbe, 0xef}, {0xca, 0xfe, 0xba, 0xbe}},
		DepositRoot:    []byte{0xab, 0xcd},
		DepositCount:   12345,
		ExecutionHash:  []byte{0x12, 0x34},
		ExecutionDepth: 67890,
	}

	res := DepositSnapshotFromConsensus(ds)
	require.NotNil(t, res)
	require.DeepEqual(t, []string{"0xdeadbeef", "0xcafebabe"}, res.Finalized)
	require.Equal(t, "0xabcd", res.DepositRoot)
	require.Equal(t, "12345", res.DepositCount)
	require.Equal(t, "0x1234", res.ExecutionBlockHash)
	require.Equal(t, "67890", res.ExecutionBlockHeight)
}
