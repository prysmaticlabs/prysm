package params_test

import (
	types "github.com/prysmaticlabs/eth2-types"
	"math"
	"testing"

	"github.com/prysmaticlabs/prysm/testing/require"
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

func TestOrderedConfigSchedule(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	for name, cfg := range params.AllConfigs() {
		t.Run(name.String(), func(t *testing.T) {
			prevVersion := [4]byte{0,0,0,0}
			// epoch 0 is genesis, and it's a uint so can't make it -1
			// so we use a pointer to detect the boundary condition and skip it
			var prevEpoch *types.Epoch
			for _, fse := range cfg.OrderedForkSchedule() {
				if prevEpoch == nil {
					prevEpoch = &fse.Epoch
					prevVersion = fse.Version
					continue
				}
				if *prevEpoch > fse.Epoch {
					t.Errorf("Epochs out of order! %#x/%d before %#x/%d", fse.Version, fse.Epoch, prevVersion, prevEpoch)
				}
				prevEpoch = &fse.Epoch
				prevVersion = fse.Version
			}
		})
	}

	bc := testForkVersionScheduleBCC()
	ofs := bc.OrderedForkSchedule()
	for i := range ofs {
		if ofs[i].Epoch != types.Epoch(math.Pow(2, float64(i))) {
			t.Errorf("expected %dth element of list w/ epoch=%d, got=%d. list=%v", i, types.Epoch(2^i), ofs[i].Epoch, ofs)
		}
	}
}

func TestVersionForEpoch(t *testing.T) {
	bc := testForkVersionScheduleBCC()
	ofs := bc.OrderedForkSchedule()
	testCases := []struct{
		name string
		version [4]byte
		epoch types.Epoch
		err error
	}{
		{
			name: "found between versions",
			version: [4]byte{2,1,2,3},
			epoch: types.Epoch(7),
		},
		{
			name: "found at end",
			version: [4]byte{4,1,2,3},
			epoch: types.Epoch(100),
		},
		{
			name: "found at start",
			version: [4]byte{0,1,2,3},
			epoch: types.Epoch(1),
		},
		{
			name: "found at boundary",
			version: [4]byte{1,1,2,3},
			epoch: types.Epoch(2),
		},
		{
			name: "not found before",
			epoch: types.Epoch(0),
			err: params.VersionForEpochNotFound,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := ofs.VersionForEpoch(tc.epoch)
			if tc.err == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.err)
			}
			require.Equal(t, tc.version, v)
		})
	}
}

func testForkVersionScheduleBCC() *params.BeaconChainConfig {
	return &params.BeaconChainConfig{
		ForkVersionSchedule: map[[4]byte]types.Epoch{
			[4]byte{1,1,2,3}: types.Epoch(2),
			[4]byte{0,1,2,3}: types.Epoch(1),
			[4]byte{4,1,2,3}: types.Epoch(16),
			[4]byte{3,1,2,3}: types.Epoch(8),
			[4]byte{2,1,2,3}: types.Epoch(4),
		},
	}
}