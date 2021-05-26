package gateway

// GenesisResponseJson is used in /beacon/genesis API endpoint.
type GenesisResponseJson struct {
	Data *GenesisResponse_GenesisJson `json:"data"`
}

// GenesisResponse_GenesisJson is used in /beacon/genesis API endpoint.
type GenesisResponse_GenesisJson struct {
	GenesisTime           string `json:"genesis_time" time:"true"`
	GenesisValidatorsRoot string `json:"genesis_validators_root" hex:"true"`
	GenesisForkVersion    string `json:"genesis_fork_version" hex:"true"`
}

// StateRootResponseJson is used in /beacon/states/{state_id}/root API endpoint.
type StateRootResponseJson struct {
	Data *StateRootResponse_StateRootJson `json:"data"`
}

// StateRootResponse_StateRootJson is used in /beacon/states/{state_id}/root API endpoint.
type StateRootResponse_StateRootJson struct {
	StateRoot string `json:"root" hex:"true"`
}

// StateForkResponseJson is used in /beacon/states/{state_id}/fork API endpoint.
type StateForkResponseJson struct {
	Data *ForkJson `json:"data"`
}

// StateFinalityCheckpointResponseJson is used in /beacon/states/{state_id}/finality_checkpoints API endpoint.
type StateFinalityCheckpointResponseJson struct {
	Data *StateFinalityCheckpointResponse_StateFinalityCheckpointJson `json:"data"`
}

// StateFinalityCheckpointResponse_StateFinalityCheckpointJson is used in /beacon/states/{state_id}/finality_checkpoints API endpoint.
type StateFinalityCheckpointResponse_StateFinalityCheckpointJson struct {
	PreviousJustified *CheckpointJson `json:"previous_justified"`
	CurrentJustified  *CheckpointJson `json:"current_justified"`
	Finalized         *CheckpointJson `json:"finalized"`
}

// StateValidatorsResponseJson is used in /beacon/states/{state_id}/validators API endpoint.
type StateValidatorsResponseJson struct {
	Data []*ValidatorContainerJson `json:"data"`
}

// StateValidatorResponseJson is used in /beacon/states/{state_id}/validators/{validator_id} API endpoint.
type StateValidatorResponseJson struct {
	Data *ValidatorContainerJson `json:"data"`
}

// ValidatorBalancesResponseJson is used in /beacon/states/{state_id}/validator_balances API endpoint.
type ValidatorBalancesResponseJson struct {
	Data []*ValidatorBalanceJson `json:"data"`
}

// StateCommitteesResponseJson is used in /beacon/states/{state_id}/committees API endpoint.
type StateCommitteesResponseJson struct {
	Data []*CommitteeJson `json:"data"`
}

// BlockHeaderResponseJson is used in /beacon/headers/{block_id} API endpoint.
type BlockHeaderResponseJson struct {
	Data *BlockHeaderContainerJson `json:"data"`
}

// BlockResponseJson is used in /beacon/blocks/{block_id} API endpoint.
type BlockResponseJson struct {
	Data *BeaconBlockContainerJson `json:"data"`
}

// BlockRootResponseJson is used in /beacon/blocks/{block_id}/root API endpoint.
type BlockRootResponseJson struct {
	Data *BlockRootContainerJson `json:"data"`
}

// BlockAttestationsResponseJson is used in /beacon/blocks/{block_id}/attestations API endpoint.
type BlockAttestationsResponseJson struct {
	Data []*AttestationJson `json:"data"`
}

// AttestationsPoolResponseJson is used in /beacon/pool/attestations GET API endpoint.
type AttestationsPoolResponseJson struct {
	Data []*AttestationJson `json:"data"`
}

// SubmitAttestationRequestJson is used in /beacon/pool/attestations POST API endpoint.
type SubmitAttestationRequestJson struct {
	Data []*AttestationJson `json:"data"`
}

// AttesterSlashingsPoolResponseJson is used in /beacon/pool/attester_slashings API endpoint.
type AttesterSlashingsPoolResponseJson struct {
	Data []*AttesterSlashingJson `json:"data"`
}

// ProposerSlashingsPoolResponseJson is used in /beacon/pool/proposer_slashings API endpoint.
type ProposerSlashingsPoolResponseJson struct {
	Data []*ProposerSlashingJson `json:"data"`
}

// VoluntaryExitsPoolResponseJson is used in /beacon/pool/voluntary_exits API endpoint.
type VoluntaryExitsPoolResponseJson struct {
	Data []*SignedVoluntaryExitJson `json:"data"`
}

// IdentityResponseJson is used in /node/identity API endpoint.
type IdentityResponseJson struct {
	Data *IdentityJson `json:"data"`
}

// PeersResponseJson is used in /node/peers API endpoint.
type PeersResponseJson struct {
	Data []*PeerJson `json:"data"`
}

// PeerResponseJson is used in /node/peers/{peer_id} API endpoint.
type PeerResponseJson struct {
	Data *PeerJson `json:"data"`
}

// PeerCountResponseJson is used in /node/peer_count API endpoint.
type PeerCountResponseJson struct {
	Data PeerCountResponse_PeerCountJson `json:"data"`
}

// PeerCountResponse_PeerCountJson is used in /node/peer_count API endpoint.
type PeerCountResponse_PeerCountJson struct {
	Disconnected  string `json:"disconnected"`
	Connecting    string `json:"connecting"`
	Connected     string `json:"connected"`
	Disconnecting string `json:"disconnecting"`
}

// VersionResponseJson is used in /node/version API endpoint.
type VersionResponseJson struct {
	Data *VersionJson `json:"data"`
}

// SyncingResponseJson is used in /node/syncing API endpoint.
type SyncingResponseJson struct {
	Data *SyncInfoJson `json:"data"`
}

// BeaconStateResponseJson is used in /debug/beacon/states/{state_id} API endpoint.
type BeaconStateResponseJson struct {
	Data *BeaconStateJson `json:"data"`
}

// BeaconStateResponseJson is used in /debug/beacon/states/{state_id} API endpoint.
type BeaconStateSszResponseJson struct {
	Data string `json:"data"`
}

// ForkChoiceHeadsResponseJson is used in /debug/beacon/heads API endpoint.
type ForkChoiceHeadsResponseJson struct {
	Data []*ForkChoiceHeadJson `json:"data"`
}

// ForkScheduleResponseJson is used in /config/fork_schedule API endpoint.
type ForkScheduleResponseJson struct {
	Data []*ForkJson `json:"data"`
}

// DepositContractResponseJson is used in /config/deposit_contract API endpoint.
type DepositContractResponseJson struct {
	Data *DepositContractJson `json:"data"`
}

// SpecResponseJson is used in /config/spec API endpoint.
type SpecResponseJson struct {
	Data interface{} `json:"data"`
}

//----------------
// Reusable types.
//----------------

// CheckpointJson is a JSON representation of a checkpoint.
type CheckpointJson struct {
	Epoch string `json:"epoch"`
	Root  string `json:"root" hex:"true"`
}

// BlockRootContainerJson is a JSON representation of a block root container.
type BlockRootContainerJson struct {
	Root string `json:"root" hex:"true"`
}

// BeaconBlockContainerJson is a JSON representation of a beacon block container.
type BeaconBlockContainerJson struct {
	Message   *BeaconBlockJson `json:"message"`
	Signature string           `json:"signature" hex:"true"`
}

// BeaconBlockJson is a JSON representation of a beacon block.
type BeaconBlockJson struct {
	Slot          string               `json:"slot"`
	ProposerIndex string               `json:"proposer_index"`
	ParentRoot    string               `json:"parent_root" hex:"true"`
	StateRoot     string               `json:"state_root" hex:"true"`
	Body          *BeaconBlockBodyJson `json:"body"`
}

// BeaconBlockBodyJson is a JSON representation of a beacon block body.
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

// BlockHeaderContainerJson is a JSON representation of a block header container.
type BlockHeaderContainerJson struct {
	Root      string                          `json:"root" hex:"true"`
	Canonical bool                            `json:"canonical"`
	Header    *BeaconBlockHeaderContainerJson `json:"header"`
}

// BeaconBlockHeaderContainerJson is a JSON representation of a beacon block header container.
type BeaconBlockHeaderContainerJson struct {
	Message   *BeaconBlockHeaderJson `json:"message"`
	Signature string                 `json:"signature" hex:"true"`
}

// SignedBeaconBlockHeaderJson is a JSON representation of a signed beacon block header.
type SignedBeaconBlockHeaderJson struct {
	Header    *BeaconBlockHeaderJson `json:"message"`
	Signature string                 `json:"signature" hex:"true"`
}

// BeaconBlockHeaderJson is a JSON representation of a beacon block header.
type BeaconBlockHeaderJson struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root" hex:"true"`
	StateRoot     string `json:"state_root" hex:"true"`
	BodyRoot      string `json:"body_root" hex:"true"`
}

// Eth1DataJson is a JSON representation of eth1data.
type Eth1DataJson struct {
	DepositRoot  string `json:"deposit_root" hex:"true"`
	DepositCount string `json:"deposit_count"`
	BlockHash    string `json:"block_hash" hex:"true"`
}

// ProposerSlashingJson is a JSON representation of a proposer slashing.
type ProposerSlashingJson struct {
	Header_1 *SignedBeaconBlockHeaderJson `json:"signed_header_1"`
	Header_2 *SignedBeaconBlockHeaderJson `json:"signed_header_2"`
}

// AttesterSlashingJson is a JSON representation of an attester slashing.
type AttesterSlashingJson struct {
	Attestation_1 *IndexedAttestationJson `json:"attestation_1"`
	Attestation_2 *IndexedAttestationJson `json:"attestation_2"`
}

// IndexedAttestationJson is a JSON representation of an indexed attestation.
type IndexedAttestationJson struct {
	AttestingIndices []string             `json:"attesting_indices"`
	Data             *AttestationDataJson `json:"data"`
	Signature        string               `json:"signature" hex:"true"`
}

// AttestationJson is a JSON representation of an attestation.
type AttestationJson struct {
	AggregationBits string               `json:"aggregation_bits" hex:"true"`
	Data            *AttestationDataJson `json:"data"`
	Signature       string               `json:"signature" hex:"true"`
}

// AttestationDataJson is a JSON representation of attestation data.
type AttestationDataJson struct {
	Slot            string          `json:"slot"`
	CommitteeIndex  string          `json:"index"`
	BeaconBlockRoot string          `json:"beacon_block_root" hex:"true"`
	Source          *CheckpointJson `json:"source"`
	Target          *CheckpointJson `json:"target"`
}

// DepositJson is a JSON representation of a deposit.
type DepositJson struct {
	Proof []string          `json:"proof" hex:"true"`
	Data  *Deposit_DataJson `json:"data"`
}

// Deposit_DataJson is a JSON representation of deposit data.
type Deposit_DataJson struct {
	PublicKey             string `json:"pubkey" hex:"true"`
	WithdrawalCredentials string `json:"withdrawal_credentials" hex:"true"`
	Amount                string `json:"amount"`
	Signature             string `json:"signature" hex:"true"`
}

// SignedVoluntaryExitJson is a JSON representation of a signed voluntary exit.
type SignedVoluntaryExitJson struct {
	Exit      *VoluntaryExitJson `json:"message"`
	Signature string             `json:"signature" hex:"true"`
}

// VoluntaryExitJson is a JSON representation of a voluntary exit.
type VoluntaryExitJson struct {
	Epoch          string `json:"epoch"`
	ValidatorIndex string `json:"validator_index"`
}

// IdentityJson is a JSON representation of a peer's identity.
type IdentityJson struct {
	PeerId             string        `json:"peer_id"`
	Enr                string        `json:"enr"`
	P2PAddresses       []string      `json:"p2p_addresses"`
	DiscoveryAddresses []string      `json:"discovery_addresses"`
	Metadata           *MetadataJson `json:"metadata"`
}

// MetadataJson is a JSON representation of p2p metadata.
type MetadataJson struct {
	SeqNumber string `json:"seq_number"`
	Attnets   string `json:"attnets" hex:"true"`
}

// PeerJson is a JSON representation of a peer.
type PeerJson struct {
	PeerId    string `json:"peer_id"`
	Enr       string `json:"enr"`
	Address   string `json:"last_seen_p2p_address"`
	State     string `json:"state" enum:"true"`
	Direction string `json:"direction" enum:"true"`
}

// VersionJson is a JSON representation of the system's version.
type VersionJson struct {
	Version string `json:"version"`
}

// BeaconStateJson is a JSON representation of the beacon state.
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

// ForkJson is a JSON representation of a fork.
type ForkJson struct {
	PreviousVersion string `json:"previous_version" hex:"true"`
	CurrentVersion  string `json:"current_version" hex:"true"`
	Epoch           string `json:"epoch"`
}

type StateValidatorsRequestJson struct {
	StateId string   `json:"state_id" hex:"true"`
	Id      []string `json:"id" hex:"true"`
	Status  []string `json:"status" enum:"true"`
}

// ValidatorContainerJson is a JSON representation of a validator container.
type ValidatorContainerJson struct {
	Index     string         `json:"index"`
	Balance   string         `json:"balance"`
	Status    string         `json:"status" enum:"true"`
	Validator *ValidatorJson `json:"validator"`
}

// ValidatorJson is a JSON representation of a validator.
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

// ValidatorBalanceJson is a JSON representation of a validator's balance.
type ValidatorBalanceJson struct {
	Index   string `json:"index"`
	Balance string `json:"balance"`
}

// CommitteeJson is a JSON representation of a committee
type CommitteeJson struct {
	Index      string   `json:"index"`
	Slot       string   `json:"slot"`
	Validators []string `json:"validators"`
}

// PendingAttestationJson is a JSON representation of a pending attestation.
type PendingAttestationJson struct {
	AggregationBits string               `json:"aggregation_bits" hex:"true"`
	Data            *AttestationDataJson `json:"data"`
	InclusionDelay  string               `json:"inclusion_delay"`
	ProposerIndex   string               `json:"proposer_index"`
}

// ForkChoiceHeadJson is a JSON representation of a fork choice head.
type ForkChoiceHeadJson struct {
	Root string `json:"root" hex:"true"`
	Slot string `json:"slot"`
}

// DepositContractJson is a JSON representation of the deposit contract.
type DepositContractJson struct {
	ChainId string `json:"chain_id"`
	Address string `json:"address"`
}

// SyncInfoJson is a JSON representation of the sync info.
type SyncInfoJson struct {
	HeadSlot     string `json:"head_slot"`
	SyncDistance string `json:"sync_distance"`
	IsSyncing    bool   `json:"is_syncing"`
}

// ---------------
// Error handling.
// ---------------

// ErrorJson describes common functionality of all JSON error representations.
type ErrorJson interface {
	StatusCode() int
	SetCode(code int)
	Msg() string
}

// DefaultErrorJson is a JSON representation of a simple error value, containing only a message and an error code.
type DefaultErrorJson struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// SubmitAttestationsErrorJson is a JSON representation of the error returned when submitting attestations.
type SubmitAttestationsErrorJson struct {
	DefaultErrorJson
	Failures []*SingleAttestationVerificationFailureJson `json:"failures"`
}

// SingleAttestationVerificationFailureJson is a JSON representation of a failure when verifying a single submitted attestation.
type SingleAttestationVerificationFailureJson struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

// StatusCode returns the error's underlying error code.
func (e *DefaultErrorJson) StatusCode() int {
	return e.Code
}

// Msg returns the error's underlying message.
func (e *DefaultErrorJson) Msg() string {
	return e.Message
}

// SetCode sets the error's underlying error code.
func (e *DefaultErrorJson) SetCode(code int) {
	e.Code = code
}
