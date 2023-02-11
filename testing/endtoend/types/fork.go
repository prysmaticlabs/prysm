package types

import (
	"fmt"
	"math"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

func StartAt(v int, c *params.BeaconChainConfig) *params.BeaconChainConfig {
	c = c.Copy()
	if v >= version.Altair {
		c.AltairForkEpoch = 0
	}
	if v >= version.Bellatrix {
		c.BellatrixForkEpoch = 0
	}
	if v >= version.Capella {
		c.CapellaForkEpoch = 0
	}
	// Time TTD to line up roughly with the bellatrix fork epoch.
	// E2E sets EL block production rate equal to SecondsPerETH1Block to keep the math simple.
	ttd := uint64(c.BellatrixForkEpoch) * uint64(c.SlotsPerEpoch) * c.SecondsPerSlot
	c.TerminalTotalDifficulty = fmt.Sprintf("%d", ttd)
	// TODO: Remove this once lighthouse has released a capella
	// compatible release.
	if c.ConfigName == params.EndToEndMainnetName {
		c.CapellaForkEpoch = math.MaxUint64
	}
	return c
}
