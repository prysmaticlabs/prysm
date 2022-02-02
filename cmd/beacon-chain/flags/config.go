package flags

import (
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/urfave/cli/v2"
)

// GlobalFlags specifies all the global flags for the
// beacon node.
type GlobalFlags struct {
	HeadSync                   bool
	DisableSync                bool
	DisableDiscv5              bool
	SubscribeToAllSubnets      bool
	MinimumSyncPeers           int
	MinimumPeersPerSubnet      int
	BlockBatchLimit            int
	BlockBatchLimitBurstFactor int
}

var globalConfig *GlobalFlags

// Get retrieves the global config.
func Get() *GlobalFlags {
	if globalConfig == nil {
		return &GlobalFlags{}
	}
	return globalConfig
}

// Init sets the global config equal to the config that is passed in.
func Init(c *GlobalFlags) {
	globalConfig = c
}

// ConfigureGlobalFlags initializes the global config.
// based on the provided cli context.
func ConfigureGlobalFlags(ctx *cli.Context) {
	cfg := &GlobalFlags{}
	if ctx.Bool(HeadSync.Name) {
		log.Warn("Using Head Sync flag, it starts syncing from last saved head.")
		cfg.HeadSync = true
	}
	if ctx.Bool(DisableSync.Name) {
		log.Warn("Using Disable Sync flag, using this flag on a live network might lead to adverse consequences.")
		cfg.DisableSync = true
	}
	if ctx.Bool(SubscribeToAllSubnets.Name) {
		log.Warn("Subscribing to All Attestation Subnets")
		cfg.SubscribeToAllSubnets = true
	}
	cfg.DisableDiscv5 = ctx.Bool(DisableDiscv5.Name)
	cfg.BlockBatchLimit = ctx.Int(BlockBatchLimit.Name)
	cfg.BlockBatchLimitBurstFactor = ctx.Int(BlockBatchLimitBurstFactor.Name)
	cfg.MinimumPeersPerSubnet = ctx.Int(MinPeersPerSubnet.Name)
	configureMinimumPeers(ctx, cfg)

	Init(cfg)
}

func configureMinimumPeers(ctx *cli.Context, cfg *GlobalFlags) {
	cfg.MinimumSyncPeers = ctx.Int(MinSyncPeers.Name)
	maxPeers := ctx.Int(cmd.P2PMaxPeers.Name)
	if cfg.MinimumSyncPeers > maxPeers {
		log.Warnf("Changing Minimum Sync Peers to %d", maxPeers)
		cfg.MinimumSyncPeers = maxPeers
	}
}
