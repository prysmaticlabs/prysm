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

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

const (
	// WalletDefaultDirName for accounts-v2.
	WalletDefaultDirName = ".prysm-wallet-v2"
	// PasswordsDefaultDirName where account passwords are stored.
	PasswordsDefaultDirName = ".prysm-wallet-v2-passwords"
	// KeymanagerConfigFileName for the keymanager used by the wallet: direct, derived, or remote.
	KeymanagerConfigFileName = "keymanageropts.json"
	// EncryptedSeedFileName for persisting a wallet's seed when using a derived keymanager.
	EncryptedSeedFileName = "seed.encrypted.json"
	// PasswordFileSuffix for passwords persisted as text to disk.
	PasswordFileSuffix = ".pass"
	// NumAccountWords for human-readable names in wallets using a direct keymanager.
	NumAccountWords = 3 // Number of words in account human-readable names.
	// AccountFilePermissions for accounts saved to disk.
	AccountFilePermissions = os.O_CREATE | os.O_RDWR
	// DirectoryPermissions for directories created under the wallet path.
	DirectoryPermissions = os.ModePerm
)

var (
	// ErrNoWalletFound signifies there was no wallet directory found on-disk.
	ErrNoWalletFound = errors.New(
		"no wallet found at path, please create a new wallet using `./prysm.sh validator wallet-v2 create`",
	)
)

// Wallet is a primitive in Prysm's v2 account management which
// has the capability of creating new accounts, reading existing accounts,
// and providing secure access to eth2 secrets depending on an
// associated keymanager (either direct, derived, or remote signing enabled).
type Wallet struct {
	accountsPath      string
	passwordsDir      string
	canUnlockAccounts bool
	keymanagerKind    v2keymanager.Kind
	walletPassword    string
}

func init() {
	petname.NonDeterministicMode() // Set random account name generation.
}

// NewWallet given a set of configuration options, will leverage
// create and write a new wallet to disk for a Prysm validator.
func NewWallet(
	cliCtx *cli.Context,
) (*Wallet, error) {
	walletDir, err := inputWalletDir(cliCtx)
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
	keymanagerKind, err := inputKeymanagerKind(cliCtx)
	if err != nil {
		return nil, err
	}
	accountsPath := path.Join(walletDir, keymanagerKind.String())
	if err := os.MkdirAll(accountsPath, DirectoryPermissions); err != nil {
		return nil, errors.Wrap(err, "could not create wallet directory")
	}
	w := &Wallet{
		accountsPath:   accountsPath,
		keymanagerKind: keymanagerKind,
	}
	if keymanagerKind == v2keymanager.Direct {
		passwordsDir := inputPasswordsDirectory(cliCtx)
		if err := os.MkdirAll(passwordsDir, DirectoryPermissions); err != nil {
			return nil, errors.Wrap(err, "could not create passwords directory")
		}
		w.passwordsDir = passwordsDir
		w.canUnlockAccounts = true
	}
	return w, nil
}

// OpenWallet instantiates a wallet from a specified path. It checks the
// type of keymanager associated with the wallet by reading files in the wallet
// path, if applicable. If a wallet does not exist, returns an appropriate error.
func OpenWallet(cliCtx *cli.Context) (*Wallet, error) {
	// Read a wallet's directory from user input.
	walletDir, err := inputWalletDir(cliCtx)
	if errors.Is(err, ErrNoWalletFound) {
		return nil, errors.New("no wallet found, create a new one with ./prysm.sh validator wallet-v2 create")
	} else if err != nil {
		return nil, err
	}
	keymanagerKind, err := readKeymanagerKindFromWalletPath(walletDir)
	if err != nil {
		return nil, errors.Wrap(err, "could not read keymanager kind for wallet")
	}
	walletPath := path.Join(walletDir, keymanagerKind.String())
	w := &Wallet{
		accountsPath:   walletPath,
		keymanagerKind: keymanagerKind,
	}
	if keymanagerKind == v2keymanager.Derived {
		walletPassword, err := inputExistingWalletPassword(cliCtx)
		if err != nil {
			return nil, err
		}
		w.walletPassword = walletPassword
	}
	if keymanagerKind == v2keymanager.Direct {
		w.passwordsDir = inputPasswordsDirectory(cliCtx)
		w.canUnlockAccounts = true
	}
	return w, nil
}

// ReadKeymanagerConfigFromDisk opens a keymanager config file
// for reading if it exists at the wallet path.
func (w *Wallet) ReadKeymanagerConfigFromDisk(ctx context.Context) (io.ReadCloser, error) {
	configFilePath := path.Join(w.accountsPath, KeymanagerConfigFileName)
	if !fileExists(configFilePath) {
		return nil, fmt.Errorf("no keymanager config file found at path: %s", w.accountsPath)
	}
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
		keymanager, err = direct.NewKeymanager(ctx, w, cfg, skipMnemonicConfirm)
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
	default:
		return nil, fmt.Errorf("keymanager kind not supported: %s", w.keymanagerKind)
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
	if err := os.MkdirAll(accountPath, DirectoryPermissions); err != nil {
		return "", errors.Wrap(err, "could not create account directory")
	}
	if err := w.writePasswordToFile(accountName, password); err != nil {
		return "", errors.Wrap(err, "could not write password to disk")
	}
	return accountName, nil
}

// WriteFileAtPath within the wallet directory given the desired path, filename, and raw data.
func (w *Wallet) WriteFileAtPath(ctx context.Context, filePath string, fileName string, data []byte) error {
	accountPath := path.Join(w.accountsPath, filePath)
	if err := os.MkdirAll(accountPath, os.ModePerm); err != nil {
		return errors.Wrapf(err, "could not create path: %s", accountPath)
	}
	fullPath := path.Join(accountPath, fileName)
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
	accountPath := path.Join(w.accountsPath, filePath)
	if err := os.MkdirAll(accountPath, os.ModePerm); err != nil {
		return nil, errors.Wrapf(err, "could not create path: %s", accountPath)
	}
	fullPath := path.Join(accountPath, fileName)
	rawData, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read %s", filePath)
	}
	return rawData, nil
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
	if err := ioutil.WriteFile(filePath, data, os.ModePerm); err != nil {
		return errors.Wrapf(err, "could not write %s", filePath)
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
	configFilePath := path.Join(w.accountsPath, KeymanagerConfigFileName)
	// Write the config file to disk.
	if err := ioutil.WriteFile(configFilePath, encoded, os.ModePerm); err != nil {
		return errors.Wrapf(err, "could not write %s", configFilePath)
	}
	log.WithField("configFilePath", configFilePath).Debug("Wrote keymanager config file to disk")
	return nil
}

// WriteEncryptedSeedToDisk writes the encrypted wallet seed configuration
// within the wallet path.
func (w *Wallet) WriteEncryptedSeedToDisk(ctx context.Context, encoded []byte) error {
	seedFilePath := path.Join(w.accountsPath, EncryptedSeedFileName)
	// Write the config file to disk.
	if err := ioutil.WriteFile(seedFilePath, encoded, os.ModePerm); err != nil {
		return errors.Wrapf(err, "could not write %s", seedFilePath)
	}
	log.WithField("seedFilePath", seedFilePath).Debug("Wrote wallet encrypted seed file to disk")
	return nil
}

// ReadEncryptedSeedFromDisk reads the encrypted wallet seed configuration from
// within the wallet path.
func (w *Wallet) ReadEncryptedSeedFromDisk(ctx context.Context) (io.ReadCloser, error) {
	if !fileExists(path.Join(w.accountsPath, EncryptedSeedFileName)) {
		return nil, fmt.Errorf("no encrypted seed file found at path: %s", w.accountsPath)
	}
	configFilePath := path.Join(w.accountsPath, EncryptedSeedFileName)
	return os.Open(configFilePath)
}

// ReadPasswordForAccount when given an account name from the wallet's passwords' path.
func (w *Wallet) ReadPasswordForAccount(accountName string) (string, error) {
	if !w.canUnlockAccounts {
		return "", errors.New("wallet has no permission to read account passwords")
	}
	passwordFilePath := path.Join(w.passwordsDir, accountName+PasswordFileSuffix)
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

func (w *Wallet) enterPasswordForAccount(cliCtx *cli.Context, accountName string) error {
	au := aurora.NewAurora(true)

	var password string
	var err error
	if cliCtx.IsSet(flags.PasswordFileFlag.Name) {
		passwordFilePath := cliCtx.String(flags.PasswordFileFlag.Name)
		data, err := ioutil.ReadFile(passwordFilePath)
		if err != nil {
			return err
		}
		password = string(data)
		err = w.checkPasswordForAccount(accountName, password)
		if err != nil && strings.Contains(err.Error(), "invalid checksum") {
			return fmt.Errorf("invalid password entered for account %s", accountName)
		}
		if err != nil {
			return err
		}
	} else {
		attemptingPassword := true
		// Loop asking for the password until the user enters it correctly.
		for attemptingPassword {
			// Ask the user for the password to their account.
			password, err = inputPasswordForAccount(cliCtx, accountName)
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

	if err := os.MkdirAll(w.passwordsDir, params.BeaconIoConfig().ReadWriteExecutePermissions); err != nil {
		return err
	}
	if err := w.writePasswordToFile(accountName, password); err != nil {
		return errors.Wrap(err, "could not write password to disk")
	}
	return nil
}

func (w *Wallet) checkPasswordForAccount(accountName string, password string) error {
	accountKeystore, err := w.keystoreForAccount(accountName)
	if err != nil {
		return errors.Wrap(err, "could not get keystore")
	}
	decryptor := keystorev4.New()
	_, err = decryptor.Decrypt(accountKeystore.Crypto, []byte(password))
	if err != nil {
		return errors.Wrap(err, "could not decrypt keystore")
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

func (w *Wallet) keystoreForAccount(accountName string) (*v2keymanager.Keystore, error) {
	encoded, err := w.ReadFileForAccount(accountName, direct.KeystoreFileName)
	if err != nil {
		return nil, errors.Wrap(err, "could not read keystore file")
	}
	keystoreJSON := &v2keymanager.Keystore{}
	if err := json.Unmarshal(encoded, &keystoreJSON); err != nil {
		return nil, errors.Wrap(err, "could not decode json")
	}
	return keystoreJSON, nil
}

// Writes the password file for an account namespace in the wallet's passwords directory.
func (w *Wallet) writePasswordToFile(accountName string, password string) error {
	passwordFilePath := path.Join(w.passwordsDir, accountName+PasswordFileSuffix)
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
		accountName = petname.Generate(NumAccountWords, "-" /* separator */)
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
