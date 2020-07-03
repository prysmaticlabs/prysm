package v2

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
)

const (
	keymanagerConfigFileName = "keymanageropts.json"
	walletDefaultDirName     = ".prysm-wallet-v2"
	passwordsDefaultDirName  = ".passwords"
)

// WalletConfig for a wallet struct, containing important information
// such as the passwords directory, the wallet's directory, and keymanager.
type WalletConfig struct {
	PasswordsDir   string
	WalletDir      string
	KeymanagerKind v2keymanager.Kind
}

// Wallet is a primitive in Prysm's v2 account management which
// has the capability of creating new accounts, reading existing accounts,
// and providing secure access to eth2 secrets depending on an
// associated keymanager (either direct, derived, or remote signing enabled).
type Wallet struct {
	accountsPath   string
	passwordsDir   string
	keymanagerKind v2keymanager.Kind
}

// CreateWallet given a set of configuration options, will leverage
// a keymanager to create and write a new wallet to disk for a Prysm validator.
func CreateWallet(ctx context.Context, cfg *WalletConfig) (*Wallet, error) {
	if cfg.WalletDir == "" || cfg.PasswordsDir == "" {
		return nil, errors.New("wallet dir and passwords dir cannot be nil")
	}
	accountsPath := path.Join(cfg.WalletDir, cfg.KeymanagerKind.String())
	if err := os.MkdirAll(accountsPath, os.ModePerm); err != nil {
		return nil, errors.Wrap(err, "could not create wallet directory")
	}
	if err := os.MkdirAll(cfg.PasswordsDir, os.ModePerm); err != nil {
		return nil, errors.Wrap(err, "could not create passwords directory")
	}
	w := &Wallet{
		accountsPath:   accountsPath,
		passwordsDir:   cfg.PasswordsDir,
		keymanagerKind: cfg.KeymanagerKind,
	}
	return w, nil
}

// OpenWallet instantiates a wallet from a specified path.
func OpenWallet(ctx context.Context, cfg *WalletConfig) (*Wallet, error) {
	walletPath := path.Join(cfg.WalletDir, cfg.KeymanagerKind.String())
	return &Wallet{
		accountsPath:   walletPath,
		passwordsDir:   cfg.PasswordsDir,
		keymanagerKind: cfg.KeymanagerKind,
	}, nil
}

// ReadKeymanagerConfigFromDisk opens a keymanager config file
// for reading if it exists at the wallet path.
func (w *Wallet) ReadKeymanagerConfigFromDisk(ctx context.Context) (io.ReadCloser, error) {
	if !fileExists(path.Join(w.accountsPath, keymanagerConfigFileName)) {
		return nil, fmt.Errorf("no keymanager config file found at path: %s", w.accountsPath)
	}
	configFilePath := path.Join(w.accountsPath, keymanagerConfigFileName)
	return os.Open(configFilePath)
}

// KeymanagerKind used by the wallet.
func (w *Wallet) KeymanagerKind() v2keymanager.Kind {
	return w.keymanagerKind
}

// AccountsPath for the wallet.
func (w *Wallet) AccountsPath() string {
	return w.accountsPath
}

// AccountPasswordsPath for the wallet's accounts.
func (w *Wallet) AccountPasswordsPath() string {
	return w.passwordsDir
}

// WriteAccountToDisk writes an encoded account by its filename
// within the wallet's directory.
func (w *Wallet) WriteAccountToDisk(ctx context.Context, filename string, encoded []byte) error {
	return errors.New("unimplemented")
}

// WriteKeymanagerConfigToDisk takes an encoded keymanager config file
// and writes it to the wallet path.
func (w *Wallet) WriteKeymanagerConfigToDisk(ctx context.Context, encoded []byte) error {
	configFilePath := path.Join(w.accountsPath, keymanagerConfigFileName)
	if fileExists(configFilePath) {
		return nil
	}
	// Open the keymanager config file for writing.
	f, err := os.Create(configFilePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatalf("Could not close keymanager opts file: %v", err)
		}
	}()
	n, err := f.Write(encoded)
	if err != nil {
		return err
	}
	if n != len(encoded) {
		return fmt.Errorf(
			"expected to write %d bytes to disk, but wrote %d",
			len(encoded),
			n,
		)
	}
	log.WithField("configFile", configFilePath).Debug("Wrote keymanager config file to disk")
	return nil
}

// Returns true if a file is not a directory and exists
// at the specified path.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
