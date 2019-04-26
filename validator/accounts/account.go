package accounts

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "accounts")

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
	if _, err := ks.GetKeys(directory, shardWithdrawalKeyFile, password); err == nil {
		return fmt.Errorf("keystore at path already exists: %s", shardWithdrawalKeyFile)
	}
	if _, err := ks.GetKeys(directory, validatorKeyFile, password); err == nil {
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
	shardWithdrawalKey, err := keystore.NewKey(rand.Reader)
	if err != nil {
		return err
	}
	shardWithdrawalKeyFile = shardWithdrawalKeyFile + hex.EncodeToString(shardWithdrawalKey.PublicKey.Marshal())[:12]
	if err := ks.StoreKey(shardWithdrawalKeyFile, shardWithdrawalKey, password); err != nil {
		return fmt.Errorf("unable to store key %v", err)
	}
	log.WithField(
		"path",
		shardWithdrawalKeyFile,
	).Info("Keystore generated for shard withdrawals at path")
	validatorKey, err := keystore.NewKey(rand.Reader)
	if err != nil {
		return err
	}
	validatorKeyFile = validatorKeyFile + hex.EncodeToString(validatorKey.PublicKey.Marshal())[:12]
	if err := ks.StoreKey(validatorKeyFile, validatorKey, password); err != nil {
		return fmt.Errorf("unable to store key %v", err)
	}
	log.WithField(
		"path",
		validatorKeyFile,
	).Info("Keystore generated for validator signatures at path")

	data, err := keystore.DepositInput(validatorKey, shardWithdrawalKey)
	if err != nil {
		return fmt.Errorf("unable to generate deposit data: %v", err)
	}
	serializedData := new(bytes.Buffer)
	if err := ssz.Encode(serializedData, data); err != nil {
		return fmt.Errorf("could not serialize deposit data: %v", err)
	}
	log.Info(`Account creation complete! Copy and paste the deposit data shown below when issuing a transaction into the ETH1.0 deposit contract to activate your validator client`)
	fmt.Printf(`
========================Deposit Data=======================

%#x

===========================================================
`, serializedData)
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
