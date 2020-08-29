package derived

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/petnames"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/iface"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	util "github.com/wealdtech/go-eth2-util"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

var log = logrus.WithField("prefix", "derived-keymanager-v2")

const (
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
	// EncryptedSeedFileName for persisting a wallet's seed when using a derived keymanager.
	EncryptedSeedFileName = "seed.encrypted.json"
)

// SeedConfig json file representation as a Go struct.
type SeedConfig struct {
	Crypto      map[string]interface{} `json:"crypto"`
	ID          string                 `json:"uuid"`
	NextAccount uint64                 `json:"next_account"`
	Version     uint                   `json:"version"`
	Name        string                 `json:"name"`
}

// KeymanagerOpts defines options for the keymanager that
// are stored to disk in the wallet.
type KeymanagerOpts struct {
	DerivedPathStructure string `json:"derived_path_structure"`
	DerivedEIPNumber     string `json:"derived_eip_number"`
}

// SetupConfig includes configuration values for initializing
// a keymanager, such as passwords, the wallet, and more.
type SetupConfig struct {
	Opts                *KeymanagerOpts
	Wallet              iface.Wallet
	SkipMnemonicConfirm bool
	WalletPassword      string
	Mnemonic            string
}

// Keymanager implementation for derived, HD keymanager using EIP-2333 and EIP-2334.
type Keymanager struct {
	wallet            iface.Wallet
	opts              *KeymanagerOpts
	mnemonicGenerator SeedPhraseFactory
	keysCache         map[[48]byte]bls.SecretKey
	lock              sync.RWMutex
	seedCfg           *SeedConfig
	seed              []byte
	accountsPassword  string
}

// DefaultKeymanagerOpts for a derived keymanager implementation.
func DefaultKeymanagerOpts() *KeymanagerOpts {
	return &KeymanagerOpts{
		DerivedPathStructure: "m / purpose / coin_type / account_index / withdrawal_key / validating_key",
		DerivedEIPNumber:     EIPVersion,
	}
}

// NewKeymanager instantiates a new derived keymanager from configuration options.
func NewKeymanager(
	ctx context.Context,
	cfg *SetupConfig,
) (*Keymanager, error) {
	// Check if the wallet seed file exists. If it does not, we initialize one
	// by creating a new mnemonic and writing the encrypted file to disk.
	var encodedSeedFile []byte
	if !fileutil.FileExists(filepath.Join(cfg.Wallet.AccountsDir(), EncryptedSeedFileName)) {
		seedConfig, err := initializeWalletSeedFile(cfg.WalletPassword, cfg.SkipMnemonicConfirm)
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize new wallet seed file")
		}
		encodedSeedFile, err = marshalEncryptedSeedFile(seedConfig)
		if err != nil {
			return nil, errors.Wrap(err, "could not marshal encrypted wallet seed file")
		}
		if err = cfg.Wallet.WriteEncryptedSeedToDisk(ctx, encodedSeedFile); err != nil {
			return nil, errors.Wrap(err, "could not write encrypted wallet seed config to disk")
		}
	} else {
		seedConfigFile, err := cfg.Wallet.ReadEncryptedSeedFromDisk(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not read encrypted seed file from disk")
		}
		encodedSeedFile, err = ioutil.ReadAll(seedConfigFile)
		if err != nil {
			return nil, errors.Wrap(err, "could not read seed configuration file contents")
		}
		defer func() {
			if err := seedConfigFile.Close(); err != nil {
				log.Errorf("Could not close keymanager config file: %v", err)
			}
		}()
	}
	seedConfig := &SeedConfig{}
	if err := json.Unmarshal(encodedSeedFile, seedConfig); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal seed configuration")
	}
	decryptor := keystorev4.New()
	seed, err := decryptor.Decrypt(seedConfig.Crypto, cfg.WalletPassword)
	if err != nil {
		return nil, errors.Wrap(err, "could not decrypt seed configuration with password")
	}
	k := &Keymanager{
		wallet: cfg.Wallet,
		opts:   cfg.Opts,
		mnemonicGenerator: &EnglishMnemonicGenerator{
			skipMnemonicConfirm: cfg.SkipMnemonicConfirm,
		},
		seedCfg:          seedConfig,
		seed:             seed,
		accountsPassword: cfg.WalletPassword,
		keysCache:        make(map[[48]byte]bls.SecretKey),
	}
	// We initialize a cache of public key -> secret keys
	// used to retrieve secrets keys for the accounts via the unlocked wallet.
	// This cache is needed to process Sign requests using a validating public key.
	if err := k.initializeSecretKeysCache(); err != nil {
		return nil, errors.Wrap(err, "could not initialize secret keys cache")
	}
	return k, nil
}

// KeymanagerForPhrase instantiates a new derived keymanager from configuration and an existing mnemonic phrase provided.
func KeymanagerForPhrase(
	ctx context.Context,
	cfg *SetupConfig,
) (*Keymanager, error) {
	// Check if the wallet seed file exists. If it does not, we initialize one
	// by creating a new mnemonic and writing the encrypted file to disk.
	var encodedSeedFile []byte
	seedConfig, err := seedFileFromMnemonic(cfg.Mnemonic, cfg.WalletPassword)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize new wallet seed file")
	}
	encodedSeedFile, err = marshalEncryptedSeedFile(seedConfig)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal encrypted wallet seed file")
	}
	if err = cfg.Wallet.WriteEncryptedSeedToDisk(ctx, encodedSeedFile); err != nil {
		return nil, errors.Wrap(err, "could not write encrypted wallet seed config to disk")
	}
	decryptor := keystorev4.New()
	seed, err := decryptor.Decrypt(seedConfig.Crypto, cfg.WalletPassword)
	if err != nil {
		return nil, errors.Wrap(err, "could not decrypt seed configuration with password")
	}
	k := &Keymanager{
		wallet: cfg.Wallet,
		opts:   cfg.Opts,
		mnemonicGenerator: &EnglishMnemonicGenerator{
			skipMnemonicConfirm: true,
		},
		seedCfg:          seedConfig,
		seed:             seed,
		accountsPassword: cfg.WalletPassword,
		keysCache:        make(map[[48]byte]bls.SecretKey),
	}
	// We initialize a cache of public key -> secret keys
	// used to retrieve secrets keys for the accounts via the unlocked wallet.
	// This cache is needed to process Sign requests using a validating public key.
	if err := k.initializeSecretKeysCache(); err != nil {
		return nil, errors.Wrap(err, "could not initialize secret keys cache")
	}
	return k, nil
}

// UnmarshalOptionsFile attempts to JSON unmarshal a derived keymanager
// options file into the *Config{} struct.
func UnmarshalOptionsFile(r io.ReadCloser) (*KeymanagerOpts, error) {
	enc, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Errorf("Could not close keymanager config file: %v", err)
		}
	}()
	opts := &KeymanagerOpts{}
	if err := json.Unmarshal(enc, opts); err != nil {
		return nil, err
	}
	return cfg, nil
}

// MarshalOptionsFile returns a marshaled options file for a keymanager.
func MarshalOptionsFile(ctx context.Context, opts *KeymanagerOpts) ([]byte, error) {
	return json.MarshalIndent(opts, "", "\t")
}

// KeymanagerOpts returns the derived keymanager options.
func (dr *Keymanager) KeymanagerOpts() *KeymanagerOpts {
	return dr.opts
}

// NextAccountNumber managed by the derived keymanager.
func (dr *Keymanager) NextAccountNumber(ctx context.Context) uint64 {
	return dr.seedCfg.NextAccount
}

// WriteEncryptedSeedToWallet given a mnemonic phrase, is able to regenerate a wallet seed
// encrypt it, and write it to the wallet's path.
func (dr *Keymanager) WriteEncryptedSeedToWallet(ctx context.Context, mnemonic string) error {
	seedConfig, err := seedFileFromMnemonic(mnemonic, dr.accountsPassword)
	if err != nil {
		return errors.Wrap(err, "could not initialize new wallet seed file")
	}
	seedConfigFile, err := marshalEncryptedSeedFile(seedConfig)
	if err != nil {
		return errors.Wrap(err, "could not marshal encrypted wallet seed file")
	}
	if err := dr.wallet.WriteEncryptedSeedToDisk(ctx, seedConfigFile); err != nil {
		return errors.Wrap(err, "could not write encrypted wallet seed config to disk")
	}
	return nil
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
		names = append(names, petnames.DeterministicName(validatingKey.PublicKey().Marshal(), "-"))
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
	// Upon confirmation of the withdrawal key, proceed to display
	// and write associated deposit data to disk.
	tx, _, err := depositutil.GenerateDepositTransaction(blsValidatingKey, blsWithdrawalKey)
	if err != nil {
		return "", errors.Wrap(err, "could not generate deposit transaction data")
	}

	// Log the deposit transaction data to the user.
	fmt.Printf(`
======================Eth1 Deposit Transaction Data================
%#x
===================================================================`, tx.Data())
	fmt.Println("")

	// Finally, write the account creation timestamps as a files.
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
	encodedCfg, err := marshalEncryptedSeedFile(dr.seedCfg)
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
		validatingKey, err := util.PrivateKeyFromSeedAndPath(dr.seed, validatingKeyPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create validating key for account %d", i)
		}
		publicKeys = append(publicKeys, bytesutil.ToBytes48(validatingKey.PublicKey().Marshal()))
	}
	return publicKeys, nil
}

// FetchWithdrawalPublicKeys fetches the list of withdrawal public keys from keymanager
func (dr *Keymanager) FetchWithdrawalPublicKeys(ctx context.Context) ([][48]byte, error) {
	publicKeys := make([][48]byte, 0)
	for i := uint64(0); i < dr.seedCfg.NextAccount; i++ {
		withdrawalKeyPath := fmt.Sprintf(WithdrawalKeyDerivationPathTemplate, i)
		withdrawalKey, err := util.PrivateKeyFromSeedAndPath(dr.seed, withdrawalKeyPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create validating key for account %d", i)
		}
		publicKeys = append(publicKeys, bytesutil.ToBytes48(withdrawalKey.PublicKey().Marshal()))
	}
	return publicKeys, nil
}

// DepositDataForAccount with a given index returns the RLP encoded eth1 deposit transaction data.
func (dr *Keymanager) DepositDataForAccount(accountIndex uint64) ([]byte, error) {
	withdrawalKeyPath := fmt.Sprintf(WithdrawalKeyDerivationPathTemplate, accountIndex)
	validatingKeyPath := fmt.Sprintf(ValidatingKeyDerivationPathTemplate, accountIndex)
	withdrawalKey, err := util.PrivateKeyFromSeedAndPath(dr.seed, withdrawalKeyPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create withdrawal key for account %d", accountIndex)
	}
	validatingKey, err := util.PrivateKeyFromSeedAndPath(dr.seed, validatingKeyPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create validating key for account %d", accountIndex)
	}

	// Upon confirmation of the withdrawal key, proceed to display
	// and write associated deposit data to disk.
	blsValidatingKey, err := bls.SecretKeyFromBytes(validatingKey.Marshal())
	if err != nil {
		return nil, err
	}
	blsWithdrawalKey, err := bls.SecretKeyFromBytes(withdrawalKey.Marshal())
	if err != nil {
		return nil, err
	}
	tx, _, err := depositutil.GenerateDepositTransaction(blsValidatingKey, blsWithdrawalKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate deposit transaction data")
	}
	return tx.Data(), nil
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

// Creates a new, encrypted seed using a password input
// and persists its encrypted file metadata to disk under the wallet path.
func initializeWalletSeedFile(password string, skipMnemonicConfirm bool) (*SeedConfig, error) {
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

// Uses the provided mnemonic seed phrase to generate the
// appropriate seed file for recovering a derived wallets.
func seedFileFromMnemonic(mnemonic string, password string) (*SeedConfig, error) {
	if ok := bip39.IsMnemonicValid(mnemonic); !ok {
		return nil, bip39.ErrInvalidMnemonic
	}
	walletSeed := bip39.NewSeed(mnemonic, "")
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

// marshalEncryptedSeedFile json encodes the seed configuration for a derived keymanager.
func marshalEncryptedSeedFile(seedCfg *SeedConfig) ([]byte, error) {
	return json.MarshalIndent(seedCfg, "", "\t")
}
