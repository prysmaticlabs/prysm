package derived

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	util "github.com/wealdtech/go-eth2-util"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"

	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/iface"
)

var log = logrus.WithField("prefix", "derived-keymanager-v2")

const (
	// TimestampFileName stores a timestamp for account creation as a
	// file for a direct keymanager account.
	TimestampFileName = "created_at.txt"
	// KeystoreFileName exposes the expected filename for the keystore file for an account.
	KeystoreFileName = "keystore.json"
	// EIPVersion used by this derived keymanager implementation.
	EIPVersion = "EIP-2334"
	// WithdrawalKeyDerivationPathTemplate defining the hierarchical path for withdrawal
	// keys for Prysm eth2 validators. According to EIP-2334, the format is as follows:
	// m / purpose / coin_type / account_index / withdrawal_key
	WithdrawalKeyDerivationPathTemplate = "m/12381/3600/%d/0"
	// ValidatingKeyDerivationPathTemplate defining the hierarchical path for validating
	// keys for Prysm eth2 validators. According to EIP-2334, the format is as follows:
	// m / purpose / coin_type / account_index / withdrawal_key / validating_key
	ValidatingKeyDerivationPathTemplate = "m/12381/3600/%d/0/0"
)

// Config for a derived keymanager.
type Config struct {
	DerivedPathStructure string
	DerivedEIPNumber     string
}

// Keymanager implementation for derived, HD keymanager using EIP-2333 and EIP-2334.
type Keymanager struct {
	wallet            iface.Wallet
	cfg               *Config
	mnemonicGenerator SeedPhraseFactory
	keysCache         map[[48]byte]bls.SecretKey
	lock              sync.RWMutex
	seedCfg           *SeedConfig
	seed              []byte
}

// Keystore json file representation as a Go struct.
type Keystore struct {
	Crypto  map[string]interface{} `json:"crypto"`
	ID      string                 `json:"uuid"`
	Pubkey  string                 `json:"pubkey"`
	Version uint                   `json:"version"`
	Name    string                 `json:"name"`
}

// SeedConfig json file representation as a Go struct.
type SeedConfig struct {
	Crypto      map[string]interface{} `json:"crypto"`
	ID          string                 `json:"uuid"`
	NextAccount uint64                 `json:"next_account"`
	Version     uint                   `json:"version"`
	Name        string                 `json:"name"`
}

// DefaultConfig for a derived keymanager implementation.
func DefaultConfig() *Config {
	return &Config{
		DerivedPathStructure: "m / purpose / coin_type / account / withdrawal_key / validating_key",
		DerivedEIPNumber:     EIPVersion,
	}
}

// NewKeymanager instantiates a new derived keymanager from configuration options.
func NewKeymanager(
	ctx context.Context,
	wallet iface.Wallet,
	cfg *Config,
	skipMnemonicConfirm bool,
	password string,
) (*Keymanager, error) {
	seedConfigFile, err := wallet.ReadEncryptedSeedFromDisk(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not read encrypted seed configuration file from disk")
	}
	enc, err := ioutil.ReadAll(seedConfigFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not read seed configuration file contents")
	}
	defer func() {
		if err := seedConfigFile.Close(); err != nil {
			log.Errorf("Could not close keymanager config file: %v", err)
		}
	}()
	seedConfig := &SeedConfig{}
	if err := json.Unmarshal(enc, seedConfig); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal seed configuration")
	}
	decryptor := keystorev4.New()
	seed, err := decryptor.Decrypt(seedConfig.Crypto, []byte(password))
	if err != nil {
		return nil, errors.Wrap(err, "could not decrypt seed configuration with password")
	}
	k := &Keymanager{
		wallet: wallet,
		cfg:    cfg,
		mnemonicGenerator: &EnglishMnemonicGenerator{
			skipMnemonicConfirm: skipMnemonicConfirm,
		},
		seedCfg: seedConfig,
		seed:    seed,
	}
	return k, nil
}

// UnmarshalConfigFile attempts to JSON unmarshal a derived keymanager
// configuration file into the *Config{} struct.
func UnmarshalConfigFile(r io.ReadCloser) (*Config, error) {
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

// MarshalConfigFile returns a marshaled configuration file for a keymanager.
func MarshalConfigFile(ctx context.Context, cfg *Config) ([]byte, error) {
	return json.MarshalIndent(cfg, "", "\t")
}

// InitializeWalletSeedFile creates a new, encrypted seed using a password input
// and persists its encrypted file metadata to disk under the wallet path.
func InitializeWalletSeedFile(ctx context.Context, password string, skipMnemonicConfirm bool) (*SeedConfig, error) {
	walletSeed := make([]byte, 32)
	n, err := rand.NewGenerator().Read(walletSeed)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize wallet seed")
	}
	if n != len(walletSeed) {
		return nil, errors.New("could not randomly create seed")
	}
	m := &EnglishMnemonicGenerator{
		skipMnemonicConfirm: skipMnemonicConfirm,
	}
	phrase, err := m.Generate(walletSeed)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate wallet seed")
	}
	if err := m.ConfirmAcknowledgement(phrase); err != nil {
		return nil, errors.Wrap(err, "could not confirm mnemonic acknowledgement")
	}
	encryptor := keystorev4.New()
	cryptoFields, err := encryptor.Encrypt(walletSeed, []byte(password))
	if err != nil {
		return nil, errors.Wrap(err, "could not encrypt seed phrase into keystore")
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "could not generate unique UUID")
	}
	return &SeedConfig{
		Crypto:      cryptoFields,
		ID:          id.String(),
		NextAccount: 0,
		Version:     encryptor.Version(),
		Name:        encryptor.Name(),
	}, nil
}

// MarshalEncryptedSeedFile json encodes the seed configuration for a derived keymanager.
func MarshalEncryptedSeedFile(ctx context.Context, seedCfg *SeedConfig) ([]byte, error) {
	return json.MarshalIndent(seedCfg, "", "\t")
}

// CreateAccount for a derived keymanager implementation. This utilizes
// the EIP-2335 keystore standard for BLS12-381 keystores. It uses the EIP-2333 and EIP-2334
// for hierarchical derivation of BLS secret keys and a common derivation path structure for
// persisting accounts to disk. Each account stores the generated keystore.json file.
// The entire derived wallet seed phrase can be recovered from a BIP-39 english mnemonic.
func (dr *Keymanager) CreateAccount(ctx context.Context, password string) (string, error) {
	withdrawalKeyPath := fmt.Sprintf(WithdrawalKeyDerivationPathTemplate, dr.seedCfg.NextAccount)
	validatingKeyPath := fmt.Sprintf(ValidatingKeyDerivationPathTemplate, dr.seedCfg.NextAccount)
	withdrawalKey, err := util.PrivateKeyFromSeedAndPath(dr.seed, withdrawalKeyPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create withdrawal key for account %d", dr.seedCfg.NextAccount)
	}
	validatingKey, err := util.PrivateKeyFromSeedAndPath(dr.seed, validatingKeyPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create validating key for account %d", dr.seedCfg.NextAccount)
	}

	// Create encrypted keystores for both the withdrawal and validating keys.
	encodedWithdrawalKeystore, err := dr.generateKeystoreFile(
		withdrawalKey.Marshal(),
		withdrawalKey.PublicKey().Marshal(),
		password,
	)
	if err != nil {
		return "", errors.Wrap(err, "could not generate keystore file for withdrawal account")
	}
	encodedValidatingKeystore, err := dr.generateKeystoreFile(
		validatingKey.Marshal(),
		validatingKey.PublicKey().Marshal(),
		password,
	)
	if err != nil {
		return "", errors.Wrap(err, "could not generate keystore file for validating account")
	}

	// Write both keystores to disk at their respective derived paths.
	if err := dr.wallet.WriteFileAtPath(ctx, withdrawalKeyPath, KeystoreFileName, encodedWithdrawalKeystore); err != nil {
		return "", errors.Wrapf(err, "could not write keystore file for account %d", dr.seedCfg.NextAccount)
	}
	if err := dr.wallet.WriteFileAtPath(ctx, validatingKeyPath, KeystoreFileName, encodedValidatingKeystore); err != nil {
		return "", errors.Wrapf(err, "could not write keystore file for account %d", dr.seedCfg.NextAccount)
	}

	// Finally, write the account creation timestamps as a files.
	createdAt := roughtime.Now().Unix()
	createdAtStr := strconv.FormatInt(createdAt, 10)
	if err := dr.wallet.WriteFileAtPath(ctx, withdrawalKeyPath, TimestampFileName, []byte(createdAtStr)); err != nil {
		return "", errors.Wrapf(err, "could not write timestamp file for account %d", dr.seedCfg.NextAccount)
	}
	if err := dr.wallet.WriteFileAtPath(ctx, validatingKeyPath, TimestampFileName, []byte(createdAtStr)); err != nil {
		return "", errors.Wrapf(err, "could not write timestamp file for account %d", dr.seedCfg.NextAccount)
	}

	newAccountNumber := dr.seedCfg.NextAccount
	log.WithFields(logrus.Fields{
		"accountNumber":       newAccountNumber,
		"withdrawalPublicKey": fmt.Sprintf("%#x", withdrawalKey.PublicKey().Marshal()),
		"validatingPublicKey": fmt.Sprintf("%#x", validatingKey.PublicKey().Marshal()),
		"withdrawalKeyPath":   path.Join(dr.wallet.AccountsDir(), withdrawalKeyPath),
		"validatingKeyPath":   path.Join(dr.wallet.AccountsDir(), validatingKeyPath),
	}).Info("Successfully created new validator account")
	dr.seedCfg.NextAccount++
	encodedCfg, err := MarshalEncryptedSeedFile(ctx, dr.seedCfg)
	if err != nil {
		return "", errors.Wrap(err, "could not marshal encrypted seed file")
	}
	if err := dr.wallet.WriteEncryptedSeedToDisk(ctx, encodedCfg); err != nil {
		return "", errors.Wrap(err, "could not write encrypted seed file to disk")
	}
	return fmt.Sprintf("%d", newAccountNumber), nil
}

// FetchValidatingPublicKeys fetches the list of public keys from the direct account keystores.
func (dr *Keymanager) FetchValidatingPublicKeys(ctx context.Context) ([][48]byte, error) {
	return nil, errors.New("unimplemented")
}

// Sign signs a message using a validator key.
func (dr *Keymanager) Sign(ctx context.Context, req *validatorpb.SignRequest) (bls.Signature, error) {
	return nil, errors.New("unimplemented")
}

func (dr *Keymanager) generateKeystoreFile(privateKey []byte, publicKey []byte, password string) ([]byte, error) {
	encryptor := keystorev4.New()
	cryptoFields, err := encryptor.Encrypt(privateKey, []byte(password))
	if err != nil {
		return nil, errors.Wrap(err, "could not encrypt validating key into keystore")
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "could not generate new, random UUID for keystore")
	}
	keystoreFile := &Keystore{
		Crypto:  cryptoFields,
		ID:      id.String(),
		Pubkey:  fmt.Sprintf("%x", publicKey),
		Version: encryptor.Version(),
		Name:    encryptor.Name(),
	}
	return json.MarshalIndent(keystoreFile, "", "\t")
}
