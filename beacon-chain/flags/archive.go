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
