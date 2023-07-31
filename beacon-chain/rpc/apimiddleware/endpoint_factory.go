package apimiddleware

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api/gateway/apimiddleware"
)

// BeaconEndpointFactory creates endpoints used for running beacon chain API calls through the API Middleware.
type BeaconEndpointFactory struct {
}

func (f *BeaconEndpointFactory) IsNil() bool {
	return f == nil
}

// Paths is a collection of all valid beacon chain API paths.
func (_ *BeaconEndpointFactory) Paths() []string {
	return []string{
		"/eth/v1/beacon/genesis",
		"/eth/v1/beacon/states/{state_id}/root",
		"/eth/v1/beacon/states/{state_id}/fork",
		"/eth/v1/beacon/states/{state_id}/finality_checkpoints",
		"/eth/v1/beacon/states/{state_id}/validators",
		"/eth/v1/beacon/states/{state_id}/validators/{validator_id}",
		"/eth/v1/beacon/states/{state_id}/validator_balances",
		"/eth/v1/beacon/states/{state_id}/committees",
		"/eth/v1/beacon/states/{state_id}/sync_committees",
		"/eth/v1/beacon/states/{state_id}/randao",
		"/eth/v1/beacon/headers",
		"/eth/v1/beacon/headers/{block_id}",
		"/eth/v1/beacon/blocks",
		"/eth/v1/beacon/blinded_blocks",
		"/eth/v1/beacon/blocks/{block_id}",
		"/eth/v2/beacon/blocks/{block_id}",
		"/eth/v1/beacon/blocks/{block_id}/root",
		"/eth/v1/beacon/blocks/{block_id}/attestations",
		"/eth/v1/beacon/blinded_blocks/{block_id}",
		"/eth/v1/beacon/pool/attestations",
		"/eth/v1/beacon/pool/attester_slashings",
		"/eth/v1/beacon/pool/proposer_slashings",
		"/eth/v1/beacon/pool/voluntary_exits",
		"/eth/v1/beacon/pool/bls_to_execution_changes",
		"/eth/v1/beacon/pool/sync_committees",
		"/eth/v1/beacon/pool/bls_to_execution_changes",
		"/eth/v1/beacon/weak_subjectivity",
		"/eth/v1/node/identity",
		"/eth/v1/node/peers",
		"/eth/v1/node/peers/{peer_id}",
		"/eth/v1/node/peer_count",
		"/eth/v1/node/version",
		"/eth/v1/node/syncing",
		"/eth/v1/node/health",
		"/eth/v1/debug/beacon/states/{state_id}",
		"/eth/v2/debug/beacon/states/{state_id}",
		"/eth/v1/debug/beacon/heads",
		"/eth/v2/debug/beacon/heads",
		"/eth/v1/debug/fork_choice",
		"/eth/v1/config/fork_schedule",
		"/eth/v1/config/deposit_contract",
		"/eth/v1/config/spec",
		"/eth/v1/events",
		"/eth/v1/validator/duties/attester/{epoch}",
		"/eth/v1/validator/duties/proposer/{epoch}",
		"/eth/v1/validator/duties/sync/{epoch}",
		"/eth/v1/validator/blocks/{slot}",
		"/eth/v2/validator/blocks/{slot}",
		"/eth/v1/validator/blinded_blocks/{slot}",
		"/eth/v1/validator/attestation_data",
		"/eth/v1/validator/beacon_committee_subscriptions",
		"/eth/v1/validator/sync_committee_subscriptions",
		"/eth/v1/validator/aggregate_and_proofs",
		"/eth/v1/validator/sync_committee_contribution",
		"/eth/v1/validator/contribution_and_proofs",
		"/eth/v1/validator/prepare_beacon_proposer",
		"/eth/v1/validator/register_validator",
		"/eth/v1/validator/liveness/{epoch}",
	}
}

// Create returns a new endpoint for the provided API path.
func (_ *BeaconEndpointFactory) Create(path string) (*apimiddleware.Endpoint, error) {
	endpoint := apimiddleware.DefaultEndpoint()
	switch path {
	case "/eth/v1/beacon/genesis":
		endpoint.GetResponse = &GenesisResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/root":
		endpoint.GetResponse = &StateRootResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/fork":
		endpoint.GetResponse = &StateForkResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/finality_checkpoints":
		endpoint.GetResponse = &StateFinalityCheckpointResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/validators":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "id", Hex: true}, {Name: "status", Enum: true}}
		endpoint.GetResponse = &StateValidatorsResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/validators/{validator_id}":
		endpoint.GetResponse = &StateValidatorResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/validator_balances":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "id", Hex: true}}
		endpoint.GetResponse = &ValidatorBalancesResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/committees":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "epoch"}, {Name: "index"}, {Name: "slot"}}
		endpoint.GetResponse = &StateCommitteesResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/sync_committees":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "epoch"}}
		endpoint.GetResponse = &SyncCommitteesResponseJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeGrpcResponseBodyIntoContainer: prepareValidatorAggregates,
		}
	case "/eth/v1/beacon/states/{state_id}/randao":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "epoch"}}
		endpoint.GetResponse = &RandaoResponseJson{}
	case "/eth/v1/beacon/headers":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "slot"}, {Name: "parent_root", Hex: true}}
		endpoint.GetResponse = &BlockHeadersResponseJson{}
	case "/eth/v1/beacon/headers/{block_id}":
		endpoint.GetResponse = &BlockHeaderResponseJson{}
	case "/eth/v1/beacon/blocks":
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer:  setInitialPublishBlockPostRequest,
			OnPostDeserializeRequestBodyIntoContainer: preparePublishedBlock,
		}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleSubmitBlockSSZ}
	case "/eth/v1/beacon/blinded_blocks":
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer:  setInitialPublishBlindedBlockPostRequest,
			OnPostDeserializeRequestBodyIntoContainer: preparePublishedBlindedBlock,
		}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleSubmitBlindedBlockSSZ}
	case "/eth/v1/beacon/blocks/{block_id}":
		endpoint.GetResponse = &BlockResponseJson{}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleGetBeaconBlockSSZ}
	case "/eth/v2/beacon/blocks/{block_id}":
		endpoint.GetResponse = &BlockV2ResponseJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreSerializeMiddlewareResponseIntoJson: serializeV2Block,
		}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleGetBeaconBlockSSZV2}
	case "/eth/v1/beacon/blocks/{block_id}/root":
		endpoint.GetResponse = &BlockRootResponseJson{}
	case "/eth/v1/beacon/blocks/{block_id}/attestations":
		endpoint.GetResponse = &BlockAttestationsResponseJson{}
	case "/eth/v1/beacon/blinded_blocks/{block_id}":
		endpoint.GetResponse = &BlindedBlockResponseJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreSerializeMiddlewareResponseIntoJson: serializeBlindedBlock,
		}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleGetBlindedBeaconBlockSSZ}
	case "/eth/v1/beacon/pool/attestations":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "slot"}, {Name: "committee_index"}}
		endpoint.GetResponse = &AttestationsPoolResponseJson{}
		endpoint.PostRequest = &SubmitAttestationRequestJson{}
		endpoint.Err = &IndexedVerificationFailureErrorJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapAttestationsArray,
		}
	case "/eth/v1/beacon/pool/attester_slashings":
		endpoint.PostRequest = &AttesterSlashingJson{}
		endpoint.GetResponse = &AttesterSlashingsPoolResponseJson{}
	case "/eth/v1/beacon/pool/proposer_slashings":
		endpoint.PostRequest = &ProposerSlashingJson{}
		endpoint.GetResponse = &ProposerSlashingsPoolResponseJson{}
	case "/eth/v1/beacon/pool/voluntary_exits":
		endpoint.PostRequest = &SignedVoluntaryExitJson{}
		endpoint.GetResponse = &VoluntaryExitsPoolResponseJson{}
	case "/eth/v1/beacon/pool/bls_to_execution_changes":
		endpoint.PostRequest = &SubmitBLSToExecutionChangesRequest{}
		endpoint.GetResponse = &BLSToExecutionChangesPoolResponseJson{}
		endpoint.Err = &IndexedVerificationFailureErrorJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapBLSChangesArray,
		}
	case "/eth/v1/beacon/pool/sync_committees":
		endpoint.PostRequest = &SubmitSyncCommitteeSignaturesRequestJson{}
		endpoint.Err = &IndexedVerificationFailureErrorJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapSyncCommitteeSignaturesArray,
		}
	case "/eth/v1/beacon/weak_subjectivity":
		endpoint.GetResponse = &WeakSubjectivityResponse{}
	case "/eth/v1/node/identity":
		endpoint.GetResponse = &IdentityResponseJson{}
	case "/eth/v1/node/peers":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "state", Enum: true}, {Name: "direction", Enum: true}}
		endpoint.GetResponse = &PeersResponseJson{}
	case "/eth/v1/node/peers/{peer_id}":
		endpoint.RequestURLLiterals = []string{"peer_id"}
		endpoint.GetResponse = &PeerResponseJson{}
	case "/eth/v1/node/peer_count":
		endpoint.GetResponse = &PeerCountResponseJson{}
	case "/eth/v1/node/version":
		endpoint.GetResponse = &VersionResponseJson{}
	case "/eth/v1/node/syncing":
		endpoint.GetResponse = &SyncingResponseJson{}
	case "/eth/v1/node/health":
		// Use default endpoint
	case "/eth/v1/debug/beacon/states/{state_id}":
		endpoint.GetResponse = &BeaconStateResponseJson{}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleGetBeaconStateSSZ}
	case "/eth/v2/debug/beacon/states/{state_id}":
		endpoint.GetResponse = &BeaconStateV2ResponseJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreSerializeMiddlewareResponseIntoJson: serializeV2State,
		}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleGetBeaconStateSSZV2}
	case "/eth/v1/debug/beacon/heads":
		endpoint.GetResponse = &ForkChoiceHeadsResponseJson{}
	case "/eth/v2/debug/beacon/heads":
		endpoint.GetResponse = &V2ForkChoiceHeadsResponseJson{}
	case "/eth/v1/debug/fork_choice":
		endpoint.GetResponse = &ForkChoiceDumpJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreSerializeMiddlewareResponseIntoJson: prepareForkChoiceResponse,
		}
	case "/eth/v1/config/fork_schedule":
		endpoint.GetResponse = &ForkScheduleResponseJson{}
	case "/eth/v1/config/deposit_contract":
		endpoint.GetResponse = &DepositContractResponseJson{}
	case "/eth/v1/config/spec":
		endpoint.GetResponse = &SpecResponseJson{}
	case "/eth/v1/events":
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleEvents}
	case "/eth/v1/validator/duties/attester/{epoch}":
		endpoint.PostRequest = &ValidatorIndicesJson{}
		endpoint.PostResponse = &AttesterDutiesResponseJson{}
		endpoint.RequestURLLiterals = []string{"epoch"}
		endpoint.Err = &NodeSyncDetailsErrorJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapValidatorIndicesArray,
		}
	case "/eth/v1/validator/duties/proposer/{epoch}":
		endpoint.GetResponse = &ProposerDutiesResponseJson{}
		endpoint.RequestURLLiterals = []string{"epoch"}
		endpoint.Err = &NodeSyncDetailsErrorJson{}
	case "/eth/v1/validator/duties/sync/{epoch}":
		endpoint.PostRequest = &ValidatorIndicesJson{}
		endpoint.PostResponse = &SyncCommitteeDutiesResponseJson{}
		endpoint.RequestURLLiterals = []string{"epoch"}
		endpoint.Err = &NodeSyncDetailsErrorJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapValidatorIndicesArray,
		}
	case "/eth/v1/validator/blocks/{slot}":
		endpoint.GetResponse = &ProduceBlockResponseJson{}
		endpoint.RequestURLLiterals = []string{"slot"}
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "randao_reveal", Hex: true}, {Name: "graffiti", Hex: true}}
	case "/eth/v2/validator/blocks/{slot}":
		endpoint.GetResponse = &ProduceBlockResponseV2Json{}
		endpoint.RequestURLLiterals = []string{"slot"}
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "randao_reveal", Hex: true}, {Name: "graffiti", Hex: true}}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreSerializeMiddlewareResponseIntoJson: serializeProducedV2Block,
		}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleProduceBlockSSZ}
	case "/eth/v1/validator/blinded_blocks/{slot}":
		endpoint.GetResponse = &ProduceBlindedBlockResponseJson{}
		endpoint.RequestURLLiterals = []string{"slot"}
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "randao_reveal", Hex: true}, {Name: "graffiti", Hex: true}}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreSerializeMiddlewareResponseIntoJson: serializeProducedBlindedBlock,
		}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleProduceBlindedBlockSSZ}
	case "/eth/v1/validator/attestation_data":
		endpoint.GetResponse = &ProduceAttestationDataResponseJson{}
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "slot"}, {Name: "committee_index"}}
	case "/eth/v1/validator/beacon_committee_subscriptions":
		endpoint.PostRequest = &SubmitBeaconCommitteeSubscriptionsRequestJson{}
		endpoint.Err = &NodeSyncDetailsErrorJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapBeaconCommitteeSubscriptionsArray,
		}
	case "/eth/v1/validator/sync_committee_subscriptions":
		endpoint.PostRequest = &SubmitSyncCommitteeSubscriptionRequestJson{}
		endpoint.Err = &NodeSyncDetailsErrorJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapSyncCommitteeSubscriptionsArray,
		}
	case "/eth/v1/validator/aggregate_and_proofs":
		endpoint.PostRequest = &SubmitAggregateAndProofsRequestJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapSignedAggregateAndProofArray,
		}
	case "/eth/v1/validator/sync_committee_contribution":
		endpoint.GetResponse = &ProduceSyncCommitteeContributionResponseJson{}
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "slot"}, {Name: "subcommittee_index"}, {Name: "beacon_block_root", Hex: true}}
	case "/eth/v1/validator/contribution_and_proofs":
		endpoint.PostRequest = &SubmitContributionAndProofsRequestJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapSignedContributionAndProofsArray,
		}
	case "/eth/v1/validator/prepare_beacon_proposer":
		endpoint.PostRequest = &FeeRecipientsRequestJSON{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapFeeRecipientsArray,
		}
	case "/eth/v1/validator/register_validator":
		endpoint.PostRequest = &SignedValidatorRegistrationsRequestJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapSignedValidatorRegistrationsArray,
		}
	case "/eth/v1/validator/liveness/{epoch}":
		endpoint.PostRequest = &ValidatorIndicesJson{}
		endpoint.PostResponse = &LivenessResponseJson{}
		endpoint.RequestURLLiterals = []string{"epoch"}
		endpoint.Err = &NodeSyncDetailsErrorJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapValidatorIndicesArray,
		}
	default:
		return nil, errors.New("invalid path")
	}

	endpoint.Path = path
	return &endpoint, nil
}
