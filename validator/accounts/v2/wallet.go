package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

const (
	// KeymanagerConfigFileName for the keymanager used by the wallet: direct, derived, or remote.
	KeymanagerConfigFileName = "keymanageropts.json"
	// DirectoryPermissions for directories created under the wallet path.
	DirectoryPermissions = os.ModePerm
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
)

// Wallet is a primitive in Prysm's v2 account management which
// has the capability of creating new accounts, reading existing accounts,
// and providing secure access to eth2 secrets depending on an
// associated keymanager (either direct, derived, or remote signing enabled).
type Wallet struct {
	walletDir      string
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
	// Check if the user has a wallet at the specified path.
	// If a user does not have a wallet, we instantiate one
	// based on specified options.
	walletExists, err := hasDir(walletDir)
	if err != nil {
		return nil, errors.Wrap(err, "could not check if wallet exists")
	}
	if walletExists {
		return nil, ErrWalletExists
	}
	accountsPath := filepath.Join(walletDir, keymanagerKind.String())
	w := &Wallet{
		accountsPath:   accountsPath,
		keymanagerKind: keymanagerKind,
		walletDir:      walletDir,
	}
	if keymanagerKind == v2keymanager.Derived {
		walletPassword, err := inputPassword(
			cliCtx,
			flags.WalletPasswordFileFlag,
			newWalletPasswordPromptText,
			confirmPass,
		)
		if err != nil {
			return nil, errors.Wrap(err, "could not get password")
		}
		w.walletPassword = walletPassword
	}
	if keymanagerKind == v2keymanager.Direct {
		passwordsDir, err := inputDirectory(cliCtx, passwordsDirPromptText, flags.WalletPasswordsDirFlag)
		if err != nil {
			return nil, errors.Wrap(err, "could not get password directory")
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
	if err != nil {
		return nil, err
	}
	ok, err := hasDir(walletDir)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse wallet directory")
	}
	if !ok {
		return nil, ErrNoWalletFound
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
		walletPassword, err := inputPassword(
			cliCtx,
			flags.WalletPasswordFileFlag,
			walletPasswordPromptText,
			noConfirmPass,
		)
		if err != nil {
			return nil, err
		}
		w.walletPassword = walletPassword
	}
	if keymanagerKind == v2keymanager.Direct {
		keymanagerCfg, err := w.ReadKeymanagerConfigFromDisk(context.Background())
		if err != nil {
			return nil, err
		}
		directCfg, err := direct.UnmarshalConfigFile(keymanagerCfg)
		if err != nil {
			return nil, err
		}
		w.passwordsDir = directCfg.AccountPasswordsDirectory
		au := aurora.NewAurora(true)
		log.Infof("%s %s", au.BrightMagenta("(account passwords path)"), w.passwordsDir)
	}
	log.Info("Successfully opened wallet")
	return w, nil
}

// SaveWallet persists the wallet's directories to disk.
func (w *Wallet) SaveWallet() error {
	if err := os.MkdirAll(w.accountsPath, DirectoryPermissions); err != nil {
		return errors.Wrap(err, "could not create wallet directory")
	}
	if w.keymanagerKind == v2keymanager.Direct {
		if err := os.MkdirAll(w.passwordsDir, DirectoryPermissions); err != nil {
			return errors.Wrap(err, "could not create passwords directory")
		}
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

// AccountTimestamp retrieves the timestamp from a given keystore file name.
func AccountTimestamp(fileName string) (time.Time, error) {
	timestampStart := strings.LastIndex(fileName, "-") + 1
	timestampEnd := strings.LastIndex(fileName, ".")
	// Return an error if the text we expect cannot be found.
	if timestampStart == -1 || timestampEnd == -1 {
		return time.Unix(0, 0), fmt.Errorf("could not find timestamp in file name %s", fileName)
	}
	unixTimestampStr, err := strconv.ParseInt(fileName[timestampStart:timestampEnd], 10, 64)
	if err != nil {
		return time.Unix(0, 0), errors.Wrapf(err, "could not parse account created at timestamp: %s", fileName)
	}
	unixTimestamp := time.Unix(unixTimestampStr, 0)
	return unixTimestamp, nil
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

// enterPasswordForAccount checks if a user has a password specified for the new account
// either from a file or from stdin. Then, it saves the password to the wallet.
func (w *Wallet) enterPasswordForAccount(cliCtx *cli.Context, accountName string, pubKey []byte) error {
	au := aurora.NewAurora(true)
	var password string
	var err error
	if cliCtx.IsSet(flags.AccountPasswordFileFlag.Name) {
		passwordFilePath := cliCtx.String(flags.AccountPasswordFileFlag.Name)
		data, err := ioutil.ReadFile(passwordFilePath)
		if err != nil {
			return err
		}
		password = string(data)
		err = w.checkPasswordForAccount(accountName, password)
		if err != nil && strings.Contains(err.Error(), "invalid checksum") {
			return fmt.Errorf("invalid password entered for account with public key %#x", pubKey)
		}
		if err != nil {
			return err
		}
	} else {
		attemptingPassword := true
		// Loop asking for the password until the user enters it correctly.
		for attemptingPassword {
			// Ask the user for the password to their account.
			password, err = inputWeakPassword(
				cliCtx,
				flags.AccountPasswordFileFlag,
				fmt.Sprintf(passwordForAccountPromptText, bytesutil.Trunc(pubKey)),
			)
			if err != nil {
				return errors.Wrap(err, "could not input password")
			}
			err = w.checkPasswordForAccount(accountName, password)
			if err != nil && strings.Contains(err.Error(), "invalid checksum") {
				fmt.Println(au.Red("Incorrect password entered, please try again"))
				continue
			}
			if err != nil {
				return err
			}

			attemptingPassword = false
		}
	}
	ctx := context.Background()
	if err := w.WritePasswordToDisk(ctx, accountName+direct.PasswordFileSuffix, password); err != nil {
		return errors.Wrap(err, "could not write password to disk")
	}
	return nil
}

func (w *Wallet) checkPasswordForAccount(accountName string, password string) error {
	encoded, err := w.ReadFileAtPath(context.Background(), accountName, direct.KeystoreFileName)
	if err != nil {
		return errors.Wrap(err, "could not read keystore file")
	}
	keystoreJSON := &v2keymanager.Keystore{}
	if err := json.Unmarshal(encoded, &keystoreJSON); err != nil {
		return errors.Wrap(err, "could not decode json")
	}
	decryptor := keystorev4.New()
	_, err = decryptor.Decrypt(keystoreJSON.Crypto, []byte(password))
	if err != nil {
		return errors.Wrap(err, "could not decrypt keystore")
	}
	return nil
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

func createOrOpenWallet(cliCtx *cli.Context, creationFunc func(cliCtx *cli.Context) (*Wallet, error)) (*Wallet, error) {
	directory := cliCtx.String(flags.WalletDirFlag.Name)
	ok, err := hasDir(directory)
	if err != nil {
		return nil, errors.Wrapf(err, "could not check if wallet dir %s exists", directory)
	}
	var wallet *Wallet
	if !ok {
		wallet, err = creationFunc(cliCtx)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not create wallet")
		}
	} else {
		wallet, err = OpenWallet(cliCtx)
		if err != nil {
			return nil, errors.Wrap(err, "could not open wallet")
		}
	}
	return wallet, nil
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
