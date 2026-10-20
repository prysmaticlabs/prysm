package structs

import "encoding/json"

// ExecutionPayloadBidHeze is the Heze execution payload bid: the Gloas fields plus inclusion_list_bits.
type ExecutionPayloadBidHeze struct {
	ExecutionPayloadBid
	InclusionListBits string `json:"inclusion_list_bits"`
}

type SignedExecutionPayloadBidHeze struct {
	Message   *ExecutionPayloadBidHeze `json:"message"`
	Signature string                   `json:"signature"`
}

// BeaconBlockBodyHeze is the Gloas body without eth1_data and deposits (EIP-8015).
type BeaconBlockBodyHeze struct {
	RandaoReveal              string                         `json:"randao_reveal"`
	Graffiti                  string                         `json:"graffiti"`
	ProposerSlashings         []*ProposerSlashing            `json:"proposer_slashings"`
	AttesterSlashings         []*AttesterSlashingElectra     `json:"attester_slashings"`
	Attestations              []*AttestationElectra          `json:"attestations"`
	VoluntaryExits            []*SignedVoluntaryExit         `json:"voluntary_exits"`
	SyncAggregate             *SyncAggregate                 `json:"sync_aggregate"`
	BLSToExecutionChanges     []*SignedBLSToExecutionChange  `json:"bls_to_execution_changes"`
	SignedExecutionPayloadBid *SignedExecutionPayloadBidHeze `json:"signed_execution_payload_bid"`
	PayloadAttestations       []*PayloadAttestation          `json:"payload_attestations"`
	ParentExecutionRequests   *ExecutionRequestsGloas        `json:"parent_execution_requests"`
}

type BeaconBlockHeze struct {
	Slot          string               `json:"slot"`
	ProposerIndex string               `json:"proposer_index"`
	ParentRoot    string               `json:"parent_root"`
	StateRoot     string               `json:"state_root"`
	Body          *BeaconBlockBodyHeze `json:"body"`
}

type SignedBeaconBlockHeze struct {
	Message   *BeaconBlockHeze `json:"message"`
	Signature string           `json:"signature"`
}

var _ SignedMessageJsoner = &SignedBeaconBlockHeze{}

func (s *SignedBeaconBlockHeze) MessageRawJson() ([]byte, error) {
	return json.Marshal(s.Message)
}

func (s *SignedBeaconBlockHeze) SigString() string {
	return s.Signature
}

type BeaconStateHeze struct {
	GenesisTime                   string                      `json:"genesis_time"`
	GenesisValidatorsRoot         string                      `json:"genesis_validators_root"`
	Slot                          string                      `json:"slot"`
	Fork                          *Fork                       `json:"fork"`
	LatestBlockHeader             *BeaconBlockHeader          `json:"latest_block_header"`
	BlockRoots                    []string                    `json:"block_roots"`
	StateRoots                    []string                    `json:"state_roots"`
	HistoricalRoots               []string                    `json:"historical_roots"`
	Validators                    []*Validator                `json:"validators"`
	Balances                      []string                    `json:"balances"`
	RandaoMixes                   []string                    `json:"randao_mixes"`
	Slashings                     []string                    `json:"slashings"`
	PreviousEpochParticipation    []string                    `json:"previous_epoch_participation"`
	CurrentEpochParticipation     []string                    `json:"current_epoch_participation"`
	JustificationBits             string                      `json:"justification_bits"`
	PreviousJustifiedCheckpoint   *Checkpoint                 `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint    *Checkpoint                 `json:"current_justified_checkpoint"`
	FinalizedCheckpoint           *Checkpoint                 `json:"finalized_checkpoint"`
	InactivityScores              []string                    `json:"inactivity_scores"`
	CurrentSyncCommittee          *SyncCommittee              `json:"current_sync_committee"`
	NextSyncCommittee             *SyncCommittee              `json:"next_sync_committee"`
	NextWithdrawalIndex           string                      `json:"next_withdrawal_index"`
	NextWithdrawalValidatorIndex  string                      `json:"next_withdrawal_validator_index"`
	HistoricalSummaries           []*HistoricalSummary        `json:"historical_summaries"`
	DepositBalanceToConsume       string                      `json:"deposit_balance_to_consume"`
	ExitBalanceToConsume          string                      `json:"exit_balance_to_consume"`
	EarliestExitEpoch             string                      `json:"earliest_exit_epoch"`
	ConsolidationBalanceToConsume string                      `json:"consolidation_balance_to_consume"`
	EarliestConsolidationEpoch    string                      `json:"earliest_consolidation_epoch"`
	PendingDeposits               []*PendingDeposit           `json:"pending_deposits"`
	PendingPartialWithdrawals     []*PendingPartialWithdrawal `json:"pending_partial_withdrawals"`
	PendingConsolidations         []*PendingConsolidation     `json:"pending_consolidations"`
	ProposerLookahead             []string                    `json:"proposer_lookahead"`
	LatestExecutionPayloadBid     *ExecutionPayloadBidHeze    `json:"latest_execution_payload_bid"`
	Builders                      []*Builder                  `json:"builders"`
	NextWithdrawalBuilderIndex    string                      `json:"next_withdrawal_builder_index"`
	ExecutionPayloadAvailability  string                      `json:"execution_payload_availability"`
	BuilderPendingPayments        []*BuilderPendingPayment    `json:"builder_pending_payments"`
	BuilderPendingWithdrawals     []*BuilderPendingWithdrawal `json:"builder_pending_withdrawals"`
	LatestBlockHash               string                      `json:"latest_block_hash"`
	PayloadExpectedWithdrawals    []*Withdrawal               `json:"payload_expected_withdrawals"`
	PtcWindow                     [][]string                  `json:"ptc_window"`
}

type BlockContentsHeze struct {
	Block                    *BeaconBlockHeze          `json:"block"`
	ExecutionPayloadEnvelope *ExecutionPayloadEnvelope `json:"execution_payload_envelope"`
	KzgProofs                []string                  `json:"kzg_proofs"`
	Blobs                    []string                  `json:"blobs"`
}

// InclusionList is the Heze inclusion list.
type InclusionList struct {
	Slot           string   `json:"slot"`
	ValidatorIndex string   `json:"validator_index"`
	DependentRoot  string   `json:"dependent_root"`
	Transactions   []string `json:"transactions"`
}

// SignedInclusionList is the Heze inclusion list with the signature of the committee member.
type SignedInclusionList struct {
	Message   *InclusionList `json:"message"`
	Signature string         `json:"signature"`
}

// InclusionListDuty is one inclusion list committee assignment.
type InclusionListDuty struct {
	Pubkey         string `json:"pubkey"`
	ValidatorIndex string `json:"validator_index"`
	Slot           string `json:"slot"`
}

// GetInclusionListDutiesResponse is the response of the inclusion list committee duties endpoint.
type GetInclusionListDutiesResponse struct {
	DependentRoot       string               `json:"dependent_root"`
	ExecutionOptimistic bool                 `json:"execution_optimistic"`
	Data                []*InclusionListDuty `json:"data"`
}

// ProduceInclusionListResponse is the response of the produce inclusion list endpoint.
type ProduceInclusionListResponse struct {
	Data []string `json:"data"`
}

// PublishInclusionListRequest is the request body of the publish inclusion list endpoint.
type PublishInclusionListRequest struct {
	Data *SignedInclusionList `json:"data"`
}

// InclusionListEvent is the data of the inclusion_list event.
type InclusionListEvent struct {
	Version string               `json:"version"`
	Data    *SignedInclusionList `json:"data"`
}
