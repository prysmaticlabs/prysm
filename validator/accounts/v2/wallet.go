package v2

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofrs/flock"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const (
	// KeymanagerConfigFileName for the keymanager used by the wallet: direct, derived, or remote.
	KeymanagerConfigFileName = "keymanageropts.json"
	// DirectoryPermissions for directories created under the wallet path.
	DirectoryPermissions        = os.ModePerm
	newWalletPasswordPromptText = "New wallet password"
	walletPasswordPromptText    = "Wallet password"
	confirmPasswordPromptText   = "Confirm password"
)

type passwordConfirm int

const (
	// An enum to indicate to the prompt that confirming the password is not needed.
	noConfirmPass passwordConfirm = iota
	// An enum to indicate to the prompt to confirm the password entered.
	confirmPass
)

var (
	// ErrNoWalletFound signifies there was no wallet directory found on-disk.
	ErrNoWalletFound = errors.New(
		"no wallet found at path, please create a new wallet using `./prysm.sh validator wallet-v2 create`",
	)
	// ErrWalletExists is an error returned when a wallet already exists in the path provided.
	ErrWalletExists = errors.New("you already have a wallet at the specified path. You can " +
		"edit your wallet configuration by running ./prysm.sh validator wallet-v2 edit-config",
	)
	keymanagerKindSelections = map[v2keymanager.Kind]string{
		v2keymanager.Derived: "HD Wallet (Recommended)",
		v2keymanager.Direct:  "Non-HD Wallet (Most Basic)",
		v2keymanager.Remote:  "Remote Signing Wallet (Advanced)",
	}
	validateExistingPass = func(input string) error {
		if input == "" {
			return errors.New("password input cannot be empty")
		}
		return nil
	}
)

// WalletConfig to open a wallet programmatically.
type WalletConfig struct {
	WalletDir      string
	KeymanagerKind v2keymanager.Kind
	WalletPassword string
}

// Wallet is a primitive in Prysm's v2 account management which
// has the capability of creating new accounts, reading existing accounts,
// and providing secure access to eth2 secrets depending on an
// associated keymanager (either direct, derived, or remote signing enabled).
type Wallet struct {
	walletDir      string
	accountsPath   string
	configFilePath string
	walletPassword string
	walletFileLock *flock.Flock
	keymanagerKind v2keymanager.Kind
}

// WalletExists check if a wallet at the specified directory
// exists and has valid information in it.
func WalletExists(walletDir string) error {
	ok, err := fileutil.HasDir(walletDir)
	if err != nil {
		return errors.Wrap(err, "could not parse wallet directory")
	}
	if ok {
		isEmptyWallet, err := isEmptyWallet(walletDir)
		if err != nil {
			return errors.Wrap(err, "could not check if wallet has files")
		}
		if isEmptyWallet {
			return ErrNoWalletFound
		}
		return nil
	}
	return ErrNoWalletFound
}

// OpenWalletOrElse tries to open the wallet and if it fails or no wallet
// is found, invokes a callback function.
func OpenWalletOrElse(cliCtx *cli.Context, otherwise func(cliCtx *cli.Context) (*Wallet, error)) (*Wallet, error) {
	if err := WalletExists(cliCtx.String(flags.WalletDirFlag.Name)); err != nil {
		if errors.Is(err, ErrNoWalletFound) {
			return otherwise(cliCtx)
		}
		return nil, errors.Wrap(err, "could not check if wallet exists")
	}
	walletDir, err := inputDirectory(cliCtx, walletDirPromptText, flags.WalletDirFlag)
	if err != nil {
		return nil, err
	}
	walletPassword, err := inputPassword(
		cliCtx,
		flags.WalletPasswordFileFlag,
		walletPasswordPromptText,
		noConfirmPass,
		validateExistingPass,
	)
	return OpenWallet(cliCtx.Context, &WalletConfig{
		WalletDir:      walletDir,
		WalletPassword: walletPassword,
	})
}

// OpenWallet instantiates a wallet from a specified path. It checks the
// type of keymanager associated with the wallet by reading files in the wallet
// path, if applicable. If a wallet does not exist, returns an appropriate error.
func OpenWallet(ctx context.Context, cfg *WalletConfig) (*Wallet, error) {
	if err := WalletExists(cfg.WalletDir); err != nil {
		if errors.Is(err, ErrNoWalletFound) {
			return nil, ErrNoWalletFound
		}
		return nil, errors.Wrap(err, "could not check if wallet exists")
	}
	keymanagerKind, err := readKeymanagerKindFromWalletPath(cfg.WalletDir)
	if err != nil {
		return nil, errors.Wrap(err, "could not read keymanager kind for wallet")
	}
	walletPath := filepath.Join(cfg.WalletDir, keymanagerKind.String())
	return &Wallet{
		walletDir:      cfg.WalletDir,
		accountsPath:   walletPath,
		keymanagerKind: keymanagerKind,
		walletPassword: cfg.WalletPassword,
	}, nil
}

// SaveWallet persists the wallet's directories to disk.
func (w *Wallet) SaveWallet() error {
	if err := os.MkdirAll(w.accountsPath, DirectoryPermissions); err != nil {
		return errors.Wrap(err, "could not create wallet directory")
	}
	return nil
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
		opts, err := direct.UnmarshalOptionsFile(configFile)
		if err != nil {
			return nil, errors.Wrap(err, "could not unmarshal keymanageropts file")
		}
		keymanager, err = direct.NewKeymanager(ctx, &direct.SetupConfig{
			Wallet: w,
			Opts:   opts,
		})
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize direct keymanager")
		}
	case v2keymanager.Derived:
		opts, err := derived.UnmarshalOptionsFile(configFile)
		if err != nil {
			return nil, errors.Wrap(err, "could not unmarshal keymanager config file")
		}
		keymanager, err = derived.NewKeymanager(ctx, &derived.SetupConfig{
			Opts:                opts,
			Wallet:              w,
			SkipMnemonicConfirm: skipMnemonicConfirm,
		})
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize derived keymanager")
		}
	case v2keymanager.Remote:
		opts, err := remote.UnmarshalOptionsFile(configFile)
		if err != nil {
			return nil, errors.Wrap(err, "could not unmarshal keymanager config file")
		}
		keymanager, err = remote.NewKeymanager(ctx, &remote.SetupConfig{
			Opts:           opts,
			MaxMessageSize: 100000000,
		})
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize remote keymanager")
		}
	default:
		return nil, fmt.Errorf("keymanager kind not supported: %s", w.keymanagerKind)
	}
	return keymanager, nil
}

// WriteFileAtPath within the wallet directory given the desired path, filename, and raw data.
func (w *Wallet) WriteFileAtPath(ctx context.Context, filePath string, fileName string, data []byte) error {
	accountPath := filepath.Join(w.accountsPath, filePath)
	if err := os.MkdirAll(accountPath, os.ModePerm); err != nil {
		return errors.Wrapf(err, "could not create path: %s", accountPath)
	}
	fullPath := filepath.Join(accountPath, fileName)
	if err := ioutil.WriteFile(fullPath, data, params.BeaconIoConfig().ReadWritePermissions); err != nil {
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
	matches, err := filepath.Glob(fullPath)
	if err != nil {
		return []byte{}, errors.Wrap(err, "could not find file")
	}
	if len(matches) == 0 {
		return []byte{}, fmt.Errorf("no files found %s", fullPath)
	}
	rawData, err := ioutil.ReadFile(matches[0])
	if err != nil {
		return nil, errors.Wrapf(err, "could not read %s", filePath)
	}
	return rawData, nil
}

// FileNameAtPath return the full file name for the requested file. It allows for finding the file
// with a regex pattern.
func (w *Wallet) FileNameAtPath(ctx context.Context, filePath string, fileName string) (string, error) {
	accountPath := filepath.Join(w.accountsPath, filePath)
	if err := os.MkdirAll(accountPath, os.ModePerm); err != nil {
		return "", errors.Wrapf(err, "could not create path: %s", accountPath)
	}
	fullPath := filepath.Join(accountPath, fileName)
	matches, err := filepath.Glob(fullPath)
	if err != nil {
		return "", errors.Wrap(err, "could not find file")
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no files found %s", fullPath)
	}
	fullFileName := filepath.Base(matches[0])
	return fullFileName, nil
}

// ReadKeymanagerConfigFromDisk opens a keymanager config file
// for reading if it exists at the wallet path.
func (w *Wallet) ReadKeymanagerConfigFromDisk(ctx context.Context) (io.ReadCloser, error) {
	configFilePath := filepath.Join(w.accountsPath, KeymanagerConfigFileName)
	if !fileutil.FileExists(configFilePath) {
		return nil, fmt.Errorf("no keymanager config file found at path: %s", w.accountsPath)
	}
	w.configFilePath = configFilePath
	return os.Open(configFilePath)

}

// LockWalletConfigFile lock read and write to wallet file in order to prevent
// two validators from using the same keys.
func (w *Wallet) LockWalletConfigFile(ctx context.Context) error {
	fileLock := flock.New(w.configFilePath)
	locked, err := fileLock.TryLock()
	if err != nil {
		return errors.Wrapf(err, "failed to lock wallet config file: %s", w.configFilePath)
	}
	if !locked {
		return fmt.Errorf("failed to lock wallet config file: %s", w.configFilePath)
	}
	w.walletFileLock = fileLock
	return nil
}

// UnlockWalletConfigFile unlock wallet file.
// should be called before client is closing in order to remove the file lock.
func (w *Wallet) UnlockWalletConfigFile() error {
	if w.walletFileLock == nil {
		return errors.New("trying to unlock a nil lock")
	}
	return w.walletFileLock.Unlock()
}

// WriteKeymanagerConfigToDisk takes an encoded keymanager config file
// and writes it to the wallet path.
func (w *Wallet) WriteKeymanagerConfigToDisk(ctx context.Context, encoded []byte) error {
	configFilePath := filepath.Join(w.accountsPath, KeymanagerConfigFileName)
	// Write the config file to disk.
	if err := ioutil.WriteFile(configFilePath, encoded, params.BeaconIoConfig().ReadWritePermissions); err != nil {
		return errors.Wrapf(err, "could not write %s", configFilePath)
	}
	log.WithField("configFilePath", configFilePath).Debug("Wrote keymanager config file to disk")
	return nil
}

// ReadEncryptedSeedFromDisk reads the encrypted wallet seed configuration from
// within the wallet path.
func (w *Wallet) ReadEncryptedSeedFromDisk(ctx context.Context) (io.ReadCloser, error) {
	configFilePath := filepath.Join(w.accountsPath, derived.EncryptedSeedFileName)
	if !fileutil.FileExists(configFilePath) {
		return nil, fmt.Errorf("no encrypted seed file found at path: %s", w.accountsPath)
	}
	return os.Open(configFilePath)
}

// WriteEncryptedSeedToDisk writes the encrypted wallet seed configuration
// within the wallet path.
func (w *Wallet) WriteEncryptedSeedToDisk(ctx context.Context, encoded []byte) error {
	seedFilePath := filepath.Join(w.accountsPath, derived.EncryptedSeedFileName)
	// Write the config file to disk.
	if err := ioutil.WriteFile(seedFilePath, encoded, params.BeaconIoConfig().ReadWritePermissions); err != nil {
		return errors.Wrapf(err, "could not write %s", seedFilePath)
	}
	log.WithField("seedFilePath", seedFilePath).Debug("Wrote wallet encrypted seed file to disk")
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
	for _, n := range list {
		keymanagerKind, err := v2keymanager.ParseKind(n)
		if err == nil {
			return keymanagerKind, nil
		}
	}
	return 0, errors.New("no keymanager folder, 'direct', 'remote', nor 'derived' found in wallet path")
}

// isEmptyWallet checks if a folder consists key directory such as `derived`, `remote` or `direct`.
// Returns true if exists, false otherwise.
func isEmptyWallet(name string) (bool, error) {
	expanded, err := fileutil.ExpandPath(name)
	if err != nil {
		return false, err
	}
	f, err := os.Open(expanded)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Debugf("Could not close directory: %s", expanded)
		}
	}()
	names, err := f.Readdirnames(-1)
	if err == io.EOF {
		return true, nil
	}

	for _, n := range names {
		// Nil error means input name is `derived`, `remote` or `direct`, the wallet is not empty.
		_, err := v2keymanager.ParseKind(n)
		if err == nil {
			return false, nil
		}
	}

	return true, err
}

func inputPassword(
	cliCtx *cli.Context,
	passwordFileFlag *cli.StringFlag,
	promptText string,
	confirmPassword passwordConfirm,
	passwordValidator func(input string) error,
) (string, error) {
	if cliCtx.IsSet(passwordFileFlag.Name) {
		passwordFilePathInput := cliCtx.String(passwordFileFlag.Name)
		data, err := fileutil.ReadFileAsBytes(passwordFilePathInput)
		if err != nil {
			return "", errors.Wrap(err, "could not read file as bytes")
		}
		enteredPassword := strings.TrimRight(string(data), "\r\n")
		if err := passwordValidator(enteredPassword); err != nil {
			return "", errors.Wrap(err, "password did not pass validation")
		}
		return enteredPassword, nil
	}
	var hasValidPassword bool
	var walletPassword string
	var err error
	for !hasValidPassword {
		walletPassword, err = promptutil.PasswordPrompt(promptText, passwordValidator)
		if err != nil {
			return "", fmt.Errorf("could not read account password: %v", err)
		}

		if confirmPassword == confirmPass {
			passwordConfirmation, err := promptutil.PasswordPrompt(confirmPasswordPromptText, passwordValidator)
			if err != nil {
				return "", fmt.Errorf("could not read password confirmation: %v", err)
			}
			if walletPassword != passwordConfirmation {
				log.Error("Passwords do not match")
				continue
			}
			hasValidPassword = true
		} else {
			return walletPassword, nil
		}
	}
	return walletPassword, nil
}
