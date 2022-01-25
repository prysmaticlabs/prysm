package remote_web3signer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/async/event"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	v1 "github.com/prysmaticlabs/prysm/validator/keymanager/remote-web3signer/v1"
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
	if cfg.BaseEndpoint == "" || len(cfg.GenesisValidatorsRoot) != fieldparams.RootLength || bytes.Equal(params.BeaconConfig().ZeroHash[:], cfg.GenesisValidatorsRoot) {
		return nil, fmt.Errorf("invalid setup config, one or more configs are empty: BaseEndpoint: %v, GenesisValidatorsRoot: %#x", cfg.BaseEndpoint, cfg.GenesisValidatorsRoot)
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

// FetchValidatingPublicKeys fetches the validating public keys
// from the remote server or from the provided keys if there are no existing public keys set
// or provides the existing keys in the keymanager.
func (km *Keymanager) FetchValidatingPublicKeys(ctx context.Context) ([][fieldparams.BLSPubkeyLength]byte, error) {
	if km.publicKeysURL != "" && len(km.providedPublicKeys) == 0 {
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

	signRequest, err := getSignRequestJson(request, km.genesisValidatorsRoot)
	if err != nil {
		return nil, err
	}

	return km.client.Sign(ctx, hexutil.Encode(request.PublicKey), signRequest)

}

// getSignRequestJson returns a json request based on the SignRequest type.
func getSignRequestJson(request *validatorpb.SignRequest, genesisValidatorsRoot []byte) (SignRequestJson, error) {
	if request == nil {
		return nil, errors.New("nil sign request provided")
	}
	if len(genesisValidatorsRoot) != fieldparams.RootLength || bytes.Equal(params.BeaconConfig().ZeroHash[:], genesisValidatorsRoot) {
		return nil, fmt.Errorf("invalid genesis validators root length, genesis root: %v", genesisValidatorsRoot)
	}
	switch request.Object.(type) {
	case *validatorpb.SignRequest_Block:
		bockSignRequest, err := v1.GetBlockSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(bockSignRequest, "", "\t")
	case *validatorpb.SignRequest_AttestationData:
		attestationSignRequest, err := v1.GetAttestationSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(attestationSignRequest, "", "\t")
	case *validatorpb.SignRequest_AggregateAttestationAndProof:
		aggregateAndProofSignRequest, err := v1.GetAggregateAndProofSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(aggregateAndProofSignRequest, "", "\t")
	case *validatorpb.SignRequest_Slot:
		aggregationSlotSignRequest, err := v1.GetAggregationSlotSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(aggregationSlotSignRequest, "", "\t")
	case *validatorpb.SignRequest_BlockV2:
		blocv2AltairSignRequest, err := v1.GetBlockV2AltairSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(blocv2AltairSignRequest, "", "\t")
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
		randaoRevealSignRequest, err := v1.GetRandaoRevealSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(randaoRevealSignRequest, "", "\t")
	case *validatorpb.SignRequest_Exit:
		voluntaryExitRequest, err := v1.GetVoluntaryExitSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(voluntaryExitRequest, "", "\t")
	case *validatorpb.SignRequest_SyncMessageBlockRoot:
		syncCommitteeMessageRequest, err := v1.GetSyncCommitteeMessageSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(syncCommitteeMessageRequest, "", "\t")
	case *validatorpb.SignRequest_SyncAggregatorSelectionData:
		syncCommitteeSelectionProofRequest, err := v1.GetSyncCommitteeSelectionProofSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(syncCommitteeSelectionProofRequest, "", "\t")
	case *validatorpb.SignRequest_ContributionAndProof:
		contributionAndProofRequest, err := v1.GetSyncCommitteeContributionAndProofSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(contributionAndProofRequest, "", "\t")
	default:
		return nil, fmt.Errorf("web3signer sign request type %T not supported", request.Object)
	}
}

// SubscribeAccountChanges returns the event subscription for changes to public keys.
func (*Keymanager) SubscribeAccountChanges(_ chan [][48]byte) event.Subscription {
	// Not used right now.
	// Returns a stub for the time being as there is a danger of being slashed if the apiClient reloads keys dynamically.
	// Because there is no way to dynamically reload keys, add or remove remote keys we are returning a stub without any event updates for the time being.
	return event.NewSubscription(func(i <-chan struct{}) error {
		return nil
	})
}
