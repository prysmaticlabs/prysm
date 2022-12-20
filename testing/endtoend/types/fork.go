package types

import (
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

func StartAt(v int, c *params.BeaconChainConfig) *params.BeaconChainConfig {
	c = c.Copy()
	c.DepositContractAddress = "0x4242424242424242424242424242424242424242"
	c.TerminalTotalDifficulty = "300"
	if v >= version.Altair {
		c.AltairForkEpoch = 0
	}
	if v >= version.Bellatrix {
		c.BellatrixForkEpoch = 0
		c.TerminalTotalDifficulty = "0"
	}
	if v >= version.Capella {
		c.CapellaForkEpoch = 0
	}
	return c
}
