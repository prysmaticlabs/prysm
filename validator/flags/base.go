// Package flags contains all configuration runtime flags for
// the validator service.
package flags

import (
	"path/filepath"
	"runtime"

	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/urfave/cli/v2"
)

const (
	// WalletDefaultDirName for accounts.
	WalletDefaultDirName = "prysm-wallet-v2"
	// DefaultGatewayHost for the validator client.
	DefaultGatewayHost = "127.0.0.1"
)

var (
	// Networking flags.

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
	// MonitoringPortFlag defines the http port used to serve prometheus metrics.
	MonitoringPortFlag = &cli.IntFlag{
		Name:  "monitoring-port",
		Usage: "Port used to listening and respond metrics for prometheus.",
		Value: 8081,
	}

	// Other flags.

	// GraffitiFlag defines the graffiti value included in proposed blocks
	GraffitiFlag = &cli.StringFlag{
		Name:  "graffiti",
		Usage: "String to include in proposed blocks",
	}
	// EnableWebFlag enables controlling the validator client via the Prysm web ui. This is a work in progress.
	EnableWebFlag = &cli.BoolFlag{
		Name:  "web",
		Usage: "Enables the web portal for the validator client (work in progress)",
		Value: false,
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
	// DisableAccountMetricsFlag disables the prometheus metrics for validator accounts, default false.
	DisableAccountMetricsFlag = &cli.BoolFlag{
		Name: "disable-account-metrics",
		Usage: "Disable prometheus metrics for validator accounts. Operators with high volumes " +
			"of validating keys may wish to disable granular prometheus metrics as it increases " +
			"the data cardinality.",
	}
	// DisablePenaltyRewardLogFlag defines the ability to not log reward/penalty information during deployment
	DisablePenaltyRewardLogFlag = &cli.BoolFlag{
		Name:  "disable-rewards-penalties-logging",
		Usage: "Disable reward/penalty logging during cluster deployment",
	}
)

// DefaultValidatorDir returns OS-specific default validator directory.
func DefaultValidatorDir() string {
	// Try to place the data folder in the user's home dir
	home := fileutil.HomeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "Eth2Validators")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "Eth2Validators")
		} else {
			return filepath.Join(home, ".eth2validators")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}
