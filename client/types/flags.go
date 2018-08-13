package types

import (
	"github.com/urfave/cli"
)

var (
	// ActorFlag defines the role of the sharding client. Either proposer, attester, or simulator.
	ActorFlag = cli.StringFlag{
		Name:  "actor",
		Usage: `use the --actor attester or --actor proposer to start a attester or proposer service in the sharding node`,
	}
	// BeaconRPCProviderFlag defines a beacon node RPC endpoint.
	BeaconRPCProviderFlag = cli.StringFlag{
		Name:  "beacon-rpc-provider",
		Usage: "Beacon node RPC provider endpoint",
		Value: "http://localhost:4000/",
	}
	// CertFlag defines a flag for the node's TLS certificate.
	CertFlag = cli.StringFlag{
		Name:  "tls-cert",
		Usage: "Certificate for secure gRPC. Pass this and the tls-key flag in order to use gRPC securely.",
	}
)
