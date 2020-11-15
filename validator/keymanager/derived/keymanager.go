package derived

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"github.com/sirupsen/logrus"
	util "github.com/wealdtech/go-eth2-util"
)

var (
	log = logrus.WithField("prefix", "derived-keymanager")
)

const (
	// EIPVersion used by this derived keymanager implementation.
	EIPVersion = "EIP-2334"
	// ValidatingKeyDerivationPathTemplate defining the hierarchical path for validating
	// keys for Prysm eth2 validators. According to EIP-2334, the format is as follows:
	// m / purpose / coin_type / account_index / withdrawal_key / validating_key
	ValidatingKeyDerivationPathTemplate = "m/12381/3600/%d/0/0"
)

// KeymanagerOpts defines options for the keymanager that
// are stored to disk in the wallet.
type KeymanagerOpts struct {
	DerivedPathStructure string `json:"derived_path_structure"`
	DerivedEIPNumber     string `json:"derived_eip_number"`
	DerivedVersion       string `json:"derived_version"`
}

// SetupConfig includes configuration values for initializing
// a keymanager, such as passwords, the wallet, and more.
type SetupConfig struct {
	Opts   *KeymanagerOpts
	Wallet iface.Wallet
}

// Keymanager implementation for derived, HD keymanager using EIP-2333 and EIP-2334.
type Keymanager struct {
	importedKM *imported.Keymanager
}

// DefaultKeymanagerOpts for a derived keymanager implementation.
func DefaultKeymanagerOpts() *KeymanagerOpts {
	return &KeymanagerOpts{
		DerivedPathStructure: "m / purpose / coin_type / account_index / withdrawal_key / validating_key",
		DerivedEIPNumber:     EIPVersion,
		DerivedVersion:       "2",
	}
}

// NewKeymanager instantiates a new derived keymanager from configuration options.
func NewKeymanager(
	ctx context.Context,
	cfg *SetupConfig,
) (*Keymanager, error) {
	importedKM, err := imported.NewKeymanager(ctx, &imported.SetupConfig{
		Wallet: cfg.Wallet,
		Opts:   imported.DefaultKeymanagerOpts(),
	})
	if err != nil {
		return nil, err
	}
	return &Keymanager{
		importedKM: importedKM,
	}, nil
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
	return opts, nil
}

// MarshalOptionsFile returns a marshaled options file for a keymanager.
func MarshalOptionsFile(_ context.Context, opts *KeymanagerOpts) ([]byte, error) {
	return json.MarshalIndent(opts, "", "\t")
}

// KeymanagerOpts returns the derived keymanager options.
func (dr *Keymanager) KeymanagerOpts() *KeymanagerOpts {
	return dr.KeymanagerOpts()
}

// WriteEncryptedKeystoresFromSeed given a mnemonic phrase, is able to regenerate N accounts
// from a derived seed, encrypt them according to the EIP-2334 JSON standard, and write them
// to disk. Then, the mnemonic is never stored nor used by the validator.
func (dr *Keymanager) RecoverAccountsFromMnemonic(
	ctx context.Context, mnemonic, mnemonicPassphrase string, numAccounts int,
) error {
	seed, err := seedFromMnemonic(mnemonic, mnemonicPassphrase)
	if err != nil {
		return errors.Wrap(err, "could not initialize new wallet seed file")
	}
	privKeys := make([][]byte, numAccounts)
	pubKeys := make([][]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		privKey, err := util.PrivateKeyFromSeedAndPath(
			seed, fmt.Sprintf(ValidatingKeyDerivationPathTemplate, i),
		)
		if err != nil {
			return err
		}
		privKeys[i] = privKey.Marshal()
		pubKeys[i] = privKey.PublicKey().Marshal()
	}
	return dr.importedKM.ImportKeypairs(ctx, privKeys, pubKeys)
}

// ValidatingAccountNames for the derived keymanager.
func (dr *Keymanager) ValidatingAccountNames(_ context.Context) ([]string, error) {
	return dr.importedKM.ValidatingAccountNames()
}

// Sign signs a message using a validator key.
func (dr *Keymanager) Sign(ctx context.Context, req *validatorpb.SignRequest) (bls.Signature, error) {
	return dr.importedKM.Sign(ctx, req)
}

// FetchValidatingPublicKeys fetches the list of validating public keys from the keymanager.
func (dr *Keymanager) FetchValidatingPublicKeys(ctx context.Context) ([][48]byte, error) {
	return dr.importedKM.FetchValidatingPublicKeys(ctx)
}

// FetchAllValidatingPublicKeys fetches the list of all public keys (including disabled ones) from the keymanager.
func (dr *Keymanager) FetchAllValidatingPublicKeys(ctx context.Context) ([][48]byte, error) {
	return dr.importedKM.FetchAllValidatingPublicKeys(ctx)
}

// FetchValidatingPrivateKeys fetches the list of validating private keys from the keymanager.
func (dr *Keymanager) FetchValidatingPrivateKeys(ctx context.Context) ([][32]byte, error) {
	return dr.importedKM.FetchValidatingPrivateKeys(ctx)
}
