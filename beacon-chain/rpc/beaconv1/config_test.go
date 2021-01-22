package beaconv1

import (
	"context"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	powchaintest "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestGetDepositContract(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.GenesisForkVersion = []byte{9, 0, 0, 0}
	params.OverrideBeaconConfig(config)

	infoFetcher := powchaintest.POWChain{}
	infoFetcher.DepositContractAddress()

	s := Server{
		PowchainInfoFetcher: &infoFetcher,
	}
	resp, err := s.GetDepositContract(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	assert.Equal(t, uint64(9), resp.Data.ChainId)
	assert.Equal(t, "0102030400000000000000000000000000000000", resp.Data.Address)
}
