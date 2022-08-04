package accounts

import (
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/remote"
	"google.golang.org/grpc"
)

// Option type for configuring the accounts cli manager.
type Option func(acc *AccountsCLIManager) error

// WithWallet provides a wallet to the accounts cli manager.
func WithWallet(wallet *wallet.Wallet) Option {
	return func(acc *AccountsCLIManager) error {
		acc.wallet = wallet
		return nil
	}
}

// WithKeymanager provides a keymanager to the accounts cli manager.
func WithKeymanager(km keymanager.IKeymanager) Option {
	return func(acc *AccountsCLIManager) error {
		acc.keymanager = km
		return nil
	}
}

// WithKeymanagerOpts provides a keymanager configuration to the accounts cli manager.
func WithKeymanagerOpts(kmo *remote.KeymanagerOpts) Option {
	return func(acc *AccountsCLIManager) error {
		acc.keymanagerOpts = kmo
		return nil
	}
}

// WithShowDepositData enables displaying deposit data in the accounts cli manager.
func WithShowDepositData() Option {
	return func(acc *AccountsCLIManager) error {
		acc.showDepositData = true
		return nil
	}
}

// WithShowPrivateKeys enables displaying private keys in the accounts cli manager.
func WithShowPrivateKeys() Option {
	return func(acc *AccountsCLIManager) error {
		acc.showPrivateKeys = true
		return nil
	}
}

// WithListValidatorIndices enables displaying validator indices in the accounts cli manager.
func WithListValidatorIndices() Option {
	return func(acc *AccountsCLIManager) error {
		acc.listValidatorIndices = true
		return nil
	}
}

// WithGRPCDialOpts adds grpc opts needed to connect to beacon nodes in the accounts cli manager.
func WithGRPCDialOpts(opts []grpc.DialOption) Option {
	return func(acc *AccountsCLIManager) error {
		acc.dialOpts = opts
		return nil
	}
}

// WithGRPCHeaders adds grpc headers used when connecting to beacon nodes in the accounts cli manager.
func WithGRPCHeaders(headers []string) Option {
	return func(acc *AccountsCLIManager) error {
		acc.grpcHeaders = headers
		return nil
	}
}

// WithBeaconRPCProvider provides a beacon node endpoint to the accounts cli manager.
func WithBeaconRPCProvider(provider string) Option {
	return func(acc *AccountsCLIManager) error {
		acc.beaconRPCProvider = provider
		return nil
	}
}

// WithWalletKeyCount tracks the number of keys in a wallet.
func WithWalletKeyCount(walletKeyCount int) Option {
	return func(acc *AccountsCLIManager) error {
		acc.walletKeyCount = walletKeyCount
		return nil
	}
}

// WithDeletePublicKeys indicates whether to delete the public keys.
func WithDeletePublicKeys(deletePublicKeys bool) Option {
	return func(acc *AccountsCLIManager) error {
		acc.deletePublicKeys = deletePublicKeys
		return nil
	}
}

// WithReadPasswordFile indicates whether to read the password from a file.
func WithReadPasswordFile(readPasswordFile bool) Option {
	return func(acc *AccountsCLIManager) error {
		acc.readPasswordFile = readPasswordFile
		return nil
	}
}

// WithImportPrivateKeys indicates whether to import private keys as accounts.
func WithImportPrivateKeys(importPrivateKeys bool) Option {
	return func(acc *AccountsCLIManager) error {
		acc.importPrivateKeys = importPrivateKeys
		return nil
	}
}

// WithPrivateKeyFile specifies the private key path.
func WithPrivateKeyFile(privateKeyFile string) Option {
	return func(acc *AccountsCLIManager) error {
		acc.privateKeyFile = privateKeyFile
		return nil
	}
}

// WithKeysDir specifies the directory keys are read from.
func WithKeysDir(keysDir string) Option {
	return func(acc *AccountsCLIManager) error {
		acc.keysDir = keysDir
		return nil
	}
}

// WithPasswordFilePath specifies where the password is stored.
func WithPasswordFilePath(passwordFilePath string) Option {
	return func(acc *AccountsCLIManager) error {
		acc.passwordFilePath = passwordFilePath
		return nil
	}
}

// WithBackupsDir specifies the directory backups are written to.
func WithBackupsDir(backupsDir string) Option {
	return func(acc *AccountsCLIManager) error {
		acc.backupsDir = backupsDir
		return nil
	}
}

// WithBackupsPassword specifies the password for backups.
func WithBackupsPassword(backupsPassword string) Option {
	return func(acc *AccountsCLIManager) error {
		acc.backupsPassword = backupsPassword
		return nil
	}
}

// WithFilteredPubKeys adds public key strings parsed from CLI.
func WithFilteredPubKeys(filteredPubKeys []bls.PublicKey) Option {
	return func(acc *AccountsCLIManager) error {
		acc.filteredPubKeys = filteredPubKeys
		return nil
	}
}

// WithRawPubKeys adds raw public key bytes parsed from CLI.
func WithRawPubKeys(rawPubKeys [][]byte) Option {
	return func(acc *AccountsCLIManager) error {
		acc.rawPubKeys = rawPubKeys
		return nil
	}
}

// WithFormattedPubKeys adds formatted public key strings parsed from CLI.
func WithFormattedPubKeys(formattedPubKeys []string) Option {
	return func(acc *AccountsCLIManager) error {
		acc.formattedPubKeys = formattedPubKeys
		return nil
	}
}
