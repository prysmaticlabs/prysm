package beaconv1

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGetDepositContract(t *testing.T) {
	const chainId = 99
	const address = "0x0000000000000000000000000000000000000009"
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.DepositChainID = chainId

	config.DepositContractAddress = address
	params.OverrideBeaconConfig(config)

	s := Server{}
	resp, err := s.GetDepositContract(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.Equal(t, uint64(chainId), resp.Data.ChainId)
	assert.Equal(t, address, resp.Data.Address)
}
