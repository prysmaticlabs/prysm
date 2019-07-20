package featureconfig

import (
	"github.com/urfave/cli"
)

var (
	// EnableCanonicalAttestationFilter filters and sends canonical attestation to RPC requests.
	EnableCanonicalAttestationFilter = cli.BoolFlag{
		Name:  "enable-canonical-attestation-filter",
		Usage: "Enable filtering and sending canonical attestations to RPC request, default is disabled.",
	}
	// DisableHistoricalStatePruningFlag allows the database to keep old historical states.
	DisableHistoricalStatePruningFlag = cli.BoolFlag{
		Name:  "disable-historical-state-pruning",
		Usage: "Disable database pruning of historical states after finalized epochs.",
	}
	// DisableGossipSubFlag uses floodsub in place of gossipsub.
	DisableGossipSubFlag = cli.BoolFlag{
		Name:  "disable-gossip-sub",
		Usage: "Disable gossip sub messaging and use floodsub messaging",
	}
	// EnableExcessDepositsFlag enables a validator to have total amount deposited as more than the
	// max deposit amount.
	EnableExcessDepositsFlag = cli.BoolFlag{
		Name:  "enables-excess-deposit",
		Usage: "Enables balances more than max deposit amount for a validator",
	}
	// NoGenesisDelayFlag disables the standard genesis delay.
	NoGenesisDelayFlag = cli.BoolFlag{
		Name:  "no-genesis-delay",
		Usage: "Process genesis event 30s after the ETH1 block time, rather than wait to midnight of the next day.",
	}
)

// ValidatorFlags contains a list of all the feature flags that apply to the validator client.
var ValidatorFlags = []cli.Flag{}

// BeaconChainFlags contains a list of all the feature flags that apply to the beacon-chain client.
var BeaconChainFlags = []cli.Flag{
	EnableCanonicalAttestationFilter,
	DisableHistoricalStatePruningFlag,
	DisableGossipSubFlag,
	EnableExcessDepositsFlag,
	NoGenesisDelayFlag,
}
