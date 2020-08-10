package direct

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/petnames"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/iface"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

var log = logrus.WithField("prefix", "direct-keymanager-v2")

const (
	// KeystoreFileName exposes the expected filename for the keystore file for an account.
	KeystoreFileName = "keystore-*.json"
	// KeystoreFileNameFormat exposes the filename the keystore should be formatted in.
	KeystoreFileNameFormat = "keystore-%d.json"
	// AccountsPath where all direct keymanager keystores are kept.
	AccountsPath                = "accounts"
	accountsKeystoreFileName    = "all-accounts.keystore.json"
	eipVersion                  = "EIP-2335"
	newWalletPasswordPromptText = "New wallet password"
	walletPasswordPromptText    = "Wallet password"
	confirmPasswordPromptText   = "Confirm password"
)

type passwordConfirm int

const (
	// An enum to indicate to the prompt that confirming the password is not needed.
	noConfirmPass passwordConfirm = iota
	// An enum to indicate to the prompt to confirm the password entered.
	confirmPass
)

// Config for a direct keymanager.
type Config struct {
	EIPVersion string `json:"direct_eip_version"`
}

// Keymanager implementation for direct keystores utilizing EIP-2335.
type Keymanager struct {
	wallet           iface.Wallet
	cfg              *Config
	keysCache        map[[48]byte]bls.SecretKey
	accountsStore    *AccountStore
	lock             sync.RWMutex
	accountsPassword string
}

// AccountStore --
type AccountStore struct {
	PrivateKeys [][]byte `json:"private_keys"`
	PublicKeys  [][]byte `json:"public_keys"`
}

// DefaultConfig for a direct keymanager implementation.
func DefaultConfig() *Config {
	return &Config{
		EIPVersion: eipVersion,
	}
}

// NewKeymanager instantiates a new direct keymanager from configuration options.
func NewKeymanager(ctx *cli.Context, wallet iface.Wallet, cfg *Config) (*Keymanager, error) {
	k := &Keymanager{
		wallet:        wallet,
		cfg:           cfg,
		keysCache:     make(map[[48]byte]bls.SecretKey),
		accountsStore: &AccountStore{},
	}

	walletFiles, err := wallet.ListDirs()
	if err != nil {
		return nil, err
	}
	var accountsPassword string
	if len(walletFiles) == 0 {
		accountsPassword, err = inputPassword(
			ctx,
			flags.WalletPasswordFileFlag,
			newWalletPasswordPromptText,
			confirmPass,
			promptutil.ValidatePasswordInput,
		)
	} else {
		validateExistingPass := func(input string) error {
			if input == "" {
				return errors.New("password input cannot be empty")
			}
			return nil
		}
		accountsPassword, err = inputPassword(
			ctx,
			flags.WalletPasswordFileFlag,
			walletPasswordPromptText,
			noConfirmPass,
			validateExistingPass,
		)
	}
	if err != nil {
		return nil, err
	}
	k.accountsPassword = accountsPassword

	// If the wallet has the capability of unlocking accounts using
	// passphrases, then we initialize a cache of public key -> secret keys
	// used to retrieve secrets keys for the accounts via password unlock.
	// This cache is needed to process Sign requests using a public key.
	if err := k.initializeSecretKeysCache(ctx); err != nil {
		return nil, errors.Wrap(err, "could not initialize keys cache")
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
	return b.String()
}

// ValidatingAccountNames for a direct keymanager.
func (dr *Keymanager) ValidatingAccountNames() ([]string, error) {
	names := make([]string, len(dr.accountsStore.PublicKeys))
	for i, pubKey := range dr.accountsStore.PublicKeys {
		names[i] = petnames.DeterministicName(pubKey, "-")
	}
	return names, nil
}

// CreateAccount for a direct keymanager implementation. This utilizes
// the EIP-2335 keystore standard for BLS12-381 keystores. It
// stores the generated keystore.json file in the wallet and additionally
// generates withdrawal credentials. At the end, it logs
// the raw deposit data hex string for users to copy.
func (dr *Keymanager) CreateAccount(ctx context.Context) (string, error) {
	// Create a petname for an account from its public key and write its password to disk.
	validatingKey := bls.RandKey()
	accountName := petnames.DeterministicName(validatingKey.PublicKey().Marshal(), "-")
	dr.accountsStore.PrivateKeys = append(dr.accountsStore.PrivateKeys, validatingKey.Marshal())
	dr.accountsStore.PublicKeys = append(dr.accountsStore.PublicKeys, validatingKey.PublicKey().Marshal())
	newStore, err := dr.createAccountsKeystore(ctx, dr.accountsStore.PrivateKeys, dr.accountsStore.PublicKeys)
	if err != nil {
		return "", errors.Wrap(err, "could not create accounts keystore")
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
	tx, _, err := depositutil.GenerateDepositTransaction(validatingKey, withdrawalKey)
	if err != nil {
		return "", errors.Wrap(err, "could not generate deposit transaction data")
	}

	// Log the deposit transaction data to the user.
	fmt.Printf(`
======================Eth1 Deposit Transaction Data================
%#x
===================================================================`, tx.Data())
	fmt.Println("")

	// Write the encoded keystore to disk with the timestamp appended
	encoded, err := json.MarshalIndent(newStore, "", "\t")
	if err != nil {
		return "", err
	}
	if err := dr.wallet.WriteFileAtPath(ctx, AccountsPath, accountsKeystoreFileName, encoded); err != nil {
		return "", errors.Wrap(err, "could not write keystore file for accounts")
	}

	log.WithFields(logrus.Fields{
		"name": accountName,
	}).Info("Successfully created new validator account")
	dr.lock.Lock()
	dr.keysCache[bytesutil.ToBytes48(validatingKey.PublicKey().Marshal())] = validatingKey
	dr.lock.Unlock()
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
	return nil, nil
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

func (dr *Keymanager) initializeSecretKeysCache(cliCtx *cli.Context) error {
	encoded, err := dr.wallet.ReadFileAtPath(context.Background(), AccountsPath, accountsKeystoreFileName)
	if err != nil && strings.Contains(err.Error(), "no files found") {
		// If there are no keys to initialize at all, just exit.
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "could not read keystore file for accounts %s", accountsKeystoreFileName)
	}
	keystoreFile := &v2keymanager.Keystore{}
	if err := json.Unmarshal(encoded, keystoreFile); err != nil {
		return errors.Wrapf(err, "could not decode keystore file for accounts %s", accountsKeystoreFileName)
	}
	// We extract the validator signing private key from the keystore
	// by utilizing the password and initialize a new BLS secret key from
	// its raw bytes.
	decryptor := keystorev4.New()
	enc, err := decryptor.Decrypt(keystoreFile.Crypto, dr.accountsPassword)
	if err != nil && strings.Contains(err.Error(), "invalid checksum") {
		// If the password fails for an individual account, we ask the user to input
		// that individual account's password until it succeeds.
		enc, dr.accountsPassword, err = dr.askUntilPasswordConfirms(decryptor, keystoreFile)
		if err != nil {
			return errors.Wrap(err, "could not confirm password via prompt")
		}
	} else if err != nil {
		return errors.Wrap(err, "could not decrypt keystore")
	}

	store := &AccountStore{}
	if err := json.Unmarshal(enc, store); err != nil {
		return err
	}
	if len(store.PublicKeys) != len(store.PrivateKeys) {
		return errors.New("unequal number of public keys and private keys")
	}
	if len(store.PublicKeys) == 0 {
		return nil
	}
	dr.lock.Lock()
	defer dr.lock.Unlock()
	for i := 0; i < len(store.PublicKeys); i++ {
		privKey, err := bls.SecretKeyFromBytes(store.PrivateKeys[i])
		if err != nil {
			return err
		}
		dr.keysCache[bytesutil.ToBytes48(store.PublicKeys[i])] = privKey
	}
	dr.accountsStore = store
	return err
}

func (dr *Keymanager) createAccountsKeystore(
	ctx context.Context,
	privateKeys [][]byte,
	publicKeys [][]byte,
) (*v2keymanager.Keystore, error) {
	au := aurora.NewAurora(true)
	encryptor := keystorev4.New()
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	if len(privateKeys) != len(publicKeys) {
		return nil, fmt.Errorf(
			"number of private keys and public keys is not equal: %d != %d", len(privateKeys), len(publicKeys),
		)
	}
	if dr.accountsStore == nil {
		dr.accountsStore = &AccountStore{
			PrivateKeys: privateKeys,
			PublicKeys:  publicKeys,
		}
	} else {
		existingPubKeys := make(map[string]bool)
		existingPrivKeys := make(map[string]bool)
		for i := 0; i < len(dr.accountsStore.PrivateKeys); i++ {
			existingPrivKeys[string(dr.accountsStore.PrivateKeys[i])] = true
			existingPubKeys[string(dr.accountsStore.PublicKeys[i])] = true
		}
		// We append to the accounts store keys only
		// if the private/secret key do not already exist, to prevent duplicates.
		for i := 0; i < len(privateKeys); i++ {
			sk := privateKeys[i]
			pk := publicKeys[i]
			_, privKeyExists := existingPrivKeys[string(sk)]
			_, pubKeyExists := existingPubKeys[string(pk)]
			if privKeyExists || pubKeyExists {
				continue
			}
			dr.accountsStore.PublicKeys = append(dr.accountsStore.PublicKeys, pk)
			dr.accountsStore.PrivateKeys = append(dr.accountsStore.PrivateKeys, sk)
		}
	}
	encodedStore, err := json.MarshalIndent(dr.accountsStore, "", "\t")
	if err != nil {
		return nil, err
	}
	cryptoFields, err := encryptor.Encrypt(encodedStore, dr.accountsPassword)
	if err != nil {
		return nil, errors.Wrap(err, "could not encrypt accounts")
	}
	return &v2keymanager.Keystore{
		Crypto:  cryptoFields,
		ID:      id.String(),
		Version: encryptor.Version(),
		Name:    encryptor.Name(),
	}, nil
}

func (dr *Keymanager) askUntilPasswordConfirms(
	decryptor *keystorev4.Encryptor, keystore *v2keymanager.Keystore,
) ([]byte, string, error) {
	au := aurora.NewAurora(true)
	// Loop asking for the password until the user enters it correctly.
	var secretKey []byte
	var password string
	var err error
	publicKey, err := hex.DecodeString(keystore.Pubkey)
	if err != nil {
		return nil, "", errors.Wrap(err, "could not decode public key")
	}
	formattedPublicKey := fmt.Sprintf("%#x", bytesutil.Trunc(publicKey))
	for {
		password, err = promptutil.PasswordPrompt(
			fmt.Sprintf("\nPlease try again, could not use password to import account %s", au.BrightGreen(formattedPublicKey)),
			promptutil.NotEmpty,
		)
		if err != nil {
			return nil, "", fmt.Errorf("could not read account password: %v", err)
		}
		secretKey, err = decryptor.Decrypt(keystore.Crypto, password)
		if err != nil && strings.Contains(err.Error(), "invalid checksum") {
			fmt.Print(au.Red("Incorrect password entered, please try again"))
			continue
		}
		if err != nil {
			return nil, "", err
		}
		break
	}
	return secretKey, password, nil
}

func inputPassword(
	cliCtx *cli.Context,
	passwordFileFlag *cli.StringFlag,
	promptText string,
	confirmPassword passwordConfirm,
	passwordValidator func(input string) error,
) (string, error) {
	if cliCtx.IsSet(passwordFileFlag.Name) {
		passwordFilePathInput := cliCtx.String(passwordFileFlag.Name)
		passwordFilePath, err := fileutil.ExpandPath(passwordFilePathInput)
		if err != nil {
			return "", errors.Wrap(err, "could not determine absolute path of password file")
		}
		data, err := ioutil.ReadFile(passwordFilePath)
		if err != nil {
			return "", errors.Wrap(err, "could not read password file")
		}
		enteredPassword := strings.TrimRight(string(data), "\r\n")
		if err := passwordValidator(enteredPassword); err != nil {
			return "", errors.Wrap(err, "password did not pass validation")
		}
		return enteredPassword, nil
	}
	var hasValidPassword bool
	var walletPassword string
	var err error
	for !hasValidPassword {
		walletPassword, err = promptutil.PasswordPrompt(promptText, passwordValidator)
		if err != nil {
			return "", fmt.Errorf("could not read account password: %v", err)
		}

		if confirmPassword == confirmPass {
			passwordConfirmation, err := promptutil.PasswordPrompt(confirmPasswordPromptText, passwordValidator)
			if err != nil {
				return "", fmt.Errorf("could not read password confirmation: %v", err)
			}
			if walletPassword != passwordConfirmation {
				log.Error("Passwords do not match")
				continue
			}
			hasValidPassword = true
		} else {
			return walletPassword, nil
		}
	}
	return walletPassword, nil
}
