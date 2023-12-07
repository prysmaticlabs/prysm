package backfill

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/network/forks"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestDomainCache(t *testing.T) {
	cfg := params.MainnetConfig()
	vRoot, err := hexutil.Decode("0x0011223344556677889900112233445566778899001122334455667788990011")
	dType := cfg.DomainBeaconProposer
	require.NoError(t, err)
	require.Equal(t, 32, len(vRoot))
	fsched := forks.NewOrderedSchedule(cfg)
	dc, err := newDomainCache(vRoot,
		dType, fsched)
	require.NoError(t, err)
	require.Equal(t, len(fsched), len(dc.forkDomains))
	for i := range fsched {
		e := fsched[i].Epoch
		ad, err := dc.forEpoch(e)
		require.NoError(t, err)
		ed, err := signing.ComputeDomain(dType, fsched[i].Version[:], vRoot)
		require.NoError(t, err)
		require.DeepEqual(t, ed, ad)
	}
}
