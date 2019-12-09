package flags

import (
	"github.com/urfave/cli"
)

var (
	// CertFlag defines a flag for the node's TLS certificate.
	CertFlag = cli.StringFlag{
		Name:  "tls-cert",
		Usage: "Certificate for secure gRPC. Pass this and the tls-key flag in order to use gRPC securely.",
	}
	// RPCPort defines a slasher node RPC port to open.
	RPCPort = cli.IntFlag{
		Name:  "rpc-port",
		Usage: "RPC port exposed by a beacon node",
		Value: 5000,
	}
	// KeyFlag defines a flag for the node's TLS key.
	KeyFlag = cli.StringFlag{
		Name:  "tls-key",
		Usage: "Key for secure gRPC. Pass this and the tls-cert flag in order to use gRPC securely.",
	}
	BeaconCert = cli.StringFlag{
		Name:  "beacon-tls-cert",
		Usage: "Certificate for secure beacon gRPC connection. Pass this in order to use becon gRPC securely.",
	}
	BeaconProvider = cli.StringFlag{
		Name:  "beacon-provider",
		Usage: "A beacon provider string endpoint. Can either be an grpc server endpoint.",
		Value: "127.0.0.1",
	}
	BeaconPort = cli.IntFlag{
		Name:  "beacon-port",
		Usage: "A beacon provider grpc port.",
		Value: 4000,
	}
)
