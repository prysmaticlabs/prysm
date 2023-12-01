package shared

import (
	"github.com/pkg/errors"
)

var errPayloadHeaderNotFound = errors.New("expected payload header not found")

type BeaconState struct {
	GenesisTime                 string                `json:"genesis_time"`
	GenesisValidatorsRoot       string                `json:"genesis_validators_root"`
	Slot                        string                `json:"slot"`
	Fork                        *Fork                 `json:"fork"`
	LatestBlockHeader           *BeaconBlockHeader    `json:"latest_block_header"`
	BlockRoots                  []string              `json:"block_roots"`
	StateRoots                  []string              `json:"state_roots"`
	HistoricalRoots             []string              `json:"historical_roots"`
	Eth1Data                    *Eth1Data             `json:"eth1_data"`
	Eth1DataVotes               []*Eth1Data           `json:"eth1_data_votes"`
	Eth1DepositIndex            string                `json:"eth1_deposit_index"`
	Validators                  []*Validator          `json:"validators"`
	Balances                    []string              `json:"balances"`
	RandaoMixes                 []string              `json:"randao_mixes"`
	Slashings                   []string              `json:"slashings"`
	PreviousEpochAttestations   []*PendingAttestation `json:"previous_epoch_attestations"`
	CurrentEpochAttestations    []*PendingAttestation `json:"current_epoch_attestations"`
	JustificationBits           string                `json:"justification_bits"`
	PreviousJustifiedCheckpoint *Checkpoint           `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint  *Checkpoint           `json:"current_justified_checkpoint"`
	FinalizedCheckpoint         *Checkpoint           `json:"finalized_checkpoint"`
}

type BeaconStateAltair struct {
	GenesisTime                 string             `json:"genesis_time"`
	GenesisValidatorsRoot       string             `json:"genesis_validators_root"`
	Slot                        string             `json:"slot"`
	Fork                        *Fork              `json:"fork"`
	LatestBlockHeader           *BeaconBlockHeader `json:"latest_block_header"`
	BlockRoots                  []string           `json:"block_roots"`
	StateRoots                  []string           `json:"state_roots"`
	HistoricalRoots             []string           `json:"historical_roots"`
	Eth1Data                    *Eth1Data          `json:"eth1_data"`
	Eth1DataVotes               []*Eth1Data        `json:"eth1_data_votes"`
	Eth1DepositIndex            string             `json:"eth1_deposit_index"`
	Validators                  []*Validator       `json:"validators"`
	Balances                    []string           `json:"balances"`
	RandaoMixes                 []string           `json:"randao_mixes"`
	Slashings                   []string           `json:"slashings"`
	PreviousEpochParticipation  []string           `json:"previous_epoch_participation"`
	CurrentEpochParticipation   []string           `json:"current_epoch_participation"`
	JustificationBits           string             `json:"justification_bits"`
	PreviousJustifiedCheckpoint *Checkpoint        `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint  *Checkpoint        `json:"current_justified_checkpoint"`
	FinalizedCheckpoint         *Checkpoint        `json:"finalized_checkpoint"`
	InactivityScores            []string           `json:"inactivity_scores"`
	CurrentSyncCommittee        *SyncCommittee     `json:"current_sync_committee"`
	NextSyncCommittee           *SyncCommittee     `json:"next_sync_committee"`
}

type BeaconStateBellatrix struct {
	GenesisTime                  string                  `json:"genesis_time"`
	GenesisValidatorsRoot        string                  `json:"genesis_validators_root"`
	Slot                         string                  `json:"slot"`
	Fork                         *Fork                   `json:"fork"`
	LatestBlockHeader            *BeaconBlockHeader      `json:"latest_block_header"`
	BlockRoots                   []string                `json:"block_roots"`
	StateRoots                   []string                `json:"state_roots"`
	HistoricalRoots              []string                `json:"historical_roots"`
	Eth1Data                     *Eth1Data               `json:"eth1_data"`
	Eth1DataVotes                []*Eth1Data             `json:"eth1_data_votes"`
	Eth1DepositIndex             string                  `json:"eth1_deposit_index"`
	Validators                   []*Validator            `json:"validators"`
	Balances                     []string                `json:"balances"`
	RandaoMixes                  []string                `json:"randao_mixes"`
	Slashings                    []string                `json:"slashings"`
	PreviousEpochParticipation   []string                `json:"previous_epoch_participation"`
	CurrentEpochParticipation    []string                `json:"current_epoch_participation"`
	JustificationBits            string                  `json:"justification_bits"`
	PreviousJustifiedCheckpoint  *Checkpoint             `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint   *Checkpoint             `json:"current_justified_checkpoint"`
	FinalizedCheckpoint          *Checkpoint             `json:"finalized_checkpoint"`
	InactivityScores             []string                `json:"inactivity_scores"`
	CurrentSyncCommittee         *SyncCommittee          `json:"current_sync_committee"`
	NextSyncCommittee            *SyncCommittee          `json:"next_sync_committee"`
	LatestExecutionPayloadHeader *ExecutionPayloadHeader `json:"latest_execution_payload_header"`
}

type BeaconStateCapella struct {
	GenesisTime                  string                         `json:"genesis_time"`
	GenesisValidatorsRoot        string                         `json:"genesis_validators_root"`
	Slot                         string                         `json:"slot"`
	Fork                         *Fork                          `json:"fork"`
	LatestBlockHeader            *BeaconBlockHeader             `json:"latest_block_header"`
	BlockRoots                   []string                       `json:"block_roots"`
	StateRoots                   []string                       `json:"state_roots"`
	HistoricalRoots              []string                       `json:"historical_roots"`
	Eth1Data                     *Eth1Data                      `json:"eth1_data"`
	Eth1DataVotes                []*Eth1Data                    `json:"eth1_data_votes"`
	Eth1DepositIndex             string                         `json:"eth1_deposit_index"`
	Validators                   []*Validator                   `json:"validators"`
	Balances                     []string                       `json:"balances"`
	RandaoMixes                  []string                       `json:"randao_mixes"`
	Slashings                    []string                       `json:"slashings"`
	PreviousEpochParticipation   []string                       `json:"previous_epoch_participation"`
	CurrentEpochParticipation    []string                       `json:"current_epoch_participation"`
	JustificationBits            string                         `json:"justification_bits"`
	PreviousJustifiedCheckpoint  *Checkpoint                    `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint   *Checkpoint                    `json:"current_justified_checkpoint"`
	FinalizedCheckpoint          *Checkpoint                    `json:"finalized_checkpoint"`
	InactivityScores             []string                       `json:"inactivity_scores"`
	CurrentSyncCommittee         *SyncCommittee                 `json:"current_sync_committee"`
	NextSyncCommittee            *SyncCommittee                 `json:"next_sync_committee"`
	LatestExecutionPayloadHeader *ExecutionPayloadHeaderCapella `json:"latest_execution_payload_header"`
	NextWithdrawalIndex          string                         `json:"next_withdrawal_index"`
	NextWithdrawalValidatorIndex string                         `json:"next_withdrawal_validator_index"`
	HistoricalSummaries          []*HistoricalSummary           `json:"historical_summaries"`
}

type BeaconStateDeneb struct {
	GenesisTime                  string                       `json:"genesis_time"`
	GenesisValidatorsRoot        string                       `json:"genesis_validators_root"`
	Slot                         string                       `json:"slot"`
	Fork                         *Fork                        `json:"fork"`
	LatestBlockHeader            *BeaconBlockHeader           `json:"latest_block_header"`
	BlockRoots                   []string                     `json:"block_roots"`
	StateRoots                   []string                     `json:"state_roots"`
	HistoricalRoots              []string                     `json:"historical_roots"`
	Eth1Data                     *Eth1Data                    `json:"eth1_data"`
	Eth1DataVotes                []*Eth1Data                  `json:"eth1_data_votes"`
	Eth1DepositIndex             string                       `json:"eth1_deposit_index"`
	Validators                   []*Validator                 `json:"validators"`
	Balances                     []string                     `json:"balances"`
	RandaoMixes                  []string                     `json:"randao_mixes"`
	Slashings                    []string                     `json:"slashings"`
	PreviousEpochParticipation   []string                     `json:"previous_epoch_participation"`
	CurrentEpochParticipation    []string                     `json:"current_epoch_participation"`
	JustificationBits            string                       `json:"justification_bits"`
	PreviousJustifiedCheckpoint  *Checkpoint                  `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint   *Checkpoint                  `json:"current_justified_checkpoint"`
	FinalizedCheckpoint          *Checkpoint                  `json:"finalized_checkpoint"`
	InactivityScores             []string                     `json:"inactivity_scores"`
	CurrentSyncCommittee         *SyncCommittee               `json:"current_sync_committee"`
	NextSyncCommittee            *SyncCommittee               `json:"next_sync_committee"`
	LatestExecutionPayloadHeader *ExecutionPayloadHeaderDeneb `json:"latest_execution_payload_header"`
	NextWithdrawalIndex          string                       `json:"next_withdrawal_index"`
	NextWithdrawalValidatorIndex string                       `json:"next_withdrawal_validator_index"`
	HistoricalSummaries          []*HistoricalSummary         `json:"historical_summaries"`
}

type Validator struct {
	PublicKey                  string `json:"pubkey"`
	WithdrawalCredentials      string `json:"withdrawal_credentials"`
	EffectiveBalance           string `json:"effective_balance"`
	Slashed                    bool   `json:"slashed"`
	ActivationEligibilityEpoch string `json:"activation_eligibility_epoch"`
	ActivationEpoch            string `json:"activation_epoch"`
	ExitEpoch                  string `json:"exit_epoch"`
	WithdrawableEpoch          string `json:"withdrawable_epoch"`
}

type PendingAttestation struct {
	AggregationBits string           `json:"aggregation_bits"`
	Data            *AttestationData `json:"data"`
	InclusionDelay  string           `json:"inclusion_delay"`
	ProposerIndex   string           `json:"proposer_index"`
}

type HistoricalSummary struct {
	BlockSummaryRoot string `json:"block_summary_root"`
	StateSummaryRoot string `json:"state_summary_root"`
}

type Attestation struct {
	AggregationBits string           `json:"aggregation_bits"`
	Data            *AttestationData `json:"data"`
	Signature       string           `json:"signature"`
}

type AttestationData struct {
	Slot            string      `json:"slot"`
	CommitteeIndex  string      `json:"index"`
	BeaconBlockRoot string      `json:"beacon_block_root"`
	Source          *Checkpoint `json:"source"`
	Target          *Checkpoint `json:"target"`
}

type Checkpoint struct {
	Epoch string `json:"epoch"`
	Root  string `json:"root"`
}

type Committee struct {
	Index      string   `json:"index"`
	Slot       string   `json:"slot"`
	Validators []string `json:"validators"`
}

type SignedContributionAndProof struct {
	Message   *ContributionAndProof `json:"message"`
	Signature string                `json:"signature"`
}

type ContributionAndProof struct {
	AggregatorIndex string                     `json:"aggregator_index"`
	Contribution    *SyncCommitteeContribution `json:"contribution"`
	SelectionProof  string                     `json:"selection_proof"`
}

type SyncCommitteeContribution struct {
	Slot              string `json:"slot"`
	BeaconBlockRoot   string `json:"beacon_block_root"`
	SubcommitteeIndex string `json:"subcommittee_index"`
	AggregationBits   string `json:"aggregation_bits"`
	Signature         string `json:"signature"`
}

type SignedAggregateAttestationAndProof struct {
	Message   *AggregateAttestationAndProof `json:"message"`
	Signature string                        `json:"signature"`
}

type AggregateAttestationAndProof struct {
	AggregatorIndex string       `json:"aggregator_index"`
	Aggregate       *Attestation `json:"aggregate"`
	SelectionProof  string       `json:"selection_proof"`
}

type SyncCommitteeSubscription struct {
	ValidatorIndex       string   `json:"validator_index"`
	SyncCommitteeIndices []string `json:"sync_committee_indices"`
	UntilEpoch           string   `json:"until_epoch"`
}

type BeaconCommitteeSubscription struct {
	ValidatorIndex   string `json:"validator_index"`
	CommitteeIndex   string `json:"committee_index"`
	CommitteesAtSlot string `json:"committees_at_slot"`
	Slot             string `json:"slot"`
	IsAggregator     bool   `json:"is_aggregator"`
}

type ValidatorRegistration struct {
	FeeRecipient string `json:"fee_recipient"`
	GasLimit     string `json:"gas_limit"`
	Timestamp    string `json:"timestamp"`
	Pubkey       string `json:"pubkey"`
}

type SignedValidatorRegistration struct {
	Message   *ValidatorRegistration `json:"message"`
	Signature string                 `json:"signature"`
}

type FeeRecipient struct {
	ValidatorIndex string `json:"validator_index"`
	FeeRecipient   string `json:"fee_recipient"`
}

type SignedVoluntaryExit struct {
	Message   *VoluntaryExit `json:"message"`
	Signature string         `json:"signature"`
}

type VoluntaryExit struct {
	Epoch          string `json:"epoch"`
	ValidatorIndex string `json:"validator_index"`
}

type Fork struct {
	PreviousVersion string `json:"previous_version"`
	CurrentVersion  string `json:"current_version"`
	Epoch           string `json:"epoch"`
}

type SignedBLSToExecutionChange struct {
	Message   *BLSToExecutionChange `json:"message"`
	Signature string                `json:"signature"`
}

type BLSToExecutionChange struct {
	ValidatorIndex     string `json:"validator_index"`
	FromBLSPubkey      string `json:"from_bls_pubkey"`
	ToExecutionAddress string `json:"to_execution_address"`
}

type SyncCommitteeMessage struct {
	Slot            string `json:"slot"`
	BeaconBlockRoot string `json:"beacon_block_root"`
	ValidatorIndex  string `json:"validator_index"`
	Signature       string `json:"signature"`
}

type SyncCommittee struct {
	Pubkeys         []string `json:"pubkeys"`
	AggregatePubkey string   `json:"aggregate_pubkey"`
}

// SyncDetails contains information about node sync status.
type SyncDetails struct {
	HeadSlot     string `json:"head_slot"`
	SyncDistance string `json:"sync_distance"`
	IsSyncing    bool   `json:"is_syncing"`
	IsOptimistic bool   `json:"is_optimistic"`
	ElOffline    bool   `json:"el_offline"`
}

// SyncDetailsContainer is a wrapper for Data.
type SyncDetailsContainer struct {
	Data *SyncDetails `json:"data"`
}

type SignedBeaconBlock struct {
	Message   *BeaconBlock `json:"message"`
	Signature string       `json:"signature"`
}

type BeaconBlock struct {
	Slot          string           `json:"slot"`
	ProposerIndex string           `json:"proposer_index"`
	ParentRoot    string           `json:"parent_root"`
	StateRoot     string           `json:"state_root"`
	Body          *BeaconBlockBody `json:"body"`
}

type BeaconBlockBody struct {
	RandaoReveal      string                 `json:"randao_reveal"`
	Eth1Data          *Eth1Data              `json:"eth1_data"`
	Graffiti          string                 `json:"graffiti"`
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings"`
	Attestations      []*Attestation         `json:"attestations"`
	Deposits          []*Deposit             `json:"deposits"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits"`
}

type SignedBeaconBlockAltair struct {
	Message   *BeaconBlockAltair `json:"message"`
	Signature string             `json:"signature"`
}

type BeaconBlockAltair struct {
	Slot          string                 `json:"slot"`
	ProposerIndex string                 `json:"proposer_index"`
	ParentRoot    string                 `json:"parent_root"`
	StateRoot     string                 `json:"state_root"`
	Body          *BeaconBlockBodyAltair `json:"body"`
}

type BeaconBlockBodyAltair struct {
	RandaoReveal      string                 `json:"randao_reveal"`
	Eth1Data          *Eth1Data              `json:"eth1_data"`
	Graffiti          string                 `json:"graffiti"`
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings"`
	Attestations      []*Attestation         `json:"attestations"`
	Deposits          []*Deposit             `json:"deposits"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits"`
	SyncAggregate     *SyncAggregate         `json:"sync_aggregate"`
}

type SignedBeaconBlockBellatrix struct {
	Message   *BeaconBlockBellatrix `json:"message"`
	Signature string                `json:"signature"`
}

type BeaconBlockBellatrix struct {
	Slot          string                    `json:"slot"`
	ProposerIndex string                    `json:"proposer_index"`
	ParentRoot    string                    `json:"parent_root"`
	StateRoot     string                    `json:"state_root"`
	Body          *BeaconBlockBodyBellatrix `json:"body"`
}

type BeaconBlockBodyBellatrix struct {
	RandaoReveal      string                 `json:"randao_reveal"`
	Eth1Data          *Eth1Data              `json:"eth1_data"`
	Graffiti          string                 `json:"graffiti"`
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings"`
	Attestations      []*Attestation         `json:"attestations"`
	Deposits          []*Deposit             `json:"deposits"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits"`
	SyncAggregate     *SyncAggregate         `json:"sync_aggregate"`
	ExecutionPayload  *ExecutionPayload      `json:"execution_payload"`
}

type SignedBlindedBeaconBlockBellatrix struct {
	Message   *BlindedBeaconBlockBellatrix `json:"message"`
	Signature string                       `json:"signature"`
}

type BlindedBeaconBlockBellatrix struct {
	Slot          string                           `json:"slot"`
	ProposerIndex string                           `json:"proposer_index"`
	ParentRoot    string                           `json:"parent_root"`
	StateRoot     string                           `json:"state_root"`
	Body          *BlindedBeaconBlockBodyBellatrix `json:"body"`
}

type BlindedBeaconBlockBodyBellatrix struct {
	RandaoReveal           string                  `json:"randao_reveal"`
	Eth1Data               *Eth1Data               `json:"eth1_data"`
	Graffiti               string                  `json:"graffiti"`
	ProposerSlashings      []*ProposerSlashing     `json:"proposer_slashings"`
	AttesterSlashings      []*AttesterSlashing     `json:"attester_slashings"`
	Attestations           []*Attestation          `json:"attestations"`
	Deposits               []*Deposit              `json:"deposits"`
	VoluntaryExits         []*SignedVoluntaryExit  `json:"voluntary_exits"`
	SyncAggregate          *SyncAggregate          `json:"sync_aggregate"`
	ExecutionPayloadHeader *ExecutionPayloadHeader `json:"execution_payload_header"`
}

type SignedBeaconBlockCapella struct {
	Message   *BeaconBlockCapella `json:"message"`
	Signature string              `json:"signature"`
}

type BeaconBlockCapella struct {
	Slot          string                  `json:"slot"`
	ProposerIndex string                  `json:"proposer_index"`
	ParentRoot    string                  `json:"parent_root"`
	StateRoot     string                  `json:"state_root"`
	Body          *BeaconBlockBodyCapella `json:"body"`
}

type BeaconBlockBodyCapella struct {
	RandaoReveal          string                        `json:"randao_reveal"`
	Eth1Data              *Eth1Data                     `json:"eth1_data"`
	Graffiti              string                        `json:"graffiti"`
	ProposerSlashings     []*ProposerSlashing           `json:"proposer_slashings"`
	AttesterSlashings     []*AttesterSlashing           `json:"attester_slashings"`
	Attestations          []*Attestation                `json:"attestations"`
	Deposits              []*Deposit                    `json:"deposits"`
	VoluntaryExits        []*SignedVoluntaryExit        `json:"voluntary_exits"`
	SyncAggregate         *SyncAggregate                `json:"sync_aggregate"`
	ExecutionPayload      *ExecutionPayloadCapella      `json:"execution_payload"`
	BLSToExecutionChanges []*SignedBLSToExecutionChange `json:"bls_to_execution_changes"`
}

type SignedBlindedBeaconBlockCapella struct {
	Message   *BlindedBeaconBlockCapella `json:"message"`
	Signature string                     `json:"signature"`
}

type BlindedBeaconBlockCapella struct {
	Slot          string                         `json:"slot"`
	ProposerIndex string                         `json:"proposer_index"`
	ParentRoot    string                         `json:"parent_root"`
	StateRoot     string                         `json:"state_root"`
	Body          *BlindedBeaconBlockBodyCapella `json:"body"`
}

type BlindedBeaconBlockBodyCapella struct {
	RandaoReveal           string                         `json:"randao_reveal"`
	Eth1Data               *Eth1Data                      `json:"eth1_data"`
	Graffiti               string                         `json:"graffiti"`
	ProposerSlashings      []*ProposerSlashing            `json:"proposer_slashings"`
	AttesterSlashings      []*AttesterSlashing            `json:"attester_slashings"`
	Attestations           []*Attestation                 `json:"attestations"`
	Deposits               []*Deposit                     `json:"deposits"`
	VoluntaryExits         []*SignedVoluntaryExit         `json:"voluntary_exits"`
	SyncAggregate          *SyncAggregate                 `json:"sync_aggregate"`
	ExecutionPayloadHeader *ExecutionPayloadHeaderCapella `json:"execution_payload_header"`
	BLSToExecutionChanges  []*SignedBLSToExecutionChange  `json:"bls_to_execution_changes"`
}

type SignedBeaconBlockContentsDeneb struct {
	SignedBlock        *SignedBeaconBlockDeneb `json:"signed_block"`
	SignedBlobSidecars []*SignedBlobSidecar    `json:"signed_blob_sidecars" `
}

type BeaconBlockContentsDeneb struct {
	Block        *BeaconBlockDeneb `json:"block"`
	BlobSidecars []*BlobSidecar    `json:"blob_sidecars"`
}

type SignedBeaconBlockDeneb struct {
	Message   *BeaconBlockDeneb `json:"message"`
	Signature string            `json:"signature"`
}

type BeaconBlockDeneb struct {
	Slot          string                `json:"slot"`
	ProposerIndex string                `json:"proposer_index"`
	ParentRoot    string                `json:"parent_root"`
	StateRoot     string                `json:"state_root"`
	Body          *BeaconBlockBodyDeneb `json:"body"`
}

type BeaconBlockBodyDeneb struct {
	RandaoReveal          string                        `json:"randao_reveal"`
	Eth1Data              *Eth1Data                     `json:"eth1_data"`
	Graffiti              string                        `json:"graffiti"`
	ProposerSlashings     []*ProposerSlashing           `json:"proposer_slashings"`
	AttesterSlashings     []*AttesterSlashing           `json:"attester_slashings"`
	Attestations          []*Attestation                `json:"attestations"`
	Deposits              []*Deposit                    `json:"deposits"`
	VoluntaryExits        []*SignedVoluntaryExit        `json:"voluntary_exits"`
	SyncAggregate         *SyncAggregate                `json:"sync_aggregate"`
	ExecutionPayload      *ExecutionPayloadDeneb        `json:"execution_payload"`
	BLSToExecutionChanges []*SignedBLSToExecutionChange `json:"bls_to_execution_changes"`
	BlobKzgCommitments    []string                      `json:"blob_kzg_commitments"`
}

type ExecutionPayloadDeneb struct {
	ParentHash    string        `json:"parent_hash"`
	FeeRecipient  string        `json:"fee_recipient"`
	StateRoot     string        `json:"state_root"`
	ReceiptsRoot  string        `json:"receipts_root"`
	LogsBloom     string        `json:"logs_bloom"`
	PrevRandao    string        `json:"prev_randao"`
	BlockNumber   string        `json:"block_number"`
	GasLimit      string        `json:"gas_limit"`
	GasUsed       string        `json:"gas_used"`
	Timestamp     string        `json:"timestamp"`
	ExtraData     string        `json:"extra_data"`
	BaseFeePerGas string        `json:"base_fee_per_gas"`
	BlobGasUsed   string        `json:"blob_gas_used"`
	ExcessBlobGas string        `json:"excess_blob_gas"`
	BlockHash     string        `json:"block_hash"`
	Transactions  []string      `json:"transactions"`
	Withdrawals   []*Withdrawal `json:"withdrawals"`
}

type SignedBlindedBeaconBlockContentsDeneb struct {
	SignedBlindedBlock        *SignedBlindedBeaconBlockDeneb `json:"signed_blinded_block"`
	SignedBlindedBlobSidecars []*SignedBlindedBlobSidecar    `json:"signed_blinded_blob_sidecars"`
}

type BlindedBeaconBlockContentsDeneb struct {
	BlindedBlock        *BlindedBeaconBlockDeneb `json:"blinded_block"`
	BlindedBlobSidecars []*BlindedBlobSidecar    `json:"blinded_blob_sidecars"`
}

type BlindedBeaconBlockDeneb struct {
	Slot          string                       `json:"slot"`
	ProposerIndex string                       `json:"proposer_index"`
	ParentRoot    string                       `json:"parent_root"`
	StateRoot     string                       `json:"state_root"`
	Body          *BlindedBeaconBlockBodyDeneb `json:"body"`
}

type SignedBlindedBeaconBlockDeneb struct {
	Message   *BlindedBeaconBlockDeneb `json:"message"`
	Signature string                   `json:"signature"`
}

type BlindedBeaconBlockBodyDeneb struct {
	RandaoReveal           string                        `json:"randao_reveal"`
	Eth1Data               *Eth1Data                     `json:"eth1_data"`
	Graffiti               string                        `json:"graffiti"`
	ProposerSlashings      []*ProposerSlashing           `json:"proposer_slashings"`
	AttesterSlashings      []*AttesterSlashing           `json:"attester_slashings"`
	Attestations           []*Attestation                `json:"attestations"`
	Deposits               []*Deposit                    `json:"deposits"`
	VoluntaryExits         []*SignedVoluntaryExit        `json:"voluntary_exits"`
	SyncAggregate          *SyncAggregate                `json:"sync_aggregate"`
	ExecutionPayloadHeader *ExecutionPayloadHeaderDeneb  `json:"execution_payload_header"`
	BLSToExecutionChanges  []*SignedBLSToExecutionChange `json:"bls_to_execution_changes"`
	BlobKzgCommitments     []string                      `json:"blob_kzg_commitments"`
}

type SignedBlindedBlobSidecar struct {
	Message   *BlindedBlobSidecar `json:"message"`
	Signature string              `json:"signature"`
}

type SignedBlobSidecar struct {
	Message   *BlobSidecar `json:"message"`
	Signature string       `json:"signature"`
}

type BlindedBlobSidecar struct {
	BlockRoot       string `json:"block_root"`
	Index           string `json:"index"`
	Slot            string `json:"slot"`
	BlockParentRoot string `json:"block_parent_root"`
	ProposerIndex   string `json:"proposer_index"`
	BlobRoot        string `json:"blob_root"`
	KzgCommitment   string `json:"kzg_commitment"` // pattern: "^0x[a-fA-F0-9]{96}$" ssz-size:"48"
	KzgProof        string `json:"kzg_proof"`      // pattern: "^0x[a-fA-F0-9]{96}$" ssz-size:"48"
}

type BlobSidecar struct {
	BlockRoot       string `json:"block_root"`
	Index           string `json:"index"`
	Slot            string `json:"slot"`
	BlockParentRoot string `json:"block_parent_root"`
	ProposerIndex   string `json:"proposer_index"`
	Blob            string `json:"blob"`           // pattern: "^0x[a-fA-F0-9]{262144}$"
	KzgCommitment   string `json:"kzg_commitment"` // pattern: "^0x[a-fA-F0-9]{96}$" ssz-size:"48"
	KzgProof        string `json:"kzg_proof"`      // pattern: "^0x[a-fA-F0-9]{96}$" ssz-size:"48"
}

type Eth1Data struct {
	DepositRoot  string `json:"deposit_root"`
	DepositCount string `json:"deposit_count"`
	BlockHash    string `json:"block_hash"`
}

type ProposerSlashing struct {
	SignedHeader1 *SignedBeaconBlockHeader `json:"signed_header_1"`
	SignedHeader2 *SignedBeaconBlockHeader `json:"signed_header_2"`
}

type AttesterSlashing struct {
	Attestation1 *IndexedAttestation `json:"attestation_1"`
	Attestation2 *IndexedAttestation `json:"attestation_2"`
}

type Deposit struct {
	Proof []string     `json:"proof"`
	Data  *DepositData `json:"data"`
}

type DepositData struct {
	Pubkey                string `json:"pubkey"`
	WithdrawalCredentials string `json:"withdrawal_credentials"`
	Amount                string `json:"amount"`
	Signature             string `json:"signature"`
}

type SignedBeaconBlockHeaderContainer struct {
	Header    *SignedBeaconBlockHeader `json:"header"`
	Root      string                   `json:"root"`
	Canonical bool                     `json:"canonical"`
}

type SignedBeaconBlockHeader struct {
	Message   *BeaconBlockHeader `json:"message"`
	Signature string             `json:"signature"`
}

type BeaconBlockHeader struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root"`
	StateRoot     string `json:"state_root"`
	BodyRoot      string `json:"body_root"`
}

type IndexedAttestation struct {
	AttestingIndices []string         `json:"attesting_indices"`
	Data             *AttestationData `json:"data"`
	Signature        string           `json:"signature"`
}

type SyncAggregate struct {
	SyncCommitteeBits      string `json:"sync_committee_bits"`
	SyncCommitteeSignature string `json:"sync_committee_signature"`
}

type ExecutionPayload struct {
	ParentHash    string   `json:"parent_hash"`
	FeeRecipient  string   `json:"fee_recipient"`
	StateRoot     string   `json:"state_root"`
	ReceiptsRoot  string   `json:"receipts_root"`
	LogsBloom     string   `json:"logs_bloom"`
	PrevRandao    string   `json:"prev_randao"`
	BlockNumber   string   `json:"block_number"`
	GasLimit      string   `json:"gas_limit"`
	GasUsed       string   `json:"gas_used"`
	Timestamp     string   `json:"timestamp"`
	ExtraData     string   `json:"extra_data"`
	BaseFeePerGas string   `json:"base_fee_per_gas"`
	BlockHash     string   `json:"block_hash"`
	Transactions  []string `json:"transactions"`
}

type ExecutionPayloadHeader struct {
	ParentHash       string `json:"parent_hash"`
	FeeRecipient     string `json:"fee_recipient"`
	StateRoot        string `json:"state_root"`
	ReceiptsRoot     string `json:"receipts_root"`
	LogsBloom        string `json:"logs_bloom"`
	PrevRandao       string `json:"prev_randao"`
	BlockNumber      string `json:"block_number"`
	GasLimit         string `json:"gas_limit"`
	GasUsed          string `json:"gas_used"`
	Timestamp        string `json:"timestamp"`
	ExtraData        string `json:"extra_data"`
	BaseFeePerGas    string `json:"base_fee_per_gas"`
	BlockHash        string `json:"block_hash"`
	TransactionsRoot string `json:"transactions_root"`
}

type ExecutionPayloadCapella struct {
	ParentHash    string        `json:"parent_hash"`
	FeeRecipient  string        `json:"fee_recipient"`
	StateRoot     string        `json:"state_root"`
	ReceiptsRoot  string        `json:"receipts_root"`
	LogsBloom     string        `json:"logs_bloom"`
	PrevRandao    string        `json:"prev_randao"`
	BlockNumber   string        `json:"block_number"`
	GasLimit      string        `json:"gas_limit"`
	GasUsed       string        `json:"gas_used"`
	Timestamp     string        `json:"timestamp"`
	ExtraData     string        `json:"extra_data"`
	BaseFeePerGas string        `json:"base_fee_per_gas"`
	BlockHash     string        `json:"block_hash"`
	Transactions  []string      `json:"transactions"`
	Withdrawals   []*Withdrawal `json:"withdrawals"`
}

type ExecutionPayloadHeaderCapella struct {
	ParentHash       string `json:"parent_hash"`
	FeeRecipient     string `json:"fee_recipient"`
	StateRoot        string `json:"state_root"`
	ReceiptsRoot     string `json:"receipts_root"`
	LogsBloom        string `json:"logs_bloom"`
	PrevRandao       string `json:"prev_randao"`
	BlockNumber      string `json:"block_number"`
	GasLimit         string `json:"gas_limit"`
	GasUsed          string `json:"gas_used"`
	Timestamp        string `json:"timestamp"`
	ExtraData        string `json:"extra_data"`
	BaseFeePerGas    string `json:"base_fee_per_gas"`
	BlockHash        string `json:"block_hash"`
	TransactionsRoot string `json:"transactions_root"`
	WithdrawalsRoot  string `json:"withdrawals_root"`
}

type ExecutionPayloadHeaderDeneb struct {
	ParentHash       string `json:"parent_hash"`
	FeeRecipient     string `json:"fee_recipient"`
	StateRoot        string `json:"state_root"`
	ReceiptsRoot     string `json:"receipts_root"`
	LogsBloom        string `json:"logs_bloom"`
	PrevRandao       string `json:"prev_randao"`
	BlockNumber      string `json:"block_number"`
	GasLimit         string `json:"gas_limit"`
	GasUsed          string `json:"gas_used"`
	Timestamp        string `json:"timestamp"`
	ExtraData        string `json:"extra_data"`
	BaseFeePerGas    string `json:"base_fee_per_gas"`
	BlobGasUsed      string `json:"blob_gas_used"`
	ExcessBlobGas    string `json:"excess_blob_gas"`
	BlockHash        string `json:"block_hash"`
	TransactionsRoot string `json:"transactions_root"`
	WithdrawalsRoot  string `json:"withdrawals_root"`
}

type Withdrawal struct {
	WithdrawalIndex  string `json:"index"`
	ValidatorIndex   string `json:"validator_index"`
	ExecutionAddress string `json:"address"`
	Amount           string `json:"amount"`
}
