package params_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestOverrideBeaconConfig(t *testing.T) {
	cfg := params.BeaconConfig().Copy()
	cfg.SlotsPerEpoch = 5
	resetCfg := params.OverrideBeaconConfigWithReset(cfg)
	defer resetCfg()
	if c := params.BeaconConfig(); c.SlotsPerEpoch != 5 {
		t.Errorf("Shardcount in BeaconConfig incorrect. Wanted %d, got %d", 5, c.SlotsPerEpoch)
	}
}

func TestOverrideBeaconConfigWithReset(t *testing.T) {
	cfg := params.BeaconConfig().Copy()
	origSlotsPerEpoch := cfg.SlotsPerEpoch
	newSlotsPerEpoch := origSlotsPerEpoch + 42

	cfg.SlotsPerEpoch = newSlotsPerEpoch
	resetFunc := params.OverrideBeaconConfigWithReset(cfg)
	if c := params.BeaconConfig(); c.SlotsPerEpoch != newSlotsPerEpoch {
		t.Errorf("Config value is incorrect, want: %d, got %d", newSlotsPerEpoch, c.SlotsPerEpoch)
	}

	resetFunc()
	if c := params.BeaconConfig(); c.SlotsPerEpoch != origSlotsPerEpoch {
		t.Errorf("Config value is incorrect, want: %d, got %d", origSlotsPerEpoch, c.SlotsPerEpoch)
	}
}
