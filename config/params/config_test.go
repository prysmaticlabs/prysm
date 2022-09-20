package params_test

import (
	"sync"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
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
	params.SetupTestConfigCleanup(t)
	wg := new(sync.WaitGroup)
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			cfg := params.BeaconConfig()
			params.OverrideBeaconConfig(cfg)
		}()
		go func() uint64 {
			defer wg.Done()
			return params.BeaconConfig().MaxDeposits
		}()
	}
	wg.Wait()
}
