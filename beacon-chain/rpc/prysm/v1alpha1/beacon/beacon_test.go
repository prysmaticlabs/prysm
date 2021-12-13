package beacon

import (
	"testing"

	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
)

func TestMain(m *testing.M) {
	// Use minimal config to reduce test setup time.
	prevConfig := params.BeaconConfig().Copy()
	defer params.OverrideBeaconConfig(prevConfig)
	cfg := params.MinimalSpecConfig()
	cfg.EpochsPerHistoricalVector = fieldparams.RandaoMixesLength
	cfg.SlotsPerHistoricalRoot = fieldparams.BlockRootsLength
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
