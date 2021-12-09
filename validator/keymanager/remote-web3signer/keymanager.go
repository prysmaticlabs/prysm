package remote_web3signer

import (
	"context"

	"github.com/prysmaticlabs/prysm/async/event"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
)

// Web3SignerKeyManager interface implements Ikeymanager interface
type Web3SignerKeyManager interface {
	keymanager.IKeymanager
}

// KeymanagerOption is a type to help conditionally configure the Keymanager
type KeymanagerOption func(*Keymanager)

func WithExternalURL(url string) KeymanagerOption {
	return func(km *Keymanager) {
		km.publicKeysURL = url
	}
}

func WithKeyList(keys [][48]byte) KeymanagerOption {
	return func(km *Keymanager) {
		km.providedPublicKeys = keys
	}
}

// Keymanager defines the web3signer keymanager
type Keymanager struct {
	opt                   *KeymanagerOption
	client                *client
	genesisValidatorsRoot []byte
	publicKeysURL         string
	providedPublicKeys    [][48]byte
	accountsChangedFeed   *event.Feed
}

//NewKeymanager instantiates a new web3signer key manager
func NewKeymanager(_ context.Context, baseEndpoint string, genesisValidatorsRoot []byte, option KeymanagerOption) (*Keymanager, error) {
	client, err := newClient(baseEndpoint)
	if err != nil {
		// fatal error?
	}
	k := &Keymanager{
		client:                client,
		genesisValidatorsRoot: genesisValidatorsRoot,
		accountsChangedFeed:   new(event.Feed),
	}
	option(k)
	return k, nil
}

func (km *Keymanager) FetchValidatingPublicKeys(ctx context.Context) ([][48]byte, error) {
	if len(km.providedPublicKeys) == 0 {
		return km.client.GetPublicKeys()
	}
	return km.providedPublicKeys, nil
}

func (km *Keymanager) Sign(ctx context.Context, request *validatorpb.SignRequest) (bls.Signature, error) {
	// get new keys before signing

	forkData := &Fork{
		PreviousVersion: string(request.Fork.PreviousVersion),
		CurrentVersion:  string(request.Fork.CurrentVersion),
		Epoch:           string(request.Fork.Epoch),
	}
	// remember to replace signing root with hex encoding remove 0x
	forkInfoData := &ForkInfo{
		Fork:                  forkData,
		GenesisValidatorsRoot: string(km.genesisValidatorsRoot),
	}
	aggregationSlotData := &AggregationSlot{Slot: string(request.AggregationSlot)}

	// remember to replace signing root with hex encoding remove 0x
	web3SignerRequest := SignRequest{
		Type:            "foo",
		ForkInfo:        forkInfoData,
		SigningRoot:     string(request.SigningRoot),
		AggregationSlot: aggregationSlotData,
	}
	return km.client.Sign(string(request.PublicKey), &web3SignerRequest)
}

func (km *Keymanager) SubscribeAccountChanges(pubKeysChan chan [][48]byte) event.Subscription {
	return km.accountsChangedFeed.Subscribe(pubKeysChan)
}

func (km *Keymanager) reloadKeys() {
	if err := km.client.ReloadSignerKeys(); err != nil {

	}
	newKeys, err := km.client.GetPublicKeys()
	if err != nil {

	}
	km.accountsChangedFeed.Send(newKeys)
}
