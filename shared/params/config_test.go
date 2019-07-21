package params

import (
	"testing"
)

func TestOverrideBeaconConfig(t *testing.T) {
	cfg := BeaconConfig()
	cfg.ShardCount = 5
	OverrideBeaconConfig(cfg)
	if c := BeaconConfig(); c.ShardCount != 5 {
		t.Errorf("Shardcount in BeaconConfig incorrect. Wanted %d, got %d", 5, c.ShardCount)
	}
}
