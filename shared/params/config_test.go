package params_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestOverrideBeaconConfig(t *testing.T) {
	cfg := params.BeaconConfig()
	cfg.SlotsPerEpoch = 5
	params.OverrideBeaconConfig(cfg)
	if c := params.BeaconConfig(); c.SlotsPerEpoch != 5 {
		t.Errorf("Shardcount in BeaconConfig incorrect. Wanted %d, got %d", 5, c.SlotsPerEpoch)
	}
}
