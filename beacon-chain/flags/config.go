package flags

import (
	"github.com/prysmaticlabs/prysm/shared/cmd"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// GlobalFlags specifies all the global flags for the
// beacon node.
type GlobalFlags struct {
	EnableArchive                     bool
	EnableArchivedValidatorSetChanges bool
	EnableArchivedBlocks              bool
	EnableArchivedAttestations        bool
	MinimumSyncPeers                  int
	MaxPageSize                       int
	DeploymentBlock                   int
	UnsafeSync                        bool
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
	if ctx.GlobalBool(ArchiveEnableFlag.Name) {
		cfg.EnableArchive = true
	}
	if ctx.GlobalBool(ArchiveValidatorSetChangesFlag.Name) {
		cfg.EnableArchivedValidatorSetChanges = true
	}
	if ctx.GlobalBool(ArchiveBlocksFlag.Name) {
		cfg.EnableArchivedBlocks = true
	}
	if ctx.GlobalBool(ArchiveAttestationsFlag.Name) {
		cfg.EnableArchivedAttestations = true
	}
	if ctx.GlobalBool(UnsafeSync.Name) {
		cfg.UnsafeSync = true
	}
	cfg.MaxPageSize = ctx.GlobalInt(RPCMaxPageSize.Name)
	cfg.DeploymentBlock = ctx.GlobalInt(ContractDeploymentBlock.Name)
	configureMinimumPeers(ctx, cfg)

	Init(cfg)
}

func configureMinimumPeers(ctx *cli.Context, cfg *GlobalFlags) {
	cfg.MinimumSyncPeers = ctx.GlobalInt(MinSyncPeers.Name)
	maxPeers := int(ctx.GlobalInt64(cmd.P2PMaxPeers.Name))
	if cfg.MinimumSyncPeers > maxPeers {
		log.Warnf("Changing Minimum Sync Peers to %d", maxPeers)
		cfg.MinimumSyncPeers = maxPeers
	}
}
