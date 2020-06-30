package v2

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
)

const keymanagerConfigSuffix = "_keymanageropts.json"

var keymanagerPrefixes = map[WalletType]string{
	DirectWallet:  "direct",
	DerivedWallet: "derived",
	RemoteWallet:  "remoteWallet",
}

// WalletConfig for a wallet struct, containing important information
// such as the passwords directory, the wallet's directory, and keymanager.
type WalletConfig struct {
	PasswordsDir string
	WalletDir    string
	WalletType   WalletType
	Keymanager   v2keymanager.IKeymanager
}

// Wallet is a primitive in Prysm's v2 account management which
// has the capability of creating new accounts, reading existing accounts,
// and providing secure access to eth2 secrets depending on an
// associated keymanager (either direct, derived, or remote signing enabled).
type Wallet struct {
	walletPath   string
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
	walletPath := path.Join(cfg.WalletDir, keymanagerPrefixes[cfg.WalletType])
	if err := os.MkdirAll(walletPath, os.ModePerm); err != nil {
		return nil, errors.Wrap(err, "could not create wallet directory")
	}
	if err := os.MkdirAll(cfg.PasswordsDir, os.ModePerm); err != nil {
		return nil, errors.Wrap(err, "could not create passwords directory")
	}
	w := &Wallet{
		walletPath:   walletPath,
		passwordsDir: cfg.PasswordsDir,
		keymanager:   cfg.Keymanager,
		walletType:   cfg.WalletType,
	}
	// Writes the keymanager's configuration file to disk if not exists.
	if err := w.writeKeymanagerConfig(ctx); err != nil {
		return nil, err
	}
	return w, nil
}

// ReadWallet parses configuration options to initialize a wallet
// struct from a keymanager configuration file at the wallet's path.
func ReadWallet(ctx context.Context, cfg *WalletConfig) (*Wallet, error) {
	walletPath := path.Join(cfg.WalletDir, keymanagerPrefixes[cfg.WalletType])
	exists, err := fileSuffixExists(path.Join(walletPath, "*"+keymanagerConfigSuffix))
	if err != nil {
		return nil, fmt.Errorf("could not check keymanager config file exists at path: %s", walletPath)
	}
	if !exists {
		return nil, fmt.Errorf("no keymanager config file found at path: %s", walletPath)
	}
	keymanagerConfigFile := keymanagerPrefixes[cfg.WalletType] + keymanagerConfigSuffix
	configFilePath := path.Join(walletPath, keymanagerConfigFile)
	f, err := os.Open(configFilePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatalf("Could not close keymanager opts file: %v", err)
		}
	}()
	keymanager, err := v2keymanager.InitializeFromConfig(f)
	if err != nil {
		return nil, err
	}
	return &Wallet{
		walletPath:   walletPath,
		passwordsDir: cfg.PasswordsDir,
		keymanager:   keymanager,
		walletType:   cfg.WalletType,
	}, nil
}

// CreateAccount --
func (w *Wallet) CreateAccount(ctx context.Context, password string) error {
	return errors.New("unimplemented")
}

// Writes a keymanager configuration file to disk at the wallet path
// if such file does not yet exist.
func (w *Wallet) writeKeymanagerConfig(ctx context.Context) error {
	keymanagerConfigFile := keymanagerPrefixes[w.walletType] + keymanagerConfigSuffix
	configFilePath := path.Join(w.walletPath, keymanagerConfigFile)
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
	// Retrieve the marshaled keymanager config file contents.
	configFileContents, err := w.keymanager.ConfigFile(ctx)
	if err != nil {
		return err
	}
	n, err := f.Write(configFileContents)
	if err != nil {
		return err
	}
	if n != len(configFileContents) {
		return fmt.Errorf(
			"expected to write %d bytes to disk, but wrote %d",
			len(configFileContents),
			n,
		)
	}
	log.WithField("configFile", configFilePath).Debug("Wrote keymanager config file to disk")
	return nil
}

// Checks if a file suffix matches any files at a file path.
func fileSuffixExists(filePath string) (bool, error) {
	matches, err := filepath.Glob(filePath)
	if err != nil {
		return false, err
	}
	return len(matches) > 0, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
