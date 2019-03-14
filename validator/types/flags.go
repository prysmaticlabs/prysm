package types

import (
	"github.com/urfave/cli"
)

var (
	// DemoConfigFlag determines whether to launch a beacon chain using demo parameters
	// such as shorter cycle length, fewer shards, and more.
	DemoConfigFlag = cli.BoolFlag{
		Name:  "demo-config",
		Usage: "Run the validator using demo paramteres (i.e. shorter cycles, fewer shards and committees)",
	}
	// BeaconRPCProviderFlag defines a beacon node RPC endpoint.
	BeaconRPCProviderFlag = cli.StringFlag{
		Name:  "beacon-rpc-provider",
		Usage: "Beacon node RPC provider endpoint",
		Value: "localhost:4000",
	}
	// CertFlag defines a flag for the node's TLS certificate.
	CertFlag = cli.StringFlag{
		Name:  "tls-cert",
		Usage: "Certificate for secure gRPC. Pass this and the tls-key flag in order to use gRPC securely.",
	}
	// KeystorePathFlag defines the location of the keystore directory for a validator's account.
	KeystorePathFlag = cli.StringFlag{
		Name:  "keystore-path",
		Usage: "path to the desired keystore directory",
	}
	// PasswordFlag defines the password value for storing and retrieving validator private keys from the keystore.
	PasswordFlag = cli.StringFlag{
		Name:  "password",
		Usage: "string value of the password for your validator private keys",
	}
)
