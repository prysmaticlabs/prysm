package direct

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/k0kubun/go-ansi"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"

	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	"github.com/prysmaticlabs/prysm/shared/mputil"
	"github.com/prysmaticlabs/prysm/shared/petnames"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/iface"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
)

var log = logrus.WithField("prefix", "direct-keymanager-v2")

const (
	// KeystoreFileName exposes the expected filename for the keystore file for an account.
	KeystoreFileName = "keystore-*.json"
	// KeystoreFileNameFormat exposes the filename the keystore should be formatted in.
	KeystoreFileNameFormat = "keystore-%d.json"
	// PasswordFileSuffix for passwords persisted as text to disk.
	PasswordFileSuffix = ".pass"
	// DepositDataFileName for the ssz-encoded deposit.
	DepositDataFileName = "deposit_data.ssz"
	eipVersion          = "EIP-2335"
	errFileNotFound     = "no such file"
	errPasswordNotMatch = "invalid checksum"
)

// Config for a direct keymanager.
type Config struct {
	EIPVersion                string `json:"direct_eip_version"`
	AccountPasswordsDirectory string `json:"direct_accounts_passwords_directory"`
}

// Keymanager implementation for direct keystores utilizing EIP-2335.
type Keymanager struct {
	wallet    iface.Wallet
	cfg       *Config
	keysCache map[[48]byte]bls.SecretKey
	lock      sync.RWMutex
}

// DefaultConfig for a direct keymanager implementation.
func DefaultConfig() *Config {
	return &Config{
		EIPVersion:                eipVersion,
		AccountPasswordsDirectory: flags.WalletPasswordsDirFlag.Value,
	}
}

// NewKeymanager instantiates a new direct keymanager from configuration options.
func NewKeymanager(ctx context.Context, wallet iface.Wallet, cfg *Config) (*Keymanager, error) {
	k := &Keymanager{
		wallet:    wallet,
		cfg:       cfg,
		keysCache: make(map[[48]byte]bls.SecretKey),
	}
	// If the wallet has the capability of unlocking accounts using
	// passphrases, then we initialize a cache of public key -> secret keys
	// used to retrieve secrets keys for the accounts via password unlock.
	// This cache is needed to process Sign requests using a public key.
	if err := k.initializeSecretKeysCache(ctx); err != nil {
		return nil, err
	}
	return k, nil
}

// UnmarshalConfigFile attempts to JSON unmarshal a direct keymanager
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

// Config for the direct keymanager.
func (dr *Keymanager) Config() *Config {
	return dr.cfg
}

// String pretty-print of a direct keymanager configuration.
func (c *Config) String() string {
	au := aurora.NewAurora(true)
	var b strings.Builder
	strAddr := fmt.Sprintf("%s: %s\n", au.BrightMagenta("EIP Version"), c.EIPVersion)
	if _, err := b.WriteString(strAddr); err != nil {
		log.Error(err)
		return ""
	}
	strCrt := fmt.Sprintf(
		"%s: %s\n", au.BrightMagenta("Accounts Passwords Directory"), c.AccountPasswordsDirectory,
	)
	if _, err := b.WriteString(strCrt); err != nil {
		log.Error(err)
		return ""
	}
	return b.String()
}

// ValidatingAccountNames for a direct keymanager.
func (dr *Keymanager) ValidatingAccountNames() ([]string, error) {
	return dr.wallet.ListDirs()
}

// CreateAccount for a direct keymanager implementation. This utilizes
// the EIP-2335 keystore standard for BLS12-381 keystores. It
// stores the generated keystore.json file in the wallet and additionally
// generates withdrawal credentials. At the end, it logs
// the raw deposit data hex string for users to copy.
func (dr *Keymanager) CreateAccount(ctx context.Context, password string) (string, error) {
	// Create a petname for an account from its public key and write its password to disk.
	validatingKey := bls.RandKey()
	accountName, err := dr.generateAccountName(validatingKey.PublicKey().Marshal())
	if err != nil {
		return "", errors.Wrap(err, "could not generate unique account name")
	}
	if err := dr.wallet.WritePasswordToDisk(ctx, accountName+".pass", password); err != nil {
		return "", errors.Wrap(err, "could not write password to disk")
	}
	// Generates a new EIP-2335 compliant keystore file
	// from a BLS private key and marshals it as JSON.
	encoded, err := dr.generateKeystoreFile(validatingKey, password)
	if err != nil {
		return "", err
	}

	// Generate a withdrawal key and confirm user
	// acknowledgement of a 256-bit entropy mnemonic phrase.
	withdrawalKey := bls.RandKey()
	log.Info(
		"Write down the private key, as it is your unique " +
			"withdrawal private key for eth2",
	)
	fmt.Printf(`
==========================Withdrawal Key===========================

%#x

===================================================================
	`, withdrawalKey.Marshal())
	fmt.Println(" ")

	// Upon confirmation of the withdrawal key, proceed to display
	// and write associated deposit data to disk.
	_, depositData, err := depositutil.GenerateDepositTransaction(validatingKey, withdrawalKey)
	if err != nil {
		return "", errors.Wrap(err, "could not generate deposit transaction data")
	}

	// We write the ssz-encoded deposit data to disk as a .ssz file.
	encodedDepositData, err := ssz.Marshal(depositData)
	if err != nil {
		return "", errors.Wrap(err, "could not marshal deposit data")
	}
	if err := dr.wallet.WriteFileAtPath(ctx, accountName, DepositDataFileName, encodedDepositData); err != nil {
		return "", errors.Wrapf(err, "could not write for account %s: %s", accountName, encodedDepositData)
	}

	// Log the deposit transaction data to the user.
	fmt.Printf(`
========================SSZ Deposit Data===============================

%#x

===================================================================`, encodedDepositData)

	// Write the encoded keystore to disk with the timestamp appended
	createdAt := roughtime.Now().Unix()
	if err := dr.wallet.WriteFileAtPath(ctx, accountName, fmt.Sprintf(KeystoreFileNameFormat, createdAt), encoded); err != nil {
		return "", errors.Wrapf(err, "could not write keystore file for account %s", accountName)
	}

	log.WithFields(logrus.Fields{
		"name": accountName,
		"path": dr.wallet.AccountsDir(),
	}).Info("Successfully created new validator account")
	return accountName, nil
}

// FetchValidatingPublicKeys fetches the list of public keys from the direct account keystores.
func (dr *Keymanager) FetchValidatingPublicKeys(ctx context.Context) ([][48]byte, error) {
	accountNames, err := dr.ValidatingAccountNames()
	if err != nil {
		return nil, err
	}

	// Return the public keys from the cache if they match the
	// number of accounts from the wallet.
	publicKeys := make([][48]byte, len(accountNames))
	dr.lock.Lock()
	defer dr.lock.Unlock()
	if dr.keysCache != nil && len(dr.keysCache) == len(accountNames) {
		var i int
		for k := range dr.keysCache {
			publicKeys[i] = k
			i++
		}
		return publicKeys, nil
	}

	for i, name := range accountNames {
		encoded, err := dr.wallet.ReadFileAtPath(ctx, name, KeystoreFileName)
		if err != nil {
			return nil, errors.Wrapf(err, "could not read keystore file for account %s", name)
		}
		keystoreFile := &v2keymanager.Keystore{}
		if err := json.Unmarshal(encoded, keystoreFile); err != nil {
			return nil, errors.Wrapf(err, "could not decode keystore json for account: %s", name)
		}
		pubKeyBytes, err := hex.DecodeString(keystoreFile.Pubkey)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode pubkey bytes: %#x", keystoreFile.Pubkey)
		}
		publicKeys[i] = bytesutil.ToBytes48(pubKeyBytes)
	}
	return publicKeys, nil
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

// PublicKeyForAccount returns the associated public key for an account name.
func (dr *Keymanager) PublicKeyForAccount(accountName string) ([48]byte, error) {
	accountKeystore, err := dr.keystoreForAccount(accountName)
	if err != nil {
		return [48]byte{}, errors.Wrap(err, "could not get keystore")
	}
	pubKey, err := hex.DecodeString(accountKeystore.Pubkey)
	if err != nil {
		return [48]byte{}, errors.Wrap(err, "could decode pubkey string")
	}
	return bytesutil.ToBytes48(pubKey), nil
}

func (dr *Keymanager) keystoreForAccount(accountName string) (*v2keymanager.Keystore, error) {
	encoded, err := dr.wallet.ReadFileAtPath(context.Background(), accountName, KeystoreFileName)
	if err != nil {
		return nil, errors.Wrap(err, "could not read keystore file")
	}
	keystoreJSON := &v2keymanager.Keystore{}
	if err := json.Unmarshal(encoded, &keystoreJSON); err != nil {
		return nil, errors.Wrap(err, "could not decode json")
	}
	return keystoreJSON, nil
}

func (dr *Keymanager) initializeSecretKeysCache(ctx context.Context) error {
	accountNames, err := dr.ValidatingAccountNames()
	if err != nil {
		return err
	}
	if len(accountNames) == 0 {
		return nil
	}
	log.Info("Decrypting validator accounts...")
	// We initialize a nice progress bar to offer the user feedback
	// during this slow operation.
	bar := initializeProgressBar(len(accountNames))
	progressChan := make(chan struct{}, len(accountNames))
	go func() {
		defer close(progressChan)
		var itemsReceived int
		for range progressChan {
			itemsReceived++
			if err := bar.Add(1); err != nil {
				log.WithError(err).Debug("Could not increase progress bar")
			}
			if itemsReceived == len(accountNames) {
				return
			}
		}
	}()

	keystoreFiles := make([]*v2keymanager.Keystore, len(accountNames))
	passwords := make([]string, len(accountNames))
	for i := 0; i < len(accountNames); i++ {
		name := accountNames[i]
		encoded, err := dr.wallet.ReadFileAtPath(ctx, name, KeystoreFileName)
		if err != nil {
			return errors.Wrapf(err, "could not read keystore file for account %s", name)
		}
		keystoreFile := &v2keymanager.Keystore{}
		if err := json.Unmarshal(encoded, keystoreFile); err != nil {
			return errors.Wrapf(err, "could not decode keystore file for account %s", name)
		}
		password, err := dr.wallet.ReadPasswordFromDisk(ctx, name+PasswordFileSuffix)
		// If the password is suddenly not found for an account, prompt the user for it.
		if err != nil && strings.Contains(err.Error(), errFileNotFound) {
			password, err = dr.promptForAccountPassword(ctx, name, keystoreFile.Pubkey)
			if err != nil {
				return err
			}
		} else if err != nil {
			return errors.Wrapf(err, "could not read password for account %s", name)
		}
		keystoreFiles[i] = keystoreFile
		passwords[i] = password
	}
	decryptor := keystorev4.New()
	_, err = mputil.Scatter(len(keystoreFiles), func(offset int, entries int, lock *sync.RWMutex) (interface{}, error) {
		lock.Lock()
		defer lock.Unlock()
		accountNameChunks := accountNames[offset : offset+entries]
		passwordChunks := passwords[offset : offset+entries]
		keystoreFileChunks := keystoreFiles[offset : offset+entries]
		for i := 0; i < len(keystoreFileChunks[offset:offset+entries]); i++ {
			// We extract the validator signing private key from the keystore
			// by utilizing the password and initialize a new BLS secret key from
			// its raw bytes.
			rawSigningKey, err := decryptor.Decrypt(keystoreFileChunks[i].Crypto, passwordChunks[i])
			if err != nil {
				return nil, errors.Wrapf(err, "could not decrypt signing key for account %s", accountNameChunks[i])
			}
			validatorSigningKey, err := bls.SecretKeyFromBytes(rawSigningKey)
			if err != nil {
				return nil, errors.Wrapf(err, "could not determine signing key for account %s", accountNameChunks[i])
			}
			// Update a simple cache of public key -> secret key utilized
			// for fast signing access in the direct keymanager.
			dr.keysCache[bytesutil.ToBytes48(validatorSigningKey.PublicKey().Marshal())] = validatorSigningKey
			progressChan <- struct{}{}
		}
		return nil, nil
	})
	return err
}

func (dr *Keymanager) promptForAccountPassword(ctx context.Context, accountName string, publicKey string) (string, error) {
	promptText := fmt.Sprintf("Enter password for account with public key %s", publicKey)
	password, err := promptutil.PasswordPrompt(promptText, promptutil.NotEmpty)
	if err != nil {
		return "", fmt.Errorf("could not read account password: %v", err)
	}
	err = dr.checkPasswordForAccount(accountName, password)
	invalidChecksum := err != nil && strings.Contains(err.Error(), errPasswordNotMatch)
	for invalidChecksum {
		password, err := promptutil.PasswordPrompt(promptText, promptutil.NotEmpty)
		if err != nil {
			return "", fmt.Errorf("could not read account password: %v", err)
		}
		err = dr.checkPasswordForAccount(accountName, password)
		invalidChecksum = err != nil && strings.Contains(err.Error(), errPasswordNotMatch)
	}
	if err := dr.wallet.WritePasswordToDisk(ctx, accountName+PasswordFileSuffix, password); err != nil {
		return "", errors.Wrap(err, "could not write password to disk")
	}
	return password, nil
}

func (dr *Keymanager) generateKeystoreFile(validatingKey bls.SecretKey, password string) ([]byte, error) {
	encryptor := keystorev4.New()
	cryptoFields, err := encryptor.Encrypt(validatingKey.Marshal(), password)
	if err != nil {
		return nil, errors.Wrap(err, "could not encrypt validating key into keystore")
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	keystoreFile := &v2keymanager.Keystore{
		Crypto:  cryptoFields,
		ID:      id.String(),
		Pubkey:  fmt.Sprintf("%x", validatingKey.PublicKey().Marshal()),
		Version: encryptor.Version(),
		Name:    encryptor.Name(),
	}
	return json.MarshalIndent(keystoreFile, "", "\t")
}

func (dr *Keymanager) generateAccountName(pubKey []byte) (string, error) {
	var accountExists bool
	var accountName string
	for !accountExists {
		accountName = petnames.DeterministicName(pubKey, "-")
		exists, err := hasDir(filepath.Join(dr.wallet.AccountsDir(), accountName))
		if err != nil {
			return "", errors.Wrapf(err, "could not check if account exists in dir: %s", dr.wallet.AccountsDir())
		}
		if !exists {
			break
		}
	}
	return accountName, nil
}

func (dr *Keymanager) checkPasswordForAccount(accountName string, password string) error {
	accountKeystore, err := dr.keystoreForAccount(accountName)
	if err != nil {
		return errors.Wrap(err, "could not get keystore")
	}
	decryptor := keystorev4.New()
	_, err = decryptor.Decrypt(accountKeystore.Crypto, password)
	if err != nil {
		return errors.Wrap(err, "could not decrypt keystore")
	}
	return nil
}

func initializeProgressBar(numItems int) *progressbar.ProgressBar {
	return progressbar.NewOptions(
		numItems,
		progressbar.OptionFullWidth(),
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
}

// Checks if a directory indeed exists at the specified path.
func hasDir(dirPath string) (bool, error) {
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return info.IsDir(), err
}
