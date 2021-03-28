package kv

import (
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	// Override network name so that hardcoded genesis files are not loaded.
	cfg := params.BeaconConfig()
	cfg.ConfigName = "test"
	params.OverrideBeaconConfig(cfg)
}
