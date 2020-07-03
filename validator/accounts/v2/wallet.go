package v2

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/pkg/errors"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/sirupsen/logrus"
)

const (
	keymanagerConfigFileName = "keymanageropts.json"
	walletDefaultDirName     = ".prysm-wallet-v2"
	passwordsDefaultDirName  = ".passwords"
	passwordFileSuffix       = ".pass"
	numAccountWords          = 3 // Number of words in account human-readable names.
	accountFilePermissions   = os.O_CREATE | os.O_RDWR
	directoryPermissions     = os.ModePerm
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

func init() {
	petname.NonDeterministicMode() // Set random account name generation.
}

// CreateWallet given a set of configuration options, will leverage
// a keymanager to create and write a new wallet to disk for a Prysm validator.
func CreateWallet(ctx context.Context, cfg *WalletConfig) (*Wallet, error) {
	if cfg.WalletDir == "" || cfg.PasswordsDir == "" {
		return nil, errors.New("wallet dir and passwords dir cannot be nil")
	}
	accountsPath := path.Join(cfg.WalletDir, cfg.KeymanagerKind.String())
	if err := os.MkdirAll(accountsPath, directoryPermissions); err != nil {
		return nil, errors.Wrap(err, "could not create wallet directory")
	}
	if err := os.MkdirAll(cfg.PasswordsDir, directoryPermissions); err != nil {
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

// AccountsDir for the wallet.
func (w *Wallet) AccountsDir() string {
	return w.accountsPath
}

// WriteAccountToDisk creates an account directory under a unique namespace
// within the wallet's path. It additionally writes the account's password to the
// wallet's passwords directory. Returns the unique account name.
func (w *Wallet) WriteAccountToDisk(ctx context.Context, password string) (string, error) {
	accountName, err := w.generateAccountName()
	if err != nil {
		return "", errors.Wrap(err, "could not generate unique account name")
	}
	// Generate a directory for the new account name and
	// write its associated password to disk.
	accountPath := path.Join(w.accountsPath, accountName)
	if err := os.MkdirAll(accountPath, directoryPermissions); err != nil {
		return "", errors.Wrap(err, "could not create account directory")
	}
	if err := w.writePasswordToFile(accountName, password); err != nil {
		return "", errors.Wrap(err, "could not write password to disk")
	}
	return accountName, nil
}

// WriteFileForAccount stores a unique ficle and its data under an account namespace
// in the wallet's directory on-disk. Creates the file if it does not exist
// and writes over it otherwise.
func (w *Wallet) WriteFileForAccount(ctx context.Context, accountName string, fileName string, data []byte) error {
	accountPath := path.Join(w.accountsPath, accountName)
	exists, err := hasDir(accountPath)
	if err != nil {
		return errors.Wrapf(err, "could not check if account exists in directory: %s", w.accountsPath)
	}
	if !exists {
		return errors.Wrapf(err, "account does not exist in wallet directory: %s", w.accountsPath)
	}
	filePath := path.Join(accountPath, fileName)
	f, err := os.OpenFile(filePath, accountFilePermissions, directoryPermissions)
	if err != nil {
		return errors.Wrapf(err, "could not open file for account: %s", filePath)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.WithError(err).Error("Could not close file after writing")
		}
	}()
	n, err := f.Write(data)
	if err != nil {
		return errors.Wrapf(err, "could not write file for account: %s", filePath)
	}
	if n != len(data) {
		return fmt.Errorf("could only write %d/%d file bytes to disk", n, len(data))
	}
	log.WithFields(logrus.Fields{
		"accountName": accountName,
		"filePath":    filePath,
	}).Debug("Wrote new file for account")
	return nil
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
			log.WithError(err).Error("Could not close keymanager opts file")
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

// Writes the password file for an account namespace in the wallet's passwords directory.
func (w *Wallet) writePasswordToFile(accountName string, password string) error {
	passwordFilePath := path.Join(w.passwordsDir, accountName+passwordFileSuffix)
	passwordFile, err := os.OpenFile(passwordFilePath, accountFilePermissions, directoryPermissions)
	if err != nil {
		return errors.Wrapf(err, "could not create password file in directory: %s", w.passwordsDir)
	}
	defer func() {
		if err := passwordFile.Close(); err != nil {
			log.WithError(err).Error("Could not close password file")
		}
	}()
	n, err := passwordFile.WriteString(password)
	if err != nil {
		return errors.Wrap(err, "could not write account password to disk")
	}
	if n != len(password) {
		return fmt.Errorf("could only write %d/%d password bytes to disk", n, len(password))
	}
	return nil
}

// Generates a human-readable name for an account. Checks for uniqueness in the accounts path.
func (w *Wallet) generateAccountName() (string, error) {
	var accountExists bool
	var accountName string
	for !accountExists {
		accountName = petname.Generate(numAccountWords, "-" /* separator */)
		exists, err := hasDir(path.Join(w.accountsPath, accountName))
		if err != nil {
			return "", errors.Wrapf(err, "could not check if account exists in dir: %s", w.accountsPath)
		}
		if !exists {
			break
		}
	}
	return accountName, nil
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

func hasDir(dirPath string) (bool, error) {
	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
