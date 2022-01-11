package apimiddleware

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/api/gateway/apimiddleware"
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
		"/eth/v1/beacon/headers",
		"/eth/v1/beacon/headers/{block_id}",
		"/eth/v1/beacon/blocks",
		"/eth/v1/beacon/blocks/{block_id}",
		"/eth/v2/beacon/blocks/{block_id}",
		"/eth/v1/beacon/blocks/{block_id}/root",
		"/eth/v1/beacon/blocks/{block_id}/attestations",
		"/eth/v1/beacon/pool/attestations",
		"/eth/v1/beacon/pool/attester_slashings",
		"/eth/v1/beacon/pool/proposer_slashings",
		"/eth/v1/beacon/pool/voluntary_exits",
		"/eth/v1/beacon/pool/sync_committees",
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
		"/eth/v1/config/fork_schedule",
		"/eth/v1/config/deposit_contract",
		"/eth/v1/config/spec",
		"/eth/v1/events",
		"/eth/v1/validator/duties/attester/{epoch}",
		"/eth/v1/validator/duties/proposer/{epoch}",
		"/eth/v1/validator/duties/sync/{epoch}",
		"/eth/v1/validator/blocks/{slot}",
		"/eth/v2/validator/blocks/{slot}",
		"/eth/v1/validator/attestation_data",
		"/eth/v1/validator/aggregate_attestation",
		"/eth/v1/validator/beacon_committee_subscriptions",
		"/eth/v1/validator/sync_committee_subscriptions",
		"/eth/v1/validator/aggregate_and_proofs",
		"/eth/v1/validator/sync_committee_contribution",
		"/eth/v1/validator/contribution_and_proofs",
	}
}

// Create returns a new endpoint for the provided API path.
func (_ *BeaconEndpointFactory) Create(path string) (*apimiddleware.Endpoint, error) {
	endpoint := apimiddleware.DefaultEndpoint()
	switch path {
	case "/eth/v1/beacon/genesis":
		endpoint.GetResponse = &genesisResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/root":
		endpoint.GetResponse = &stateRootResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/fork":
		endpoint.GetResponse = &stateForkResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/finality_checkpoints":
		endpoint.GetResponse = &stateFinalityCheckpointResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/validators":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "id", Hex: true}, {Name: "status", Enum: true}}
		endpoint.GetResponse = &stateValidatorsResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/validators/{validator_id}":
		endpoint.GetResponse = &stateValidatorResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/validator_balances":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "id", Hex: true}}
		endpoint.GetResponse = &validatorBalancesResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/committees":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "epoch"}, {Name: "index"}, {Name: "slot"}}
		endpoint.GetResponse = &stateCommitteesResponseJson{}
	case "/eth/v1/beacon/states/{state_id}/sync_committees":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "epoch"}}
		endpoint.GetResponse = &syncCommitteesResponseJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeGrpcResponseBodyIntoContainer: prepareValidatorAggregates,
		}
	case "/eth/v1/beacon/headers":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "slot"}, {Name: "parent_root", Hex: true}}
		endpoint.GetResponse = &blockHeadersResponseJson{}
	case "/eth/v1/beacon/headers/{block_id}":
		endpoint.GetResponse = &blockHeaderResponseJson{}
	case "/eth/v1/beacon/blocks":
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer:  setInitialPublishBlockPostRequest,
			OnPostDeserializeRequestBodyIntoContainer: preparePublishedBlock,
		}
	case "/eth/v1/beacon/blocks/{block_id}":
		endpoint.GetResponse = &blockResponseJson{}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleGetBeaconBlockSSZ}
	case "/eth/v2/beacon/blocks/{block_id}":
		endpoint.GetResponse = &blockV2ResponseJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreSerializeMiddlewareResponseIntoJson: serializeV2Block,
		}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleGetBeaconBlockSSZV2}
	case "/eth/v1/beacon/blocks/{block_id}/root":
		endpoint.GetResponse = &blockRootResponseJson{}
	case "/eth/v1/beacon/blocks/{block_id}/attestations":
		endpoint.GetResponse = &blockAttestationsResponseJson{}
	case "/eth/v1/beacon/pool/attestations":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "slot"}, {Name: "committee_index"}}
		endpoint.GetResponse = &attestationsPoolResponseJson{}
		endpoint.PostRequest = &submitAttestationRequestJson{}
		endpoint.Err = &indexedVerificationFailureErrorJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapAttestationsArray,
		}
	case "/eth/v1/beacon/pool/attester_slashings":
		endpoint.PostRequest = &attesterSlashingJson{}
		endpoint.GetResponse = &attesterSlashingsPoolResponseJson{}
	case "/eth/v1/beacon/pool/proposer_slashings":
		endpoint.PostRequest = &proposerSlashingJson{}
		endpoint.GetResponse = &proposerSlashingsPoolResponseJson{}
	case "/eth/v1/beacon/pool/voluntary_exits":
		endpoint.PostRequest = &signedVoluntaryExitJson{}
		endpoint.GetResponse = &voluntaryExitsPoolResponseJson{}
	case "/eth/v1/beacon/pool/sync_committees":
		endpoint.PostRequest = &submitSyncCommitteeSignaturesRequestJson{}
		endpoint.Err = &indexedVerificationFailureErrorJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapSyncCommitteeSignaturesArray,
		}
	case "/eth/v1/node/identity":
		endpoint.GetResponse = &identityResponseJson{}
	case "/eth/v1/node/peers":
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "state", Enum: true}, {Name: "direction", Enum: true}}
		endpoint.GetResponse = &peersResponseJson{}
	case "/eth/v1/node/peers/{peer_id}":
		endpoint.RequestURLLiterals = []string{"peer_id"}
		endpoint.GetResponse = &peerResponseJson{}
	case "/eth/v1/node/peer_count":
		endpoint.GetResponse = &peerCountResponseJson{}
	case "/eth/v1/node/version":
		endpoint.GetResponse = &versionResponseJson{}
	case "/eth/v1/node/syncing":
		endpoint.GetResponse = &syncingResponseJson{}
	case "/eth/v1/node/health":
		// Use default endpoint
	case "/eth/v1/debug/beacon/states/{state_id}":
		endpoint.GetResponse = &beaconStateResponseJson{}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleGetBeaconStateSSZ}
	case "/eth/v2/debug/beacon/states/{state_id}":
		endpoint.GetResponse = &beaconStateV2ResponseJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreSerializeMiddlewareResponseIntoJson: serializeV2State,
		}
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleGetBeaconStateSSZV2}
	case "/eth/v1/debug/beacon/heads":
		endpoint.GetResponse = &forkChoiceHeadsResponseJson{}
	case "/eth/v1/config/fork_schedule":
		endpoint.GetResponse = &forkScheduleResponseJson{}
	case "/eth/v1/config/deposit_contract":
		endpoint.GetResponse = &depositContractResponseJson{}
	case "/eth/v1/config/spec":
		endpoint.GetResponse = &specResponseJson{}
	case "/eth/v1/events":
		endpoint.CustomHandlers = []apimiddleware.CustomHandler{handleEvents}
	case "/eth/v1/validator/duties/attester/{epoch}":
		endpoint.PostRequest = &dutiesRequestJson{}
		endpoint.PostResponse = &attesterDutiesResponseJson{}
		endpoint.RequestURLLiterals = []string{"epoch"}
		endpoint.Err = &nodeSyncDetailsErrorJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapValidatorIndicesArray,
		}
	case "/eth/v1/validator/duties/proposer/{epoch}":
		endpoint.GetResponse = &proposerDutiesResponseJson{}
		endpoint.RequestURLLiterals = []string{"epoch"}
		endpoint.Err = &nodeSyncDetailsErrorJson{}
	case "/eth/v1/validator/duties/sync/{epoch}":
		endpoint.PostRequest = &dutiesRequestJson{}
		endpoint.PostResponse = &syncCommitteeDutiesResponseJson{}
		endpoint.RequestURLLiterals = []string{"epoch"}
		endpoint.Err = &nodeSyncDetailsErrorJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapValidatorIndicesArray,
		}
	case "/eth/v1/validator/blocks/{slot}":
		endpoint.GetResponse = &produceBlockResponseJson{}
		endpoint.RequestURLLiterals = []string{"slot"}
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "randao_reveal", Hex: true}, {Name: "graffiti", Hex: true}}
	case "/eth/v2/validator/blocks/{slot}":
		endpoint.GetResponse = &produceBlockResponseV2Json{}
		endpoint.RequestURLLiterals = []string{"slot"}
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "randao_reveal", Hex: true}, {Name: "graffiti", Hex: true}}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreSerializeMiddlewareResponseIntoJson: serializeProducedV2Block,
		}
	case "/eth/v1/validator/attestation_data":
		endpoint.GetResponse = &produceAttestationDataResponseJson{}
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "slot"}, {Name: "committee_index"}}
	case "/eth/v1/validator/aggregate_attestation":
		endpoint.GetResponse = &aggregateAttestationResponseJson{}
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "attestation_data_root", Hex: true}, {Name: "slot"}}
	case "/eth/v1/validator/beacon_committee_subscriptions":
		endpoint.PostRequest = &submitBeaconCommitteeSubscriptionsRequestJson{}
		endpoint.Err = &nodeSyncDetailsErrorJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapBeaconCommitteeSubscriptionsArray,
		}
	case "/eth/v1/validator/sync_committee_subscriptions":
		endpoint.PostRequest = &submitSyncCommitteeSubscriptionRequestJson{}
		endpoint.Err = &nodeSyncDetailsErrorJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapSyncCommitteeSubscriptionsArray,
		}
	case "/eth/v1/validator/aggregate_and_proofs":
		endpoint.PostRequest = &submitAggregateAndProofsRequestJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapSignedAggregateAndProofArray,
		}
	case "/eth/v1/validator/sync_committee_contribution":
		endpoint.GetResponse = &produceSyncCommitteeContributionResponseJson{}
		endpoint.RequestQueryParams = []apimiddleware.QueryParam{{Name: "slot"}, {Name: "subcommittee_index"}, {Name: "beacon_block_root", Hex: true}}
	case "/eth/v1/validator/contribution_and_proofs":
		endpoint.PostRequest = &submitContributionAndProofsRequestJson{}
		endpoint.Hooks = apimiddleware.HookCollection{
			OnPreDeserializeRequestBodyIntoContainer: wrapSignedContributionAndProofsArray,
		}
	default:
		return nil, errors.New("invalid path")
	}

	endpoint.Path = path
	return &endpoint, nil
}
