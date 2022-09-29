package apimiddleware

import (
	"strings"

	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/eth/helpers"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
)

//----------------
// Requests and responses.
//----------------

type genesisResponseJson struct {
	Data *genesisResponse_GenesisJson `json:"data"`
}

type genesisResponse_GenesisJson struct {
	GenesisTime           string `json:"genesis_time" time:"true"`
	GenesisValidatorsRoot string `json:"genesis_validators_root" hex:"true"`
	GenesisForkVersion    string `json:"genesis_fork_version" hex:"true"`
}

// WeakSubjectivityResponse is used to marshal/unmarshal the response for the
// /eth/v1/beacon/weak_subjectivity endpoint.
type WeakSubjectivityResponse struct {
	Data *struct {
		Checkpoint *checkpointJson `json:"ws_checkpoint"`
		StateRoot  string          `json:"state_root" hex:"true"`
	} `json:"data"`
}

type feeRecipientsRequestJSON struct {
	Recipients []*feeRecipientJson `json:"recipients"`
}

type stateRootResponseJson struct {
	Data                *stateRootResponse_StateRootJson `json:"data"`
	ExecutionOptimistic bool                             `json:"execution_optimistic"`
}

type stateRootResponse_StateRootJson struct {
	StateRoot string `json:"root" hex:"true"`
}

type stateForkResponseJson struct {
	Data                *forkJson `json:"data"`
	ExecutionOptimistic bool      `json:"execution_optimistic"`
}

type stateFinalityCheckpointResponseJson struct {
	Data                *stateFinalityCheckpointResponse_StateFinalityCheckpointJson `json:"data"`
	ExecutionOptimistic bool                                                         `json:"execution_optimistic"`
}

type stateFinalityCheckpointResponse_StateFinalityCheckpointJson struct {
	PreviousJustified *checkpointJson `json:"previous_justified"`
	CurrentJustified  *checkpointJson `json:"current_justified"`
	Finalized         *checkpointJson `json:"finalized"`
}

type stateValidatorsResponseJson struct {
	Data                []*validatorContainerJson `json:"data"`
	ExecutionOptimistic bool                      `json:"execution_optimistic"`
}

type stateValidatorResponseJson struct {
	Data                *validatorContainerJson `json:"data"`
	ExecutionOptimistic bool                    `json:"execution_optimistic"`
}

type validatorBalancesResponseJson struct {
	Data                []*validatorBalanceJson `json:"data"`
	ExecutionOptimistic bool                    `json:"execution_optimistic"`
}

type stateCommitteesResponseJson struct {
	Data                []*committeeJson `json:"data"`
	ExecutionOptimistic bool             `json:"execution_optimistic"`
}

type syncCommitteesResponseJson struct {
	Data                *syncCommitteeValidatorsJson `json:"data"`
	ExecutionOptimistic bool                         `json:"execution_optimistic"`
}

type blockHeadersResponseJson struct {
	Data                []*blockHeaderContainerJson `json:"data"`
	ExecutionOptimistic bool                        `json:"execution_optimistic"`
}

type blockHeaderResponseJson struct {
	Data                *blockHeaderContainerJson `json:"data"`
	ExecutionOptimistic bool                      `json:"execution_optimistic"`
}

type blockResponseJson struct {
	Data *signedBeaconBlockContainerJson `json:"data"`
}

type blockV2ResponseJson struct {
	Version             string                            `json:"version" enum:"true"`
	Data                *signedBeaconBlockContainerV2Json `json:"data"`
	ExecutionOptimistic bool                              `json:"execution_optimistic"`
}

type blockRootResponseJson struct {
	Data                *blockRootContainerJson `json:"data"`
	ExecutionOptimistic bool                    `json:"execution_optimistic"`
}

type blockAttestationsResponseJson struct {
	Data                []*attestationJson `json:"data"`
	ExecutionOptimistic bool               `json:"execution_optimistic"`
}

type attestationsPoolResponseJson struct {
	Data []*attestationJson `json:"data"`
}

type submitAttestationRequestJson struct {
	Data []*attestationJson `json:"data"`
}

type attesterSlashingsPoolResponseJson struct {
	Data []*attesterSlashingJson `json:"data"`
}

type proposerSlashingsPoolResponseJson struct {
	Data []*proposerSlashingJson `json:"data"`
}

type voluntaryExitsPoolResponseJson struct {
	Data []*signedVoluntaryExitJson `json:"data"`
}

type submitSyncCommitteeSignaturesRequestJson struct {
	Data []*syncCommitteeMessageJson `json:"data"`
}

type identityResponseJson struct {
	Data *identityJson `json:"data"`
}

type peersResponseJson struct {
	Data []*peerJson `json:"data"`
}

type peerResponseJson struct {
	Data *peerJson `json:"data"`
}

type peerCountResponseJson struct {
	Data peerCountResponse_PeerCountJson `json:"data"`
}

type peerCountResponse_PeerCountJson struct {
	Disconnected  string `json:"disconnected"`
	Connecting    string `json:"connecting"`
	Connected     string `json:"connected"`
	Disconnecting string `json:"disconnecting"`
}

type versionResponseJson struct {
	Data *versionJson `json:"data"`
}

type syncingResponseJson struct {
	Data *helpers.SyncDetailsJson `json:"data"`
}

type beaconStateResponseJson struct {
	Data *beaconStateJson `json:"data"`
}

type beaconStateV2ResponseJson struct {
	Version             string                      `json:"version" enum:"true"`
	Data                *beaconStateContainerV2Json `json:"data"`
	ExecutionOptimistic bool                        `json:"execution_optimistic"`
}

type forkChoiceHeadsResponseJson struct {
	Data []*forkChoiceHeadJson `json:"data"`
}

type v2ForkChoiceHeadsResponseJson struct {
	Data []*v2ForkChoiceHeadJson `json:"data"`
}

type forkScheduleResponseJson struct {
	Data []*forkJson `json:"data"`
}

type depositContractResponseJson struct {
	Data *depositContractJson `json:"data"`
}

type specResponseJson struct {
	Data interface{} `json:"data"`
}

type dutiesRequestJson struct {
	Index []string `json:"index"`
}

type attesterDutiesResponseJson struct {
	DependentRoot       string              `json:"dependent_root" hex:"true"`
	Data                []*attesterDutyJson `json:"data"`
	ExecutionOptimistic bool                `json:"execution_optimistic"`
}

type proposerDutiesResponseJson struct {
	DependentRoot       string              `json:"dependent_root" hex:"true"`
	Data                []*proposerDutyJson `json:"data"`
	ExecutionOptimistic bool                `json:"execution_optimistic"`
}

type syncCommitteeDutiesResponseJson struct {
	Data                []*syncCommitteeDuty `json:"data"`
	ExecutionOptimistic bool                 `json:"execution_optimistic"`
}

type produceBlockResponseJson struct {
	Data *beaconBlockJson `json:"data"`
}

type produceBlockResponseV2Json struct {
	Version string                      `json:"version"`
	Data    *beaconBlockContainerV2Json `json:"data"`
}

type produceBlindedBlockResponseJson struct {
	Version string                           `json:"version"`
	Data    *blindedBeaconBlockContainerJson `json:"data"`
}

type produceAttestationDataResponseJson struct {
	Data *attestationDataJson `json:"data"`
}

type aggregateAttestationResponseJson struct {
	Data *attestationJson `json:"data"`
}

type submitBeaconCommitteeSubscriptionsRequestJson struct {
	Data []*beaconCommitteeSubscribeJson `json:"data"`
}

type beaconCommitteeSubscribeJson struct {
	ValidatorIndex   string `json:"validator_index"`
	CommitteeIndex   string `json:"committee_index"`
	CommitteesAtSlot string `json:"committees_at_slot"`
	Slot             string `json:"slot"`
	IsAggregator     bool   `json:"is_aggregator"`
}

type submitSyncCommitteeSubscriptionRequestJson struct {
	Data []*syncCommitteeSubscriptionJson `json:"data"`
}

type syncCommitteeSubscriptionJson struct {
	ValidatorIndex       string   `json:"validator_index"`
	SyncCommitteeIndices []string `json:"sync_committee_indices"`
	UntilEpoch           string   `json:"until_epoch"`
}

type submitAggregateAndProofsRequestJson struct {
	Data []*signedAggregateAttestationAndProofJson `json:"data"`
}

type produceSyncCommitteeContributionResponseJson struct {
	Data *syncCommitteeContributionJson `json:"data"`
}

type submitContributionAndProofsRequestJson struct {
	Data []*signedContributionAndProofJson `json:"data"`
}

type forkchoiceResponse struct {
	JustifiedCheckpoint           *checkpointJson       `json:"justified_checkpoint"`
	FinalizedCheckpoint           *checkpointJson       `json:"finalized_checkpoint"`
	BestJustifiedCheckpoint       *checkpointJson       `json:"best_justified_checkpoint"`
	UnrealizedJustifiedCheckpoint *checkpointJson       `json:"unrealized_justified_checkpoint"`
	UnrealizedFinalizedCheckpoint *checkpointJson       `json:"unrealized_finalized_checkpoint"`
	ProposerBoostRoot             string                `json:"proposer_boost_root" hex:"true"`
	PreviousProposerBoostRoot     string                `json:"previous_proposer_boost_root" hex:"true"`
	HeadRoot                      string                `json:"head_root" hex:"true"`
	ForkChoiceNodes               []*forkChoiceNodeJson `json:"forkchoice_nodes"`
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
	Phase0Block    *beaconBlockJson          `json:"phase0_block"`
	AltairBlock    *beaconBlockAltairJson    `json:"altair_block"`
	BellatrixBlock *beaconBlockBellatrixJson `json:"bellatrix_block"`
	Signature      string                    `json:"signature" hex:"true"`
}

type beaconBlockContainerV2Json struct {
	Phase0Block    *beaconBlockJson          `json:"phase0_block"`
	AltairBlock    *beaconBlockAltairJson    `json:"altair_block"`
	BellatrixBlock *beaconBlockBellatrixJson `json:"bellatrix_block"`
}

type blindedBeaconBlockContainerJson struct {
	Phase0Block    *beaconBlockJson                 `json:"phase0_block"`
	AltairBlock    *beaconBlockAltairJson           `json:"altair_block"`
	BellatrixBlock *blindedBeaconBlockBellatrixJson `json:"bellatrix_block"`
}

type signedBeaconBlockAltairContainerJson struct {
	Message   *beaconBlockAltairJson `json:"message"`
	Signature string                 `json:"signature" hex:"true"`
}

type signedBeaconBlockBellatrixContainerJson struct {
	Message   *beaconBlockBellatrixJson `json:"message"`
	Signature string                    `json:"signature" hex:"true"`
}

type signedBlindedBeaconBlockBellatrixContainerJson struct {
	Message   *blindedBeaconBlockBellatrixJson `json:"message"`
	Signature string                           `json:"signature" hex:"true"`
}

type beaconBlockAltairJson struct {
	Slot          string                     `json:"slot"`
	ProposerIndex string                     `json:"proposer_index"`
	ParentRoot    string                     `json:"parent_root" hex:"true"`
	StateRoot     string                     `json:"state_root" hex:"true"`
	Body          *beaconBlockBodyAltairJson `json:"body"`
}

type beaconBlockBellatrixJson struct {
	Slot          string                        `json:"slot"`
	ProposerIndex string                        `json:"proposer_index"`
	ParentRoot    string                        `json:"parent_root" hex:"true"`
	StateRoot     string                        `json:"state_root" hex:"true"`
	Body          *beaconBlockBodyBellatrixJson `json:"body"`
}

type blindedBeaconBlockBellatrixJson struct {
	Slot          string                               `json:"slot"`
	ProposerIndex string                               `json:"proposer_index"`
	ParentRoot    string                               `json:"parent_root" hex:"true"`
	StateRoot     string                               `json:"state_root" hex:"true"`
	Body          *blindedBeaconBlockBodyBellatrixJson `json:"body"`
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

type beaconBlockBodyBellatrixJson struct {
	RandaoReveal      string                     `json:"randao_reveal" hex:"true"`
	Eth1Data          *eth1DataJson              `json:"eth1_data"`
	Graffiti          string                     `json:"graffiti" hex:"true"`
	ProposerSlashings []*proposerSlashingJson    `json:"proposer_slashings"`
	AttesterSlashings []*attesterSlashingJson    `json:"attester_slashings"`
	Attestations      []*attestationJson         `json:"attestations"`
	Deposits          []*depositJson             `json:"deposits"`
	VoluntaryExits    []*signedVoluntaryExitJson `json:"voluntary_exits"`
	SyncAggregate     *syncAggregateJson         `json:"sync_aggregate"`
	ExecutionPayload  *executionPayloadJson      `json:"execution_payload"`
}

type blindedBeaconBlockBodyBellatrixJson struct {
	RandaoReveal           string                      `json:"randao_reveal" hex:"true"`
	Eth1Data               *eth1DataJson               `json:"eth1_data"`
	Graffiti               string                      `json:"graffiti" hex:"true"`
	ProposerSlashings      []*proposerSlashingJson     `json:"proposer_slashings"`
	AttesterSlashings      []*attesterSlashingJson     `json:"attester_slashings"`
	Attestations           []*attestationJson          `json:"attestations"`
	Deposits               []*depositJson              `json:"deposits"`
	VoluntaryExits         []*signedVoluntaryExitJson  `json:"voluntary_exits"`
	SyncAggregate          *syncAggregateJson          `json:"sync_aggregate"`
	ExecutionPayloadHeader *executionPayloadHeaderJson `json:"execution_payload_header"`
}

type executionPayloadJson struct {
	ParentHash    string   `json:"parent_hash" hex:"true"`
	FeeRecipient  string   `json:"fee_recipient" hex:"true"`
	StateRoot     string   `json:"state_root" hex:"true"`
	ReceiptsRoot  string   `json:"receipts_root" hex:"true"`
	LogsBloom     string   `json:"logs_bloom" hex:"true"`
	PrevRandao    string   `json:"prev_randao" hex:"true"`
	BlockNumber   string   `json:"block_number"`
	GasLimit      string   `json:"gas_limit"`
	GasUsed       string   `json:"gas_used"`
	TimeStamp     string   `json:"timestamp"`
	ExtraData     string   `json:"extra_data" hex:"true"`
	BaseFeePerGas string   `json:"base_fee_per_gas" uint256:"true"`
	BlockHash     string   `json:"block_hash" hex:"true"`
	Transactions  []string `json:"transactions" hex:"true"`
}

type executionPayloadHeaderJson struct {
	ParentHash       string `json:"parent_hash" hex:"true"`
	FeeRecipient     string `json:"fee_recipient" hex:"true"`
	StateRoot        string `json:"state_root" hex:"true"`
	ReceiptsRoot     string `json:"receipts_root" hex:"true"`
	LogsBloom        string `json:"logs_bloom" hex:"true"`
	PrevRandao       string `json:"prev_randao" hex:"true"`
	BlockNumber      string `json:"block_number"`
	GasLimit         string `json:"gas_limit"`
	GasUsed          string `json:"gas_used"`
	TimeStamp        string `json:"timestamp"`
	ExtraData        string `json:"extra_data" hex:"true"`
	BaseFeePerGas    string `json:"base_fee_per_gas" uint256:"true"`
	BlockHash        string `json:"block_hash" hex:"true"`
	TransactionsRoot string `json:"transactions_root" hex:"true"`
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

type feeRecipientJson struct {
	ValidatorIndex string `json:"validator_index"`
	FeeRecipient   string `json:"fee_recipient" hex:"true"`
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

type beaconStateAltairJson struct {
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

type beaconStateBellatrixJson struct {
	GenesisTime                  string                      `json:"genesis_time"`
	GenesisValidatorsRoot        string                      `json:"genesis_validators_root" hex:"true"`
	Slot                         string                      `json:"slot"`
	Fork                         *forkJson                   `json:"fork"`
	LatestBlockHeader            *beaconBlockHeaderJson      `json:"latest_block_header"`
	BlockRoots                   []string                    `json:"block_roots" hex:"true"`
	StateRoots                   []string                    `json:"state_roots" hex:"true"`
	HistoricalRoots              []string                    `json:"historical_roots" hex:"true"`
	Eth1Data                     *eth1DataJson               `json:"eth1_data"`
	Eth1DataVotes                []*eth1DataJson             `json:"eth1_data_votes"`
	Eth1DepositIndex             string                      `json:"eth1_deposit_index"`
	Validators                   []*validatorJson            `json:"validators"`
	Balances                     []string                    `json:"balances"`
	RandaoMixes                  []string                    `json:"randao_mixes" hex:"true"`
	Slashings                    []string                    `json:"slashings"`
	PreviousEpochParticipation   EpochParticipation          `json:"previous_epoch_participation"`
	CurrentEpochParticipation    EpochParticipation          `json:"current_epoch_participation"`
	JustificationBits            string                      `json:"justification_bits" hex:"true"`
	PreviousJustifiedCheckpoint  *checkpointJson             `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint   *checkpointJson             `json:"current_justified_checkpoint"`
	FinalizedCheckpoint          *checkpointJson             `json:"finalized_checkpoint"`
	InactivityScores             []string                    `json:"inactivity_scores"`
	CurrentSyncCommittee         *syncCommitteeJson          `json:"current_sync_committee"`
	NextSyncCommittee            *syncCommitteeJson          `json:"next_sync_committee"`
	LatestExecutionPayloadHeader *executionPayloadHeaderJson `json:"latest_execution_payload_header"`
}

type beaconStateContainerV2Json struct {
	Phase0State    *beaconStateJson          `json:"phase0_state"`
	AltairState    *beaconStateAltairJson    `json:"altair_state"`
	BellatrixState *beaconStateBellatrixJson `json:"bellatrix_state"`
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

type v2ForkChoiceHeadJson struct {
	Root                string `json:"root" hex:"true"`
	Slot                string `json:"slot"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
}

type depositContractJson struct {
	ChainId string `json:"chain_id"`
	Address string `json:"address"`
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

type validatorRegistrationJson struct {
	FeeRecipient string `json:"fee_recipient" hex:"true"`
	GasLimit     string `json:"gas_limit"`
	Timestamp    string `json:"timestamp"`
	Pubkey       string `json:"pubkey" hex:"true"`
}

type signedValidatorRegistrationJson struct {
	Message   *validatorRegistrationJson `json:"message"`
	Signature string                     `json:"signature" hex:"true"`
}

type signedValidatorRegistrationsRequestJson struct {
	Registrations []*signedValidatorRegistrationJson `json:"registrations"`
}

type forkChoiceNodeJson struct {
	Slot                     string `json:"slot"`
	Root                     string `json:"root" hex:"true"`
	ParentRoot               string `json:"parent_root" hex:"true"`
	JustifiedEpoch           string `json:"justified_epoch"`
	FinalizedEpoch           string `json:"finalized_epoch"`
	UnrealizedJustifiedEpoch string `json:"unrealized_justified_epoch"`
	UnrealizedFinalizedEpoch string `json:"unrealized_finalized_epoch"`
	Balance                  string `json:"balance"`
	Weight                   string `json:"weight"`
	ExecutionOptimistic      bool   `json:"execution_optimistic"`
	ExecutionPayload         string `json:"execution_payload" hex:"true"`
	TimeStamp                string `json:"timestamp"`
}

//----------------
// SSZ
// ---------------

type sszRequestJson struct {
	Data string `json:"data"`
}

// sszResponse is a common abstraction over all SSZ responses.
type sszResponse interface {
	SSZVersion() string
	SSZData() string
}

type sszResponseJson struct {
	Data string `json:"data"`
}

func (ssz *sszResponseJson) SSZData() string {
	return ssz.Data
}

func (*sszResponseJson) SSZVersion() string {
	return strings.ToLower(ethpbv2.Version_PHASE0.String())
}

type versionedSSZResponseJson struct {
	Version string `json:"version"`
	Data    string `json:"data"`
}

func (ssz *versionedSSZResponseJson) SSZData() string {
	return ssz.Data
}

func (ssz *versionedSSZResponseJson) SSZVersion() string {
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
	ExecutionOptimistic       bool   `json:"execution_optimistic"`
	PreviousDutyDependentRoot string `json:"previous_duty_dependent_root" hex:"true"`
	CurrentDutyDependentRoot  string `json:"current_duty_dependent_root" hex:"true"`
}

type receivedBlockDataJson struct {
	Slot                string `json:"slot"`
	Block               string `json:"block" hex:"true"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
}

type aggregatedAttReceivedDataJson struct {
	Aggregate *attestationJson `json:"aggregate"`
}

type eventFinalizedCheckpointJson struct {
	Block               string `json:"block" hex:"true"`
	State               string `json:"state" hex:"true"`
	Epoch               string `json:"epoch"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
}

type eventChainReorgJson struct {
	Slot                string `json:"slot"`
	Depth               string `json:"depth"`
	OldHeadBlock        string `json:"old_head_block" hex:"true"`
	NewHeadBlock        string `json:"old_head_state" hex:"true"`
	OldHeadState        string `json:"new_head_block" hex:"true"`
	NewHeadState        string `json:"new_head_state" hex:"true"`
	Epoch               string `json:"epoch"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
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
	SyncDetails helpers.SyncDetailsJson `json:"sync_details"`
}

type eventErrorJson struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}
