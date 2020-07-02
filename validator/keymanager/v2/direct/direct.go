package direct

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/brianium/mnemonic"
	"github.com/brianium/mnemonic/entropy"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/manifoldco/promptui"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	contract "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

var log = logrus.WithField("prefix", "keymanager-v2")

const (
	keystoreFileName           = "keystore.json"
	depositDataFileName        = "deposit_data.ssz"
	depositTransactionFileName = "deposit_transaction.rlp"
	mnemonicLanguage           = mnemonic.English
)

// Wallet defines a struct which has capabilities and knowledge of how
// to read and write important accounts-related files to the filesystem.
// Useful for keymanager to have persistent capabilities for accounts on-disk.
type Wallet interface {
	WriteAccountToDisk(ctx context.Context, password string) (string, error)
	WriteFileForAccount(ctx context.Context, accountName string, fileName string, data []byte) error
	WriteKeymanagerConfigToDisk(ctx context.Context, encoded []byte) error
	ReadKeymanagerConfigFromDisk(ctx context.Context) (io.ReadCloser, error)
}

// Config for a direct keymanager.
type Config struct{}

// Keymanager implementation for direct keystores.
type Keymanager struct {
	wallet Wallet
}

// DefaultConfig for a direct keymanager implementation.
func DefaultConfig() *Config {
	return &Config{}
}

// NewKeymanager instantiates a new direct keymanager from configuration options.
func NewKeymanager(ctx context.Context, wallet Wallet, cfg *Config) *Keymanager {
	return &Keymanager{
		wallet: wallet,
	}
}

// NewKeymanagerFromConfigFile instantiates a direct keymanager instance
// from a configuration file accesed via a wallet.
// TODO(#6220): Implement.
func NewKeymanagerFromConfigFile(ctx context.Context, wallet Wallet) (*Keymanager, error) {
	return &Keymanager{
		wallet: wallet,
	}, nil
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
		return err
	}
	// Generates a new EIP-2335 compliant keystore file
	// from a BLS private key and marshals it as JSON.
	encryptor := keystorev4.New()
	validatingKey := bls.RandKey()
	keystoreFile, err := encryptor.Encrypt(validatingKey.Marshal(), []byte(password))
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(keystoreFile, "", "\t")
	if err != nil {
		return err
	}
	if err := dr.wallet.WriteFileForAccount(ctx, accountName, keystoreFileName, encoded); err != nil {
		return err
	}

	// Generate a withdrawal key and confirm user
	// acknowledgement of a 256-bit entropy mnemonic phrase.
	withdrawalKey := bls.RandKey()
	if err := dr.confirmWithdrawalMnemonic(withdrawalKey); err != nil {
		return err
	}

	// Upon confirmation of the withdrawal key, proceed to display
	// and write associated deposit data to disk.
	tx, depositData, err := generateDepositData(validatingKey, withdrawalKey)
	if err != nil {
		return err
	}

	// Log the deposit transaction data to the user.
	logDepositTransaction(tx)

	// We write the raw deposit transaction as an .rlp encoded file.
	if err := dr.wallet.WriteFileForAccount(ctx, accountName, depositTransactionFileName, tx.Data()); err != nil {
		return err
	}
	// We write the ssz-encoded deposit data to disk as a .ssz file.
	encodedDepositData, err := depositData.MarshalSSZ()
	if err != nil {
		return err
	}
	if err := dr.wallet.WriteFileForAccount(ctx, accountName, depositDataFileName, encodedDepositData); err != nil {
		return err
	}
	fmt.Println("***Enter the above deposit data into step 3 on https://prylabs.net/participate***")
	return nil
}

// MarshalConfigFile returns a marshaled configuration file for a direct keymanager.
// TODO(#6220): Implement.
func (dr *Keymanager) MarshalConfigFile(ctx context.Context) ([]byte, error) {
	return nil, nil
}

// FetchValidatingPublicKeys fetches the list of public keys from the direct account keystores.
func (dr *Keymanager) FetchValidatingPublicKeys() ([][48]byte, error) {
	return nil, errors.New("unimplemented")
}

// Sign signs a message using a validator key.
func (dr *Keymanager) Sign(context.Context, interface{}) (bls.Signature, error) {
	return nil, errors.New("unimplemented")
}

func (dr *Keymanager) confirmWithdrawalMnemonic(withdrawalKey bls.SecretKey) error {
	key := withdrawalKey.Marshal()[:]
	ent, err := entropy.FromHex(fmt.Sprintf("%x", key))
	if err != nil {
		return err
	}
	en, err := mnemonic.New(ent, mnemonicLanguage)
	if err != nil {
		return err
	}
	log.Info(
		"Write down the following sentence somewhere safe, as it is your only " +
			"means of recovering your validator withdrawal key",
	)
	fmt.Printf(`
	=================Withdrawal Key Recovery Phrase====================

	%s

	===================================================================
	`, en.Sentence())
	prompt := promptui.Prompt{
		Label:     "Confirm you have written down the words above somewhere safe (offline)",
		IsConfirm: true,
	}
	expected := "y"
	var result string
	for result != expected {
		result, err = prompt.Run()
		if err != nil {
			return fmt.Errorf("could not confirm acknowledgement: %v", formatPromptError(err))
		}
	}
	return nil
}

func generateDepositData(
	validatingKey bls.SecretKey,
	withdrawalKey bls.SecretKey,
) (*types.Transaction, *ethpb.Deposit_Data, error) {
	depositData, depositRoot, err := depositInput(
		validatingKey, withdrawalKey, params.BeaconConfig().MaxEffectiveBalance,
	)
	if err != nil {
		return nil, nil, err
	}
	testAcc, err := contract.Setup()
	if err != nil {
		return nil, nil, err
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
		"Copy and paste the raw deposit data shown below when issuing a transaction into the " +
			"ETH1.0 deposit contract to activate your validator client")
	fmt.Printf(`
	========================Deposit Data=======================

	%#x

	===================================================================
	`, tx.Data())
	fmt.Println("***Enter the above deposit data into step 3 on https://prylabs.net/participate***")
}

func depositInput(
	depositKey bls.SecretKey,
	withdrawalKey bls.SecretKey,
	amountInGwei uint64,
) (*ethpb.Deposit_Data, [32]byte, error) {
	di := &ethpb.Deposit_Data{
		PublicKey:             depositKey.Marshal(),
		WithdrawalCredentials: withdrawalCredentialsHash(withdrawalKey),
		Amount:                amountInGwei,
	}

	sr, err := ssz.SigningRoot(di)
	if err != nil {
		return nil, [32]byte{}, err
	}

	domain, err := helpers.ComputeDomain(params.BeaconConfig().DomainDeposit, nil /*forkVersion*/, nil /*genesisValidatorsRoot*/)
	if err != nil {
		return nil, [32]byte{}, err
	}
	root, err := ssz.HashTreeRoot(&pb.SigningData{ObjectRoot: sr[:], Domain: domain})
	if err != nil {
		return nil, [32]byte{}, err
	}
	di.Signature = depositKey.Sign(root[:]).Marshal()

	dr, err := ssz.HashTreeRoot(di)
	if err != nil {
		return nil, [32]byte{}, err
	}

	return di, dr, nil
}

// withdrawalCredentialsHash forms a 32 byte hash of the withdrawal public
// address.
//
// The specification is as follows:
//   withdrawal_credentials[:1] == BLS_WITHDRAWAL_PREFIX_BYTE
//   withdrawal_credentials[1:] == hash(withdrawal_pubkey)[1:]
// where withdrawal_credentials is of type bytes32.
func withdrawalCredentialsHash(withdrawalKey bls.SecretKey) []byte {
	h := hashutil.Hash(withdrawalKey.PublicKey().Marshal())
	return append([]byte{params.BeaconConfig().BLSWithdrawalPrefixByte}, h[1:]...)[:32]
}

func formatPromptError(err error) error {
	switch err {
	case promptui.ErrAbort:
		return errors.New("wallet creation aborted, closing")
	case promptui.ErrInterrupt:
		return errors.New("keyboard interrupt, closing")
	case promptui.ErrEOF:
		return errors.New("no input received, closing")
	default:
		return err
	}
}
