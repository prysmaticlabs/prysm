package beaconv1

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"

	ptypes "github.com/gogo/protobuf/types"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestGetGenesis(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.GenesisForkVersion = []byte("genesis")
	params.OverrideBeaconConfig(config)
	genesis := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	validatorsRoot := [32]byte{1, 2, 3, 4, 5, 6}
	chainService := &chainMock.ChainService{}
	s := Server{
		GenesisTimeFetcher: chainService,
		ChainInfoFetcher:   chainService,
	}

	t.Run("OK", func(t *testing.T) {
		chainService.Genesis = genesis
		chainService.ValidatorsRoot = validatorsRoot
		resp, err := s.GetGenesis(context.Background(), &ptypes.Empty{})
		require.NoError(t, err)
		assert.Equal(t, resp.GenesisTime.Seconds, genesis.Unix())
		assert.Equal(t, int64(resp.GenesisTime.Nanos), genesis.UnixNano())
		assert.DeepEqual(t, resp.GenesisValidatorsRoot, validatorsRoot[:])
		assert.DeepEqual(t, resp.GenesisForkVersion, []byte("genesis"))
	})

	t.Run("No genesis time", func(t *testing.T) {

	})

	t.Run("No genesis validator root", func(t *testing.T) {

	})
}
