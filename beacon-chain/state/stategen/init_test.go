package stategen

import (
	"github.com/prysmaticlabs/prysm/config/params"
)

func init() {
	// Override network name so that hardcoded genesis files are not loaded.
	cfg := params.BeaconConfig().Copy()
	cfg.ConfigName = "test"
	params.SetTestForkVersions(cfg, params.TestForkVersionSuffix)
	params.OverrideBeaconConfig(cfg)
}
