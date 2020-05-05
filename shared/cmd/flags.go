// Package cmd defines the command line flags for the shared utlities.
package cmd

import (
	"gopkg.in/urfave/cli.v2"
)

var (
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
	BootstrapNode = &cli.StringFlag{
		Name:  "bootstrap-node",
		Usage: "The address of bootstrap node. Beacon node will connect for peer discovery via DHT.  Multiple nodes can be separated with a comma",
		Value: "/dns4/prylabs.net/tcp/30001/p2p/16Uiu2HAm7Qwe19vz9WzD2Mxn7fXd1vgHHp4iccuyq7TxwRXoAGfc,enr:-Ku4QAGwOT9StqmwI5LHaIymIO4ooFKfNkEjWa0f1P8OsElgBh2Ijb-GrD_-b9W4kcPFcwmHQEy5RncqXNqdpVo1heoBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpAAAAAAAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQJxCnE6v_x2ekgY_uoE1rtwzvGy40mq9eD66XfHPBWgIIN1ZHCCD6A",
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
	P2PMaxPeers = &cli.Int64Flag{
		Name:  "p2p-max-peers",
		Usage: "The max number of p2p peers to maintain.",
		Value: 30,
	}
	// P2PWhitelist defines a CIDR subnet to exclusively allow connections.
	P2PWhitelist = &cli.StringFlag{
		Name: "p2p-whitelist",
		Usage: "The CIDR subnet for whitelisting peer connections. Example: 192.168.0.0/16 " +
			"would whitelist connections to peers on your local network only. The default " +
			"is to accept all connections.",
	}
	// P2PEncoding defines the encoding format for p2p messages.
	P2PEncoding = &cli.StringFlag{
		Name:  "p2p-encoding",
		Usage: "The encoding format of messages sent over the wire. The default is 0, which represents ssz",
		Value: "ssz-snappy",
	}
	// P2PPubsub defines the pubsub router to use for p2p messages.
	P2PPubsub = &cli.StringFlag{
		Name:  "p2p-pubsub",
		Usage: "The name of the pubsub router to use. Supported values are: gossip, flood, random",
		Value: "gossip",
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
	MaxGoroutines = &cli.Int64Flag{
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
)
