// Package cmd defines the command line flags for the shared utlities.
package cmd

import (
	"github.com/urfave/cli/v2"
)

var (
	// MinimalConfigFlag declares to use the minimal config for running Eth2.0.
	MinimalConfigFlag = &cli.BoolFlag{
		Name:  "minimal-config",
		Usage: "Use minimal config with parameters as defined in the spec.",
	}
	// E2EConfigFlag declares to use a testing specific config for running Eth2.0 in end-to-end testing.
	E2EConfigFlag = &cli.BoolFlag{
		Name:  "e2e-config",
		Usage: "Use the E2E testing config, only for use within end-to-end testing.",
	}
	// RPCMaxPageSizeFlag defines the maximum numbers per page returned in RPC responses from this
	// beacon node (default: 500).
	RPCMaxPageSizeFlag = &cli.IntFlag{
		Name:  "rpc-max-page-size",
		Usage: "Max number of items returned per page in RPC responses for paginated endpoints.",
	}
	// VerbosityFlag defines the logrus configuration.
	VerbosityFlag = &cli.StringFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity (trace, debug, info=default, warn, error, fatal, panic)",
		Value: "info",
	}
	// DataDirFlag defines a path on disk.
	DataDirFlag = &cli.StringFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases and keystore",
		Value: DefaultDataDir(),
	}
	// EnableTracingFlag defines a flag to enable p2p message tracing.
	EnableTracingFlag = &cli.BoolFlag{
		Name:  "enable-tracing",
		Usage: "Enable request tracing.",
	}
	// TracingProcessNameFlag defines a flag to specify a process name.
	TracingProcessNameFlag = &cli.StringFlag{
		Name:  "tracing-process-name",
		Usage: "The name to apply to tracing tag \"process_name\"",
	}
	// TracingEndpointFlag flag defines the http endpoint for serving traces to Jaeger.
	TracingEndpointFlag = &cli.StringFlag{
		Name:  "tracing-endpoint",
		Usage: "Tracing endpoint defines where beacon chain traces are exposed to Jaeger.",
		Value: "http://127.0.0.1:14268/api/traces",
	}
	// TraceSampleFractionFlag defines a flag to indicate what fraction of p2p
	// messages are sampled for tracing.
	TraceSampleFractionFlag = &cli.Float64Flag{
		Name:  "trace-sample-fraction",
		Usage: "Indicate what fraction of p2p messages are sampled for tracing.",
		Value: 0.20,
	}
	// MonitoringHostFlag defines the host used to serve prometheus metrics.
	MonitoringHostFlag = &cli.StringFlag{
		Name:  "monitoring-host",
		Usage: "Host used for listening and responding metrics for prometheus.",
		Value: "127.0.0.1",
	}
	// DisableMonitoringFlag defines a flag to disable the metrics collection.
	DisableMonitoringFlag = &cli.BoolFlag{
		Name:  "disable-monitoring",
		Usage: "Disable monitoring service.",
	}
	// NoDiscovery specifies whether we are running a local network and have no need for connecting
	// to the bootstrap nodes in the cloud
	NoDiscovery = &cli.BoolFlag{
		Name:  "no-discovery",
		Usage: "Enable only local network p2p and do not connect to cloud bootstrap nodes.",
	}
	// StaticPeers specifies a set of peers to connect to explicitly.
	StaticPeers = &cli.StringSliceFlag{
		Name:  "peer",
		Usage: "Connect with this peer. This flag may be used multiple times.",
	}
	// BootstrapNode tells the beacon node which bootstrap node to connect to
	BootstrapNode = &cli.StringSliceFlag{
		Name:  "bootstrap-node",
		Usage: "The address of bootstrap node. Beacon node will connect for peer discovery via DHT.  Multiple nodes can be passed by using the flag multiple times but not comma-separated. You can also pass YAML files containing multiple nodes.",
		Value: cli.NewStringSlice("enr:-Ku4QMKVC_MowDsmEa20d5uGjrChI0h8_KsKXDmgVQbIbngZV0idV6_RL7fEtZGo-kTNZ5o7_EJI_vCPJ6scrhwX0Z4Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQJxCnE6v_x2ekgY_uoE1rtwzvGy40mq9eD66XfHPBWgIIN1ZHCCD6A"),
	}
	// RelayNode tells the beacon node which relay node to connect to.
	RelayNode = &cli.StringFlag{
		Name: "relay-node",
		Usage: "The address of relay node. The beacon node will connect to the " +
			"relay node and advertise their address via the relay node to other peers",
		Value: "",
	}
	// P2PUDPPort defines the port to be used by discv5.
	P2PUDPPort = &cli.IntFlag{
		Name:  "p2p-udp-port",
		Usage: "The port used by discv5.",
		Value: 12000,
	}
	// P2PTCPPort defines the port to be used by libp2p.
	P2PTCPPort = &cli.IntFlag{
		Name:  "p2p-tcp-port",
		Usage: "The port used by libp2p.",
		Value: 13000,
	}
	// P2PIP defines the local IP to be used by libp2p.
	P2PIP = &cli.StringFlag{
		Name:  "p2p-local-ip",
		Usage: "The local ip address to listen for incoming data.",
		Value: "",
	}
	// P2PHost defines the host IP to be used by libp2p.
	P2PHost = &cli.StringFlag{
		Name:  "p2p-host-ip",
		Usage: "The IP address advertised by libp2p. This may be used to advertise an external IP.",
		Value: "",
	}
	// P2PHostDNS defines the host DNS to be used by libp2p.
	P2PHostDNS = &cli.StringFlag{
		Name:  "p2p-host-dns",
		Usage: "The DNS address advertised by libp2p. This may be used to advertise an external DNS.",
		Value: "",
	}
	// P2PPrivKey defines a flag to specify the location of the private key file for libp2p.
	P2PPrivKey = &cli.StringFlag{
		Name:  "p2p-priv-key",
		Usage: "The file containing the private key to use in communications with other peers.",
		Value: "",
	}
	// P2PMetadata defines a flag to specify the location of the peer metadata file.
	P2PMetadata = &cli.StringFlag{
		Name:  "p2p-metadata",
		Usage: "The file containing the metadata to communicate with other peers.",
		Value: "",
	}
	// P2PMaxPeers defines a flag to specify the max number of peers in libp2p.
	P2PMaxPeers = &cli.IntFlag{
		Name:  "p2p-max-peers",
		Usage: "The max number of p2p peers to maintain.",
		Value: 30,
	}
	// P2PAllowList defines a CIDR subnet to exclusively allow connections.
	P2PAllowList = &cli.StringFlag{
		Name: "p2p-allowlist",
		Usage: "The CIDR subnet for allowing only certain peer connections. Example: " +
			"192.168.0.0/16 would permit connections to peers on your local network only. The " +
			"default is to accept all connections.",
	}
	// P2PDenyList defines a list of CIDR subnets to disallow connections from them.
	P2PDenyList = &cli.StringSliceFlag{
		Name: "p2p-denylist",
		Usage: "The CIDR subnets for denying certainy peer connections. Example: " +
			"192.168.0.0/16 would deny connections from peers on your local network only. The " +
			"default is to accept all connections.",
	}
	// ForceClearDB removes any previously stored data at the data directory.
	ForceClearDB = &cli.BoolFlag{
		Name:  "force-clear-db",
		Usage: "Clear any previously stored data at the data directory",
	}
	// ClearDB prompts user to see if they want to remove any previously stored data at the data directory.
	ClearDB = &cli.BoolFlag{
		Name:  "clear-db",
		Usage: "Prompt for clearing any previously stored data at the data directory",
	}
	// LogFormat specifies the log output format.
	LogFormat = &cli.StringFlag{
		Name:  "log-format",
		Usage: "Specify log formatting. Supports: text, json, fluentd.",
		Value: "text",
	}
	// MaxGoroutines specifies the maximum amount of goroutines tolerated, before a status check fails.
	MaxGoroutines = &cli.IntFlag{
		Name:  "max-goroutines",
		Usage: "Specifies the upper limit of goroutines running before a status check fails",
		Value: 5000,
	}
	// LogFileName specifies the log output file name.
	LogFileName = &cli.StringFlag{
		Name:  "log-file",
		Usage: "Specify log file name, relative or absolute",
	}
	// EnableUPnPFlag specifies if UPnP should be enabled or not. The default value is false.
	EnableUPnPFlag = &cli.BoolFlag{
		Name:  "enable-upnp",
		Usage: "Enable the service (Beacon chain or Validator) to use UPnP when possible.",
	}
	// ConfigFileFlag specifies the filepath to load flag values.
	ConfigFileFlag = &cli.StringFlag{
		Name:  "config-file",
		Usage: "The filepath to a yaml file with flag values",
	}
	// ChainConfigFileFlag specifies the filepath to load flag values.
	ChainConfigFileFlag = &cli.StringFlag{
		Name:  "chain-config-file",
		Usage: "The path to a YAML file with chain config values",
	}
	// GrpcMaxCallRecvMsgSizeFlag defines the max call message size for GRPC
	GrpcMaxCallRecvMsgSizeFlag = &cli.IntFlag{
		Name:  "grpc-max-msg-size",
		Usage: "Integer to define max recieve message call size (default: 4194304 (for 4MB))",
		Value: 1 << 22,
	}
)
