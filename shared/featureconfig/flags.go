package featureconfig

import (
	"github.com/urfave/cli"
)

var (
	// NoGenesisDelayFlag disables the standard genesis delay.
	NoGenesisDelayFlag = cli.BoolFlag{
		Name:  "no-genesis-delay",
		Usage: "Process genesis event 30s after the ETH1 block time, rather than wait to midnight of the next day.",
	}
	// MinimalConfigFlag enables the minimal configuration.
	MinimalConfigFlag = cli.BoolFlag{
		Name:  "minimal-config",
		Usage: "Use minimal config with parameters as defined in the spec.",
	}
	writeSSZStateTransitionsFlag = cli.BoolFlag{
		Name:  "interop-write-ssz-state-transitions",
		Usage: "Write ssz states to disk after attempted state transition",
	}
	// EnableAttestationCacheFlag see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableAttestationCacheFlag = cli.BoolFlag{
		Name:  "enable-attestation-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	// EnableEth1DataVoteCacheFlag see https://github.com/prysmaticlabs/prysm/issues/3106.
	EnableEth1DataVoteCacheFlag = cli.BoolFlag{
		Name:  "enable-eth1-data-vote-cache",
		Usage: "Enable unsafe cache mechanism. See https://github.com/prysmaticlabs/prysm/issues/3106",
	}
	// InitSyncNoVerifyFlag enables the initial sync no verify configuration.
	InitSyncNoVerifyFlag = cli.BoolFlag{
		Name:  "init-sync-no-verify",
		Usage: "Initial sync to finalized check point w/o verifying block's signature, RANDAO and attestation's aggregated signatures",
	}
	// NewCacheFlag enables the node to use the new caching scheme.
	NewCacheFlag = cli.BoolFlag{
		Name:  "new-cache",
		Usage: "Use the new shuffled indices cache for committee. Much improvement than previous caching implementations",
	}
	// SkipBLSVerifyFlag skips BLS signature verification across the runtime for development purposes.
	SkipBLSVerifyFlag = cli.BoolFlag{
		Name:  "skip-bls-verify",
		Usage: "Whether or not to skip BLS verification of signature at runtime, this is unsafe and should only be used for development",
	}
	enableBackupWebhookFlag = cli.BoolFlag{
		Name:  "enable-db-backup-webhook",
		Usage: "Serve HTTP handler to initiate database backups. The handler is served on the monitoring port at path /db/backup.",
	}
	enableBLSPubkeyCacheFlag = cli.BoolFlag{
		Name:  "enable-bls-pubkey-cache",
		Usage: "Enable BLS pubkey cache to improve wall time of PubkeyFromBytes",
	}
)

// ValidatorFlags contains a list of all the feature flags that apply to the validator client.
var ValidatorFlags = []cli.Flag{
	MinimalConfigFlag,
}

// BeaconChainFlags contains a list of all the feature flags that apply to the beacon-chain client.
var BeaconChainFlags = []cli.Flag{
	NoGenesisDelayFlag,
	MinimalConfigFlag,
	writeSSZStateTransitionsFlag,
	EnableAttestationCacheFlag,
	EnableEth1DataVoteCacheFlag,
	InitSyncNoVerifyFlag,
	NewCacheFlag,
	SkipBLSVerifyFlag,
	enableBackupWebhookFlag,
	enableBLSPubkeyCacheFlag,
}
