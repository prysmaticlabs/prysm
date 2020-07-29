package derived

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	"github.com/prysmaticlabs/prysm/shared/petnames"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/iface"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	util "github.com/wealdtech/go-eth2-util"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

var log = logrus.WithField("prefix", "derived-keymanager-v2")

const (
	// TimestampFileName stores a timestamp for account creation as a
	// file for a direct keymanager account.
	TimestampFileName = "created_at.txt"
	// KeystoreFilePattern exposes the expected filename for the keystore file for an account.
	KeystoreFilePattern = "keystore.json"
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
	// DepositTransactionFileName for the encoded, eth1 raw deposit tx data
	// for a validator account.
	DepositTransactionFileName = "deposit_transaction.rlp"
	// DepositDataFileName for the raw, ssz-encoded deposit data object.
	DepositDataFileName = "deposit_data.ssz"
	// EncryptedSeedFileName for persisting a wallet's seed when using a derived keymanager.
	EncryptedSeedFileName = "seed.encrypted.json"
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
	walletPassword    string
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
		DerivedPathStructure: "m / purpose / coin_type / account_index / withdrawal_key / validating_key",
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
		return nil, errors.Wrap(err, "could not read encrypted seed file from disk")
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
	log.Info(seedConfig)
	decryptor := keystorev4.New()
	seed, err := decryptor.Decrypt(seedConfig.Crypto, password)
	if err != nil {
		return nil, errors.Wrap(err, "could not decrypt seed configuration with password")
	}
	k := &Keymanager{
		wallet: wallet,
		cfg:    cfg,
		mnemonicGenerator: &EnglishMnemonicGenerator{
			skipMnemonicConfirm: skipMnemonicConfirm,
		},
		seedCfg:        seedConfig,
		seed:           seed,
		walletPassword: password,
		keysCache:      make(map[[48]byte]bls.SecretKey),
	}
	// We initialize a cache of public key -> secret keys
	// used to retrieve secrets keys for the accounts via the unlocked wallet.
	// This cache is needed to process Sign requests using a validating public key.
	if err := k.initializeSecretKeysCache(); err != nil {
		return nil, errors.Wrap(err, "could not initialize secret keys cache")
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
	mnemonicRandomness := make([]byte, 32)
	if _, err := rand.NewGenerator().Read(mnemonicRandomness); err != nil {
		return nil, errors.Wrap(err, "could not initialize mnemonic source of randomness")
	}
	m := &EnglishMnemonicGenerator{
		skipMnemonicConfirm: skipMnemonicConfirm,
	}
	phrase, err := m.Generate(mnemonicRandomness)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate wallet seed")
	}
	if err := m.ConfirmAcknowledgement(phrase); err != nil {
		return nil, errors.Wrap(err, "could not confirm mnemonic acknowledgement")
	}
	walletSeed := bip39.NewSeed(phrase, "")
	fmt.Printf("New wallet seed: %#x", walletSeed)
	encryptor := keystorev4.New()
	cryptoFields, err := encryptor.Encrypt(walletSeed, password)
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

// SeedFileFromMnemonic uses the provided mnemonic seed phrase to generate the
// appropriate seed file for recovering a derived wallets.
func SeedFileFromMnemonic(ctx context.Context, mnemonic string, password string) (*SeedConfig, error) {
	if ok := bip39.IsMnemonicValid(mnemonic); !ok {
		return nil, bip39.ErrInvalidMnemonic
	}
	walletSeed := bip39.NewSeed(mnemonic, "")
	encryptor := keystorev4.New()
	fmt.Printf("Seed: %#x\n", walletSeed)
	cryptoFields, err := encryptor.Encrypt(walletSeed, password)
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

// Config returns the derived keymanager configuration.
func (dr *Keymanager) Config() *Config {
	return dr.cfg
}

// NextAccountNumber managed by the derived keymanager.
func (dr *Keymanager) NextAccountNumber(ctx context.Context) uint64 {
	return dr.seedCfg.NextAccount
}

// ValidatingAccountNames for the derived keymanager.
func (dr *Keymanager) ValidatingAccountNames(ctx context.Context) ([]string, error) {
	names := make([]string, 0)
	for i := uint64(0); i < dr.seedCfg.NextAccount; i++ {
		validatingKeyPath := fmt.Sprintf(ValidatingKeyDerivationPathTemplate, i)
		validatingKey, err := util.PrivateKeyFromSeedAndPath(dr.seed, validatingKeyPath)
		if err != nil {
			return nil, errors.Wrap(err, "could not derive validating key")
		}
		names = append(names, petnames.DeterministicName(validatingKey.Marshal(), "-"))
	}
	return names, nil
}

// CreateAccount for a derived keymanager implementation. This utilizes
// the EIP-2335 keystore standard for BLS12-381 keystores. It uses the EIP-2333 and EIP-2334
// for hierarchical derivation of BLS secret keys and a common derivation path structure for
// persisting accounts to disk. Each account stores the generated keystore.json file.
// The entire derived wallet seed phrase can be recovered from a BIP-39 english mnemonic.
func (dr *Keymanager) CreateAccount(ctx context.Context, logAccountInfo bool) (string, error) {
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
	fmt.Printf("Seed: %#x\n", dr.seed)
	fmt.Printf("Withdrawal key: %#x\n", withdrawalKey.Marshal())
	fmt.Printf("Withdrawal key path: %s\n", withdrawalKeyPath)

	// Create encrypted keystores for both the withdrawal and validating keys.
	encodedWithdrawalKeystore, err := dr.generateKeystoreFile(
		withdrawalKey.Marshal(),
		withdrawalKey.PublicKey().Marshal(),
		dr.walletPassword,
	)
	if err != nil {
		return "", errors.Wrap(err, "could not generate keystore file for withdrawal account")
	}
	encodedValidatingKeystore, err := dr.generateKeystoreFile(
		validatingKey.Marshal(),
		validatingKey.PublicKey().Marshal(),
		dr.walletPassword,
	)
	if err != nil {
		return "", errors.Wrap(err, "could not generate keystore file for validating account")
	}

	// Write both keystores to disk at their respective derived paths.
	if err := dr.wallet.WriteFileAtPath(ctx, withdrawalKeyPath, KeystoreFilePattern, encodedWithdrawalKeystore); err != nil {
		return "", errors.Wrapf(err, "could not write keystore file for account %d", dr.seedCfg.NextAccount)
	}
	if err := dr.wallet.WriteFileAtPath(ctx, validatingKeyPath, KeystoreFilePattern, encodedValidatingKeystore); err != nil {
		return "", errors.Wrapf(err, "could not write keystore file for account %d", dr.seedCfg.NextAccount)
	}

	// Upon confirmation of the withdrawal key, proceed to display
	// and write associated deposit data to disk.
	blsValidatingKey, err := bls.SecretKeyFromBytes(validatingKey.Marshal())
	if err != nil {
		return "", err
	}
	blsWithdrawalKey, err := bls.SecretKeyFromBytes(withdrawalKey.Marshal())
	if err != nil {
		return "", err
	}
	tx, depositData, err := depositutil.GenerateDepositTransaction(blsValidatingKey, blsWithdrawalKey)
	if err != nil {
		return "", errors.Wrap(err, "could not generate deposit transaction data")
	}

	if logAccountInfo {
		// Log the deposit transaction data to the user.
		depositutil.LogDepositTransaction(log, tx)
	}

	// We write the raw deposit transaction as an .rlp encoded file.
	if err := dr.wallet.WriteFileAtPath(ctx, withdrawalKeyPath, DepositTransactionFileName, tx.Data()); err != nil {
		return "", errors.Wrapf(err, "could not write for account %s: %s", withdrawalKeyPath, DepositTransactionFileName)
	}

	// We write the ssz-encoded deposit data to disk as a .ssz file.
	encodedDepositData, err := ssz.Marshal(depositData)
	if err != nil {
		return "", errors.Wrap(err, "could not marshal deposit data")
	}
	if err := dr.wallet.WriteFileAtPath(ctx, withdrawalKeyPath, DepositDataFileName, encodedDepositData); err != nil {
		return "", errors.Wrapf(err, "could not write for account %s: %s", withdrawalKeyPath, encodedDepositData)
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
	if logAccountInfo {
		log.WithFields(logrus.Fields{
			"accountNumber":       newAccountNumber,
			"withdrawalPublicKey": fmt.Sprintf("%#x", withdrawalKey.PublicKey().Marshal()),
			"validatingPublicKey": fmt.Sprintf("%#x", validatingKey.PublicKey().Marshal()),
			"withdrawalKeyPath":   path.Join(dr.wallet.AccountsDir(), withdrawalKeyPath),
			"validatingKeyPath":   path.Join(dr.wallet.AccountsDir(), validatingKeyPath),
		}).Info("Successfully created new validator account")
	}
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

// Sign signs a message using a validator key.
func (dr *Keymanager) Sign(ctx context.Context, req *validatorpb.SignRequest) (bls.Signature, error) {
	rawPubKey := req.PublicKey
	if rawPubKey == nil {
		return nil, errors.New("nil public key in request")
	}
	dr.lock.RLock()
	defer dr.lock.RUnlock()
	secretKey, ok := dr.keysCache[bytesutil.ToBytes48(rawPubKey)]
	if !ok {
		return nil, errors.New("no signing key found in keys cache")
	}
	return secretKey.Sign(req.SigningRoot), nil
}

// FetchValidatingPublicKeys fetches the list of validating public keys from the keymanager.
func (dr *Keymanager) FetchValidatingPublicKeys(ctx context.Context) ([][48]byte, error) {
	// Return the public keys from the cache if they match the
	// number of accounts from the wallet.
	publicKeys := make([][48]byte, dr.seedCfg.NextAccount)
	dr.lock.RLock()
	defer dr.lock.RUnlock()
	if dr.keysCache != nil && uint64(len(dr.keysCache)) == dr.seedCfg.NextAccount {
		var i int
		for k := range dr.keysCache {
			publicKeys[i] = k
			i++
		}
		return publicKeys, nil
	}
	for i := uint64(0); i < dr.seedCfg.NextAccount; i++ {
		validatingKeyPath := fmt.Sprintf(ValidatingKeyDerivationPathTemplate, i)
		validatingKeystore, err := dr.wallet.ReadFileAtPath(ctx, validatingKeyPath, KeystoreFilePattern)
		if err != nil {
			return nil, err
		}
		keystoreFile := &v2keymanager.Keystore{}
		if err := json.Unmarshal(validatingKeystore, keystoreFile); err != nil {
			return nil, errors.Wrapf(err, "could not decode keystore json for account: %s", validatingKeyPath)
		}
		pubKeyBytes, err := hex.DecodeString(keystoreFile.Pubkey)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode pubkey bytes: %#x", keystoreFile.Pubkey)
		}
		publicKeys = append(publicKeys, bytesutil.ToBytes48(pubKeyBytes))
	}
	return publicKeys, nil
}

// FetchWithdrawalPublicKeys fetches the list of withdrawal public keys from keymanager
func (dr *Keymanager) FetchWithdrawalPublicKeys(ctx context.Context) ([][48]byte, error) {
	publicKeys := make([][48]byte, 0)
	for i := uint64(0); i < dr.seedCfg.NextAccount; i++ {
		withdrawalKeyPath := fmt.Sprintf(WithdrawalKeyDerivationPathTemplate, i)
		withdrawalKeystore, err := dr.wallet.ReadFileAtPath(ctx, withdrawalKeyPath, KeystoreFilePattern)
		if err != nil {
			return nil, err
		}
		keystoreFile := &v2keymanager.Keystore{}
		if err := json.Unmarshal(withdrawalKeystore, keystoreFile); err != nil {
			return nil, errors.Wrapf(err, "could not decode keystore json for account: %s", withdrawalKeyPath)
		}
		pubKeyBytes, err := hex.DecodeString(keystoreFile.Pubkey)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode pubkey bytes: %#x", keystoreFile.Pubkey)
		}
		publicKeys = append(publicKeys, bytesutil.ToBytes48(pubKeyBytes))
	}
	return publicKeys, nil
}

func (dr *Keymanager) initializeSecretKeysCache() error {
	dr.lock.Lock()
	defer dr.lock.Unlock()
	for i := uint64(0); i < dr.seedCfg.NextAccount; i++ {
		validatingKeyPath := fmt.Sprintf(ValidatingKeyDerivationPathTemplate, i)
		derivedKey, err := util.PrivateKeyFromSeedAndPath(dr.seed, validatingKeyPath)
		if err != nil {
			return errors.Wrapf(err, "failed to derive validating key for account %s", validatingKeyPath)
		}
		validatorSigningKey, err := bls.SecretKeyFromBytes(derivedKey.Marshal())
		if err != nil {
			return errors.Wrapf(
				err,
				"could not instantiate bls secret key from bytes for account: %s",
				validatingKeyPath,
			)
		}

		// Update a simple cache of public key -> secret key utilized
		// for fast signing access in the keymanager.
		dr.keysCache[bytesutil.ToBytes48(validatorSigningKey.PublicKey().Marshal())] = validatorSigningKey
	}
	return nil
}

func (dr *Keymanager) generateKeystoreFile(privateKey []byte, publicKey []byte, password string) ([]byte, error) {
	encryptor := keystorev4.New()
	cryptoFields, err := encryptor.Encrypt(privateKey, password)
	if err != nil {
		return nil, errors.Wrap(err, "could not encrypt validating key into keystore")
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "could not generate new, random UUID for keystore")
	}
	keystoreFile := &v2keymanager.Keystore{
		Crypto:  cryptoFields,
		ID:      id.String(),
		Pubkey:  fmt.Sprintf("%x", publicKey),
		Version: encryptor.Version(),
		Name:    encryptor.Name(),
	}
	return json.MarshalIndent(keystoreFile, "", "\t")
}
