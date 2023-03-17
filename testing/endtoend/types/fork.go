package types

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
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
	return c
}
