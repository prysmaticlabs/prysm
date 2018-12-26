package types

import (
	"github.com/urfave/cli"
)

var (
	// BeaconRPCProviderFlag defines a beacon node RPC endpoint.
	BeaconRPCProviderFlag = cli.StringFlag{
		Name:  "beacon-rpc-provider",
		Usage: "Beacon node RPC provider endpoint",
		Value: "http://localhost:4000/",
	}
	// PubKeyFlag defines a flag for validator's public key on the mainchain
	PubKeyFlag = cli.StringFlag{
		Name:  "pubkey",
		Usage: "Validator's public key. The public key will be used to identify the validator to the beacon-node",
	}
	// CertFlag defines a flag for the node's TLS certificate.
	CertFlag = cli.StringFlag{
		Name:  "tls-cert",
		Usage: "Certificate for secure gRPC. Pass this and the tls-key flag in order to use gRPC securely.",
	}
)
