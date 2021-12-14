package remote_web3signer

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/pkg/errors"

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
type PublicKeysOption func(*Keymanager)

// WithExternalURL sets the external url for the keymanager
// explain usecases for this option
func WithExternalURL(url string) PublicKeysOption {
	return func(km *Keymanager) {
		km.publicKeysURL = url
	}
}

// WithKeyList is a function to set the key list
// explain usecases for this option
func WithKeyList(keys [][48]byte) PublicKeysOption {
	return func(km *Keymanager) {
		km.providedPublicKeys = keys
	}
}

// SetupConfig includes configuration values for initializing
// a keymanager, such as passwords, the wallet, and more.
type SetupConfig struct {
	Option                *PublicKeysOption
	BaseEndpoint          string
	GenesisValidatorsRoot []byte
}

// Keymanager defines the web3signer keymanager
type Keymanager struct {
	opt                   *PublicKeysOption
	client                Web3SignerClient
	genesisValidatorsRoot []byte
	publicKeysURL         string
	providedPublicKeys    [][48]byte
	accountsChangedFeed   *event.Feed
}

// NewKeymanager instantiates a new web3signer key manager
func NewKeymanager(_ context.Context, cfg *SetupConfig) (*Keymanager, error) {
	if cfg.Option == nil || cfg.BaseEndpoint == "" || cfg.GenesisValidatorsRoot == nil {
		return nil, errors.New("invalid setup config, one or more configs are empty: " + fmt.Sprintf("Option: %v, BaseEndpoint: %v, GenesisValidatorsRoot: %v.", cfg.Option, cfg.BaseEndpoint, cfg.GenesisValidatorsRoot))
	}
	client, err := newClient(cfg.BaseEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not create client")
	}
	km := &Keymanager{
		client:                Web3SignerClient(client),
		genesisValidatorsRoot: cfg.GenesisValidatorsRoot,
		accountsChangedFeed:   new(event.Feed),
	}
	optionFunction := *cfg.Option
	optionFunction(km)
	if km.publicKeysURL == "" && len(km.providedPublicKeys) == 0 {
		return nil, errors.New("no valid public key options provided")
	}
	return km, nil
}

// FetchValidatingPublicKeys fetches the validating public keys from the remote server or from the provided keys
func (km *Keymanager) FetchValidatingPublicKeys(ctx context.Context) ([][48]byte, error) {
	if km.publicKeysURL != "" {
		return km.client.GetPublicKeys(km.publicKeysURL)
	}
	return km.providedPublicKeys, nil
}

// Sign signs the message by using a remote web3signer server
func (km *Keymanager) Sign(ctx context.Context, request *validatorpb.SignRequest) (bls.Signature, error) {

	if request.Fork == nil {
		return nil, errors.New("invalid sign request: Fork is nil")
	}
	if request.AggregationSlot == 0 {
		return nil, errors.New("invalid sign request: AggregationSlot is 0")
	}

	// get new keys before signing
	signRequestType, err := getSignRequestType(request)
	if err != nil {
		return nil, err
	}
	forkData := &Fork{
		PreviousVersion: hexutil.Encode(request.Fork.PreviousVersion),
		CurrentVersion:  hexutil.Encode(request.Fork.CurrentVersion),
		Epoch:           fmt.Sprint(request.Fork.Epoch),
	}
	forkInfoData := &ForkInfo{
		Fork:                  forkData,
		GenesisValidatorsRoot: hexutil.Encode(km.genesisValidatorsRoot),
	}
	aggregationSlotData := &AggregationSlot{Slot: fmt.Sprint(request.AggregationSlot)}
	web3SignerRequest := SignRequest{
		Type:            signRequestType,
		ForkInfo:        forkInfoData,
		SigningRoot:     hexutil.Encode(request.SigningRoot),
		AggregationSlot: aggregationSlotData,
	}
	return km.client.Sign(string(request.PublicKey), &web3SignerRequest)
}

// getSignRequestType returns the type of the sign request
func getSignRequestType(request *validatorpb.SignRequest) (string, error) {
	//	*SignRequest_Slot // check where this is used
	//	*SignRequest_Epoch // check where this is used
	//	*SignRequest_SyncAggregatorSelectionData
	//	*SignRequest_SyncMessageBlockRoot
	//	*SignRequest_BlockV3 // check where this is used
	switch request.Object.(type) {
	case *validatorpb.SignRequest_Block:
		return "BLOCK", nil
	case *validatorpb.SignRequest_AttestationData:
		return "ATTESTATION", nil
	case *validatorpb.SignRequest_AggregateAttestationAndProof:
		return "AGGREGATE_AND_PROOF", nil
	case *validatorpb.SignRequest_SyncAggregatorSelectionData:
		return "AGGREGATION_SLOT", nil
	case *validatorpb.SignRequest_BlockV2:
		return "BLOCK_V2", nil
	//case *validatorpb.:
	//	return "DEPOSIT", nil
	//case *validatorpb.:
	//	return "RANDAO_REVEAL", nil
	case *validatorpb.SignRequest_Exit: //not sure
		return "VOLUNTARY_EXIT", nil
	//case *validatorpb.:
	//	return "SYNC_COMMITTEE_MESSAGE", nil
	//case *validatorpb.:
	//	return "SYNC_COMMITTEE_SELECTION_PROOF", nil
	case *validatorpb.SignRequest_ContributionAndProof:
		return "SYNC_COMMITTEE_CONTRIBUTION_AND_PROOF", nil
	default:
		return "", errors.New("Web3signer sign request type not found")
	}
}

// SubscribeAccountChanges returns the event subscription for changes to public keys
func (km *Keymanager) SubscribeAccountChanges(pubKeysChan chan [][48]byte) event.Subscription {
	// not used right now
	return event.NewSubscription(func(i <-chan struct{}) error {
		return nil
	})
}

// reloadKeys reloads the public keys from the remote server
func (km *Keymanager) reloadKeys() {
	// not used right now
}
