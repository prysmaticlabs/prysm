package accounts

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"

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
	shardWithdrawalKeyFile := directory + params.BeaconConfig().WithdrawalPrivkeyFileName
	validatorKeyFile := directory + params.BeaconConfig().ValidatorPrivkeyFileName
	// First, if the keystore already exists, throws an error as there can only be
	// one keystore per validator client.
	ks := keystore.NewKeystore(directory)
	if _, err := ks.GetKey(shardWithdrawalKeyFile, password); err == nil {
		return fmt.Errorf("keystore at path already exists: %s", shardWithdrawalKeyFile)
	}
	if _, err := ks.GetKey(validatorKeyFile, password); err == nil {
		return fmt.Errorf("keystore at path already exists: %s", validatorKeyFile)
	}
	return nil
}

// NewValidatorAccount sets up a validator client's secrets and generates the necessary deposit data
// parameters needed to deposit into the deposit contract on the ETH1.0 chain. Specifically, this
// generates a BLS private and public key, and then logs the serialized deposit input hex string
// to be used in an ETH1.0 transaction by the validator.
func NewValidatorAccount(directory string, password string) error {
	// First, if the keystore already exists, throws an error as there can only be
	// one keystore per validator client.
	if err := VerifyAccountNotExists(directory, password); err != nil {
		return fmt.Errorf("validator account exists: %v", err)
	}
	shardWithdrawalKeyFile := directory + params.BeaconConfig().WithdrawalPrivkeyFileName
	validatorKeyFile := directory + params.BeaconConfig().ValidatorPrivkeyFileName
	ks := keystore.NewKeystore(directory)
	// If the keystore does not exists at the path, we create a new one for the validator.
	shardWithdrawalKey, err := keystore.NewKey(rand.Reader)
	if err != nil {
		return err
	}
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
