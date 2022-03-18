package params_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
)

// Test cases can be executed in an arbitrary order. TestOverrideBeaconConfigTestTeardown checks
// that there's no state mutation leak from the previous test, therefore we need a sentinel flag,
// to make sure that previous test case has already been completed and check can be run.
var testOverrideBeaconConfigExecuted bool

func TestConfig_OverrideBeaconConfig(t *testing.T) {
	// Ensure that param modifications are safe.
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.SlotsPerEpoch = 5
	params.OverrideBeaconConfig(cfg)
	if c := params.BeaconConfig(); c.SlotsPerEpoch != 5 {
		t.Errorf("Shardcount in BeaconConfig incorrect. Wanted %d, got %d", 5, c.SlotsPerEpoch)
	}
	testOverrideBeaconConfigExecuted = true
}

func TestConfig_OverrideBeaconConfigTestTeardown(t *testing.T) {
	if !testOverrideBeaconConfigExecuted {
		t.Skip("State leak can occur only if state mutating test has already completed")
	}
	cfg := params.BeaconConfig()
	if cfg.SlotsPerEpoch == 5 {
		t.Fatal("Parameter update has been leaked out of previous test")
	}
}

func TestConfig_DataRace(t *testing.T) {
	for i := 0; i < 10; i++ {
		go func() {
			cfg := params.BeaconConfig()
			params.OverrideBeaconConfig(cfg)
		}()
		go func() uint64 {
			return params.BeaconConfig().MaxDeposits
		}()
	}
}
