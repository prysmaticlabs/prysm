package flags

import (
	"time"

	"github.com/urfave/cli/v2"
)

var (
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
