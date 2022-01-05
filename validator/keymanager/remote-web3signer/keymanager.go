package remote_web3signer

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/async/event"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
)

// SetupConfig includes configuration values for initializing.
// a keymanager, such as passwords, the wallet, and more.
// Web3Signer contains one public keys option. Either through a URL or a static key list.
type SetupConfig struct {
	BaseEndpoint          string
	GenesisValidatorsRoot []byte

	// Either URL or keylist must be set.
	// If the URL is set, the keymanager will fetch the public keys from the URL.
	// caution: this option is susceptible to slashing if the web3signer's validator keys are shared across validators
	PublicKeysURL string

	// Either URL or keylist must be set.
	// a static list of public keys to be passed by the user to determine what accounts should sign.
	// This will provide a layer of safety against slashing if the web3signer is shared across validators.
	ProvidedPublicKeys [][48]byte
}

// Keymanager defines the web3signer keymanager.
type Keymanager struct {
	client                httpSignerClient
	genesisValidatorsRoot []byte
	publicKeysURL         string
	providedPublicKeys    [][48]byte
	accountsChangedFeed   *event.Feed
}

// NewKeymanager instantiates a new web3signer key manager.
func NewKeymanager(_ context.Context, cfg *SetupConfig) (*Keymanager, error) {
	if cfg.BaseEndpoint == "" || len(cfg.GenesisValidatorsRoot) == 0 {
		return nil, fmt.Errorf("invalid setup config, one or more configs are empty: BaseEndpoint: %v, GenesisValidatorsRoot: %v", cfg.BaseEndpoint, cfg.GenesisValidatorsRoot)
	}
	if cfg.PublicKeysURL != "" && len(cfg.ProvidedPublicKeys) != 0 {
		return nil, errors.New("Either a provided list of public keys or a URL to a list of public keys must be provided, but not both")
	}
	if cfg.PublicKeysURL == "" && len(cfg.ProvidedPublicKeys) == 0 {
		return nil, errors.New("no valid public key options provided")
	}
	client, err := newApiClient(cfg.BaseEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not create apiClient")
	}
	return &Keymanager{
		client:                httpSignerClient(client),
		genesisValidatorsRoot: cfg.GenesisValidatorsRoot,
		accountsChangedFeed:   new(event.Feed),
		publicKeysURL:         cfg.PublicKeysURL,
		providedPublicKeys:    cfg.ProvidedPublicKeys,
	}, nil
}

// FetchValidatingPublicKeys fetches the validating public keys from the remote server or from the provided keys.
func (km *Keymanager) FetchValidatingPublicKeys(ctx context.Context) ([][48]byte, error) {
	if km.publicKeysURL != "" {
		providedPublicKeys, err := km.client.GetPublicKeys(ctx, km.publicKeysURL)
		if err != nil {
			return nil, err
		}
		km.providedPublicKeys = providedPublicKeys
	}
	return km.providedPublicKeys, nil
}

// Sign signs the message by using a remote web3signer server.
func (km *Keymanager) Sign(ctx context.Context, request *validatorpb.SignRequest) (bls.Signature, error) {
	if request.Fork == nil {
		return nil, errors.New("invalid sign request: Fork is nil")
	}

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
	return km.client.Sign(ctx, hexutil.Encode(request.PublicKey), &web3SignerRequest)
}

// getSignRequestType returns the type of the sign request.
func getSignRequestType(request *validatorpb.SignRequest) (string, error) {
	switch request.Object.(type) {
	case *validatorpb.SignRequest_Block:
		return "BLOCK", nil
	case *validatorpb.SignRequest_AttestationData:
		return "ATTESTATION", nil
	case *validatorpb.SignRequest_AggregateAttestationAndProof:
		return "AGGREGATE_AND_PROOF", nil
	case *validatorpb.SignRequest_Slot:
		return "AGGREGATION_SLOT", nil
	case *validatorpb.SignRequest_BlockV2:
		return "BLOCK_V2", nil
	// TODO(#10053): Need to add support for merge blocks.
	/*
		case *validatorpb.SignRequest_BlockV3:
		return "BLOCK_V3", nil
	*/

	// We do not support "DEPOSIT" type.
	/*
		case *validatorpb.:
		return "DEPOSIT", nil
	*/
	case *validatorpb.SignRequest_Epoch:
		return "RANDAO_REVEAL", nil
	case *validatorpb.SignRequest_Exit:
		return "VOLUNTARY_EXIT", nil
	case *validatorpb.SignRequest_SyncMessageBlockRoot:
		return "SYNC_COMMITTEE_MESSAGE", nil
	case *validatorpb.SignRequest_SyncAggregatorSelectionData:
		return "SYNC_COMMITTEE_SELECTION_PROOF", nil
	case *validatorpb.SignRequest_ContributionAndProof:
		return "SYNC_COMMITTEE_CONTRIBUTION_AND_PROOF", nil
	default:
		return "", errors.New(fmt.Sprintf("Web3signer sign request type: %T  not found", request.Object))
	}
}

// SubscribeAccountChanges returns the event subscription for changes to public keys.
func (_ *Keymanager) SubscribeAccountChanges(_ chan [][48]byte) event.Subscription {
	// Not used right now.
	// Returns a stub for the time being as there is a danger of being slashed if the apiClient reloads keys dynamically.
	// Because there is no way to dynamically reload keys, add or remove remote keys we are returning a stub without any event updates for the time being.
	return event.NewSubscription(func(i <-chan struct{}) error {
		return nil
	})
}
