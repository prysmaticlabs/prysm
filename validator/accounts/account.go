// Package accounts defines tools to manage an encrypted validator keystore.
package accounts

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	contract "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"
)

var log = logrus.WithField("prefix", "accounts")

// DecryptKeysFromKeystore extracts a set of validator private keys from
// an encrypted keystore directory and a password string.
func DecryptKeysFromKeystore(directory string, password string) (map[string]*keystore.Key, error) {
	validatorPrefix := params.BeaconConfig().ValidatorPrivkeyFileName
	ks := keystore.NewKeystore(directory)
	validatorKeys, err := ks.GetKeys(directory, validatorPrefix, password, true /* warnOnFail */)
	if err != nil {
		return nil, errors.Wrap(err, "could not get private key")
	}
	return validatorKeys, nil
}

// VerifyAccountNotExists checks if a validator has not yet created an account
// and keystore in the provided directory string.
func VerifyAccountNotExists(directory string, password string) error {
	if directory == "" || password == "" {
		return errors.New("expected a path to the validator keystore and password to be provided, received nil")
	}
	shardWithdrawalKeyFile := params.BeaconConfig().WithdrawalPrivkeyFileName
	validatorKeyFile := params.BeaconConfig().ValidatorPrivkeyFileName
	// First, if the keystore already exists, throws an error as there can only be
	// one keystore per validator client.
	ks := keystore.NewKeystore(directory)
	if _, err := ks.GetKeys(directory, shardWithdrawalKeyFile, password, false /* warnOnFail */); err == nil {
		return fmt.Errorf("keystore at path already exists: %s", shardWithdrawalKeyFile)
	}
	if _, err := ks.GetKeys(directory, validatorKeyFile, password, false /* warnOnFail */); err == nil {
		return fmt.Errorf("keystore at path already exists: %s", validatorKeyFile)
	}
	return nil
}

// NewValidatorAccount sets up a validator client's secrets and generates the necessary deposit data
// parameters needed to deposit into the deposit contract on the ETH1.0 chain. Specifically, this
// generates a BLS private and public key, and then logs the serialized deposit input hex string
// to be used in an ETH1.0 transaction by the validator.
func NewValidatorAccount(directory string, password string) error {
	shardWithdrawalKeyFile := directory + params.BeaconConfig().WithdrawalPrivkeyFileName
	validatorKeyFile := directory + params.BeaconConfig().ValidatorPrivkeyFileName
	ks := keystore.NewKeystore(directory)
	// If the keystore does not exists at the path, we create a new one for the validator.
	shardWithdrawalKey, err := keystore.NewKey()
	if err != nil {
		return err
	}
	shardWithdrawalKeyFile = shardWithdrawalKeyFile + hex.EncodeToString(shardWithdrawalKey.PublicKey.Marshal())[:12]
	if err := ks.StoreKey(shardWithdrawalKeyFile, shardWithdrawalKey, password); err != nil {
		return errors.Wrap(err, "unable to store key")
	}
	log.WithField(
		"path",
		shardWithdrawalKeyFile,
	).Info("Keystore generated for shard withdrawals at path")
	validatorKey, err := keystore.NewKey()
	if err != nil {
		return err
	}
	validatorKeyFile = validatorKeyFile + hex.EncodeToString(validatorKey.PublicKey.Marshal())[:12]
	if err := ks.StoreKey(validatorKeyFile, validatorKey, password); err != nil {
		return errors.Wrap(err, "unable to store key")
	}
	log.WithField(
		"path",
		validatorKeyFile,
	).Info("Keystore generated for validator signatures at path")

	data, depositRoot, err := keystore.DepositInput(validatorKey, shardWithdrawalKey, params.BeaconConfig().MaxEffectiveBalance)
	if err != nil {
		return errors.Wrap(err, "unable to generate deposit data")
	}
	testAcc, err := contract.Setup()
	if err != nil {
		return errors.Wrap(err, "unable to create simulated backend")
	}
	testAcc.TxOpts.GasLimit = 1000000

	tx, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoot)
	if err != nil {
		return errors.Wrap(err, "unable to create deposit transaction")
	}
	log.Info(`Account creation complete! Copy and paste the raw transaction data shown below when issuing a transaction into the ETH1.0 deposit contract to activate your validator client`)
	fmt.Printf(`
========================Raw Transaction Data=======================

%#x

===================================================================
`, tx.Data())
	publicKey := validatorKey.PublicKey.Marshal()[:]
	log.Infof("Deposit data displayed for public key: %#x", publicKey)
	return nil
}

// Exists checks if a validator account at a given keystore path exists.
func Exists(keystorePath string) (bool, error) {
	/* #nosec */
	f, err := os.Open(keystorePath)
	if err != nil {
		return false, nil
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return false, nil
	}
	return true, err
}

// CreateValidatorAccount creates a validator account from the given cli context.
func CreateValidatorAccount(path string, passphrase string) (string, string, error) {
	// Forces user to create directory if using non-default path.
	if path != DefaultValidatorDir() {
		exists, err := Exists(path)
		if err != nil {
			return path, passphrase, err
		}
		if !exists {
			return path, passphrase, fmt.Errorf("path %q does not exist", path)
		}
	}
	if err := NewValidatorAccount(path, passphrase); err != nil {
		return "", "", errors.Wrapf(err, "could not initialize validator account")
	}
	return path, passphrase, nil
}

// PrintPublicAndPrivateKeys uses the passed in path and prints out the public and private keys in that directory.
func PrintPublicAndPrivateKeys(path string, passphrase string) error {
	keystores, err := DecryptKeysFromKeystore(path, passphrase)
	if err != nil {
		return errors.Wrapf(err, "failed to decrypt keystore keys at path %s", path)
	}
	for _, v := range keystores {
		fmt.Printf("Public key: %#x private key: %#x\n", v.PublicKey.Marshal(), v.SecretKey.Marshal())
	}
	return nil
}

// DefaultValidatorDir returns OS-specific default keystore directory.
func DefaultValidatorDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "Eth2Validators")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "Eth2Validators")
		} else {
			return filepath.Join(home, ".eth2validators")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

// HandleEmptyFlags checks what the set flags are and allows the user to manually enter them if they're empty.
func HandleEmptyFlags(cliCtx *cli.Context, confirmPassword bool) (string, string, error) {
	path := cliCtx.String(flags.KeystorePathFlag.Name)
	passphrase := cliCtx.String(flags.PasswordFlag.Name)

	if path == "" {
		path = DefaultValidatorDir()
		log.Infof("Please specify the keystore path for your private keys (default: %q):", path)
		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')
		if err != nil {
			return path, passphrase, errors.Wrap(err, "could not read input path")
		}
		if text = strings.Replace(text, "\n", "", -1); text != "" {
			path = text
		}
	}

	if passphrase == "" {
		log.Info("Please enter the password for your private keys")
		enteredPassphrase, err := cmd.EnterPassword(confirmPassword, cmd.StdInPasswordReader{})
		if err != nil {
			return path, enteredPassphrase, errors.Wrap(err, "could not read entered passphrase")
		}
		passphrase = enteredPassphrase
	}

	return path, passphrase, nil
}

// homeDir returns home directory path.
func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
