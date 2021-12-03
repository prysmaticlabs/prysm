// Package flags defines beacon-node specific runtime flags for
// setting important values such as ports, eth1 endpoints, and more.
package flags

import (
	"encoding/hex"
	"strings"

	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/urfave/cli/v2"
)

var (
	// HTTPWeb3ProviderFlag provides an HTTP access endpoint to an ETH 1.0 RPC.
	HTTPWeb3ProviderFlag = &cli.StringFlag{
		Name:  "http-web3provider",
		Usage: "A mainchain web3 provider string http endpoint. Can contain auth header as well in the format --http-web3provider=\"https://goerli.infura.io/v3/xxxx,Basic xxx\" for project secret (base64 encoded) and --http-web3provider=\"https://goerli.infura.io/v3/xxxx,Bearer xxx\" for jwt use",
		Value: "",
	}
	// FallbackWeb3ProviderFlag provides a fallback endpoint to an ETH 1.0 RPC.
	FallbackWeb3ProviderFlag = &cli.StringSliceFlag{
		Name:  "fallback-web3provider",
		Usage: "A mainchain web3 provider string http endpoint. This is our fallback web3 provider, this flag may be used multiple times.",
	}
	// DepositContractFlag defines a flag for the deposit contract address.
	DepositContractFlag = &cli.StringFlag{
		Name:  "deposit-contract",
		Usage: "Deposit contract address. Beacon chain node will listen logs coming from the deposit contract to determine when validator is eligible to participate.",
		Value: params.BeaconConfig().DepositContractAddress,
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
	// MonitoringPortFlag defines the http port used to serve prometheus metrics.
	MonitoringPortFlag = &cli.IntFlag{
		Name:  "monitoring-port",
		Usage: "Port used to listening and respond metrics for prometheus.",
		Value: 8080,
	}
	// CertFlag defines a flag for the node's TLS certificate.
	CertFlag = &cli.StringFlag{
		Name:  "tls-cert",
		Usage: "Certificate for secure gRPC. Pass this and the tls-key flag in order to use gRPC securely.",
	}
	// KeyFlag defines a flag for the node's TLS key.
	KeyFlag = &cli.StringFlag{
		Name:  "tls-key",
		Usage: "Key for secure gRPC. Pass this and the tls-cert flag in order to use gRPC securely.",
	}
	// HTTPModules define the set of enabled HTTP APIs.
	HTTPModules = &cli.StringFlag{
		Name:  "http-modules",
		Usage: "Comma-separated list of API module names. Possible values: `" + PrysmAPIModule + `,` + EthAPIModule + "`.",
		Value: strings.Join([]string{PrysmAPIModule, EthAPIModule}, ","),
	}
	// DisableGRPCGateway for JSON-HTTP requests to the beacon node.
	DisableGRPCGateway = &cli.BoolFlag{
		Name:  "disable-grpc-gateway",
		Usage: "Disable the gRPC gateway for JSON-HTTP requests",
	}
	// GRPCGatewayHost specifies a gRPC gateway host for Prysm.
	GRPCGatewayHost = &cli.StringFlag{
		Name:  "grpc-gateway-host",
		Usage: "The host on which the gateway server runs on",
		Value: "127.0.0.1",
	}
	// GRPCGatewayPort specifies a gRPC gateway port for Prysm.
	GRPCGatewayPort = &cli.IntFlag{
		Name:  "grpc-gateway-port",
		Usage: "The port on which the gateway server runs on",
		Value: 3500,
	}
	// GPRCGatewayCorsDomain serves preflight requests when serving gRPC JSON gateway.
	GPRCGatewayCorsDomain = &cli.StringFlag{
		Name: "grpc-gateway-corsdomain",
		Usage: "Comma separated list of domains from which to accept cross origin requests " +
			"(browser enforced). This flag has no effect if not used with --grpc-gateway-port.",
		Value: "http://localhost:4200,http://localhost:7500,http://127.0.0.1:4200,http://127.0.0.1:7500,http://0.0.0.0:4200,http://0.0.0.0:7500",
	}
	// MinSyncPeers specifies the required number of successful peer handshakes in order
	// to start syncing with external peers.
	MinSyncPeers = &cli.Uint64Flag{
		Name:  "min-sync-peers",
		Usage: "The required number of valid peers to connect with before syncing.",
		Value: 3,
	}
	// ContractDeploymentBlock is the block in which the eth1 deposit contract was deployed.
	ContractDeploymentBlock = &cli.IntFlag{
		Name:  "contract-deployment-block",
		Usage: "The eth1 block in which the deposit contract was deployed.",
		Value: 11184524,
	}
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
	// section of beaconDB.
	SlotsPerArchivedPoint = &cli.IntFlag{
		Name:  "slots-per-archive-point",
		Usage: "The slot durations of when an archived state gets saved in the beaconDB.",
		Value: 2048,
	}
	// DisableDiscv5 disables running discv5.
	DisableDiscv5 = &cli.BoolFlag{
		Name:  "disable-discv5",
		Usage: "Does not run the discoveryV5 dht.",
	}
	// BlockBatchLimit specifies the requested block batch size.
	BlockBatchLimit = &cli.Uint64Flag{
		Name:  "block-batch-limit",
		Usage: "The amount of blocks the local peer is bounded to request and respond to in a batch.",
		Value: 64,
	}
	// BlockBatchLimitBurstFactor specifies the factor by which block batch size may increase.
	BlockBatchLimitBurstFactor = &cli.Uint64Flag{
		Name:  "block-batch-limit-burst-factor",
		Usage: "The factor by which block batch limit may increase on burst.",
		Value: 10,
	}
	// DisableSync disables a node from syncing at start-up. Instead the node enters regular sync
	// immediately.
	DisableSync = &cli.BoolFlag{
		Name:  "disable-sync",
		Usage: "Starts the beacon node without entering initial sync and instead exits to regular sync immediately.",
	}
	// EnableDebugRPCEndpoints as /v1/beacon/state.
	EnableDebugRPCEndpoints = &cli.BoolFlag{
		Name:  "enable-debug-rpc-endpoints",
		Usage: "Enables the debug rpc service, containing utility endpoints such as /eth/v1alpha1/beacon/state.",
	}
	// SubscribeToAllSubnets defines a flag to specify whether to subscribe to all possible attestation/sync subnets or not.
	SubscribeToAllSubnets = &cli.BoolFlag{
		Name:  "subscribe-all-subnets",
		Usage: "Subscribe to all possible attestation and sync subnets.",
	}
	// HistoricalSlasherNode is a set of beacon node flags required for performing historical detection with a slasher.
	HistoricalSlasherNode = &cli.BoolFlag{
		Name:  "historical-slasher-node",
		Usage: "Enables required flags for serving historical data to a slasher client. Results in additional storage usage",
	}
	// ChainID defines a flag to set the chain id. If none is set, it derives this value from NetworkConfig
	ChainID = &cli.Uint64Flag{
		Name:  "chain-id",
		Usage: "Sets the chain id of the beacon chain",
	}
	// NetworkID defines a flag to set the network id. If none is set, it derives this value from NetworkConfig
	NetworkID = &cli.Uint64Flag{
		Name:  "network-id",
		Usage: "Sets the network id of the beacon chain.",
	}
	// WeakSubjectivityCheckpt defines the weak subjectivity checkpoint the node must sync through to defend against long range attacks.
	WeakSubjectivityCheckpt = &cli.StringFlag{
		Name: "weak-subjectivity-checkpoint",
		Usage: "Input in `block_root:epoch_number` format. This guarantees that syncing leads to the given Weak Subjectivity Checkpoint along the canonical chain. " +
			"If such a sync is not possible, the node will treat it a critical and irrecoverable failure",
		Value: "",
	}
	// Eth1HeaderReqLimit defines a flag to set the maximum number of headers that a deposit log query can fetch. If none is set, 1000 will be the limit.
	Eth1HeaderReqLimit = &cli.Uint64Flag{
		Name:  "eth1-header-req-limit",
		Usage: "Sets the maximum number of headers that a deposit log query can fetch.",
		Value: uint64(1000),
	}
	// GenesisStatePath defines a flag to start the beacon chain from a give genesis state file.
	GenesisStatePath = &cli.StringFlag{
		Name: "genesis-state",
		Usage: "Load a genesis state from ssz file. Testnet genesis files can be found in the " +
			"eth2-clients/eth2-testnets repository on github.",
	}
	// MinPeersPerSubnet defines a flag to set the minimum number of peers that a node will attempt to peer with for a subnet.
	MinPeersPerSubnet = &cli.Uint64Flag{
		Name:  "minimum-peers-per-subnet",
		Usage: "Sets the minimum number of peers that a node will attempt to peer with that are subscribed to a subnet.",
		Value: 6,
	}
	// TerminalTotalDifficultyOverride specifies the total difficulty to manual overrides the `TERMINAL_TOTAL_DIFFICULTY` parameter.
	TerminalTotalDifficultyOverride = &cli.Uint64Flag{
		Name: "terminal-total-difficulty-override",
		Usage: "Sets the total difficulty to manual overrides the default TERMINAL_TOTAL_DIFFICULTY value. " +
			"WARNING: This flag should be used only if you have a clear understanding that community has decided to override the terminal difficulty. " +
			"Incorrect usage will result in your node experience consensus failure.",
	}
	// TerminalBlockHashOverride specifies the terminal block hash to manual overrides the `TERMINAL_BLOCK_HASH` parameter.
	TerminalBlockHashOverride = &cli.StringFlag{
		Name: "terminal-block-hash-override",
		Usage: "Sets the block hash to manual overrides the default TERMINAL_BLOCK_HASH value. " +
			"WARNING: This flag should be used only if you have a clear understanding that community has decided to override the terminal block hash. " +
			"Incorrect usage will result in your node experience consensus failure.",
	}
	// TerminalBlockHashActivationEpochOverride specifies the terminal block hash epoch to manual overrides the `TERMINAL_BLOCK_HASH_ACTIVATION_EPOCH` parameter.
	TerminalBlockHashActivationEpochOverride = &cli.Uint64Flag{
		Name: "terminal-block-hash-epoch-override",
		Usage: "Sets the block hash epoch to manual overrides the default TERMINAL_BLOCK_HASH_ACTIVATION_EPOCH value. " +
			"WARNING: This flag should be used only if you have a clear understanding that community has decided to override the terminal block hash activation epoch. " +
			"Incorrect usage will result in your node experience consensus failure.",
	}
	// FeeRecipient specifies the fee recipient for the transaction fees.
	FeeRecipient = &cli.StringFlag{
		Name:  "fee-recipient",
		Usage: "Post merge, this address will receive the transaction fees produced by any blocks from this node. Default to junk whilst merge is in development state.",
		Value: hex.EncodeToString([]byte("0x0000000000000000000000000000000000000001")),
	}
)
