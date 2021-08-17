package beacon

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/shared/params"
	sharedtestutil "github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestListSyncCommittees(t *testing.T) {
	ctx := context.Background()
	st, _ := sharedtestutil.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
	stRoot, err := st.HashTreeRoot(ctx)
	require.NoError(t, err)

	s := &Server{
		StateFetcher: &testutil.MockFetcher{
			BeaconState: st,
		},
	}
	req := &eth.StateSyncCommitteesRequest{StateId: stRoot[:]}
	resp, err := s.ListSyncCommittees(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
	committeeVals := resp.Data.Validators
	require.NotNil(t, committeeVals)
	require.Equal(t, params.BeaconConfig().SyncCommitteeSize, uint64(len(committeeVals)), "incorrect committee size")
	for i := uint64(0); i < params.BeaconConfig().SyncCommitteeSize; i++ {
		assert.Equal(t, types.ValidatorIndex(i), committeeVals[i])
	}
	require.NotNil(t, resp.Data.ValidatorAggregates)
	assert.Equal(t, params.BeaconConfig().SyncCommitteeSubnetCount, uint64(len(resp.Data.ValidatorAggregates)))
	for i := uint64(0); i < params.BeaconConfig().SyncCommitteeSubnetCount; i++ {
		vStartIndex := types.ValidatorIndex(params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount * i)
		vEndIndex := types.ValidatorIndex(params.BeaconConfig().SyncCommitteeSize/params.BeaconConfig().SyncCommitteeSubnetCount*(i+1) - 1)
		j := 0
		for vIndex := vStartIndex; vIndex <= vEndIndex; vIndex++ {
			assert.Equal(t, vIndex, resp.Data.ValidatorAggregates[i].Validators[j])
			j++
		}
	}
}
