package remote_web3signer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/async/event"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/validator/keymanager/remote-web3signer/internal"
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
	client                internal.HttpSignerClient
	genesisValidatorsRoot []byte
	publicKeysURL         string
	providedPublicKeys    [][48]byte
	accountsChangedFeed   *event.Feed
	validator             *validator.Validate
}

// NewKeymanager instantiates a new web3signer key manager.
func NewKeymanager(_ context.Context, cfg *SetupConfig) (*Keymanager, error) {
	if cfg.BaseEndpoint == "" || !bytesutil.NonZeroRoot(cfg.GenesisValidatorsRoot) {
		return nil, fmt.Errorf("invalid setup config, one or more configs are empty: BaseEndpoint: %v, GenesisValidatorsRoot: %#x", cfg.BaseEndpoint, cfg.GenesisValidatorsRoot)
	}
	if cfg.PublicKeysURL != "" && len(cfg.ProvidedPublicKeys) != 0 {
		return nil, errors.New("Either a provided list of public keys or a URL to a list of public keys must be provided, but not both")
	}
	if cfg.PublicKeysURL == "" && len(cfg.ProvidedPublicKeys) == 0 {
		return nil, errors.New("no valid public key options provided")
	}
	client, err := internal.NewApiClient(cfg.BaseEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not create apiClient")
	}
	return &Keymanager{
		client:                internal.HttpSignerClient(client),
		genesisValidatorsRoot: cfg.GenesisValidatorsRoot,
		accountsChangedFeed:   new(event.Feed),
		publicKeysURL:         cfg.PublicKeysURL,
		providedPublicKeys:    cfg.ProvidedPublicKeys,
		validator:             validator.New(),
	}, nil
}

// FetchValidatingPublicKeys fetches the validating public keys
// from the remote server or from the provided keys if there are no existing public keys set
// or provides the existing keys in the keymanager.
func (km *Keymanager) FetchValidatingPublicKeys(ctx context.Context) ([][fieldparams.BLSPubkeyLength]byte, error) {
	if km.publicKeysURL != "" && len(km.providedPublicKeys) == 0 {
		providedPublicKeys, err := km.client.GetPublicKeys(ctx, km.publicKeysURL)
		if err != nil {
			erroredResponsesTotal.Inc()
			return nil, err
		}
		km.providedPublicKeys = providedPublicKeys
	}
	return km.providedPublicKeys, nil
}

// Sign signs the message by using a remote web3signer server.
func (km *Keymanager) Sign(ctx context.Context, request *validatorpb.SignRequest) (bls.Signature, error) {
	signRequest, err := getSignRequestJson(ctx, km.validator, request, km.genesisValidatorsRoot)
	if err != nil {
		erroredResponsesTotal.Inc()
		return nil, err
	}

	signRequestsTotal.Inc()

	return km.client.Sign(ctx, hexutil.Encode(request.PublicKey), signRequest)
}

// getSignRequestJson returns a json request based on the SignRequest type.
func getSignRequestJson(ctx context.Context, validator *validator.Validate, request *validatorpb.SignRequest, genesisValidatorsRoot []byte) (internal.SignRequestJson, error) {
	if request == nil {
		return nil, errors.New("nil sign request provided")
	}
	if !bytesutil.NonZeroRoot(genesisValidatorsRoot) {
		return nil, fmt.Errorf("invalid genesis validators root length, genesis root: %v", genesisValidatorsRoot)
	}
	switch request.Object.(type) {
	case *validatorpb.SignRequest_Block:
		bockSignRequest, err := v1.GetBlockSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		if err = validator.StructCtx(ctx, bockSignRequest); err != nil {
			return nil, err
		}
		blockSignRequestsTotal.Inc()
		return json.Marshal(bockSignRequest)
	case *validatorpb.SignRequest_AttestationData:
		attestationSignRequest, err := v1.GetAttestationSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		if err = validator.StructCtx(ctx, attestationSignRequest); err != nil {
			return nil, err
		}
		attestationSignRequestsTotal.Inc()
		return json.Marshal(attestationSignRequest)
	case *validatorpb.SignRequest_AggregateAttestationAndProof:
		aggregateAndProofSignRequest, err := v1.GetAggregateAndProofSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		if err = validator.StructCtx(ctx, aggregateAndProofSignRequest); err != nil {
			return nil, err
		}
		aggregateAndProofSignRequestsTotal.Inc()
		return json.Marshal(aggregateAndProofSignRequest)
	case *validatorpb.SignRequest_Slot:
		aggregationSlotSignRequest, err := v1.GetAggregationSlotSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		if err = validator.StructCtx(ctx, aggregationSlotSignRequest); err != nil {
			return nil, err
		}
		aggregationSlotSignRequestsTotal.Inc()
		return json.Marshal(aggregationSlotSignRequest)
	case *validatorpb.SignRequest_BlockV2:
		blocv2AltairSignRequest, err := v1.GetBlockV2AltairSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		if err = validator.StructCtx(ctx, blocv2AltairSignRequest); err != nil {
			return nil, err
		}
		blockV2SignRequestsTotal.Inc()
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
		randaoRevealSignRequest, err := v1.GetRandaoRevealSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		if err = validator.StructCtx(ctx, randaoRevealSignRequest); err != nil {
			return nil, err
		}
		randaoRevealSignRequestsTotal.Inc()
		return json.Marshal(randaoRevealSignRequest)
	case *validatorpb.SignRequest_Exit:
		voluntaryExitRequest, err := v1.GetVoluntaryExitSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		if err = validator.StructCtx(ctx, voluntaryExitRequest); err != nil {
			return nil, err
		}
		voluntaryExitSignRequestsTotal.Inc()
		return json.Marshal(voluntaryExitRequest)
	case *validatorpb.SignRequest_SyncMessageBlockRoot:
		syncCommitteeMessageRequest, err := v1.GetSyncCommitteeMessageSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		if err = validator.StructCtx(ctx, syncCommitteeMessageRequest); err != nil {
			return nil, err
		}
		syncCommitteeMessageSignRequestsTotal.Inc()
		return json.Marshal(syncCommitteeMessageRequest)
	case *validatorpb.SignRequest_SyncAggregatorSelectionData:
		syncCommitteeSelectionProofRequest, err := v1.GetSyncCommitteeSelectionProofSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		if err = validator.StructCtx(ctx, syncCommitteeSelectionProofRequest); err != nil {
			return nil, err
		}
		syncCommitteeSelectionProofSignRequestsTotal.Inc()
		return json.Marshal(syncCommitteeSelectionProofRequest)
	case *validatorpb.SignRequest_ContributionAndProof:
		contributionAndProofRequest, err := v1.GetSyncCommitteeContributionAndProofSignRequest(request, genesisValidatorsRoot)
		if err != nil {
			return nil, err
		}
		if err = validator.StructCtx(ctx, contributionAndProofRequest); err != nil {
			return nil, err
		}
		syncCommitteeContributionAndProofSignRequestsTotal.Inc()
		return json.Marshal(contributionAndProofRequest)
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
