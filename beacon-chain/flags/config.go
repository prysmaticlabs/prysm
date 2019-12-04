package flags

import (
	"github.com/urfave/cli"
)

// GlobalFlags specifies all the global flags for the
// beacon node.
type GlobalFlags struct {
	EnableArchive                     bool
	EnableArchivedValidatorSetChanges bool
	EnableArchivedBlocks              bool
	EnableArchivedAttestations        bool
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

// ConfigureGlobalFlags initializes the archiver config
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
	Init(cfg)
}
