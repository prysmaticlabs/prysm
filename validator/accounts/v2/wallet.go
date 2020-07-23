package v2

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	// ErrNoWalletFound signifies there was no wallet directory found on-disk.
	ErrNoWalletFound = errors.New(
		"no wallet found at path, please create a new wallet using `./prysm.sh validator wallet-v2 create`",
	)
	keymanagerKindSelections = map[v2keymanager.Kind]string{
		v2keymanager.Derived: "HD Wallet (Recommended)",
		v2keymanager.Direct:  "Non-HD Wallet (Most Basic)",
		v2keymanager.Remote:  "Remote Signing Wallet (Advanced)",
	}
)

const (
	// WalletDefaultDirName for accounts-v2.
	WalletDefaultDirName = ".prysm-wallet-v2"
	// PasswordsDefaultDirName where account passwords are stored.
	PasswordsDefaultDirName = ".prysm-wallet-v2-passwords"
	// KeymanagerConfigFileName for the keymanager used by the wallet: direct, derived, or remote.
	KeymanagerConfigFileName = "keymanageropts.json"
	// DirectoryPermissions for directories created under the wallet path.
	DirectoryPermissions = os.ModePerm
)

// Wallet is a primitive in Prysm's v2 account management which
// has the capability of creating new accounts, reading existing accounts,
// and providing secure access to eth2 secrets depending on an
// associated keymanager (either direct, derived, or remote signing enabled).
type Wallet struct {
	accountsPath   string
	passwordsDir   string
	keymanagerKind v2keymanager.Kind
	walletPassword string
}

func init() {
	petname.NonDeterministicMode() // Set random account name generation.
}

// NewWallet given a set of configuration options, will leverage
// create and write a new wallet to disk for a Prysm validator.
func NewWallet(
	cliCtx *cli.Context,
	keymanagerKind v2keymanager.Kind,
) (*Wallet, error) {
	walletDir, err := inputDirectory(cliCtx, walletDirPromptText, flags.WalletDirFlag)
	if err != nil && !errors.Is(err, ErrNoWalletFound) {
		return nil, errors.Wrap(err, "could not parse wallet directory")
	}
	// Check if the user has a wallet at the specified path.
	// If a user does not have a wallet, we instantiate one
	// based on specified options.
	walletExists, err := hasDir(walletDir)
	if err != nil {
		return nil, errors.Wrap(err, "could not check if wallet exists")
	}
	if walletExists {
		return nil, errors.New(
			"you already have a wallet at the specified path. You can " +
				"edit your wallet configuration by running ./prysm.sh validator wallet-v2 edit",
		)
	}
	accountsPath := filepath.Join(walletDir, keymanagerKind.String())
	if err := os.MkdirAll(accountsPath, DirectoryPermissions); err != nil {
		return nil, errors.Wrap(err, "could not create wallet directory")
	}
	w := &Wallet{
		accountsPath:   accountsPath,
		keymanagerKind: keymanagerKind,
	}
	if keymanagerKind == v2keymanager.Derived {
		walletPassword, err := inputPassword(cliCtx, newWalletPasswordPromptText, confirmPass)
		if err != nil {
			return nil, err
		}
		w.walletPassword = walletPassword
	}
	if keymanagerKind == v2keymanager.Direct {
		passwordsDir, err := inputDirectory(cliCtx, passwordsDirPromptText, flags.WalletPasswordsDirFlag)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(passwordsDir, DirectoryPermissions); err != nil {
			return nil, errors.Wrap(err, "could not create passwords directory")
		}
		w.passwordsDir = passwordsDir
	}
	return w, nil
}

// OpenWallet instantiates a wallet from a specified path. It checks the
// type of keymanager associated with the wallet by reading files in the wallet
// path, if applicable. If a wallet does not exist, returns an appropriate error.
func OpenWallet(cliCtx *cli.Context) (*Wallet, error) {
	// Read a wallet's directory from user input.
	walletDir, err := inputDirectory(cliCtx, walletDirPromptText, flags.WalletDirFlag)
	if errors.Is(err, ErrNoWalletFound) {
		return nil, errors.New("no wallet found, create a new one with ./prysm.sh validator wallet-v2 create")
	} else if err != nil {
		return nil, err
	}
	keymanagerKind, err := readKeymanagerKindFromWalletPath(walletDir)
	if err != nil {
		return nil, errors.Wrap(err, "could not read keymanager kind for wallet")
	}
	walletPath := filepath.Join(walletDir, keymanagerKind.String())
	w := &Wallet{
		accountsPath:   walletPath,
		keymanagerKind: keymanagerKind,
	}
	if keymanagerKind == v2keymanager.Derived {
		walletPassword, err := inputPassword(cliCtx, walletPasswordPromptText, noConfirmPass)
		if err != nil {
			return nil, err
		}
		w.walletPassword = walletPassword
	}
	if keymanagerKind == v2keymanager.Direct {
		passwordsDir, err := inputDirectory(cliCtx, passwordsDirPromptText, flags.WalletPasswordsDirFlag)
		if err != nil {
			return nil, err
		}
		w.passwordsDir = passwordsDir
	}
	return w, nil
}

// KeymanagerKind used by the wallet.
func (w *Wallet) KeymanagerKind() v2keymanager.Kind {
	return w.keymanagerKind
}

// AccountsDir for the wallet.
func (w *Wallet) AccountsDir() string {
	return w.accountsPath
}

// InitializeKeymanager reads a keymanager config from disk at the wallet path,
// unmarshals it based on the wallet's keymanager kind, and returns its value.
func (w *Wallet) InitializeKeymanager(
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
		keymanager, err = direct.NewKeymanager(ctx, w, cfg)
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize direct keymanager")
		}
	case v2keymanager.Derived:
		cfg, err := derived.UnmarshalConfigFile(configFile)
		if err != nil {
			return nil, errors.Wrap(err, "could not unmarshal keymanager config file")
		}
		keymanager, err = derived.NewKeymanager(ctx, w, cfg, skipMnemonicConfirm, w.walletPassword)
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize derived keymanager")
		}
	case v2keymanager.Remote:
		cfg, err := remote.UnmarshalConfigFile(configFile)
		if err != nil {
			return nil, errors.Wrap(err, "could not unmarshal keymanager config file")
		}
		keymanager, err = remote.NewKeymanager(ctx, 100000000, cfg)
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize remote keymanager")
		}
	default:
		return nil, fmt.Errorf("keymanager kind not supported: %s", w.keymanagerKind)
	}
	return keymanager, nil
}

// ListDirs in wallet accounts path.
func (w *Wallet) ListDirs() ([]string, error) {
	accountsDir, err := os.Open(w.AccountsDir())
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := accountsDir.Close(); err != nil {
			log.WithField(
				"directory", w.AccountsDir(),
			).Errorf("Could not close accounts directory: %v", err)
		}
	}()

	list, err := accountsDir.Readdirnames(0) // 0 to read all files and folders.
	if err != nil {
		return nil, errors.Wrapf(err, "could not read files in directory: %s", w.AccountsDir())
	}
	dirNames := make([]string, 0)
	for _, item := range list {
		ok, err := hasDir(filepath.Join(w.AccountsDir(), item))
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse directory: %v", err)
		}
		if ok {
			dirNames = append(dirNames, item)
		}
	}
	return dirNames, nil
}

// WriteFileAtPath within the wallet directory given the desired path, filename, and raw data.
func (w *Wallet) WriteFileAtPath(ctx context.Context, filePath string, fileName string, data []byte) error {
	accountPath := filepath.Join(w.accountsPath, filePath)
	if err := os.MkdirAll(accountPath, os.ModePerm); err != nil {
		return errors.Wrapf(err, "could not create path: %s", accountPath)
	}
	fullPath := filepath.Join(accountPath, fileName)
	if err := ioutil.WriteFile(fullPath, data, os.ModePerm); err != nil {
		return errors.Wrapf(err, "could not write %s", filePath)
	}
	log.WithFields(logrus.Fields{
		"path":     fullPath,
		"fileName": fileName,
	}).Debug("Wrote new file at path")
	return nil
}

// ReadFileAtPath within the wallet directory given the desired path and filename.
func (w *Wallet) ReadFileAtPath(ctx context.Context, filePath string, fileName string) ([]byte, error) {
	accountPath := filepath.Join(w.accountsPath, filePath)
	if err := os.MkdirAll(accountPath, os.ModePerm); err != nil {
		return nil, errors.Wrapf(err, "could not create path: %s", accountPath)
	}
	fullPath := filepath.Join(accountPath, fileName)
	rawData, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read %s", filePath)
	}
	return rawData, nil
}

// ReadKeymanagerConfigFromDisk opens a keymanager config file
// for reading if it exists at the wallet path.
func (w *Wallet) ReadKeymanagerConfigFromDisk(ctx context.Context) (io.ReadCloser, error) {
	configFilePath := filepath.Join(w.accountsPath, KeymanagerConfigFileName)
	if !fileExists(configFilePath) {
		return nil, fmt.Errorf("no keymanager config file found at path: %s", w.accountsPath)
	}
	return os.Open(configFilePath)
}

// WriteKeymanagerConfigToDisk takes an encoded keymanager config file
// and writes it to the wallet path.
func (w *Wallet) WriteKeymanagerConfigToDisk(ctx context.Context, encoded []byte) error {
	configFilePath := filepath.Join(w.accountsPath, KeymanagerConfigFileName)
	// Write the config file to disk.
	if err := ioutil.WriteFile(configFilePath, encoded, os.ModePerm); err != nil {
		return errors.Wrapf(err, "could not write %s", configFilePath)
	}
	log.WithField("configFilePath", configFilePath).Debug("Wrote keymanager config file to disk")
	return nil
}

// ReadEncryptedSeedFromDisk reads the encrypted wallet seed configuration from
// within the wallet path.
func (w *Wallet) ReadEncryptedSeedFromDisk(ctx context.Context) (io.ReadCloser, error) {
	configFilePath := filepath.Join(w.accountsPath, derived.EncryptedSeedFileName)
	if !fileExists(configFilePath) {
		return nil, fmt.Errorf("no encrypted seed file found at path: %s", w.accountsPath)
	}
	return os.Open(configFilePath)
}

// WriteEncryptedSeedToDisk writes the encrypted wallet seed configuration
// within the wallet path.
func (w *Wallet) WriteEncryptedSeedToDisk(ctx context.Context, encoded []byte) error {
	seedFilePath := filepath.Join(w.accountsPath, derived.EncryptedSeedFileName)
	// Write the config file to disk.
	if err := ioutil.WriteFile(seedFilePath, encoded, os.ModePerm); err != nil {
		return errors.Wrapf(err, "could not write %s", seedFilePath)
	}
	log.WithField("seedFilePath", seedFilePath).Debug("Wrote wallet encrypted seed file to disk")
	return nil
}

// ReadPasswordFromDisk --
func (w *Wallet) ReadPasswordFromDisk(ctx context.Context, passwordFileName string) (string, error) {
	fullPath := filepath.Join(w.passwordsDir, passwordFileName)
	rawData, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return "", errors.Wrapf(err, "could not read %s", fullPath)
	}
	return string(rawData), nil
}

// WritePasswordToDisk --
func (w *Wallet) WritePasswordToDisk(ctx context.Context, passwordFileName string, password string) error {
	passwordPath := filepath.Join(w.passwordsDir, passwordFileName)
	if err := ioutil.WriteFile(passwordPath, []byte(password), os.ModePerm); err != nil {
		return errors.Wrapf(err, "could not write %s", passwordPath)
	}
	return nil
}

func readKeymanagerKindFromWalletPath(walletPath string) (v2keymanager.Kind, error) {
	walletItem, err := os.Open(walletPath)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := walletItem.Close(); err != nil {
			log.WithField(
				"path", walletPath,
			).Errorf("Could not close wallet directory: %v", err)
		}
	}()
	list, err := walletItem.Readdirnames(0) // 0 to read all files and folders.
	if err != nil {
		return 0, fmt.Errorf("could not read files in directory: %s", walletPath)
	}
	if len(list) != 1 {
		return 0, fmt.Errorf("wanted 1 directory in wallet dir, received %d", len(list))
	}
	return v2keymanager.ParseKind(list[0])
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
