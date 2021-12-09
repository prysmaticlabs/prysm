package beacon

import (
	"testing"

	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/config/params"
)

func TestMain(m *testing.M) {
	// Use minimal config to reduce test setup time.
	prevConfig := params.BeaconConfig().Copy()
	defer params.OverrideBeaconConfig(prevConfig)
	cfg := params.MinimalSpecConfig()
	cfg.EpochsPerHistoricalVector = customtypes.RandaoMixesSize
	cfg.SlotsPerHistoricalRoot = customtypes.BlockRootsSize
	params.OverrideBeaconConfig(cfg)

	resetFlags := flags.Get()
	flags.Init(&flags.GlobalFlags{
		MinimumSyncPeers: 30,
	})
	defer func() {
		flags.Init(resetFlags)
	}()

	m.Run()
}
