package apimiddleware

import (
	"strings"

	"github.com/prysmaticlabs/prysm/api/gateway/apimiddleware"
	ethpbv2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
)

// genesisResponseJson is used in /beacon/genesis API endpoint.
type genesisResponseJson struct {
	Data *genesisResponse_GenesisJson `json:"data"`
}

// genesisResponse_GenesisJson is used in /beacon/genesis API endpoint.
type genesisResponse_GenesisJson struct {
	GenesisTime           string `json:"genesis_time" time:"true"`
	GenesisValidatorsRoot string `json:"genesis_validators_root" hex:"true"`
	GenesisForkVersion    string `json:"genesis_fork_version" hex:"true"`
}

// stateRootResponseJson is used in /beacon/states/{state_id}/root API endpoint.
type stateRootResponseJson struct {
	Data *stateRootResponse_StateRootJson `json:"data"`
}

// stateRootResponse_StateRootJson is used in /beacon/states/{state_id}/root API endpoint.
type stateRootResponse_StateRootJson struct {
	StateRoot string `json:"root" hex:"true"`
}

// stateForkResponseJson is used in /beacon/states/{state_id}/fork API endpoint.
type stateForkResponseJson struct {
	Data *forkJson `json:"data"`
}

// stateFinalityCheckpointResponseJson is used in /beacon/states/{state_id}/finality_checkpoints API endpoint.
type stateFinalityCheckpointResponseJson struct {
	Data *stateFinalityCheckpointResponse_StateFinalityCheckpointJson `json:"data"`
}

// stateFinalityCheckpointResponse_StateFinalityCheckpointJson is used in /beacon/states/{state_id}/finality_checkpoints API endpoint.
type stateFinalityCheckpointResponse_StateFinalityCheckpointJson struct {
	PreviousJustified *checkpointJson `json:"previous_justified"`
	CurrentJustified  *checkpointJson `json:"current_justified"`
	Finalized         *checkpointJson `json:"finalized"`
}

// stateValidatorResponseJson is used in /beacon/states/{state_id}/validators API endpoint.
type stateValidatorsResponseJson struct {
	Data []*validatorContainerJson `json:"data"`
}

// stateValidatorResponseJson is used in /beacon/states/{state_id}/validators/{validator_id} API endpoint.
type stateValidatorResponseJson struct {
	Data *validatorContainerJson `json:"data"`
}

// validatorBalancesResponseJson is used in /beacon/states/{state_id}/validator_balances API endpoint.
type validatorBalancesResponseJson struct {
	Data []*validatorBalanceJson `json:"data"`
}

// stateCommitteesResponseJson is used in /beacon/states/{state_id}/committees API endpoint.
type stateCommitteesResponseJson struct {
	Data []*committeeJson `json:"data"`
}

// syncCommitteesResponseJson is used in /beacon/states/{state_id}/sync_committees API endpoint.
type syncCommitteesResponseJson struct {
	Data *syncCommitteeValidatorsJson `json:"data"`
}

// blockHeadersResponseJson is used in /beacon/headers API endpoint.
type blockHeadersResponseJson struct {
	Data []*blockHeaderContainerJson `json:"data"`
}

// blockHeaderResponseJson is used in /beacon/headers/{block_id} API endpoint.
type blockHeaderResponseJson struct {
	Data *blockHeaderContainerJson `json:"data"`
}

// blockResponseJson is used in /beacon/blocks/{block_id} API endpoint.
type blockResponseJson struct {
	Data *signedBeaconBlockContainerJson `json:"data"`
}

// blockV2ResponseJson is used in /v2/beacon/blocks/{block_id} API endpoint.
type blockV2ResponseJson struct {
	Version string                            `json:"version" enum:"true"`
	Data    *signedBeaconBlockContainerV2Json `json:"data"`
}

// blockRootResponseJson is used in /beacon/blocks/{block_id}/root API endpoint.
type blockRootResponseJson struct {
	Data *blockRootContainerJson `json:"data"`
}

// blockAttestationsResponseJson is used in /beacon/blocks/{block_id}/attestations API endpoint.
type blockAttestationsResponseJson struct {
	Data []*attestationJson `json:"data"`
}

// attestationsPoolResponseJson is used in /beacon/pool/attestations GET API endpoint.
type attestationsPoolResponseJson struct {
	Data []*attestationJson `json:"data"`
}

// submitAttestationRequestJson is used in /beacon/pool/attestations POST API endpoint.
type submitAttestationRequestJson struct {
	Data []*attestationJson `json:"data"`
}

// attesterSlashingsPoolResponseJson is used in /beacon/pool/attester_slashings API endpoint.
type attesterSlashingsPoolResponseJson struct {
	Data []*attesterSlashingJson `json:"data"`
}

// proposerSlashingsPoolResponseJson is used in /beacon/pool/proposer_slashings API endpoint.
type proposerSlashingsPoolResponseJson struct {
	Data []*proposerSlashingJson `json:"data"`
}

// voluntaryExitsPoolResponseJson is used in /beacon/pool/voluntary_exits API endpoint.
type voluntaryExitsPoolResponseJson struct {
	Data []*signedVoluntaryExitJson `json:"data"`
}

// submitSyncCommitteeSignaturesRequestJson is used in /beacon/pool/sync_committees API endpoint.
type submitSyncCommitteeSignaturesRequestJson struct {
	Data []*syncCommitteeMessageJson `json:"data"`
}

// identityResponseJson is used in /node/identity API endpoint.
type identityResponseJson struct {
	Data *identityJson `json:"data"`
}

// peersResponseJson is used in /node/peers API endpoint.
type peersResponseJson struct {
	Data []*peerJson `json:"data"`
}

// peerResponseJson is used in /node/peers/{peer_id} API endpoint.
type peerResponseJson struct {
	Data *peerJson `json:"data"`
}

// peerCountResponseJson is used in /node/peer_count API endpoint.
type peerCountResponseJson struct {
	Data peerCountResponse_PeerCountJson `json:"data"`
}

// peerCountResponse_PeerCountJson is used in /node/peer_count API endpoint.
type peerCountResponse_PeerCountJson struct {
	Disconnected  string `json:"disconnected"`
	Connecting    string `json:"connecting"`
	Connected     string `json:"connected"`
	Disconnecting string `json:"disconnecting"`
}

// versionResponseJson is used in /node/version API endpoint.
type versionResponseJson struct {
	Data *versionJson `json:"data"`
}

// syncingResponseJson is used in /node/syncing API endpoint.
type syncingResponseJson struct {
	Data *syncInfoJson `json:"data"`
}

// beaconStateResponseJson is used in /debug/beacon/states/{state_id} API endpoint.
type beaconStateResponseJson struct {
	Data *beaconStateJson `json:"data"`
}

// beaconStateV2ResponseJson is used in /v2/debug/beacon/states/{state_id} API endpoint.
type beaconStateV2ResponseJson struct {
	Version string                      `json:"version" enum:"true"`
	Data    *beaconStateContainerV2Json `json:"data"`
}

// forkChoiceHeadsResponseJson is used in /debug/beacon/heads API endpoint.
type forkChoiceHeadsResponseJson struct {
	Data []*forkChoiceHeadJson `json:"data"`
}

// forkScheduleResponseJson is used in /config/fork_schedule API endpoint.
type forkScheduleResponseJson struct {
	Data []*forkJson `json:"data"`
}

// depositContractResponseJson is used in /config/deposit_contract API endpoint.
type depositContractResponseJson struct {
	Data *depositContractJson `json:"data"`
}

// specResponseJson is used in /config/spec API endpoint.
type specResponseJson struct {
	Data interface{} `json:"data"`
}

// dutiesRequestJson is used in several duties-related API endpoints.
type dutiesRequestJson struct {
	Index []string `json:"index"`
}

// attesterDutiesResponseJson is used in /validator/duties/attester/{epoch} API endpoint.
type attesterDutiesResponseJson struct {
	DependentRoot string              `json:"dependent_root" hex:"true"`
	Data          []*attesterDutyJson `json:"data"`
}

// proposerDutiesResponseJson is used in /validator/duties/proposer/{epoch} API endpoint.
type proposerDutiesResponseJson struct {
	DependentRoot string              `json:"dependent_root" hex:"true"`
	Data          []*proposerDutyJson `json:"data"`
}

// syncCommitteeDutiesResponseJson is used in /validator/duties/sync/{epoch} API endpoint.
type syncCommitteeDutiesResponseJson struct {
	Data []*syncCommitteeDuty `json:"data"`
}

// produceBlockResponseJson is used in /validator/blocks/{slot} API endpoint.
type produceBlockResponseJson struct {
	Data *beaconBlockJson `json:"data"`
}

// produceBlockResponseV2Json is used in /v2/validator/blocks/{slot} API endpoint.
type produceBlockResponseV2Json struct {
	Version string                      `json:"version"`
	Data    *beaconBlockContainerV2Json `json:"data"`
}

// produceAttestationDataResponseJson is used in /validator/attestation_data API endpoint.
type produceAttestationDataResponseJson struct {
	Data *attestationDataJson `json:"data"`
}

// aggregateAttestationResponseJson is used in /validator/aggregate_attestation API endpoint.
type aggregateAttestationResponseJson struct {
	Data *attestationJson `json:"data"`
}

// submitBeaconCommitteeSubscriptionsRequestJson is used in /validator/beacon_committee_subscriptions API endpoint.
type submitBeaconCommitteeSubscriptionsRequestJson struct {
	Data []*beaconCommitteeSubscribeJson `json:"data"`
}

// beaconCommitteeSubscribeJson is used in /validator/beacon_committee_subscriptions API endpoint.
type beaconCommitteeSubscribeJson struct {
	ValidatorIndex   string `json:"validator_index"`
	CommitteeIndex   string `json:"committee_index"`
	CommitteesAtSlot string `json:"committees_at_slot"`
	Slot             string `json:"slot"`
	IsAggregator     bool   `json:"is_aggregator"`
}

// submitBeaconCommitteeSubscriptionsRequestJson is used in /validator/sync_committee_subscriptions API endpoint.
type submitSyncCommitteeSubscriptionRequestJson struct {
	Data []*syncCommitteeSubscriptionJson `json:"data"`
}

// syncCommitteeSubscriptionJson is used in /validator/sync_committee_subscriptions API endpoint.
type syncCommitteeSubscriptionJson struct {
	ValidatorIndex       string   `json:"validator_index"`
	SyncCommitteeIndices []string `json:"sync_committee_indices"`
	UntilEpoch           string   `json:"until_epoch"`
}

// submitAggregateAndProofsRequestJson is used in /validator/aggregate_and_proofs API endpoint.
type submitAggregateAndProofsRequestJson struct {
	Data []*signedAggregateAttestationAndProofJson `json:"data"`
}

// produceSyncCommitteeContributionResponseJson is used in /validator/sync_committee_contribution API endpoint.
type produceSyncCommitteeContributionResponseJson struct {
	Data *syncCommitteeContributionJson `json:"data"`
}

// submitContributionAndProofsRequestJson is used in /validator/contribution_and_proofs API endpoint.
type submitContributionAndProofsRequestJson struct {
	Data []*signedContributionAndProofJson `json:"data"`
}

//----------------
// Reusable types.
//----------------

type checkpointJson struct {
	Epoch string `json:"epoch"`
	Root  string `json:"root" hex:"true"`
}

type blockRootContainerJson struct {
	Root string `json:"root" hex:"true"`
}

type signedBeaconBlockContainerJson struct {
	Message   *beaconBlockJson `json:"message"`
	Signature string           `json:"signature" hex:"true"`
}

type beaconBlockJson struct {
	Slot          string               `json:"slot"`
	ProposerIndex string               `json:"proposer_index"`
	ParentRoot    string               `json:"parent_root" hex:"true"`
	StateRoot     string               `json:"state_root" hex:"true"`
	Body          *beaconBlockBodyJson `json:"body"`
}

type beaconBlockBodyJson struct {
	RandaoReveal      string                     `json:"randao_reveal" hex:"true"`
	Eth1Data          *eth1DataJson              `json:"eth1_data"`
	Graffiti          string                     `json:"graffiti" hex:"true"`
	ProposerSlashings []*proposerSlashingJson    `json:"proposer_slashings"`
	AttesterSlashings []*attesterSlashingJson    `json:"attester_slashings"`
	Attestations      []*attestationJson         `json:"attestations"`
	Deposits          []*depositJson             `json:"deposits"`
	VoluntaryExits    []*signedVoluntaryExitJson `json:"voluntary_exits"`
}

type signedBeaconBlockContainerV2Json struct {
	Phase0Block *beaconBlockJson       `json:"phase0_block"`
	AltairBlock *beaconBlockAltairJson `json:"altair_block"`
	Signature   string                 `json:"signature" hex:"true"`
}

type beaconBlockContainerV2Json struct {
	Phase0Block *beaconBlockJson       `json:"phase0_block"`
	AltairBlock *beaconBlockAltairJson `json:"altair_block"`
}

type signedBeaconBlockAltairContainerJson struct {
	Message   *beaconBlockAltairJson `json:"message"`
	Signature string                 `json:"signature" hex:"true"`
}

type beaconBlockAltairJson struct {
	Slot          string                     `json:"slot"`
	ProposerIndex string                     `json:"proposer_index"`
	ParentRoot    string                     `json:"parent_root" hex:"true"`
	StateRoot     string                     `json:"state_root" hex:"true"`
	Body          *beaconBlockBodyAltairJson `json:"body"`
}

type beaconBlockBodyAltairJson struct {
	RandaoReveal      string                     `json:"randao_reveal" hex:"true"`
	Eth1Data          *eth1DataJson              `json:"eth1_data"`
	Graffiti          string                     `json:"graffiti" hex:"true"`
	ProposerSlashings []*proposerSlashingJson    `json:"proposer_slashings"`
	AttesterSlashings []*attesterSlashingJson    `json:"attester_slashings"`
	Attestations      []*attestationJson         `json:"attestations"`
	Deposits          []*depositJson             `json:"deposits"`
	VoluntaryExits    []*signedVoluntaryExitJson `json:"voluntary_exits"`
	SyncAggregate     *syncAggregateJson         `json:"sync_aggregate"`
}

type syncAggregateJson struct {
	SyncCommitteeBits      string `json:"sync_committee_bits" hex:"true"`
	SyncCommitteeSignature string `json:"sync_committee_signature" hex:"true"`
}

type blockHeaderContainerJson struct {
	Root      string                          `json:"root" hex:"true"`
	Canonical bool                            `json:"canonical"`
	Header    *beaconBlockHeaderContainerJson `json:"header"`
}

type beaconBlockHeaderContainerJson struct {
	Message   *beaconBlockHeaderJson `json:"message"`
	Signature string                 `json:"signature" hex:"true"`
}

type signedBeaconBlockHeaderJson struct {
	Header    *beaconBlockHeaderJson `json:"message"`
	Signature string                 `json:"signature" hex:"true"`
}

type beaconBlockHeaderJson struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root" hex:"true"`
	StateRoot     string `json:"state_root" hex:"true"`
	BodyRoot      string `json:"body_root" hex:"true"`
}

type eth1DataJson struct {
	DepositRoot  string `json:"deposit_root" hex:"true"`
	DepositCount string `json:"deposit_count"`
	BlockHash    string `json:"block_hash" hex:"true"`
}

type proposerSlashingJson struct {
	Header_1 *signedBeaconBlockHeaderJson `json:"signed_header_1"`
	Header_2 *signedBeaconBlockHeaderJson `json:"signed_header_2"`
}

type attesterSlashingJson struct {
	Attestation_1 *indexedAttestationJson `json:"attestation_1"`
	Attestation_2 *indexedAttestationJson `json:"attestation_2"`
}

type indexedAttestationJson struct {
	AttestingIndices []string             `json:"attesting_indices"`
	Data             *attestationDataJson `json:"data"`
	Signature        string               `json:"signature" hex:"true"`
}

type attestationJson struct {
	AggregationBits string               `json:"aggregation_bits" hex:"true"`
	Data            *attestationDataJson `json:"data"`
	Signature       string               `json:"signature" hex:"true"`
}

type attestationDataJson struct {
	Slot            string          `json:"slot"`
	CommitteeIndex  string          `json:"index"`
	BeaconBlockRoot string          `json:"beacon_block_root" hex:"true"`
	Source          *checkpointJson `json:"source"`
	Target          *checkpointJson `json:"target"`
}

type depositJson struct {
	Proof []string          `json:"proof" hex:"true"`
	Data  *deposit_DataJson `json:"data"`
}

type deposit_DataJson struct {
	PublicKey             string `json:"pubkey" hex:"true"`
	WithdrawalCredentials string `json:"withdrawal_credentials" hex:"true"`
	Amount                string `json:"amount"`
	Signature             string `json:"signature" hex:"true"`
}

type signedVoluntaryExitJson struct {
	Exit      *voluntaryExitJson `json:"message"`
	Signature string             `json:"signature" hex:"true"`
}

type voluntaryExitJson struct {
	Epoch          string `json:"epoch"`
	ValidatorIndex string `json:"validator_index"`
}

type syncCommitteeMessageJson struct {
	Slot            string `json:"slot"`
	BeaconBlockRoot string `json:"beacon_block_root" hex:"true"`
	ValidatorIndex  string `json:"validator_index"`
	Signature       string `json:"signature" hex:"true"`
}

type identityJson struct {
	PeerId             string        `json:"peer_id"`
	Enr                string        `json:"enr"`
	P2PAddresses       []string      `json:"p2p_addresses"`
	DiscoveryAddresses []string      `json:"discovery_addresses"`
	Metadata           *metadataJson `json:"metadata"`
}

type metadataJson struct {
	SeqNumber string `json:"seq_number"`
	Attnets   string `json:"attnets" hex:"true"`
}

type peerJson struct {
	PeerId    string `json:"peer_id"`
	Enr       string `json:"enr"`
	Address   string `json:"last_seen_p2p_address"`
	State     string `json:"state" enum:"true"`
	Direction string `json:"direction" enum:"true"`
}

type versionJson struct {
	Version string `json:"version"`
}

type beaconStateJson struct {
	GenesisTime                 string                    `json:"genesis_time"`
	GenesisValidatorsRoot       string                    `json:"genesis_validators_root" hex:"true"`
	Slot                        string                    `json:"slot"`
	Fork                        *forkJson                 `json:"fork"`
	LatestBlockHeader           *beaconBlockHeaderJson    `json:"latest_block_header"`
	BlockRoots                  []string                  `json:"block_roots" hex:"true"`
	StateRoots                  []string                  `json:"state_roots" hex:"true"`
	HistoricalRoots             []string                  `json:"historical_roots" hex:"true"`
	Eth1Data                    *eth1DataJson             `json:"eth1_data"`
	Eth1DataVotes               []*eth1DataJson           `json:"eth1_data_votes"`
	Eth1DepositIndex            string                    `json:"eth1_deposit_index"`
	Validators                  []*validatorJson          `json:"validators"`
	Balances                    []string                  `json:"balances"`
	RandaoMixes                 []string                  `json:"randao_mixes" hex:"true"`
	Slashings                   []string                  `json:"slashings"`
	PreviousEpochAttestations   []*pendingAttestationJson `json:"previous_epoch_attestations"`
	CurrentEpochAttestations    []*pendingAttestationJson `json:"current_epoch_attestations"`
	JustificationBits           string                    `json:"justification_bits" hex:"true"`
	PreviousJustifiedCheckpoint *checkpointJson           `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint  *checkpointJson           `json:"current_justified_checkpoint"`
	FinalizedCheckpoint         *checkpointJson           `json:"finalized_checkpoint"`
}

type beaconStateV2Json struct {
	GenesisTime                 string                 `json:"genesis_time"`
	GenesisValidatorsRoot       string                 `json:"genesis_validators_root" hex:"true"`
	Slot                        string                 `json:"slot"`
	Fork                        *forkJson              `json:"fork"`
	LatestBlockHeader           *beaconBlockHeaderJson `json:"latest_block_header"`
	BlockRoots                  []string               `json:"block_roots" hex:"true"`
	StateRoots                  []string               `json:"state_roots" hex:"true"`
	HistoricalRoots             []string               `json:"historical_roots" hex:"true"`
	Eth1Data                    *eth1DataJson          `json:"eth1_data"`
	Eth1DataVotes               []*eth1DataJson        `json:"eth1_data_votes"`
	Eth1DepositIndex            string                 `json:"eth1_deposit_index"`
	Validators                  []*validatorJson       `json:"validators"`
	Balances                    []string               `json:"balances"`
	RandaoMixes                 []string               `json:"randao_mixes" hex:"true"`
	Slashings                   []string               `json:"slashings"`
	PreviousEpochParticipation  EpochParticipation     `json:"previous_epoch_participation"`
	CurrentEpochParticipation   EpochParticipation     `json:"current_epoch_participation"`
	JustificationBits           string                 `json:"justification_bits" hex:"true"`
	PreviousJustifiedCheckpoint *checkpointJson        `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint  *checkpointJson        `json:"current_justified_checkpoint"`
	FinalizedCheckpoint         *checkpointJson        `json:"finalized_checkpoint"`
	InactivityScores            []string               `json:"inactivity_scores"`
	CurrentSyncCommittee        *syncCommitteeJson     `json:"current_sync_committee"`
	NextSyncCommittee           *syncCommitteeJson     `json:"next_sync_committee"`
}

type beaconStateContainerV2Json struct {
	Phase0State *beaconStateJson   `json:"phase0_state"`
	AltairState *beaconStateV2Json `json:"altair_state"`
}

type forkJson struct {
	PreviousVersion string `json:"previous_version" hex:"true"`
	CurrentVersion  string `json:"current_version" hex:"true"`
	Epoch           string `json:"epoch"`
}

type validatorContainerJson struct {
	Index     string         `json:"index"`
	Balance   string         `json:"balance"`
	Status    string         `json:"status" enum:"true"`
	Validator *validatorJson `json:"validator"`
}

type validatorJson struct {
	PublicKey                  string `json:"pubkey" hex:"true"`
	WithdrawalCredentials      string `json:"withdrawal_credentials" hex:"true"`
	EffectiveBalance           string `json:"effective_balance"`
	Slashed                    bool   `json:"slashed"`
	ActivationEligibilityEpoch string `json:"activation_eligibility_epoch"`
	ActivationEpoch            string `json:"activation_epoch"`
	ExitEpoch                  string `json:"exit_epoch"`
	WithdrawableEpoch          string `json:"withdrawable_epoch"`
}

type validatorBalanceJson struct {
	Index   string `json:"index"`
	Balance string `json:"balance"`
}

type committeeJson struct {
	Index      string   `json:"index"`
	Slot       string   `json:"slot"`
	Validators []string `json:"validators"`
}

type syncCommitteeJson struct {
	Pubkeys         []string `json:"pubkeys" hex:"true"`
	AggregatePubkey string   `json:"aggregate_pubkey" hex:"true"`
}

type syncCommitteeValidatorsJson struct {
	Validators          []string   `json:"validators"`
	ValidatorAggregates [][]string `json:"validator_aggregates"`
}

type pendingAttestationJson struct {
	AggregationBits string               `json:"aggregation_bits" hex:"true"`
	Data            *attestationDataJson `json:"data"`
	InclusionDelay  string               `json:"inclusion_delay"`
	ProposerIndex   string               `json:"proposer_index"`
}

type forkChoiceHeadJson struct {
	Root string `json:"root" hex:"true"`
	Slot string `json:"slot"`
}

type depositContractJson struct {
	ChainId string `json:"chain_id"`
	Address string `json:"address"`
}

type syncInfoJson struct {
	HeadSlot     string `json:"head_slot"`
	SyncDistance string `json:"sync_distance"`
	IsSyncing    bool   `json:"is_syncing"`
}

type attesterDutyJson struct {
	Pubkey                  string `json:"pubkey" hex:"true"`
	ValidatorIndex          string `json:"validator_index"`
	CommitteeIndex          string `json:"committee_index"`
	CommitteeLength         string `json:"committee_length"`
	CommitteesAtSlot        string `json:"committees_at_slot"`
	ValidatorCommitteeIndex string `json:"validator_committee_index"`
	Slot                    string `json:"slot"`
}

type proposerDutyJson struct {
	Pubkey         string `json:"pubkey" hex:"true"`
	ValidatorIndex string `json:"validator_index"`
	Slot           string `json:"slot"`
}

type syncCommitteeDuty struct {
	Pubkey                        string   `json:"pubkey" hex:"true"`
	ValidatorIndex                string   `json:"validator_index"`
	ValidatorSyncCommitteeIndices []string `json:"validator_sync_committee_indices"`
}

type signedAggregateAttestationAndProofJson struct {
	Message   *aggregateAttestationAndProofJson `json:"message"`
	Signature string                            `json:"signature" hex:"true"`
}

type aggregateAttestationAndProofJson struct {
	AggregatorIndex string           `json:"aggregator_index"`
	Aggregate       *attestationJson `json:"aggregate"`
	SelectionProof  string           `json:"selection_proof" hex:"true"`
}

type signedContributionAndProofJson struct {
	Message   *contributionAndProofJson `json:"message"`
	Signature string                    `json:"signature" hex:"true"`
}

type contributionAndProofJson struct {
	AggregatorIndex string                         `json:"aggregator_index"`
	Contribution    *syncCommitteeContributionJson `json:"contribution"`
	SelectionProof  string                         `json:"selection_proof" hex:"true"`
}

type syncCommitteeContributionJson struct {
	Slot              string `json:"slot"`
	BeaconBlockRoot   string `json:"beacon_block_root" hex:"true"`
	SubcommitteeIndex string `json:"subcommittee_index"`
	AggregationBits   string `json:"aggregation_bits" hex:"true"`
	Signature         string `json:"signature" hex:"true"`
}

//----------------
// SSZ
// ---------------

// sszResponseJson is a common abstraction over all SSZ responses.
type sszResponseJson interface {
	SSZVersion() string
	SSZData() string
}

// blockSSZResponseJson is used in /beacon/blocks/{block_id} API endpoint.
type blockSSZResponseJson struct {
	Data string `json:"data"`
}

func (ssz *blockSSZResponseJson) SSZData() string {
	return ssz.Data
}

func (*blockSSZResponseJson) SSZVersion() string {
	return strings.ToLower(ethpbv2.Version_PHASE0.String())
}

// blockSSZResponseV2Json is used in /v2/beacon/blocks/{block_id} API endpoint.
type blockSSZResponseV2Json struct {
	Version string `json:"version"`
	Data    string `json:"data"`
}

func (ssz *blockSSZResponseV2Json) SSZData() string {
	return ssz.Data
}

func (ssz *blockSSZResponseV2Json) SSZVersion() string {
	return ssz.Version
}

// beaconStateSSZResponseJson is used in /debug/beacon/states/{state_id} API endpoint.
type beaconStateSSZResponseJson struct {
	Data string `json:"data"`
}

func (ssz *beaconStateSSZResponseJson) SSZData() string {
	return ssz.Data
}

func (*beaconStateSSZResponseJson) SSZVersion() string {
	return strings.ToLower(ethpbv2.Version_PHASE0.String())
}

// beaconStateSSZResponseV2Json is used in /v2/debug/beacon/states/{state_id} API endpoint.
type beaconStateSSZResponseV2Json struct {
	Version string `json:"version"`
	Data    string `json:"data"`
}

func (ssz *beaconStateSSZResponseV2Json) SSZData() string {
	return ssz.Data
}

func (ssz *beaconStateSSZResponseV2Json) SSZVersion() string {
	return ssz.Version
}

// ---------------
// Events.
// ---------------

type eventHeadJson struct {
	Slot                      string `json:"slot"`
	Block                     string `json:"block" hex:"true"`
	State                     string `json:"state" hex:"true"`
	EpochTransition           bool   `json:"epoch_transition"`
	PreviousDutyDependentRoot string `json:"previous_duty_dependent_root" hex:"true"`
	CurrentDutyDependentRoot  string `json:"current_duty_dependent_root" hex:"true"`
}

type receivedBlockDataJson struct {
	Slot  string `json:"slot"`
	Block string `json:"block" hex:"true"`
}

type aggregatedAttReceivedDataJson struct {
	Aggregate *attestationJson `json:"aggregate"`
}

type eventFinalizedCheckpointJson struct {
	Block string `json:"block" hex:"true"`
	State string `json:"state" hex:"true"`
	Epoch string `json:"epoch"`
}

type eventChainReorgJson struct {
	Slot         string `json:"slot"`
	Depth        string `json:"depth"`
	OldHeadBlock string `json:"old_head_block" hex:"true"`
	NewHeadBlock string `json:"old_head_state" hex:"true"`
	OldHeadState string `json:"new_head_block" hex:"true"`
	NewHeadState string `json:"new_head_state" hex:"true"`
	Epoch        string `json:"epoch"`
}

// ---------------
// Error handling.
// ---------------

// indexedVerificationFailureErrorJson is a JSON representation of the error returned when verifying an indexed object.
type indexedVerificationFailureErrorJson struct {
	apimiddleware.DefaultErrorJson
	Failures []*singleIndexedVerificationFailureJson `json:"failures"`
}

// singleIndexedVerificationFailureJson is a JSON representation of a an issue when verifying a single indexed object e.g. an item in an array.
type singleIndexedVerificationFailureJson struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

type nodeSyncDetailsErrorJson struct {
	apimiddleware.DefaultErrorJson
	SyncDetails syncDetails `json:"sync_details"`
}

type eventErrorJson struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

type syncDetails struct {
	HeadSlot     string `json:"head_slot"`
	SyncDistance string `json:"sync_distance"`
	IsSyncing    bool   `json:"is_syncing"`
}
