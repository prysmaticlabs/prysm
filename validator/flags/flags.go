// Package flags contains all configuration runtime flags for
// the validator service.
package flags

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"github.com/prysmaticlabs/prysm/validator/accounts/v2/consts"
	"github.com/urfave/cli/v2"
)

var (
	// DisableAccountMetricsFlag defines the graffiti value included in proposed blocks, default false.
	DisableAccountMetricsFlag = &cli.BoolFlag{
		Name: "disable-account-metrics",
		Usage: "Disable prometheus metrics for validator accounts. Operators with high volumes " +
			"of validating keys may wish to disable granular prometheus metrics as it increases " +
			"the data cardinality.",
	}
	// BeaconRPCProviderFlag defines a beacon node RPC endpoint.
	BeaconRPCProviderFlag = &cli.StringFlag{
		Name:  "beacon-rpc-provider",
		Usage: "Beacon node RPC provider endpoint",
		Value: "127.0.0.1:4000",
	}
	// CertFlag defines a flag for the node's TLS certificate.
	CertFlag = &cli.StringFlag{
		Name:  "tls-cert",
		Usage: "Certificate for secure gRPC. Pass this and the tls-key flag in order to use gRPC securely.",
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
	// DisablePenaltyRewardLogFlag defines the ability to not log reward/penalty information during deployment
	DisablePenaltyRewardLogFlag = &cli.BoolFlag{
		Name:  "disable-rewards-penalties-logging",
		Usage: "Disable reward/penalty logging during cluster deployment",
	}
	// GraffitiFlag defines the graffiti value included in proposed blocks
	GraffitiFlag = &cli.StringFlag{
		Name:  "graffiti",
		Usage: "String to include in proposed blocks",
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
	// KeyManager specifies the key manager to use.
	KeyManager = &cli.StringFlag{
		Name:  "keymanager",
		Usage: "The keymanger to use (unencrypted, interop, keystore, wallet)",
		Value: "",
	}
	// KeyManagerOpts specifies the key manager options.
	KeyManagerOpts = &cli.StringFlag{
		Name:  "keymanageropts",
		Usage: "The options for the keymanger, either a JSON string or path to same",
		Value: "",
	}
	// KeystorePathFlag defines the location of the keystore directory for a validator's account.
	KeystorePathFlag = &cli.StringFlag{
		Name:  "keystore-path",
		Usage: "Path to the desired keystore directory",
	}
	// MonitoringPortFlag defines the http port used to serve prometheus metrics.
	MonitoringPortFlag = &cli.IntFlag{
		Name:  "monitoring-port",
		Usage: "Port used to listening and respond metrics for prometheus.",
		Value: 8081,
	}
	// PasswordFlag defines the password value for storing and retrieving validator private keys from the keystore.
	PasswordFlag = &cli.StringFlag{
		Name:  "password",
		Usage: "String value of the password for your validator private keys",
	}
	// SourceDirectories defines the locations of the source validator databases while managing validators.
	SourceDirectories = &cli.StringFlag{
		Name:  "source-dirs",
		Usage: "The directory of source validator databases",
	}
	// SourceDirectory defines the location of the source validator database while managing validators.
	SourceDirectory = &cli.StringFlag{
		Name:  "source-dir",
		Usage: "The directory of the source validator database",
	}
	// TargetDirectory defines the location of the target validator database while managing validators.
	TargetDirectory = &cli.StringFlag{
		Name:  "target-dir",
		Usage: "The directory of the target validator database",
	}
	// UnencryptedKeysFlag specifies a file path of a JSON file of unencrypted validator keys as an
	// alternative from launching the validator client from decrypting a keystore directory.
	UnencryptedKeysFlag = &cli.StringFlag{
		Name:  "unencrypted-keys",
		Usage: "Filepath to a JSON file of unencrypted validator keys for easier launching of the validator client",
		Value: "",
	}
	// WalletDirFlag defines the path to a wallet directory for Prysm accounts-v2.
	WalletDirFlag = &cli.StringFlag{
		Name:  "wallet-dir",
		Usage: "Path to a wallet directory on-disk for Prysm validator accounts",
		Value: filepath.Join(DefaultValidatorDir(), consts.WalletDefaultDirName),
	}
	// WalletPasswordsDirFlag defines the path for a passwords directory for
	// Prysm accounts-v2.
	WalletPasswordsDirFlag = &cli.StringFlag{
		Name:  "passwords-dir",
		Usage: "Path to a directory on-disk where wallet passwords are stored",
		Value: filepath.Join(DefaultValidatorDir(), consts.PasswordsDefaultDirName),
	}
	// PasswordFileFlag is used to enter a file to get a password for new account creation, non-interactively.
	PasswordFileFlag = &cli.StringFlag{
		Name:  "password-file",
		Usage: "File to retrieve password for writing to the password dir when making a new account",
	}
	// MnemonicFileFlag is used to enter a file to mnemonic phrase for new wallet creation, non-interactively.
	MnemonicFileFlag = &cli.StringFlag{
		Name:  "mnemonic-file",
		Usage: "File to retrieve mnemonic for non-interactively passing a mnemonic phrase into wallet recover.",
	}
	// SkipMnemonicConfirmFlag is used to skip the withdrawal key mnemonic phrase prompt confirmation.
	SkipMnemonicConfirmFlag = &cli.BoolFlag{
		Name:  "skip-mnemonic-confirm",
		Usage: "Skip the withdrawal key mnemonic phrase prompt confirmation",
	}
	// ShowDepositDataFlag for accounts-v2.
	ShowDepositDataFlag = &cli.BoolFlag{
		Name:  "show-deposit-data",
		Usage: "Display raw eth1 tx deposit data for validator accounts-v2",
		Value: false,
	}
	// AccountsFlag for non-interactive usage of accounts exporting, sets a list of account names or all to be exported.
	AccountsFlag = &cli.StringSliceFlag{
		Name:  "accounts",
		Usage: "List of account names to export, or \"all\" to backup all accounts",
	}
	// BackupDirFlag defines the path for the zip backup of the wallet will be created.
	BackupDirFlag = &cli.StringFlag{
		Name:  "backup-dir",
		Usage: "Path to a directory where accounts will be exported into a zip file",
		Value: DefaultValidatorDir(),
	}
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
	// KeymanagerKindFlag defines the kind of keymanager desired by a user during wallet creation.
	KeymanagerKindFlag = &cli.StringFlag{
		Name:  "keymanager-kind",
		Usage: "Kind of keymanager, either direct, derived, or remote, specified during wallet creation",
		Value: "",
	}
)

// DefaultValidatorDir returns OS-specific default validator directory.
func DefaultValidatorDir() string {
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

// homeDir returns home directory path.
func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
