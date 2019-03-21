package types

import (
	"github.com/urfave/cli"
)

var (
	// NoCustomConfigFlag determines whether to launch a beacon chain using real parameters or demo parameters.
	NoCustomConfigFlag = cli.BoolFlag{
		Name:  "no-custom-config",
		Usage: "Run the beacon chain with the real parameters from phase 0.",
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
