package remote_web3signer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-playground/validator/v10"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/async/event"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	remote_utils "github.com/prysmaticlabs/prysm/validator/keymanager/remote-utils"
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
	publicKeysUrlCalled   bool
}

// NewKeymanager instantiates a new web3signer key manager.
func NewKeymanager(_ context.Context, cfg *SetupConfig) (*Keymanager, error) {
	if cfg.BaseEndpoint == "" || !bytesutil.IsValidRoot(cfg.GenesisValidatorsRoot) {
		return nil, fmt.Errorf("invalid setup config, one or more configs are empty: BaseEndpoint: %v, GenesisValidatorsRoot: %#x", cfg.BaseEndpoint, cfg.GenesisValidatorsRoot)
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
		publicKeysUrlCalled:   false,
	}, nil
}

// FetchValidatingPublicKeys fetches the validating public keys
// from the remote server or from the provided keys if there are no existing public keys set
// or provides the existing keys in the keymanager.
func (km *Keymanager) FetchValidatingPublicKeys(ctx context.Context) ([][fieldparams.BLSPubkeyLength]byte, error) {
	if km.publicKeysURL != "" && !km.publicKeysUrlCalled {
		providedPublicKeys, err := km.client.GetPublicKeys(ctx, km.publicKeysURL)
		if err != nil {
			erroredResponsesTotal.Inc()
			return nil, errors.Wrap(err, fmt.Sprintf("could not get public keys from remote server url: %v", km.publicKeysURL))
		}
		// makes sure that if the public keys are deleted the validator does not call URL again.
		km.publicKeysUrlCalled = true
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
	if !bytesutil.IsValidRoot(genesisValidatorsRoot) {
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
func (km *Keymanager) SubscribeAccountChanges(pubKeysChan chan [][fieldparams.BLSPubkeyLength]byte) event.Subscription {
	return km.accountsChangedFeed.Subscribe(pubKeysChan)
}

// ExtractKeystores is not supported for the remote-web3signer keymanager type.
func (*Keymanager) ExtractKeystores(
	ctx context.Context, publicKeys []bls.PublicKey, password string,
) ([]*keymanager.Keystore, error) {
	return nil, errors.New("extracting keys is not supported for a web3signer keymanager")
}

// DeleteKeystores is not supported for the remote-web3signer keymanager type.
func (km *Keymanager) DeleteKeystores(context.Context, [][]byte) ([]*ethpbservice.DeletedKeystoreStatus, error) {
	return nil, errors.New("Wrong wallet type: web3-signer. Only Imported or Derived wallets can delete accounts")
}

func (km *Keymanager) ListKeymanagerAccounts(ctx context.Context, cfg keymanager.ListKeymanagerAccountConfig) error {
	au := aurora.NewAurora(true)
	fmt.Printf("(keymanager kind) %s\n", au.BrightGreen("web3signer").Bold())
	fmt.Printf(
		"(configuration file path) %s\n",
		au.BrightGreen(filepath.Join(cfg.WalletAccountsDir, cfg.KeymanagerConfigFileName)).Bold(),
	)
	fmt.Println(" ")
	fmt.Printf("%s\n", au.BrightGreen("Setup Configuration").Bold())
	fmt.Println(" ")
	//TODO: add config options, may require refactor again
	validatingPubKeys, err := km.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return errors.Wrap(err, "could not fetch validating public keys")
	}
	if len(validatingPubKeys) == 1 {
		fmt.Print("Showing 1 validator account\n")
	} else if len(validatingPubKeys) == 0 {
		fmt.Print("No accounts found\n")
		return nil
	} else {
		fmt.Printf("Showing %d validator accounts\n", len(validatingPubKeys))
	}
	remote_utils.DisplayRemotePublicKeys(validatingPubKeys)
	return nil
}

// AddPublicKeys imports a list of public keys into the keymanager for web3signer use. will return status with message.
func (km *Keymanager) AddPublicKeys(ctx context.Context, pubKeys [][fieldparams.BLSPubkeyLength]byte) ([]*ethpbservice.ImportedRemoteKeysStatus, error) {
	if ctx == nil {
		return nil, errors.New("context is nil")
	}
	importedRemoteKeysStatuses := make([]*ethpbservice.ImportedRemoteKeysStatus, len(pubKeys))
	for i, pubKey := range pubKeys {
		found := false
		for _, key := range km.providedPublicKeys {
			if bytes.Equal(key[:], pubKey[:]) {
				found = true
				break
			}
		}
		if found {
			importedRemoteKeysStatuses[i] = &ethpbservice.ImportedRemoteKeysStatus{
				Status:  ethpbservice.ImportedRemoteKeysStatus_DUPLICATE,
				Message: fmt.Sprintf("Duplicate pubkey: %v, already in use", hexutil.Encode(pubKey[:])),
			}
			continue
		}
		km.providedPublicKeys = append(km.providedPublicKeys, pubKey)
		importedRemoteKeysStatuses[i] = &ethpbservice.ImportedRemoteKeysStatus{
			Status:  ethpbservice.ImportedRemoteKeysStatus_IMPORTED,
			Message: fmt.Sprintf("Successfully added pubkey: %v", hexutil.Encode(pubKey[:])),
		}
		km.accountsChangedFeed.Send(pubKey)
	}
	return importedRemoteKeysStatuses, nil
}

// DeletePublicKeys removes a list of public keys from the keymanager for web3signer use. will return status with message.
func (km *Keymanager) DeletePublicKeys(ctx context.Context, pubKeys [][fieldparams.BLSPubkeyLength]byte) ([]*ethpbservice.DeletedRemoteKeysStatus, error) {
	if ctx == nil {
		return nil, errors.New("context is nil")
	}
	deletedRemoteKeysStatuses := make([]*ethpbservice.DeletedRemoteKeysStatus, len(pubKeys))
	for i, pubkey := range pubKeys {
		if len(km.providedPublicKeys) == 0 {
			deletedRemoteKeysStatuses[i] = &ethpbservice.DeletedRemoteKeysStatus{
				Status:  ethpbservice.DeletedRemoteKeysStatus_NOT_FOUND,
				Message: fmt.Sprintf("No pubkeys are set in validator"),
			}
			continue
		}
		for in, key := range km.providedPublicKeys {
			if bytes.Equal(key[:], pubkey[:]) {
				km.providedPublicKeys = append(km.providedPublicKeys[:in], km.providedPublicKeys[in+1:]...)
				deletedRemoteKeysStatuses[i] = &ethpbservice.DeletedRemoteKeysStatus{
					Status:  ethpbservice.DeletedRemoteKeysStatus_DELETED,
					Message: fmt.Sprintf("Successfully deleted pubkey: %v", hexutil.Encode(pubkey[:])),
				}
				break
			}
		}
		if deletedRemoteKeysStatuses[i] == nil {
			deletedRemoteKeysStatuses[i] = &ethpbservice.DeletedRemoteKeysStatus{
				Status:  ethpbservice.DeletedRemoteKeysStatus_NOT_FOUND,
				Message: fmt.Sprintf("Pubkey: %v not found", hexutil.Encode(pubkey[:])),
			}
		}
	}
	return deletedRemoteKeysStatuses, nil
}
