package keymanager

import (
	"context"

	"github.com/prysmaticlabs/prysm/async/event"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
)

const (
	Local   = "imported"
	Derived = "derived"
	Remote  = "remote"
)

// IKeymanager defines a struct which can be used to manage keys with important
// actions such as listing public keys, signing data, and subscribing to key changes.
// It is defined by the simple composition of a few base features. We can assert whether
// or not a keymanager has "extra" functionality by casting it to other, useful key
// management interfaces defined in this package.
type IKeymanager interface {
	PublicKeysFetcher
	Signer
	KeyChangeSubscriber
}

// KeysFetcher for validating private and public keys.
type KeysFetcher interface {
	FetchValidatingPrivateKeys(ctx context.Context) ([][32]byte, error)
	PublicKeysFetcher
}

// PublicKeysFetcher for validating public keys.
type PublicKeysFetcher interface {
	FetchValidatingPublicKeys(ctx context.Context) ([][48]byte, error)
}

// Signer allows signing messages using a validator private key.
type Signer interface {
	Sign(context.Context, *validatorpb.SignRequest) (bls.Signature, error)
}

// Importer can import new keystores into the keymanager.
type Importer interface {
	ImportKeystores(ctx context.Context, keystores []*Keystore, importsPassword string) error
}

// KeyChangeSubscriber allows subscribing to changes made to the underlying keys.
type KeyChangeSubscriber interface {
	SubscribeAccountChanges(pubKeysChan chan [][48]byte) event.Subscription
}

// KeyRecoverer allows recovering a keystore from a recovery phrase.
type KeyRecoverer interface {
	RecoverKeystoresFromMnemonic(
		ctx context.Context, mnemonic, mnemonicPassphrase string, numKeys int,
	) error
}

// Keystore json file representation as a Go struct.
type Keystore struct {
	Crypto  map[string]interface{} `json:"crypto"`
	ID      string                 `json:"uuid"`
	Pubkey  string                 `json:"pubkey"`
	Version uint                   `json:"version"`
	Name    string                 `json:"name"`
}

// IncorrectPasswordErrMsg defines a common error string representing an EIP-2335
// keystore password was incorrect.
const IncorrectPasswordErrMsg = "invalid checksum"
