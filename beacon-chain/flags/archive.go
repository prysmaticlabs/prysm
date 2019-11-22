package flags

import (
	"github.com/urfave/cli"
)

var (
	// ArchiveEnableFlag defines whether or not the beacon chain should archive
	// historical blocks, attestations, and validator set changes.
	ArchiveEnableFlag = cli.BoolFlag{
		Name:  "archive",
		Usage: "Whether or not beacon chain should archive historical data including blocks, attestations, and validator set changes",
	}
	// ArchiveValidatorSetChangesFlag defines whether or not the beacon chain should archive
	// historical validator set changes in persistent storage.
	ArchiveValidatorSetChangesFlag = cli.BoolFlag{
		Name:  "archive-validator-set-changes",
		Usage: "Whether or not beacon chain should archive historical validator set changes",
	}
	// ArchiveBlocksFlag defines whether or not the beacon chain should archive
	// historical block data in persistent storage.
	ArchiveBlocksFlag = cli.BoolFlag{
		Name:  "archive-blocks",
		Usage: "Whether or not beacon chain should archive historical blocks",
	}
	// ArchiveAttestationsFlag defines whether or not the beacon chain should archive
	// historical attestation data in persistent storage.
	ArchiveAttestationsFlag = cli.BoolFlag{
		Name:  "archive-attestations",
		Usage: "Whether or not beacon chain should archive historical blocks",
	}
)

// ArchiveFlags specifies all the archive flags for the
// archiver service.
type ArchiveFlags struct {
	EnableArchive                     bool
	EnableArchivedValidatorSetChanges bool
	EnableArchivedBlocks              bool
	EnableArchivedAttestations        bool
}

var archiveConfig *ArchiveFlags

// Get retrieves archive config.
func Get() *ArchiveFlags {
	if archiveConfig == nil {
		return &ArchiveFlags{}
	}
	return archiveConfig
}

// Init sets the archive config equal to the config that is passed in.
func Init(c *ArchiveFlags) {
	archiveConfig = c
}

// ConfigureArchiveFlags initializes the archiver config
// based on the provided cli context.
func ConfigureArchiveFlags(ctx *cli.Context) {
	cfg := &ArchiveFlags{}
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
