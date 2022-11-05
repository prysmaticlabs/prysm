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

type GenesisResponseJson struct {
	Data *GenesisResponse_GenesisJson `json:"data"`
}

type GenesisResponse_GenesisJson struct {
	GenesisTime           string `json:"genesis_time" time:"true"`
	GenesisValidatorsRoot string `json:"genesis_validators_root" hex:"true"`
	GenesisForkVersion    string `json:"genesis_fork_version" hex:"true"`
}

// WeakSubjectivityResponse is used to marshal/unmarshal the response for the
// /eth/v1/beacon/weak_subjectivity endpoint.
type WeakSubjectivityResponse struct {
	Data *struct {
		Checkpoint *CheckpointJson `json:"ws_checkpoint"`
		StateRoot  string          `json:"state_root" hex:"true"`
	} `json:"data"`
}

type FeeRecipientsRequestJSON struct {
	Recipients []*FeeRecipientJson `json:"recipients"`
}

type StateRootResponseJson struct {
	Data                *StateRootResponse_StateRootJson `json:"data"`
	ExecutionOptimistic bool                             `json:"execution_optimistic"`
}

type StateRootResponse_StateRootJson struct {
	StateRoot string `json:"root" hex:"true"`
}

type StateForkResponseJson struct {
	Data                *ForkJson `json:"data"`
	ExecutionOptimistic bool      `json:"execution_optimistic"`
}

type StateFinalityCheckpointResponseJson struct {
	Data                *StateFinalityCheckpointResponse_StateFinalityCheckpointJson `json:"data"`
	ExecutionOptimistic bool                                                         `json:"execution_optimistic"`
}

type StateFinalityCheckpointResponse_StateFinalityCheckpointJson struct {
	PreviousJustified *CheckpointJson `json:"previous_justified"`
	CurrentJustified  *CheckpointJson `json:"current_justified"`
	Finalized         *CheckpointJson `json:"finalized"`
}

type StateValidatorsResponseJson struct {
	Data                []*ValidatorContainerJson `json:"data"`
	ExecutionOptimistic bool                      `json:"execution_optimistic"`
}

type StateValidatorResponseJson struct {
	Data                *ValidatorContainerJson `json:"data"`
	ExecutionOptimistic bool                    `json:"execution_optimistic"`
}

type ValidatorBalancesResponseJson struct {
	Data                []*ValidatorBalanceJson `json:"data"`
	ExecutionOptimistic bool                    `json:"execution_optimistic"`
}

type StateCommitteesResponseJson struct {
	Data                []*CommitteeJson `json:"data"`
	ExecutionOptimistic bool             `json:"execution_optimistic"`
}

type SyncCommitteesResponseJson struct {
	Data                *SyncCommitteeValidatorsJson `json:"data"`
	ExecutionOptimistic bool                         `json:"execution_optimistic"`
}

type RandaoResponseJson struct {
	Data *struct {
		Randao string `json:"randao" hex:"true"`
	} `json:"data"`
	ExecutionOptimistic bool `json:"execution_optimistic"`
}

type BlockHeadersResponseJson struct {
	Data                []*BlockHeaderContainerJson `json:"data"`
	ExecutionOptimistic bool                        `json:"execution_optimistic"`
}

type BlockHeaderResponseJson struct {
	Data                *BlockHeaderContainerJson `json:"data"`
	ExecutionOptimistic bool                      `json:"execution_optimistic"`
}

type BlockResponseJson struct {
	Data *SignedBeaconBlockContainerJson `json:"data"`
}

type BlockV2ResponseJson struct {
	Version             string                            `json:"version" enum:"true"`
	Data                *SignedBeaconBlockContainerV2Json `json:"data"`
	ExecutionOptimistic bool                              `json:"execution_optimistic"`
}

type BlindedBlockResponseJson struct {
	Version             string                                 `json:"version" enum:"true"`
	Data                *SignedBlindedBeaconBlockContainerJson `json:"data"`
	ExecutionOptimistic bool                                   `json:"execution_optimistic"`
}

type BlockRootResponseJson struct {
	Data                *BlockRootContainerJson `json:"data"`
	ExecutionOptimistic bool                    `json:"execution_optimistic"`
}

type BlockAttestationsResponseJson struct {
	Data                []*AttestationJson `json:"data"`
	ExecutionOptimistic bool               `json:"execution_optimistic"`
}

type AttestationsPoolResponseJson struct {
	Data []*AttestationJson `json:"data"`
}

type SubmitAttestationRequestJson struct {
	Data []*AttestationJson `json:"data"`
}

type AttesterSlashingsPoolResponseJson struct {
	Data []*AttesterSlashingJson `json:"data"`
}

type ProposerSlashingsPoolResponseJson struct {
	Data []*ProposerSlashingJson `json:"data"`
}

type VoluntaryExitsPoolResponseJson struct {
	Data []*SignedVoluntaryExitJson `json:"data"`
}

type SubmitSyncCommitteeSignaturesRequestJson struct {
	Data []*SyncCommitteeMessageJson `json:"data"`
}

type IdentityResponseJson struct {
	Data *IdentityJson `json:"data"`
}

type PeersResponseJson struct {
	Data []*PeerJson `json:"data"`
}

type PeerResponseJson struct {
	Data *PeerJson `json:"data"`
}

type PeerCountResponseJson struct {
	Data PeerCountResponse_PeerCountJson `json:"data"`
}

type PeerCountResponse_PeerCountJson struct {
	Disconnected  string `json:"disconnected"`
	Connecting    string `json:"connecting"`
	Connected     string `json:"connected"`
	Disconnecting string `json:"disconnecting"`
}

type VersionResponseJson struct {
	Data *VersionJson `json:"data"`
}

type SyncingResponseJson struct {
	Data *helpers.SyncDetailsJson `json:"data"`
}

type BeaconStateResponseJson struct {
	Data *BeaconStateJson `json:"data"`
}

type BeaconStateV2ResponseJson struct {
	Version             string                      `json:"version" enum:"true"`
	Data                *BeaconStateContainerV2Json `json:"data"`
	ExecutionOptimistic bool                        `json:"execution_optimistic"`
}

type ForkChoiceHeadsResponseJson struct {
	Data []*ForkChoiceHeadJson `json:"data"`
}

type V2ForkChoiceHeadsResponseJson struct {
	Data []*V2ForkChoiceHeadJson `json:"data"`
}

type ForkScheduleResponseJson struct {
	Data []*ForkJson `json:"data"`
}

type DepositContractResponseJson struct {
	Data *DepositContractJson `json:"data"`
}

type SpecResponseJson struct {
	Data interface{} `json:"data"`
}

type DutiesRequestJson struct {
	Index []string `json:"index"`
}

type AttesterDutiesResponseJson struct {
	DependentRoot       string              `json:"dependent_root" hex:"true"`
	Data                []*AttesterDutyJson `json:"data"`
	ExecutionOptimistic bool                `json:"execution_optimistic"`
}

type ProposerDutiesResponseJson struct {
	DependentRoot       string              `json:"dependent_root" hex:"true"`
	Data                []*ProposerDutyJson `json:"data"`
	ExecutionOptimistic bool                `json:"execution_optimistic"`
}

type SyncCommitteeDutiesResponseJson struct {
	Data                []*SyncCommitteeDuty `json:"data"`
	ExecutionOptimistic bool                 `json:"execution_optimistic"`
}

type ProduceBlockResponseJson struct {
	Data *BeaconBlockJson `json:"data"`
}

type ProduceBlockResponseV2Json struct {
	Version string                      `json:"version"`
	Data    *BeaconBlockContainerV2Json `json:"data"`
}

type ProduceBlindedBlockResponseJson struct {
	Version string                           `json:"version"`
	Data    *BlindedBeaconBlockContainerJson `json:"data"`
}

type ProduceAttestationDataResponseJson struct {
	Data *AttestationDataJson `json:"data"`
}

type AggregateAttestationResponseJson struct {
	Data *AttestationJson `json:"data"`
}

type SubmitBeaconCommitteeSubscriptionsRequestJson struct {
	Data []*BeaconCommitteeSubscribeJson `json:"data"`
}

type BeaconCommitteeSubscribeJson struct {
	ValidatorIndex   string `json:"validator_index"`
	CommitteeIndex   string `json:"committee_index"`
	CommitteesAtSlot string `json:"committees_at_slot"`
	Slot             string `json:"slot"`
	IsAggregator     bool   `json:"is_aggregator"`
}

type SubmitSyncCommitteeSubscriptionRequestJson struct {
	Data []*SyncCommitteeSubscriptionJson `json:"data"`
}

type SyncCommitteeSubscriptionJson struct {
	ValidatorIndex       string   `json:"validator_index"`
	SyncCommitteeIndices []string `json:"sync_committee_indices"`
	UntilEpoch           string   `json:"until_epoch"`
}

type SubmitAggregateAndProofsRequestJson struct {
	Data []*SignedAggregateAttestationAndProofJson `json:"data"`
}

type ProduceSyncCommitteeContributionResponseJson struct {
	Data *SyncCommitteeContributionJson `json:"data"`
}

type SubmitContributionAndProofsRequestJson struct {
	Data []*SignedContributionAndProofJson `json:"data"`
}

type ForkchoiceResponse struct {
	JustifiedCheckpoint           *CheckpointJson       `json:"justified_checkpoint"`
	FinalizedCheckpoint           *CheckpointJson       `json:"finalized_checkpoint"`
	BestJustifiedCheckpoint       *CheckpointJson       `json:"best_justified_checkpoint"`
	UnrealizedJustifiedCheckpoint *CheckpointJson       `json:"unrealized_justified_checkpoint"`
	UnrealizedFinalizedCheckpoint *CheckpointJson       `json:"unrealized_finalized_checkpoint"`
	ProposerBoostRoot             string                `json:"proposer_boost_root" hex:"true"`
	PreviousProposerBoostRoot     string                `json:"previous_proposer_boost_root" hex:"true"`
	HeadRoot                      string                `json:"head_root" hex:"true"`
	ForkChoiceNodes               []*ForkChoiceNodeJson `json:"forkchoice_nodes"`
}

//----------------
// Reusable types.
//----------------

type CheckpointJson struct {
	Epoch string `json:"epoch"`
	Root  string `json:"root" hex:"true"`
}

type BlockRootContainerJson struct {
	Root string `json:"root" hex:"true"`
}

type SignedBeaconBlockContainerJson struct {
	Message   *BeaconBlockJson `json:"message"`
	Signature string           `json:"signature" hex:"true"`
}

type BeaconBlockJson struct {
	Slot          string               `json:"slot"`
	ProposerIndex string               `json:"proposer_index"`
	ParentRoot    string               `json:"parent_root" hex:"true"`
	StateRoot     string               `json:"state_root" hex:"true"`
	Body          *BeaconBlockBodyJson `json:"body"`
}

type BeaconBlockBodyJson struct {
	RandaoReveal      string                     `json:"randao_reveal" hex:"true"`
	Eth1Data          *Eth1DataJson              `json:"eth1_data"`
	Graffiti          string                     `json:"graffiti" hex:"true"`
	ProposerSlashings []*ProposerSlashingJson    `json:"proposer_slashings"`
	AttesterSlashings []*AttesterSlashingJson    `json:"attester_slashings"`
	Attestations      []*AttestationJson         `json:"attestations"`
	Deposits          []*DepositJson             `json:"deposits"`
	VoluntaryExits    []*SignedVoluntaryExitJson `json:"voluntary_exits"`
}

type SignedBeaconBlockContainerV2Json struct {
	Phase0Block    *BeaconBlockJson          `json:"phase0_block"`
	AltairBlock    *BeaconBlockAltairJson    `json:"altair_block"`
	BellatrixBlock *BeaconBlockBellatrixJson `json:"bellatrix_block"`
	Signature      string                    `json:"signature" hex:"true"`
}

type SignedBlindedBeaconBlockContainerJson struct {
	Phase0Block    *BeaconBlockJson                 `json:"phase0_block"`
	AltairBlock    *BeaconBlockAltairJson           `json:"altair_block"`
	BellatrixBlock *BlindedBeaconBlockBellatrixJson `json:"bellatrix_block"`
	Signature      string                           `json:"signature" hex:"true"`
}

type BeaconBlockContainerV2Json struct {
	Phase0Block    *BeaconBlockJson          `json:"phase0_block"`
	AltairBlock    *BeaconBlockAltairJson    `json:"altair_block"`
	BellatrixBlock *BeaconBlockBellatrixJson `json:"bellatrix_block"`
}

type BlindedBeaconBlockContainerJson struct {
	Phase0Block    *BeaconBlockJson                 `json:"phase0_block"`
	AltairBlock    *BeaconBlockAltairJson           `json:"altair_block"`
	BellatrixBlock *BlindedBeaconBlockBellatrixJson `json:"bellatrix_block"`
}

type SignedBeaconBlockAltairContainerJson struct {
	Message   *BeaconBlockAltairJson `json:"message"`
	Signature string                 `json:"signature" hex:"true"`
}

type SignedBeaconBlockBellatrixContainerJson struct {
	Message   *BeaconBlockBellatrixJson `json:"message"`
	Signature string                    `json:"signature" hex:"true"`
}

type SignedBlindedBeaconBlockBellatrixContainerJson struct {
	Message   *BlindedBeaconBlockBellatrixJson `json:"message"`
	Signature string                           `json:"signature" hex:"true"`
}

type BeaconBlockAltairJson struct {
	Slot          string                     `json:"slot"`
	ProposerIndex string                     `json:"proposer_index"`
	ParentRoot    string                     `json:"parent_root" hex:"true"`
	StateRoot     string                     `json:"state_root" hex:"true"`
	Body          *BeaconBlockBodyAltairJson `json:"body"`
}

type BeaconBlockBellatrixJson struct {
	Slot          string                        `json:"slot"`
	ProposerIndex string                        `json:"proposer_index"`
	ParentRoot    string                        `json:"parent_root" hex:"true"`
	StateRoot     string                        `json:"state_root" hex:"true"`
	Body          *BeaconBlockBodyBellatrixJson `json:"body"`
}

type BlindedBeaconBlockBellatrixJson struct {
	Slot          string                               `json:"slot"`
	ProposerIndex string                               `json:"proposer_index"`
	ParentRoot    string                               `json:"parent_root" hex:"true"`
	StateRoot     string                               `json:"state_root" hex:"true"`
	Body          *BlindedBeaconBlockBodyBellatrixJson `json:"body"`
}

type BeaconBlockBodyAltairJson struct {
	RandaoReveal      string                     `json:"randao_reveal" hex:"true"`
	Eth1Data          *Eth1DataJson              `json:"eth1_data"`
	Graffiti          string                     `json:"graffiti" hex:"true"`
	ProposerSlashings []*ProposerSlashingJson    `json:"proposer_slashings"`
	AttesterSlashings []*AttesterSlashingJson    `json:"attester_slashings"`
	Attestations      []*AttestationJson         `json:"attestations"`
	Deposits          []*DepositJson             `json:"deposits"`
	VoluntaryExits    []*SignedVoluntaryExitJson `json:"voluntary_exits"`
	SyncAggregate     *SyncAggregateJson         `json:"sync_aggregate"`
}

type BeaconBlockBodyBellatrixJson struct {
	RandaoReveal      string                     `json:"randao_reveal" hex:"true"`
	Eth1Data          *Eth1DataJson              `json:"eth1_data"`
	Graffiti          string                     `json:"graffiti" hex:"true"`
	ProposerSlashings []*ProposerSlashingJson    `json:"proposer_slashings"`
	AttesterSlashings []*AttesterSlashingJson    `json:"attester_slashings"`
	Attestations      []*AttestationJson         `json:"attestations"`
	Deposits          []*DepositJson             `json:"deposits"`
	VoluntaryExits    []*SignedVoluntaryExitJson `json:"voluntary_exits"`
	SyncAggregate     *SyncAggregateJson         `json:"sync_aggregate"`
	ExecutionPayload  *ExecutionPayloadJson      `json:"execution_payload"`
}

type BlindedBeaconBlockBodyBellatrixJson struct {
	RandaoReveal           string                      `json:"randao_reveal" hex:"true"`
	Eth1Data               *Eth1DataJson               `json:"eth1_data"`
	Graffiti               string                      `json:"graffiti" hex:"true"`
	ProposerSlashings      []*ProposerSlashingJson     `json:"proposer_slashings"`
	AttesterSlashings      []*AttesterSlashingJson     `json:"attester_slashings"`
	Attestations           []*AttestationJson          `json:"attestations"`
	Deposits               []*DepositJson              `json:"deposits"`
	VoluntaryExits         []*SignedVoluntaryExitJson  `json:"voluntary_exits"`
	SyncAggregate          *SyncAggregateJson          `json:"sync_aggregate"`
	ExecutionPayloadHeader *ExecutionPayloadHeaderJson `json:"execution_payload_header"`
}

type ExecutionPayloadJson struct {
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

type ExecutionPayloadHeaderJson struct {
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

type SyncAggregateJson struct {
	SyncCommitteeBits      string `json:"sync_committee_bits" hex:"true"`
	SyncCommitteeSignature string `json:"sync_committee_signature" hex:"true"`
}

type BlockHeaderContainerJson struct {
	Root      string                          `json:"root" hex:"true"`
	Canonical bool                            `json:"canonical"`
	Header    *BeaconBlockHeaderContainerJson `json:"header"`
}

type BeaconBlockHeaderContainerJson struct {
	Message   *BeaconBlockHeaderJson `json:"message"`
	Signature string                 `json:"signature" hex:"true"`
}

type SignedBeaconBlockHeaderJson struct {
	Header    *BeaconBlockHeaderJson `json:"message"`
	Signature string                 `json:"signature" hex:"true"`
}

type BeaconBlockHeaderJson struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root" hex:"true"`
	StateRoot     string `json:"state_root" hex:"true"`
	BodyRoot      string `json:"body_root" hex:"true"`
}

type Eth1DataJson struct {
	DepositRoot  string `json:"deposit_root" hex:"true"`
	DepositCount string `json:"deposit_count"`
	BlockHash    string `json:"block_hash" hex:"true"`
}

type ProposerSlashingJson struct {
	Header_1 *SignedBeaconBlockHeaderJson `json:"signed_header_1"`
	Header_2 *SignedBeaconBlockHeaderJson `json:"signed_header_2"`
}

type AttesterSlashingJson struct {
	Attestation_1 *IndexedAttestationJson `json:"attestation_1"`
	Attestation_2 *IndexedAttestationJson `json:"attestation_2"`
}

type IndexedAttestationJson struct {
	AttestingIndices []string             `json:"attesting_indices"`
	Data             *AttestationDataJson `json:"data"`
	Signature        string               `json:"signature" hex:"true"`
}

type FeeRecipientJson struct {
	ValidatorIndex string `json:"validator_index"`
	FeeRecipient   string `json:"fee_recipient" hex:"true"`
}

type AttestationJson struct {
	AggregationBits string               `json:"aggregation_bits" hex:"true"`
	Data            *AttestationDataJson `json:"data"`
	Signature       string               `json:"signature" hex:"true"`
}

type AttestationDataJson struct {
	Slot            string          `json:"slot"`
	CommitteeIndex  string          `json:"index"`
	BeaconBlockRoot string          `json:"beacon_block_root" hex:"true"`
	Source          *CheckpointJson `json:"source"`
	Target          *CheckpointJson `json:"target"`
}

type DepositJson struct {
	Proof []string          `json:"proof" hex:"true"`
	Data  *Deposit_DataJson `json:"data"`
}

type Deposit_DataJson struct {
	PublicKey             string `json:"pubkey" hex:"true"`
	WithdrawalCredentials string `json:"withdrawal_credentials" hex:"true"`
	Amount                string `json:"amount"`
	Signature             string `json:"signature" hex:"true"`
}

type SignedVoluntaryExitJson struct {
	Exit      *VoluntaryExitJson `json:"message"`
	Signature string             `json:"signature" hex:"true"`
}

type VoluntaryExitJson struct {
	Epoch          string `json:"epoch"`
	ValidatorIndex string `json:"validator_index"`
}

type SyncCommitteeMessageJson struct {
	Slot            string `json:"slot"`
	BeaconBlockRoot string `json:"beacon_block_root" hex:"true"`
	ValidatorIndex  string `json:"validator_index"`
	Signature       string `json:"signature" hex:"true"`
}

type IdentityJson struct {
	PeerId             string        `json:"peer_id"`
	Enr                string        `json:"enr"`
	P2PAddresses       []string      `json:"p2p_addresses"`
	DiscoveryAddresses []string      `json:"discovery_addresses"`
	Metadata           *MetadataJson `json:"metadata"`
}

type MetadataJson struct {
	SeqNumber string `json:"seq_number"`
	Attnets   string `json:"attnets" hex:"true"`
}

type PeerJson struct {
	PeerId    string `json:"peer_id"`
	Enr       string `json:"enr"`
	Address   string `json:"last_seen_p2p_address"`
	State     string `json:"state" enum:"true"`
	Direction string `json:"direction" enum:"true"`
}

type VersionJson struct {
	Version string `json:"version"`
}

type BeaconStateJson struct {
	GenesisTime                 string                    `json:"genesis_time"`
	GenesisValidatorsRoot       string                    `json:"genesis_validators_root" hex:"true"`
	Slot                        string                    `json:"slot"`
	Fork                        *ForkJson                 `json:"fork"`
	LatestBlockHeader           *BeaconBlockHeaderJson    `json:"latest_block_header"`
	BlockRoots                  []string                  `json:"block_roots" hex:"true"`
	StateRoots                  []string                  `json:"state_roots" hex:"true"`
	HistoricalRoots             []string                  `json:"historical_roots" hex:"true"`
	Eth1Data                    *Eth1DataJson             `json:"eth1_data"`
	Eth1DataVotes               []*Eth1DataJson           `json:"eth1_data_votes"`
	Eth1DepositIndex            string                    `json:"eth1_deposit_index"`
	Validators                  []*ValidatorJson          `json:"validators"`
	Balances                    []string                  `json:"balances"`
	RandaoMixes                 []string                  `json:"randao_mixes" hex:"true"`
	Slashings                   []string                  `json:"slashings"`
	PreviousEpochAttestations   []*PendingAttestationJson `json:"previous_epoch_attestations"`
	CurrentEpochAttestations    []*PendingAttestationJson `json:"current_epoch_attestations"`
	JustificationBits           string                    `json:"justification_bits" hex:"true"`
	PreviousJustifiedCheckpoint *CheckpointJson           `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint  *CheckpointJson           `json:"current_justified_checkpoint"`
	FinalizedCheckpoint         *CheckpointJson           `json:"finalized_checkpoint"`
}

type BeaconStateAltairJson struct {
	GenesisTime                 string                 `json:"genesis_time"`
	GenesisValidatorsRoot       string                 `json:"genesis_validators_root" hex:"true"`
	Slot                        string                 `json:"slot"`
	Fork                        *ForkJson              `json:"fork"`
	LatestBlockHeader           *BeaconBlockHeaderJson `json:"latest_block_header"`
	BlockRoots                  []string               `json:"block_roots" hex:"true"`
	StateRoots                  []string               `json:"state_roots" hex:"true"`
	HistoricalRoots             []string               `json:"historical_roots" hex:"true"`
	Eth1Data                    *Eth1DataJson          `json:"eth1_data"`
	Eth1DataVotes               []*Eth1DataJson        `json:"eth1_data_votes"`
	Eth1DepositIndex            string                 `json:"eth1_deposit_index"`
	Validators                  []*ValidatorJson       `json:"validators"`
	Balances                    []string               `json:"balances"`
	RandaoMixes                 []string               `json:"randao_mixes" hex:"true"`
	Slashings                   []string               `json:"slashings"`
	PreviousEpochParticipation  EpochParticipation     `json:"previous_epoch_participation"`
	CurrentEpochParticipation   EpochParticipation     `json:"current_epoch_participation"`
	JustificationBits           string                 `json:"justification_bits" hex:"true"`
	PreviousJustifiedCheckpoint *CheckpointJson        `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint  *CheckpointJson        `json:"current_justified_checkpoint"`
	FinalizedCheckpoint         *CheckpointJson        `json:"finalized_checkpoint"`
	InactivityScores            []string               `json:"inactivity_scores"`
	CurrentSyncCommittee        *SyncCommitteeJson     `json:"current_sync_committee"`
	NextSyncCommittee           *SyncCommitteeJson     `json:"next_sync_committee"`
}

type BeaconStateBellatrixJson struct {
	GenesisTime                  string                      `json:"genesis_time"`
	GenesisValidatorsRoot        string                      `json:"genesis_validators_root" hex:"true"`
	Slot                         string                      `json:"slot"`
	Fork                         *ForkJson                   `json:"fork"`
	LatestBlockHeader            *BeaconBlockHeaderJson      `json:"latest_block_header"`
	BlockRoots                   []string                    `json:"block_roots" hex:"true"`
	StateRoots                   []string                    `json:"state_roots" hex:"true"`
	HistoricalRoots              []string                    `json:"historical_roots" hex:"true"`
	Eth1Data                     *Eth1DataJson               `json:"eth1_data"`
	Eth1DataVotes                []*Eth1DataJson             `json:"eth1_data_votes"`
	Eth1DepositIndex             string                      `json:"eth1_deposit_index"`
	Validators                   []*ValidatorJson            `json:"validators"`
	Balances                     []string                    `json:"balances"`
	RandaoMixes                  []string                    `json:"randao_mixes" hex:"true"`
	Slashings                    []string                    `json:"slashings"`
	PreviousEpochParticipation   EpochParticipation          `json:"previous_epoch_participation"`
	CurrentEpochParticipation    EpochParticipation          `json:"current_epoch_participation"`
	JustificationBits            string                      `json:"justification_bits" hex:"true"`
	PreviousJustifiedCheckpoint  *CheckpointJson             `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint   *CheckpointJson             `json:"current_justified_checkpoint"`
	FinalizedCheckpoint          *CheckpointJson             `json:"finalized_checkpoint"`
	InactivityScores             []string                    `json:"inactivity_scores"`
	CurrentSyncCommittee         *SyncCommitteeJson          `json:"current_sync_committee"`
	NextSyncCommittee            *SyncCommitteeJson          `json:"next_sync_committee"`
	LatestExecutionPayloadHeader *ExecutionPayloadHeaderJson `json:"latest_execution_payload_header"`
}

type BeaconStateContainerV2Json struct {
	Phase0State    *BeaconStateJson          `json:"phase0_state"`
	AltairState    *BeaconStateAltairJson    `json:"altair_state"`
	BellatrixState *BeaconStateBellatrixJson `json:"bellatrix_state"`
}

type ForkJson struct {
	PreviousVersion string `json:"previous_version" hex:"true"`
	CurrentVersion  string `json:"current_version" hex:"true"`
	Epoch           string `json:"epoch"`
}

type ValidatorContainerJson struct {
	Index     string         `json:"index"`
	Balance   string         `json:"balance"`
	Status    string         `json:"status" enum:"true"`
	Validator *ValidatorJson `json:"validator"`
}

type ValidatorJson struct {
	PublicKey                  string `json:"pubkey" hex:"true"`
	WithdrawalCredentials      string `json:"withdrawal_credentials" hex:"true"`
	EffectiveBalance           string `json:"effective_balance"`
	Slashed                    bool   `json:"slashed"`
	ActivationEligibilityEpoch string `json:"activation_eligibility_epoch"`
	ActivationEpoch            string `json:"activation_epoch"`
	ExitEpoch                  string `json:"exit_epoch"`
	WithdrawableEpoch          string `json:"withdrawable_epoch"`
}

type ValidatorBalanceJson struct {
	Index   string `json:"index"`
	Balance string `json:"balance"`
}

type CommitteeJson struct {
	Index      string   `json:"index"`
	Slot       string   `json:"slot"`
	Validators []string `json:"validators"`
}

type SyncCommitteeJson struct {
	Pubkeys         []string `json:"pubkeys" hex:"true"`
	AggregatePubkey string   `json:"aggregate_pubkey" hex:"true"`
}

type SyncCommitteeValidatorsJson struct {
	Validators          []string   `json:"validators"`
	ValidatorAggregates [][]string `json:"validator_aggregates"`
}

type PendingAttestationJson struct {
	AggregationBits string               `json:"aggregation_bits" hex:"true"`
	Data            *AttestationDataJson `json:"data"`
	InclusionDelay  string               `json:"inclusion_delay"`
	ProposerIndex   string               `json:"proposer_index"`
}

type ForkChoiceHeadJson struct {
	Root string `json:"root" hex:"true"`
	Slot string `json:"slot"`
}

type V2ForkChoiceHeadJson struct {
	Root                string `json:"root" hex:"true"`
	Slot                string `json:"slot"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
}

type DepositContractJson struct {
	ChainId string `json:"chain_id"`
	Address string `json:"address"`
}

type AttesterDutyJson struct {
	Pubkey                  string `json:"pubkey" hex:"true"`
	ValidatorIndex          string `json:"validator_index"`
	CommitteeIndex          string `json:"committee_index"`
	CommitteeLength         string `json:"committee_length"`
	CommitteesAtSlot        string `json:"committees_at_slot"`
	ValidatorCommitteeIndex string `json:"validator_committee_index"`
	Slot                    string `json:"slot"`
}

type ProposerDutyJson struct {
	Pubkey         string `json:"pubkey" hex:"true"`
	ValidatorIndex string `json:"validator_index"`
	Slot           string `json:"slot"`
}

type SyncCommitteeDuty struct {
	Pubkey                        string   `json:"pubkey" hex:"true"`
	ValidatorIndex                string   `json:"validator_index"`
	ValidatorSyncCommitteeIndices []string `json:"validator_sync_committee_indices"`
}

type SignedAggregateAttestationAndProofJson struct {
	Message   *AggregateAttestationAndProofJson `json:"message"`
	Signature string                            `json:"signature" hex:"true"`
}

type AggregateAttestationAndProofJson struct {
	AggregatorIndex string           `json:"aggregator_index"`
	Aggregate       *AttestationJson `json:"aggregate"`
	SelectionProof  string           `json:"selection_proof" hex:"true"`
}

type SignedContributionAndProofJson struct {
	Message   *ContributionAndProofJson `json:"message"`
	Signature string                    `json:"signature" hex:"true"`
}

type ContributionAndProofJson struct {
	AggregatorIndex string                         `json:"aggregator_index"`
	Contribution    *SyncCommitteeContributionJson `json:"contribution"`
	SelectionProof  string                         `json:"selection_proof" hex:"true"`
}

type SyncCommitteeContributionJson struct {
	Slot              string `json:"slot"`
	BeaconBlockRoot   string `json:"beacon_block_root" hex:"true"`
	SubcommitteeIndex string `json:"subcommittee_index"`
	AggregationBits   string `json:"aggregation_bits" hex:"true"`
	Signature         string `json:"signature" hex:"true"`
}

type ValidatorRegistrationJson struct {
	FeeRecipient string `json:"fee_recipient" hex:"true"`
	GasLimit     string `json:"gas_limit"`
	Timestamp    string `json:"timestamp"`
	Pubkey       string `json:"pubkey" hex:"true"`
}

type SignedValidatorRegistrationJson struct {
	Message   *ValidatorRegistrationJson `json:"message"`
	Signature string                     `json:"signature" hex:"true"`
}

type SignedValidatorRegistrationsRequestJson struct {
	Registrations []*SignedValidatorRegistrationJson `json:"registrations"`
}

type ForkChoiceNodeJson struct {
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

type SszRequestJson struct {
	Data string `json:"data"`
}

// SszResponse is a common abstraction over all SSZ responses.
type SszResponse interface {
	SSZVersion() string
	SSZOptimistic() bool
	SSZData() string
}

type SszResponseJson struct {
	Data string `json:"data"`
}

func (ssz *SszResponseJson) SSZData() string {
	return ssz.Data
}

func (*SszResponseJson) SSZVersion() string {
	return strings.ToLower(ethpbv2.Version_PHASE0.String())
}

func (*SszResponseJson) SSZOptimistic() bool {
	return false
}

type VersionedSSZResponseJson struct {
	Version             string `json:"version"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
	Data                string `json:"data"`
}

func (ssz *VersionedSSZResponseJson) SSZData() string {
	return ssz.Data
}

func (ssz *VersionedSSZResponseJson) SSZVersion() string {
	return ssz.Version
}

func (ssz *VersionedSSZResponseJson) SSZOptimistic() bool {
	return ssz.ExecutionOptimistic
}

// ---------------
// Events.
// ---------------

type EventHeadJson struct {
	Slot                      string `json:"slot"`
	Block                     string `json:"block" hex:"true"`
	State                     string `json:"state" hex:"true"`
	EpochTransition           bool   `json:"epoch_transition"`
	ExecutionOptimistic       bool   `json:"execution_optimistic"`
	PreviousDutyDependentRoot string `json:"previous_duty_dependent_root" hex:"true"`
	CurrentDutyDependentRoot  string `json:"current_duty_dependent_root" hex:"true"`
}

type ReceivedBlockDataJson struct {
	Slot                string `json:"slot"`
	Block               string `json:"block" hex:"true"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
}

type AggregatedAttReceivedDataJson struct {
	Aggregate *AttestationJson `json:"aggregate"`
}

type UnaggregatedAttReceivedDataJson struct {
	AggregationBits string               `json:"aggregation_bits" hex:"true"`
	Data            *AttestationDataJson `json:"data"`
	Signature       string               `json:"signature" hex:"true"`
}

type EventFinalizedCheckpointJson struct {
	Block               string `json:"block" hex:"true"`
	State               string `json:"state" hex:"true"`
	Epoch               string `json:"epoch"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
}

type EventChainReorgJson struct {
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

// IndexedVerificationFailureErrorJson is a JSON representation of the error returned when verifying an indexed object.
type IndexedVerificationFailureErrorJson struct {
	apimiddleware.DefaultErrorJson
	Failures []*SingleIndexedVerificationFailureJson `json:"failures"`
}

// SingleIndexedVerificationFailureJson is a JSON representation of a an issue when verifying a single indexed object e.g. an item in an array.
type SingleIndexedVerificationFailureJson struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

type NodeSyncDetailsErrorJson struct {
	apimiddleware.DefaultErrorJson
	SyncDetails helpers.SyncDetailsJson `json:"sync_details"`
}

type EventErrorJson struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}
