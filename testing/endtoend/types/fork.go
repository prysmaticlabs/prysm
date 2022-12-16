package types

import "github.com/prysmaticlabs/prysm/v3/config/params"

func StartAtBellatrix(c *params.BeaconChainConfig) *params.BeaconChainConfig {
	c = c.Copy()
	c.DepositContractAddress = "0x4242424242424242424242424242424242424242"
	c.AltairForkEpoch = 0
	c.BellatrixForkEpoch = 0
	c.TerminalTotalDifficulty = "0"
	return c
}
