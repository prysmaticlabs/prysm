package remote_web3signer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/async/event"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
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
	signRequest, err := km.getSignRequestJson(request)
	if err != nil {
		return nil, err
	}

	return km.client.Sign(ctx, hexutil.Encode(request.PublicKey), signRequest)

}

// getSignRequestJson returns a json request based on the SignRequest type.
func (km *Keymanager) getSignRequestJson(request *validatorpb.SignRequest) (SignRequestJson, error) {
	switch request.Object.(type) {
	case *validatorpb.SignRequest_Block:
		bockSignRequest, err := v1.GetBlockSignRequest(request, km.genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.Marshal(bockSignRequest)
	case *validatorpb.SignRequest_AttestationData:
		attestationSignRequest, err := v1.GetAttestationSignRequest(request, km.genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.Marshal(attestationSignRequest)
	case *validatorpb.SignRequest_AggregateAttestationAndProof:
		aggregateAndProofSignRequest, err := v1.GetAggregateAndProofSignRequest(request, km.genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.Marshal(aggregateAndProofSignRequest)
	case *validatorpb.SignRequest_Slot:
		aggregationSlotSignRequest, err := v1.GetAggregationSlotSignRequest(request, km.genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.Marshal(aggregationSlotSignRequest)
	case *validatorpb.SignRequest_BlockV2:
		blocv2AltairSignRequest, err := v1.GetBlockV2AltairSignRequest(request, km.genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.Marshal(blocv2AltairSignRequest)
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
		randaoRevealSignRequest, err := v1.GetRandaoRevealSignRequest(request, km.genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.Marshal(randaoRevealSignRequest)
	case *validatorpb.SignRequest_Exit:
		voluntaryExitRequest, err := v1.GetVoluntaryExitSignRequest(request, km.genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.Marshal(voluntaryExitRequest)
	case *validatorpb.SignRequest_SyncMessageBlockRoot:
		syncCommitteeMessageRequest, err := v1.GetSyncCommitteeMessageSignRequest(request, km.genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.Marshal(syncCommitteeMessageRequest)
	case *validatorpb.SignRequest_SyncAggregatorSelectionData:
		syncCommitteeSelectionProofRequest, err := v1.GetSyncCommitteeSelectionProofSignRequest(request, km.genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.Marshal(syncCommitteeSelectionProofRequest)
	case *validatorpb.SignRequest_ContributionAndProof:
		contributionAndProofRequest, err := v1.GetSyncCommitteeContributionAndProofSignRequest(request, km.genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		return json.Marshal(contributionAndProofRequest)
	default:
		return nil, errors.New(fmt.Sprintf("Web3signer sign request type: %T  not found", request.Object))
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

// reloadKeys reloads the public keys from the remote server
func (*Keymanager) reloadKeys() {
	// Not used right now.
	// The feature of needing to dynamically reload from the validator instead of from the web3signer is yet to be determined.
	// In the future there may be an api provided to add remote sign keys to the static list or remove from the static list.
}

// UnmarshalConfigFile attempts to JSON unmarshal a keymanager
// config file into a SetupConfig struct.
func UnmarshalConfigFile(r io.ReadCloser) (*SetupConfig, error) {
	enc, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "could not read config")
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Errorf("Could not close keymanager config file: %v", err)
		}
	}()
	config := &SetupConfig{}
	if err := json.Unmarshal(enc, config); err != nil {
		return nil, errors.Wrap(err, "could not JSON unmarshal")
	}
	return config, nil
}

// MarshalConfigFile for the keymanager.
func MarshalConfigFile(_ context.Context, config *SetupConfig) ([]byte, error) {
	return json.MarshalIndent(config, "", "\t")
}
