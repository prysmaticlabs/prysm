// Package flags defines beacon-node specific runtime flags for
// setting important values such as ports, eth1 endpoints, and more.
package flags

import (
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/urfave/cli/v2"
)

var (
	// Network-related flags.

	// HTTPWeb3ProviderFlag provides an HTTP access endpoint to an ETH 1.0 RPC.
	HTTPWeb3ProviderFlag = &cli.StringFlag{
		Name:  "http-web3provider",
		Usage: "A mainchain web3 provider string http endpoint",
		Value: "",
	}
	// DepositContractFlag defines a flag for the deposit contract address.
	DepositContractFlag = &cli.StringFlag{
		Name:  "deposit-contract",
		Usage: "Deposit contract address. Beacon chain node will listen logs coming from the deposit contract to determine when validator is eligible to participate.",
		Value: params.BeaconNetworkConfig().DepositContractAddress,
	}
	// MonitoringPortFlag defines the http port used to serve prometheus metrics.
	MonitoringPortFlag = &cli.IntFlag{
		Name:  "monitoring-port",
		Usage: "Port used to listening and respond metrics for prometheus.",
		Value: 8080,
	}
	// RPCHost defines the host on which the RPC server should listen.
	RPCHost = &cli.StringFlag{
		Name:  "rpc-host",
		Usage: "Host on which the RPC server should listen",
		Value: "127.0.0.1",
	}
	// RPCPort defines a beacon node RPC port to open.
	RPCPort = &cli.IntFlag{
		Name:  "rpc-port",
		Usage: "RPC port exposed by a beacon node",
		Value: 4000,
	}
	// ContractDeploymentBlock is the block in which the eth1 deposit contract was deployed.
	ContractDeploymentBlock = &cli.IntFlag{
		Name:  "contract-deployment-block",
		Usage: "The eth1 block in which the deposit contract was deployed.",
		Value: 11184524,
	}

	// Beacon node management flags.

	// SetGCPercent is the percentage of current live allocations at which the garbage collector is to run.
	SetGCPercent = &cli.IntFlag{
		Name:  "gc-percent",
		Usage: "The percentage of freshly allocated data to live data on which the gc will be run again.",
		Value: 100,
	}
	// HeadSync starts the beacon node from the previously saved head state and syncs from there.
	HeadSync = &cli.BoolFlag{
		Name:  "head-sync",
		Usage: "Starts the beacon node with the previously saved head state instead of finalized state.",
	}
	// SlotsPerArchivedPoint specifies the number of slots between the archived points, to save beacon state in the cold
	// section of DB.
	SlotsPerArchivedPoint = &cli.IntFlag{
		Name:  "slots-per-archive-point",
		Usage: "The slot durations of when an archived state gets saved in the DB.",
		Value: 2048,
	}
	// WeakSubjectivityCheckpt defines the weak subjectivity checkpoint the node must sync through to defend against long range attacks.
	WeakSubjectivityCheckpt = &cli.StringFlag{
		Name: "weak-subjectivity-checkpoint",
		Usage: "Input in `block_root:epoch_number` format. This guarantee that syncing leads to the given Weak Subjectivity Checkpoint being in the canonical chain. " +
			"If such a sync is not possible, the node will treat it a critical and irrecoverable failure",
		Value: "",
	}
	// DisableSync disables a node from syncing at start-up. Instead the node enters regular sync
	// immediately.
	DisableSync = &cli.BoolFlag{
		Name:  "disable-sync",
		Usage: "Starts the beacon node without entering initial sync and instead exits to regular sync immediately.",
	}
	// HistoricalSlasherNode is a set of beacon node flags required for performing historical detection with a slasher.
	HistoricalSlasherNode = &cli.BoolFlag{
		Name:  "historical-slasher-node",
		Usage: "Enables required flags for serving historical data to a slasher client. Results in additional storage usage",
	}
	// ChainID defines a flag to set the chain id. If none is set, it derives this value from NetworkConfig
	ChainID = &cli.Uint64Flag{
		Name:  "chain-id",
		Usage: "Sets the chain id of the beacon chain.",
	}
	// NetworkID defines a flag to set the network id. If none is set, it derives this value from NetworkConfig
	NetworkID = &cli.Uint64Flag{
		Name:  "network-id",
		Usage: "Sets the network id of the beacon chain.",
	}

	// Backup flags.

	// EnableBackupWebhookFlag for users to trigger db backups via an HTTP webhook.
	EnableBackupWebhookFlag = &cli.BoolFlag{
		Name:  "enable-db-backup-webhook",
		Usage: "Serve HTTP handler to initiate database backups. The handler is served on the monitoring port at path /db/backup.",
	}
	// BackupWebhookOutputDir to customize the output directory for db backups.
	BackupWebhookOutputDir = &cli.StringFlag{
		Name:  "db-backup-output-dir",
		Usage: "Output directory for db backups",
	}
)
