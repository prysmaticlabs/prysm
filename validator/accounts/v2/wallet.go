package v2

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/pkg/errors"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

const (
	// WalletDefaultDirName for accounts-v2.
	WalletDefaultDirName = ".prysm-wallet-v2"
	// PasswordsDefaultDirName where account passwords are stored.
	PasswordsDefaultDirName  = ".prysm-wallet-v2-passwords"
	keymanagerConfigFileName = "keymanageropts.json"
	passwordFileSuffix       = ".pass"
	numAccountWords          = 3 // Number of words in account human-readable names.
	accountFilePermissions   = os.O_CREATE | os.O_RDWR
	directoryPermissions     = os.ModePerm
)

var (
	// ErrNoWalletFound signifies there is no data at the given wallet path.
	ErrNoWalletFound = errors.New("no wallet found at path")
)

// WalletConfig for a wallet struct, containing important information
// such as the passwords directory, the wallet's directory, and keymanager.
type WalletConfig struct {
	PasswordsDir      string
	WalletDir         string
	KeymanagerKind    v2keymanager.Kind
	CanUnlockAccounts bool
}

// Wallet is a primitive in Prysm's v2 account management which
// has the capability of creating new accounts, reading existing accounts,
// and providing secure access to eth2 secrets depending on an
// associated keymanager (either direct, derived, or remote signing enabled).
type Wallet struct {
	accountsPath      string
	passwordsDir      string
	canUnlockAccounts bool
	keymanagerKind    v2keymanager.Kind
}

func init() {
	petname.NonDeterministicMode() // Set random account name generation.
}

// CreateWallet given a set of configuration options, will leverage
// a keymanager to create and write a new wallet to disk for a Prysm validator.
func CreateWallet(ctx context.Context, cfg *WalletConfig) (*Wallet, error) {
	if cfg.WalletDir == "" || (cfg.CanUnlockAccounts && cfg.PasswordsDir == "") {
		return nil, errors.New("wallet dir and passwords dir cannot be nil")
	}
	accountsPath := path.Join(cfg.WalletDir, cfg.KeymanagerKind.String())
	if err := os.MkdirAll(accountsPath, directoryPermissions); err != nil {
		return nil, errors.Wrap(err, "could not create wallet directory")
	}
	if cfg.PasswordsDir != "" {
		if err := os.MkdirAll(cfg.PasswordsDir, directoryPermissions); err != nil {
			return nil, errors.Wrap(err, "could not create passwords directory")
		}
	}
	w := &Wallet{
		accountsPath:      accountsPath,
		passwordsDir:      cfg.PasswordsDir,
		keymanagerKind:    cfg.KeymanagerKind,
		canUnlockAccounts: cfg.CanUnlockAccounts,
	}
	return w, nil
}

// OpenWallet instantiates a wallet from a specified path. It checks the
// type of keymanager associated with the wallet by reading files in the wallet
// path, if applicable. If a wallet does not exist, returns an appropriate error.
func OpenWallet(ctx context.Context, cfg *WalletConfig) (*Wallet, error) {
	ok, err := hasDir(cfg.WalletDir)
	if err != nil {
		return nil, errors.Wrapf(err, "could not check if wallet exists at %s", cfg.WalletDir)
	}
	if !ok {
		return nil, ErrNoWalletFound
	}
	walletPath := path.Join(cfg.WalletDir, cfg.KeymanagerKind.String())
	walletDir, err := os.Open(cfg.WalletDir)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := walletDir.Close(); err != nil {
			log.WithField(
				"path", walletPath,
			).Errorf("Could not close wallet directory: %v", err)
		}
	}()
	// Retrieve the type of keymanager the wallet uses by looking at
	// directories in its directory path.
	list, err := walletDir.Readdirnames(0) // 0 to read all files and folders.
	if err != nil {
		return nil, errors.Wrapf(err, "could not read files in directory: %s", walletPath)
	}
	if len(list) != 1 {
		return nil, fmt.Errorf("expected a single directory in the wallet path: %s", walletPath)
	}
	keymanagerKind, err := v2keymanager.ParseKind(list[0])
	if err != nil {
		return nil, errors.Wrap(err, "could not parse keymanager kind from wallet path")
	}
	return &Wallet{
		accountsPath:      walletPath,
		passwordsDir:      cfg.PasswordsDir,
		keymanagerKind:    keymanagerKind,
		canUnlockAccounts: cfg.CanUnlockAccounts,
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

// CanUnlockAccounts determines whether a wallet has capabilities
// of unlocking validator accounts using passphrases.
func (w *Wallet) CanUnlockAccounts() bool {
	return w.canUnlockAccounts
}

// AccountNames reads all account names at the wallet's path.
func (w *Wallet) AccountNames() ([]string, error) {
	accountsDir, err := os.Open(w.accountsPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := accountsDir.Close(); err != nil {
			log.WithField(
				"directory", w.accountsPath,
			).Errorf("Could not close accounts directory: %v", err)
		}
	}()

	list, err := accountsDir.Readdirnames(0) // 0 to read all files and folders.
	if err != nil {
		return nil, errors.Wrapf(err, "could not read files in directory: %s", w.accountsPath)
	}
	accountNames := make([]string, 0)
	for _, item := range list {
		ok, err := hasDir(path.Join(w.accountsPath, item))
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse directory: %v", err)
		}
		if ok {
			accountNames = append(accountNames, item)
		}
	}
	return accountNames, err
}

// ExistingKeyManager reads a keymanager config from disk at the wallet path,
// unmarshals it based on the wallet's keymanager kind, and returns its value.
func (w *Wallet) ExistingKeyManager(
	ctx context.Context,
	skipMnemonicConfirm bool,
) (v2keymanager.IKeymanager, error) {
	configFile, err := w.ReadKeymanagerConfigFromDisk(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not read keymanager config")
	}
	var keymanager v2keymanager.IKeymanager
	switch w.KeymanagerKind() {
	case v2keymanager.Direct:
		cfg, err := direct.UnmarshalConfigFile(configFile)
		if err != nil {
			return nil, errors.Wrap(err, "could not unmarshal keymanager config file")
		}
		keymanager, err = direct.NewKeymanager(ctx, w, cfg, skipMnemonicConfirm)
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize keymanager")
		}
	case v2keymanager.Derived:
		return nil, errors.New("derived keymanager is unimplemented, work in progress")
	case v2keymanager.Remote:
		return nil, errors.New("remote keymanager is unimplemented, work in progress")
	default:
		return nil, errors.New("keymanager kind must be specified")
	}
	return keymanager, nil
}

// CreateKeymanager determines if a config file exists in the wallet, it
// reads the config file and initializes the keymanager that way. Otherwise,
// writes a new configuration file to the wallet and returns the initialized
// keymanager for use.
func (w *Wallet) CreateKeymanager(ctx context.Context, skipMnemonicConfirm bool) (v2keymanager.IKeymanager, error) {
	var keymanager v2keymanager.IKeymanager
	var err error
	switch w.KeymanagerKind() {
	case v2keymanager.Direct:
		keymanager, err = direct.NewKeymanager(ctx, w, direct.DefaultConfig(), skipMnemonicConfirm)
		if err != nil {
			return nil, errors.Wrap(err, "could not read keymanager")
		}
	case v2keymanager.Derived:
		return nil, errors.New("derived keymanager is unimplemented, work in progress")
	case v2keymanager.Remote:
		return nil, errors.New("remote keymanager is unimplemented, work in progress")
	default:
		return nil, errors.New("keymanager type must be specified")
	}
	keymanagerConfig, err := keymanager.MarshalConfigFile(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal keymanager config file")
	}
	if err := w.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
		return nil, errors.Wrap(err, "could not write keymanager config file to disk")
	}
	return keymanager, nil
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

// WriteFileForAccount stores a unique file and its data under an account namespace
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
		"name": accountName,
		"path": filePath,
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

// ReadPasswordForAccount when given an account name from the wallet's passwords' path.
func (w *Wallet) ReadPasswordForAccount(accountName string) (string, error) {
	if !w.canUnlockAccounts {
		return "", errors.New("wallet has no permission to read account passwords")
	}
	passwordFilePath := path.Join(w.passwordsDir, accountName+passwordFileSuffix)
	passwordFile, err := os.Open(passwordFilePath)
	if err != nil {
		return "", errors.Wrapf(err, "could not read password file from directory: %s", w.passwordsDir)
	}
	defer func() {
		if err := passwordFile.Close(); err != nil {
			log.Errorf("Could not close password file: %s", passwordFilePath)
		}
	}()
	password, err := ioutil.ReadAll(passwordFile)
	if err != nil {
		return "", errors.Wrapf(err, "could not read data from password file: %s", passwordFilePath)
	}
	return string(password), nil
}

func (w *Wallet) enterPasswordForAccount(cliCtx *cli.Context, accountName string) error {
	attemptingPassword := true
	// Loop asking for the password until the user enters it correctly.
	for attemptingPassword {
		// Ask the user for the password to their account.
		password, err := inputPasswordForAccount(cliCtx, accountName)
		if err != nil {
			return errors.Wrap(err, "could not input password")
		}
		accountKeystore, err := w.keystoreForAccount(accountName)
		if err != nil {
			return errors.Wrap(err, "could not get keystore")
		}
		decryptor := keystorev4.New()
		_, err = decryptor.Decrypt(accountKeystore.Crypto, []byte(password))
		if err != nil && strings.Contains(err.Error(), "invalid checksum") {
			fmt.Println("Incorrect password entered, please try again")
			continue
		}
		if err != nil {
			return errors.Wrap(err, "could not decrypt keystore")
		}

		if err := w.writePasswordToFile(accountName, password); err != nil {
			return errors.Wrap(err, "could not write password to disk")
		}
		attemptingPassword = false
	}
	return nil
}

func (w *Wallet) publicKeyForAccount(accountName string) ([48]byte, error) {
	accountKeystore, err := w.keystoreForAccount(accountName)
	if err != nil {
		return [48]byte{}, errors.Wrap(err, "could not get keystore")
	}
	pubKey, err := hex.DecodeString(accountKeystore.Pubkey)
	if err != nil {
		return [48]byte{}, errors.Wrap(err, "could decode pubkey string")
	}
	return bytesutil.ToBytes48(pubKey), nil
}

func (w *Wallet) keystoreForAccount(accountName string) (*direct.Keystore, error) {
	encoded, err := w.ReadFileForAccount(accountName, direct.KeystoreFileName)
	if err != nil {
		return nil, errors.Wrap(err, "could not read keystore file")
	}
	keystoreJSON := &direct.Keystore{}
	if err := json.Unmarshal(encoded, &keystoreJSON); err != nil {
		return nil, errors.Wrap(err, "could not decode json")
	}
	return keystoreJSON, nil
}

// ReadFileForAccount from the wallet's accounts directory.
func (w *Wallet) ReadFileForAccount(accountName string, fileName string) ([]byte, error) {
	accountPath := path.Join(w.accountsPath, accountName)
	exists, err := hasDir(accountPath)
	if err != nil {
		return nil, errors.Wrapf(err, "could not check if account exists in directory: %s", w.accountsPath)
	}
	if !exists {
		return nil, errors.Wrapf(err, "account does not exist in wallet directory: %s", w.accountsPath)
	}
	filePath := path.Join(accountPath, fileName)
	f, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read file for account: %s", filePath)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Errorf("Could not close file after writing: %s", filePath)
		}
	}()
	return ioutil.ReadAll(f)
}

// Writes the password file for an account namespace in the wallet's passwords directory.
func (w *Wallet) writePasswordToFile(accountName string, password string) error {
	passwordFilePath := path.Join(w.passwordsDir, accountName+passwordFileSuffix)
	// Removing any file that exists to make sure the existing is overwritten.
	if _, err := os.Stat(passwordFilePath); os.IsExist(err) {
		if err := os.Remove(passwordFilePath); err != nil {
			return errors.Wrap(err, "could not rewrite password file")
		}
	}
	passwordFile, err := os.Create(passwordFilePath)
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

// Checks if a directory indeed exists at the specified path.
func hasDir(dirPath string) (bool, error) {
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return info.IsDir(), err
}
