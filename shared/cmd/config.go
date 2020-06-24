package cmd

import (
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/urfave/cli/v2"
)

// Flags is a struct to represent which features the client will perform on runtime.
type Flags struct {
	// Configuration related flags.
	MinimalConfig      bool   // MinimalConfig as defined in the spec.
	E2EConfig          bool   // E2EConfig made specifically for testing, do not use except in E2E.
	CustomGenesisDelay uint64 // CustomGenesisDelay signals how long of a delay to set to start the chain.
	MaxRPCPageSize     int    //MaxRPCPageSize is used for a cap of page sizes in RPC requests.
}

var sharedConfig *Flags

// Get retrieves feature config.
func Get() *Flags {
	if sharedConfig == nil {
		return &Flags{}
	}
	return sharedConfig
}

// Init sets the global config equal to the config that is passed in.
func Init(c *Flags) {
	sharedConfig = c
}

// InitWithReset sets the global config and returns function that is used to reset configuration.
func InitWithReset(c *Flags) func() {
	resetFunc := func() {
		Init(&Flags{})
	}
	Init(c)
	return resetFunc
}

// ConfigureBeaconChain sets the global config based
// on what flags are enabled for the beacon-chain client.
func ConfigureBeaconChain(ctx *cli.Context) {
	cfg := newConfig(ctx)
	delay := params.BeaconConfig().GenesisDelay
	if ctx.IsSet(CustomGenesisDelayFlag.Name) {
		delay = ctx.Uint64(CustomGenesisDelayFlag.Name)
		log.Warnf("Starting ETH2 with genesis delay of %d seconds", delay)
	}
	cfg.CustomGenesisDelay = delay
	maxPageSize := params.BeaconConfig().DefaultPageSize
	if ctx.IsSet(RPCMaxPageSize.Name) {
		maxPageSize = ctx.Int(RPCMaxPageSize.Name)
		log.Warnf("Starting beacon chain with max RPC page size of %d", maxPageSize)
	}
	cfg.MaxRPCPageSize = maxPageSize
	Init(cfg)
}

// ConfigureSlasher sets the global config based
// on what flags are enabled for the slasher client.
func ConfigureSlasher(ctx *cli.Context) {
	cfg := newConfig(ctx)
	maxPageSize := params.BeaconConfig().DefaultPageSize
	if ctx.IsSet(RPCMaxPageSize.Name) {
		maxPageSize = ctx.Int(RPCMaxPageSize.Name)
		log.Warnf("Starting slasher with max RPC page size of %d", maxPageSize)
	}
	cfg.MaxRPCPageSize = maxPageSize
	Init(cfg)
}

// ConfigureValidator sets the global config based
// on what flags are enabled for the validator client.
func ConfigureValidator(ctx *cli.Context) {
	cfg := newConfig(ctx)
	Init(cfg)
}

func newConfig(ctx *cli.Context) *Flags {
	cfg := &Flags{}
	if ctx.Bool(MinimalConfigFlag.Name) {
		log.Warn("Using minimal config")
		cfg.MinimalConfig = true
		params.UseMinimalConfig()
	}
	if ctx.Bool(E2EConfigFlag.Name) {
		log.Warn("Using end-to-end testing config")
		cfg.MinimalConfig = true
		params.UseE2EConfig()
	}
	return cfg
}
