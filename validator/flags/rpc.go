package flags

import (
	"time"

	"github.com/urfave/cli/v2"
)

var (
	// BeaconRPCProviderFlag defines a beacon node RPC endpoint.
	BeaconRPCProviderFlag = &cli.StringFlag{
		Name:  "beacon-rpc-provider",
		Usage: "Beacon node RPC provider endpoint",
		Value: "127.0.0.1:4000",
	}
	// BeaconRPCGatewayProviderFlag defines a beacon node JSON-RPC endpoint.
	BeaconRPCGatewayProviderFlag = &cli.StringFlag{
		Name:  "beacon-rpc-gateway-provider",
		Usage: "Beacon node RPC gateway provider endpoint",
		Value: "127.0.0.1:3500",
	}
	// SlasherRPCProviderFlag defines a slasher node RPC endpoint.
	SlasherRPCProviderFlag = &cli.StringFlag{
		Name:  "slasher-rpc-provider",
		Usage: "Slasher node RPC provider endpoint",
		Value: "127.0.0.1:4002",
	}
	// SlasherCertFlag defines a flag for the slasher node's TLS certificate.
	SlasherCertFlag = &cli.StringFlag{
		Name:  "slasher-tls-cert",
		Usage: "Certificate for secure slasher gRPC. Pass this and the tls-key flag in order to use gRPC securely.",
	}
	// CertFlag defines a flag for the node's TLS certificate.
	CertFlag = &cli.StringFlag{
		Name:  "tls-cert",
		Usage: "Certificate for secure gRPC. Pass this and the tls-key flag in order to use gRPC securely.",
	}
	// EnableRPCFlag enables controlling the validator client via gRPC (without web UI).
	EnableRPCFlag = &cli.BoolFlag{
		Name:  "rpc",
		Usage: "Enables the RPC server for the validator client (without Web UI)",
		Value: false,
	}
	// RPCHost defines the host on which the RPC server should listen.
	RPCHost = &cli.StringFlag{
		Name:  "rpc-host",
		Usage: "Host on which the RPC server should listen",
		Value: "127.0.0.1",
	}
	// RPCPort defines a validator client RPC port to open.
	RPCPort = &cli.IntFlag{
		Name:  "rpc-port",
		Usage: "RPC port exposed by a validator client",
		Value: 7000,
	}
	// GrpcRetriesFlag defines the number of times to retry a failed gRPC request.
	GrpcRetriesFlag = &cli.UintFlag{
		Name:  "grpc-retries",
		Usage: "Number of attempts to retry gRPC requests",
		Value: 5,
	}
	// GrpcRetryDelayFlag defines the interval to retry a failed gRPC request.
	GrpcRetryDelayFlag = &cli.DurationFlag{
		Name:  "grpc-retry-delay",
		Usage: "The amount of time between gRPC retry requests.",
		Value: 1 * time.Second,
	}
	// GrpcHeadersFlag defines a list of headers to send with all gRPC requests.
	GrpcHeadersFlag = &cli.StringFlag{
		Name: "grpc-headers",
		Usage: "A comma separated list of key value pairs to pass as gRPC headers for all gRPC " +
			"calls. Example: --grpc-headers=key=value",
	}

	// gRPC gateway flags.

	// GRPCGatewayHost specifies a gRPC gateway host for the validator client.
	GRPCGatewayHost = &cli.StringFlag{
		Name:  "grpc-gateway-host",
		Usage: "The host on which the gateway server runs on",
		Value: DefaultGatewayHost,
	}
	// GRPCGatewayPort enables a gRPC gateway to be exposed for the validator client.
	GRPCGatewayPort = &cli.IntFlag{
		Name:  "grpc-gateway-port",
		Usage: "Enable gRPC gateway for JSON requests",
		Value: 7500,
	}
	// GPRCGatewayCorsDomain serves preflight requests when serving gRPC JSON gateway.
	GPRCGatewayCorsDomain = &cli.StringFlag{
		Name: "grpc-gateway-corsdomain",
		Usage: "Comma separated list of domains from which to accept cross origin requests " +
			"(browser enforced). This flag has no effect if not used with --grpc-gateway-port.",
		Value: "http://localhost:4242,http://127.0.0.1:4242,http://localhost:4200,http://0.0.0.0:4242,http://0.0.0.0:4200",
	}

	// Remote keymanager flags.

	// GrpcRemoteAddressFlag defines the host:port address for a remote keymanager to connect to.
	GrpcRemoteAddressFlag = &cli.StringFlag{
		Name:  "grpc-remote-address",
		Usage: "Host:port of a gRPC server for a remote keymanager",
		Value: "",
	}
	// RemoteSignerCertPathFlag defines the path to a client.crt file for a wallet to connect to
	// a secure signer via TLS and gRPC.
	RemoteSignerCertPathFlag = &cli.StringFlag{
		Name:  "remote-signer-crt-path",
		Usage: "/path/to/client.crt for establishing a secure, TLS gRPC connection to a remote signer server",
		Value: "",
	}
	// RemoteSignerKeyPathFlag defines the path to a client.key file for a wallet to connect to
	// a secure signer via TLS and gRPC.
	RemoteSignerKeyPathFlag = &cli.StringFlag{
		Name:  "remote-signer-key-path",
		Usage: "/path/to/client.key for establishing a secure, TLS gRPC connection to a remote signer server",
		Value: "",
	}
	// RemoteSignerCACertPathFlag defines the path to a ca.crt file for a wallet to connect to
	// a secure signer via TLS and gRPC.
	RemoteSignerCACertPathFlag = &cli.StringFlag{
		Name:  "remote-signer-ca-crt-path",
		Usage: "/path/to/ca.crt for establishing a secure, TLS gRPC connection to a remote signer server",
		Value: "",
	}
)
