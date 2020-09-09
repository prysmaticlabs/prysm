// Package flags contains all configuration runtime flags for
// the validator service.
package flags

import (
	"path/filepath"
	"runtime"
	"time"

	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const (
	// WalletDefaultDirName for accounts-v2.
	WalletDefaultDirName = "prysm-wallet-v2"
)

var log = logrus.WithField("prefix", "flags")

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
	// GRPCGatewayHost specifies a gRPC gateway host for the validator client.
	GRPCGatewayHost = &cli.StringFlag{
		Name:  "grpc-gateway-host",
		Usage: "The host on which the gateway server runs on",
		Value: "127.0.0.1",
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
		Value: "http://localhost:4200",
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
		Value: filepath.Join(DefaultValidatorDir(), WalletDefaultDirName),
	}
	// AccountPasswordFileFlag is path to a file containing a password for a validator account.
	AccountPasswordFileFlag = &cli.StringFlag{
		Name:  "account-password-file",
		Usage: "Path to a plain-text, .txt file containing a password for a validator account",
	}
	// WalletPasswordFileFlag is the path to a file containing your wallet password.
	WalletPasswordFileFlag = &cli.StringFlag{
		Name:  "wallet-password-file",
		Usage: "Path to a plain-text, .txt file containing your wallet password",
	}
	// ImportPrivateKeyFileFlag allows for directly importing a private key hex string as an account.
	ImportPrivateKeyFileFlag = &cli.StringFlag{
		Name:  "import-private-key-file",
		Usage: "Path to a plain-text, .txt file containing a hex string representation of a private key to import",
	}
	// MnemonicFileFlag is used to enter a file to mnemonic phrase for new wallet creation, non-interactively.
	MnemonicFileFlag = &cli.StringFlag{
		Name:  "mnemonic-file",
		Usage: "File to retrieve mnemonic for non-interactively passing a mnemonic phrase into wallet recover.",
	}
	// ShowDepositDataFlag for accounts-v2.
	ShowDepositDataFlag = &cli.BoolFlag{
		Name:  "show-deposit-data",
		Usage: "Display raw eth1 tx deposit data for validator accounts-v2",
		Value: false,
	}
	// NumAccountsFlag defines the amount of accounts to generate for derived wallets.
	NumAccountsFlag = &cli.Int64Flag{
		Name:  "num-accounts",
		Usage: "Number of accounts to generate for derived wallets",
		Value: 1,
	}
	// DeletePublicKeysFlag defines a comma-separated list of hex string public keys
	// for accounts which a user desires to delete from their wallet.
	DeletePublicKeysFlag = &cli.StringFlag{
		Name:  "delete-public-keys",
		Usage: "Comma-separated list of public key hex strings to specify which validator accounts to delete",
		Value: "",
	}
	// BackupPublicKeysFlag defines a comma-separated list of hex string public keys
	// for accounts which a user desires to backup from their wallet.
	BackupPublicKeysFlag = &cli.StringFlag{
		Name:  "backup-public-keys",
		Usage: "Comma-separated list of public key hex strings to specify which validator accounts to backup",
		Value: "",
	}
	// VoluntaryExitPublicKeysFlag defines a comma-separated list of hex string public keys
	// for accounts on which a user wants to perform a voluntary exit.
	VoluntaryExitPublicKeysFlag = &cli.StringFlag{
		Name: "public-keys",
		Usage: "Comma-separated list of public key hex strings to specify on which validator accounts to perform " +
			"a voluntary exit",
		Value: "",
	}
	// BackupPasswordFile for encrypting accounts a user wishes to back up.
	BackupPasswordFile = &cli.StringFlag{
		Name:  "backup-password-file",
		Usage: "Path to a plain-text, .txt file containing the desired password for your backed up accounts",
		Value: "",
	}
	// BackupDirFlag defines the path for the zip backup of the wallet will be created.
	BackupDirFlag = &cli.StringFlag{
		Name:  "backup-dir",
		Usage: "Path to a directory where accounts will be backed up into a zip file",
		Value: DefaultValidatorDir(),
	}
	// KeysDirFlag defines the path for a directory where keystores to be imported at stored.
	KeysDirFlag = &cli.StringFlag{
		Name:  "keys-dir",
		Usage: "Path to a directory where keystores to be imported are stored",
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
	// Eth1KeystoreUTCPathFlag defines the path to an eth1 utc keystore containing eth1 private keys.
	Eth1KeystoreUTCPathFlag = &cli.StringFlag{
		Name:  "eth1-keystore-utc-path",
		Usage: "Path to an eth1 utc keystore containing eth1 private keys",
		Value: "",
	}
	// Eth1KeystorePasswordFileFlag to unlock an eth1 keystores.
	Eth1KeystorePasswordFileFlag = &cli.StringFlag{
		Name:  "eth1-keystore-password-file",
		Value: "",
		Usage: "Password file for unlock account",
	}
	// HTTPWeb3ProviderFlag provides an HTTP access endpoint to an ETH 1.0 RPC.
	HTTPWeb3ProviderFlag = &cli.StringFlag{
		Name:  "http-web3provider",
		Usage: "An eth1 web3 provider string http endpoint",
		Value: "https://goerli.prylabs.net",
	}
	// Eth1PrivateKeyFileFlag containing a hex string for sending deposit transactions from eth1.
	Eth1PrivateKeyFileFlag = &cli.StringFlag{
		Name:  "eth1-private-key-file",
		Usage: "File containing a private key for sending deposit transactions from eth1",
		Value: "",
	}
	// DepositDelaySecondsFlag to delay sending deposit transactions by a fixed interval.
	DepositDelaySecondsFlag = &cli.Int64Flag{
		Name:  "deposit-delay-seconds",
		Usage: "The time delay between sending the deposits to the contract (in seconds)",
		Value: 5,
	}
	// DepositContractAddressFlag for the validator deposit contract on eth1.
	DepositContractAddressFlag = &cli.StringFlag{
		Name:  "deposit-contract",
		Usage: "Address of the deposit contract",
		Value: "0x07b39F4fDE4A38bACe212b546dAc87C58DfE3fDC", // Medalla deposit contract address.
	}
	// DepositPublicKeysFlag for validating public keys a user wishes to deposit for.
	DepositPublicKeysFlag = &cli.StringFlag{
		Name:  "deposit-public-keys",
		Usage: "Comma-separated list of validating public key hex strings to specify which validator accounts to deposit",
		Value: "",
	}
	// SkipDepositConfirmationFlag skips the y/n confirmation prompt for sending a deposit to the deposit contract.
	SkipDepositConfirmationFlag = &cli.BoolFlag{
		Name:  "skip-deposit-confirmation",
		Usage: "Skips the y/n confirmation prompt for sending a deposit to the deposit contract",
		Value: false,
	}
	// DepositAllAccountsFlag is an easy way for a user to send deposit transactions for all accounts in their wallet.
	DepositAllAccountsFlag = &cli.BoolFlag{
		Name:  "deposit-all-accounts",
		Usage: "Sends a 32 ETH deposit for each of a user's validator accounts in their wallet",
		Value: false,
	}
	// EnableWebFlag enables controlling the validator client via the Prysm web ui. This is a work in progress.
	EnableWebFlag = &cli.BoolFlag{
		Name:  "web",
		Usage: "Enables the web portal for the validator client (work in progress)",
		Value: false,
	}
)

// Deprecated flags list.
const deprecatedUsage = "DEPRECATED. DO NOT USE."

var (
	// DeprecatedPasswordsDirFlag is a deprecated flag.
	DeprecatedPasswordsDirFlag = &cli.StringFlag{
		Name:   "passwords-dir",
		Usage:  deprecatedUsage,
		Hidden: true,
	}
)

// DeprecatedFlags is a slice holding all of the validator client's deprecated flags.
var DeprecatedFlags = []cli.Flag{
	DeprecatedPasswordsDirFlag,
}

// ComplainOnDeprecatedFlags logs out a error log if a deprecated flag is used, letting the user know it will be removed soon.
func ComplainOnDeprecatedFlags(ctx *cli.Context) {
	for _, f := range DeprecatedFlags {
		if ctx.IsSet(f.Names()[0]) {
			log.Errorf("%s is deprecated and has no effect. Do not use this flag, it will be deleted soon.", f.Names()[0])
		}
	}
}

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
