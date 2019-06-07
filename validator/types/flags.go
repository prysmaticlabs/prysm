package types

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/prysmaticlabs/prysm/shared/cmd"
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
	KeystorePathFlag = cmd.DirectoryFlag{
		Name:  "keystore-path",
		Usage: "path to the desired keystore directory",
		Value: cmd.DirectoryString{Value: defaultValidatorDir()},
	}
	// PasswordFlag defines the password value for storing and retrieving validator private keys from the keystore.
	PasswordFlag = cli.StringFlag{
		Name:  "password",
		Usage: "string value of the password for your validator private keys",
	}
	// DisablePenaltyRewardLogFlag defines the ability to not log reward/penalty information during deployment
	DisablePenaltyRewardLogFlag = cli.BoolFlag{
		Name:  "disable-rewards-penalties-logging",
		Usage: "Disable reward/penalty logging during cluster deployment",
	}
)

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

func defaultValidatorDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
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
