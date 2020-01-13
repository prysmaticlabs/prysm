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
	// BeaconCertFlag defines a flag for the beacon api certificate.
	BeaconCertFlag = cli.StringFlag{
		Name:  "beacon-tls-cert",
		Usage: "Certificate for secure beacon gRPC connection. Pass this in order to use beacon gRPC securely.",
	}
	// BeaconRPCProviderFlag defines a flag for the beacon host ip or address.
	BeaconRPCProviderFlag = cli.StringFlag{
		Name:  "beacon-rpc-provider",
		Usage: "Beacon node RPC provider endpoint",
		Value: "localhost:4000",
	}
	// UseSpanCacheFlag enables the slasher to use span cache.
	UseSpanCacheFlag = cli.BoolFlag{
		Name:  "span-map-cache",
		Usage: "Enable span map cache",
	}
	// RebuildSpanMapsFlag iterate through all indexed attestations in db and update all validators span maps from scratch.
	RebuildSpanMapsFlag = cli.BoolFlag{
		Name:  "rebuild-span-maps",
		Usage: "Rebuild span maps from indexed attestations in db",
	}
)
