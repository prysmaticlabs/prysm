package derived

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/async/event"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/local"
	util "github.com/wealdtech/go-eth2-util"
)

const (
	// DerivationPathFormat describes the structure of how keys are derived from a master key.
	DerivationPathFormat = "m / purpose / coin_type / account_index / withdrawal_key / validating_key"
	// ValidatingKeyDerivationPathTemplate defining the hierarchical path for validating
	// keys for Prysm Ethereum validators. According to EIP-2334, the format is as follows:
	// m / purpose / coin_type / account_index / withdrawal_key / validating_key
	ValidatingKeyDerivationPathTemplate = "m/12381/3600/%d/0/0"
)

// SetupConfig includes configuration values for initializing
// a keymanager, such as passwords, the wallet, and more.
type SetupConfig struct {
	Wallet           iface.Wallet
	ListenForChanges bool
}

// Keymanager implementation for derived, HD keymanager using EIP-2333 and EIP-2334.
type Keymanager struct {
	localKM *local.Keymanager
}

// NewKeymanager instantiates a new derived keymanager from configuration options.
func NewKeymanager(
	ctx context.Context,
	cfg *SetupConfig,
) (*Keymanager, error) {
	localKM, err := local.NewKeymanager(ctx, &local.SetupConfig{
		Wallet:           cfg.Wallet,
		ListenForChanges: cfg.ListenForChanges,
	})
	if err != nil {
		return nil, err
	}
	return &Keymanager{
		localKM: localKM,
	}, nil
}

// RecoverAccountsFromMnemonic given a mnemonic phrase, is able to regenerate N accounts
// from a derived seed, encrypt them according to the EIP-2334 JSON standard, and write them
// to disk. Then, the mnemonic is never stored nor used by the validator.
func (km *Keymanager) RecoverAccountsFromMnemonic(
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
	return km.localKM.ImportKeypairs(ctx, privKeys, pubKeys)
}

// ExtractKeystores retrieves the secret keys for specified public keys
// in the function input, encrypts them using the specified password,
// and returns their respective EIP-2335 keystores.
func (km *Keymanager) ExtractKeystores(
	ctx context.Context, publicKeys []bls.PublicKey, password string,
) ([]*keymanager.Keystore, error) {
	return km.localKM.ExtractKeystores(ctx, publicKeys, password)
}

// ValidatingAccountNames for the derived keymanager.
func (km *Keymanager) ValidatingAccountNames(_ context.Context) ([]string, error) {
	return km.localKM.ValidatingAccountNames()
}

// Sign signs a message using a validator key.
func (km *Keymanager) Sign(ctx context.Context, req *validatorpb.SignRequest) (bls.Signature, error) {
	return km.localKM.Sign(ctx, req)
}

// FetchValidatingPublicKeys fetches the list of validating public keys from the keymanager.
func (km *Keymanager) FetchValidatingPublicKeys(ctx context.Context) ([][fieldparams.BLSPubkeyLength]byte, error) {
	return km.localKM.FetchValidatingPublicKeys(ctx)
}

// FetchValidatingPrivateKeys fetches the list of validating private keys from the keymanager.
func (km *Keymanager) FetchValidatingPrivateKeys(ctx context.Context) ([][32]byte, error) {
	return km.localKM.FetchValidatingPrivateKeys(ctx)
}

// ImportKeystores for a derived keymanager.
func (km *Keymanager) ImportKeystores(
	ctx context.Context, keystores []*keymanager.Keystore, passwords []string,
) ([]*ethpbservice.ImportedKeystoreStatus, error) {
	return km.localKM.ImportKeystores(ctx, keystores, passwords)
}

// DeleteKeystores for a derived keymanager.
func (km *Keymanager) DeleteKeystores(
	ctx context.Context, publicKeys [][]byte,
) ([]*ethpbservice.DeletedKeystoreStatus, error) {
	return km.localKM.DeleteKeystores(ctx, publicKeys)
}

// SubscribeAccountChanges creates an event subscription for a channel
// to listen for public key changes at runtime, such as when new validator accounts
// are imported into the keymanager while the validator process is running.
func (km *Keymanager) SubscribeAccountChanges(pubKeysChan chan [][fieldparams.BLSPubkeyLength]byte) event.Subscription {
	return km.localKM.SubscribeAccountChanges(pubKeysChan)
}
