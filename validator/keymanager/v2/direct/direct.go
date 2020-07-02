package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	contract "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

var log = logrus.WithField("prefix", "keymanager-v2")

const (
	keystoreFileName           = "keystore.json"
	depositDataFileName        = "deposit_data.ssz"
	depositTransactionFileName = "deposit_transaction.rlp"
)

// Wallet defines a struct which has capabilities and knowledge of how
// to read and write important accounts-related files to the filesystem.
// Useful for keymanager to have persistent capabilities for accounts on-disk.
type Wallet interface {
	AccountsDir() string
	WriteAccountToDisk(ctx context.Context, password string) (string, error)
	WriteFileForAccount(ctx context.Context, accountName string, fileName string, data []byte) error
}

// Config for a direct keymanager.
type Config struct {
	EIPVersion string `json:"direct_eip_version"`
}

// Keymanager implementation for direct keystores utilizing EIP-2335.
type Keymanager struct {
	wallet            Wallet
	cfg               *Config
	mnemonicGenerator SeedPhraseFactory
}

// DefaultConfig for a direct keymanager implementation.
func DefaultConfig() *Config {
	return &Config{
		EIPVersion: "EIP-2335",
	}
}

// NewKeymanager instantiates a new direct keymanager from configuration options.
func NewKeymanager(ctx context.Context, wallet Wallet, cfg *Config) *Keymanager {
	return &Keymanager{
		wallet:            wallet,
		cfg:               cfg,
		mnemonicGenerator: &EnglishMnemonicGenerator{},
	}
}

// UnmarshalConfigFile attempts to JSON unmarshal a direct keymanager
// configuration file into the *Config{} struct.
func UnmarshalConfigFile(r io.ReadCloser) (interface{}, error) {
	enc, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Errorf("Could not close keymanager config file: %v", err)
		}
	}()
	cfg := &Config{}
	if err := json.Unmarshal(enc, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// CreateAccount for a direct keymanager implementation. This utilizes
// the EIP-2335 keystore standard for BLS12-381 keystores. It
// stores the generated keystore.json file in the wallet and additionally
// generates a mnemonic for withdrawal credentials. At the end, it logs
// the raw deposit data hex string for users to copy.
func (dr *Keymanager) CreateAccount(ctx context.Context, password string) error {
	// Create a new, unique account name and write its password + directory to disk.
	accountName, err := dr.wallet.WriteAccountToDisk(ctx, password)
	if err != nil {
		return errors.Wrap(err, "could not write account to disk")
	}
	// Generates a new EIP-2335 compliant keystore file
	// from a BLS private key and marshals it as JSON.
	encryptor := keystorev4.New()
	validatingKey := bls.RandKey()
	keystoreFile, err := encryptor.Encrypt(validatingKey.Marshal(), []byte(password))
	if err != nil {
		return errors.Wrap(err, "could not encrypt validating key into keystore")
	}
	encoded, err := json.MarshalIndent(keystoreFile, "", "\t")
	if err != nil {
		return errors.Wrap(err, "could not json marshal keystore file")
	}

	// Generate a withdrawal key and confirm user
	// acknowledgement of a 256-bit entropy mnemonic phrase.
	withdrawalKey := bls.RandKey()
	rawWithdrawalKey := withdrawalKey.Marshal()[:]
	seedPhrase, err := dr.mnemonicGenerator.Generate(rawWithdrawalKey)
	if err != nil {
		return errors.Wrap(err, "could not generate mnemonic for withdrawal key")
	}
	if err := dr.mnemonicGenerator.ConfirmAcknowledgement(seedPhrase); err != nil {
		return errors.Wrap(err, "could not confirm acknowledgement of mnemonic")
	}

	// Upon confirmation of the withdrawal key, proceed to display
	// and write associated deposit data to disk.
	tx, depositData, err := generateDepositTransaction(validatingKey, withdrawalKey)
	if err != nil {
		return errors.Wrap(err, "could not generate deposit transaction data")
	}

	// Log the deposit transaction data to the user.
	logDepositTransaction(tx)

	// We write the raw deposit transaction as an .rlp encoded file.
	if err := dr.wallet.WriteFileForAccount(ctx, accountName, depositTransactionFileName, tx.Data()); err != nil {
		return errors.Wrapf(err, "could not write for account %s: %s", accountName, depositTransactionFileName)
	}

	// We write the ssz-encoded deposit data to disk as a .ssz file.
	encodedDepositData, err := ssz.Marshal(depositData)
	if err != nil {
		return errors.Wrap(err, "could not marshal deposit data")
	}
	if err := dr.wallet.WriteFileForAccount(ctx, accountName, depositDataFileName, encodedDepositData); err != nil {
		return errors.Wrapf(err, "could not write for account %s: %s", accountName, encodedDepositData)
	}

	// Finally, write the encoded keystore to disk.
	if err := dr.wallet.WriteFileForAccount(ctx, accountName, keystoreFileName, encoded); err != nil {
		return errors.Wrapf(err, "could not write keystore file for account %s", accountName)
	}
	log.WithFields(logrus.Fields{
		"name": accountName,
		"path": dr.wallet.AccountsDir(),
	}).Info("Successfully created new validator account")
	return nil
}

// MarshalConfigFile returns a marshaled configuration file for a direct keymanager.
func (dr *Keymanager) MarshalConfigFile(ctx context.Context) ([]byte, error) {
	return json.MarshalIndent(dr.cfg, "", "\t")
}

// FetchValidatingPublicKeys fetches the list of public keys from the direct account keystores.
func (dr *Keymanager) FetchValidatingPublicKeys() ([][48]byte, error) {
	return nil, errors.New("unimplemented")
}

// Sign signs a message using a validator key.
func (dr *Keymanager) Sign(context.Context, interface{}) (bls.Signature, error) {
	return nil, errors.New("unimplemented")
}

func generateDepositTransaction(
	validatingKey bls.SecretKey,
	withdrawalKey bls.SecretKey,
) (*types.Transaction, *ethpb.Deposit_Data, error) {
	depositData, depositRoot, err := depositutil.DepositInput(
		validatingKey, withdrawalKey, params.BeaconConfig().MaxEffectiveBalance,
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not generate deposit input")
	}
	testAcc, err := contract.Setup()
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not load deposit contract")
	}
	testAcc.TxOpts.GasLimit = 1000000

	tx, err := testAcc.Contract.Deposit(
		testAcc.TxOpts,
		depositData.PublicKey,
		depositData.WithdrawalCredentials,
		depositData.Signature,
		depositRoot,
	)
	return tx, depositData, nil
}

func logDepositTransaction(tx *types.Transaction) {
	log.Info(
		"Copy + paste the deposit data below when using the " +
			"eth1 deposit contract")
	fmt.Printf(`
========================Deposit Data===============================

%#x

===================================================================`, tx.Data())
	fmt.Printf(`
***Enter the above deposit data into step 3 on https://prylabs.net/participate***
`)
}
