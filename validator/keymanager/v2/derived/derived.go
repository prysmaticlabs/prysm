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
	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/sirupsen/logrus"
	util "github.com/wealdtech/go-eth2-util"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

var log = logrus.WithField("prefix", "derived-keymanager-v2")

const (
	// TimestampFileName stores a timestamp for account creation as a
	// file for a direct keymanager account.
	TimestampFileName = "created_at.txt"
	// KeystoreFileName exposes the expected filename for the keystore file for an account.
	KeystoreFileName = "keystore.json"
	// SeedFileName defines a json file storing an encrypted derived wallet seed.
	SeedFileName = "seed.json"
	eipVersion   = "EIP-2334"
)

// Wallet defines a struct which has capabilities and knowledge of how
// to read and write important accounts-related files to the filesystem.
// Useful for keymanager to have persistent capabilities for accounts on-disk.
type Wallet interface {
	AccountsDir() string
	CanUnlockAccounts() bool
	WriteFileAtPath(ctx context.Context, pathName string, fileName string, data []byte) error
	ReadEncryptedSeedFromDisk(ctx context.Context) (io.ReadCloser, error)
}

// Config for a derived keymanager.
type Config struct {
	DerivedPathStructure string
	DerivedEIPNumber     string
}

// Keymanager implementation for derived, HD keymanager using EIP-2333 and EIP-2334.
type Keymanager struct {
	wallet            Wallet
	cfg               *Config
	mnemonicGenerator SeedPhraseFactory
	keysCache         map[[48]byte]bls.SecretKey
	lock              sync.RWMutex
	accountNum        uint64
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
		DerivedEIPNumber:     eipVersion,
	}
}

// NewKeymanager instantiates a new direct keymanager from configuration options.
func NewKeymanager(
	ctx context.Context,
	wallet Wallet,
	cfg *Config,
	skipMnemonicConfirm bool,
	password string,
) (*Keymanager, error) {
	seedConfigFile, err := wallet.ReadEncryptedSeedFromDisk(ctx)
	if err != nil {
		return nil, err
	}
	enc, err := ioutil.ReadAll(seedConfigFile)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := seedConfigFile.Close(); err != nil {
			log.Errorf("Could not close keymanager config file: %v", err)
		}
	}()
	seedConfig := &SeedConfig{}
	log.Info(enc)
	if err := json.Unmarshal(enc, cfg); err != nil {
		return nil, err
	}
	decryptor := keystorev4.New()
	seed, err := decryptor.Derypt(seedConfig.Crypto, []byte(password))
	if err != nil {
		return nil, err
	}
	k := &Keymanager{
		wallet: wallet,
		cfg:    cfg,
		mnemonicGenerator: &EnglishMnemonicGenerator{
			skipMnemonicConfirm: skipMnemonicConfirm,
		},
		seedCfg:    seedConfig,
		seed:       seed,
		accountNum: seedConfig.NextAccount - 1, // TODO: Check for underflow.
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

func InitializeWalletSeed(ctx context.Context) ([]byte, error) {
	walletSeed := make([]byte, 32)
	n, err := rand.NewGenerator().Read(walletSeed)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize wallet seed")
	}
	if n != len(walletSeed) {
		return nil, errors.New("could not randomly create seed")
	}
	m := &EnglishMnemonicGenerator{
		skipMnemonicConfirm: false,
	}
	phrase, err := m.Generate(walletSeed)
	if err != nil {
		return nil, err
	}
	if err := m.ConfirmAcknowledgement(phrase); err != nil {
		return nil, err
	}
	return walletSeed, nil
}

func MarshalEncryptedSeedFile(ctx context.Context, seed []byte, password string) ([]byte, error) {
	encryptor := keystorev4.New()
	cryptoFields, err := encryptor.Encrypt(seed, []byte(password))
	if err != nil {
		return nil, errors.Wrap(err, "could not encrypt seed phrase into keystore")
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	seedFile := &SeedConfig{}
	seedFile.Crypto = cryptoFields
	seedFile.ID = id.String()
	seedFile.NextAccount = 1
	seedFile.Version = encryptor.Version()
	seedFile.Name = encryptor.Name()
	return json.MarshalIndent(seedFile, "", "\t")
}

// CreateAccount for a derived keymanager implementation. This utilizes
// the EIP-2335 keystore standard for BLS12-381 keystores. It uses the EIP-2333 and EIP-2334
// for hierarchical derivation of BLS secret keys and a common derivation path structure for
// persisting accounts to disk. Each account stores the generated keystore.json file.
// The entire derived wallet seed phrase can be recovered from a BIP-39 english mnemonic.
func (dr *Keymanager) CreateAccount(ctx context.Context, password string) (string, error) {
	// TODO: needs better formatting at the top
	withdrawalKeyPath := fmt.Sprintf("m/12381/3600/%d/0", dr.accountNum)
	validatingKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", dr.accountNum)
	// TODO: better seed.
	seed := make([]byte, 32)
	copy(seed, "hello world")

	withdrawalKey, err := util.PrivateKeyFromSeedAndPath(seed, withdrawalKeyPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create withdrawal key for account %d", dr.accountNum)
	}
	validatingKey, err := util.PrivateKeyFromSeedAndPath(seed, validatingKeyPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create validating key for account %d", dr.accountNum)
	}

	// Create encrypted keystores for both the withdrawal and validating keys.
	encodedWithdrawalKeystore, err := dr.generateKeystoreFile(
		withdrawalKey.Marshal(),
		withdrawalKey.PublicKey().Marshal(),
		password,
	)
	if err != nil {
		return "", err
	}
	encodedValidatingKeystore, err := dr.generateKeystoreFile(
		validatingKey.Marshal(),
		validatingKey.PublicKey().Marshal(),
		password,
	)
	if err != nil {
		return "", err
	}

	// Write both keystores to disk at their respective derived paths.
	if err := dr.wallet.WriteFileAtPath(ctx, withdrawalKeyPath, KeystoreFileName, encodedWithdrawalKeystore); err != nil {
		return "", errors.Wrapf(err, "could not write keystore file for account %d", dr.accountNum)
	}
	if err := dr.wallet.WriteFileAtPath(ctx, validatingKeyPath, KeystoreFileName, encodedValidatingKeystore); err != nil {
		return "", errors.Wrapf(err, "could not write keystore file for account %d", dr.accountNum)
	}

	// Finally, write the account creation timestamps as a files.
	createdAt := roughtime.Now().Unix()
	createdAtStr := strconv.FormatInt(createdAt, 10)
	if err := dr.wallet.WriteFileAtPath(ctx, withdrawalKeyPath, TimestampFileName, []byte(createdAtStr)); err != nil {
		return "", errors.Wrapf(err, "could not write timestamp file for account %d", dr.accountNum)
	}
	if err := dr.wallet.WriteFileAtPath(ctx, validatingKeyPath, TimestampFileName, []byte(createdAtStr)); err != nil {
		return "", errors.Wrapf(err, "could not write timestamp file for account %d", dr.accountNum)
	}

	log.WithFields(logrus.Fields{
		"accountNumber":       dr.accountNum,
		"validatingPublicKey": fmt.Sprintf("%#x", validatingKey.PublicKey().Marshal()),
		"path":                path.Join(dr.wallet.AccountsDir(), withdrawalKeyPath),
	}).Info("Successfully created new validator account")
	dr.accountNum++
	return fmt.Sprintf("%d", dr.accountNum), nil
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
		return nil, err
	}
	keystoreFile := &Keystore{}
	keystoreFile.Crypto = cryptoFields
	keystoreFile.ID = id.String()
	keystoreFile.Pubkey = fmt.Sprintf("%x", publicKey)
	keystoreFile.Version = encryptor.Version()
	keystoreFile.Name = encryptor.Name()
	return json.MarshalIndent(keystoreFile, "", "\t")
}
