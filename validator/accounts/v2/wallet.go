package v2

import (
	"context"
	"os"
	"path"

	"github.com/pkg/errors"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
)

const keymanagerOptsSuffix = "_keymanageropts.json"

var keymanagerOptsPrefixes = map[WalletType]string{
	directWallet:  "direct",
	derivedWallet: "derived",
	remoteWallet:  "remoteWallet",
}

// WalletConfig --
type WalletConfig struct {
	PasswordsDir string
	WalletDir    string
	WalletType   WalletType
	Keymanager   v2keymanager.IKeymanager
}

// Wallet --
type Wallet struct {
	walletDir    string
	passwordsDir string
	walletType   WalletType
	keymanager   v2keymanager.IKeymanager
}

// CreateWallet given a set of configuration options, will leverage
// a keymanager to create and write a new wallet to disk for a Prysm validator.
func CreateWallet(ctx context.Context, cfg *WalletConfig) (*Wallet, error) {
	if cfg.WalletDir == "" || cfg.PasswordsDir == "" {
		return nil, errors.New("wallet dir and passwords dir cannot be nil")
	}
	if err := os.MkdirAll(cfg.WalletDir, os.ModePerm); err != nil {
		return nil, errors.Wrap(err, "could not create wallet directory")
	}
	if err := os.MkdirAll(cfg.PasswordsDir, os.ModePerm); err != nil {
		return nil, errors.Wrap(err, "could not create passwords directory")
	}

	return &Wallet{
		walletDir:    cfg.WalletDir,
		passwordsDir: cfg.PasswordsDir,
		keymanager:   cfg.Keymanager,
		walletType:   cfg.WalletType,
	}, nil
}

// ReadWallet --
func ReadWallet(ctx context.Context, walletPath string) (*Wallet, error) {
	return &Wallet{}, errors.New("unimplemented")
}

// CreateAccount --
func (w *Wallet) CreateAccount(ctx context.Context, password string) error {
	// Writes the keymanager's configuration file to disk if not exists.
	keymanagerOptsFileName := keymanagerOptsPrefixes[w.walletType] + keymanagerOptsSuffix
	if !keymanagerFileExists(keymanagerOptsFileName) {
		f, err := os.Open(path.Join(w.walletDir, keymanagerOptsFileName))
		if err != nil {
			return err
		}
	}
	// Writes the account to disk at a path with a human readable name.
	return errors.New("unimplemented")
}

func keymanagerFileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
