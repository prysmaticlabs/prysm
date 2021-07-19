package apimiddleware

import "github.com/prysmaticlabs/prysm/shared/gateway"

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
	Data *beaconBlockContainerJson `json:"data"`
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

// attesterDutiesRequestJson is used in /validator/duties/attester/{epoch} API endpoint.
type attesterDutiesRequestJson struct {
	Index []string `json:"index"`
}

// attesterDutiesResponseJson is used in /validator/duties/attester/{epoch} API endpoint.
type attesterDutiesResponseJson struct {
	DependentRoot string              `json:"dependent_root" hex:"true"`
	AttesterDuty  []*attesterDutyJson `json:"data"`
}

//----------------
// Reusable types.
//----------------

// checkpointJson is a JSON representation of a checkpoint.
type checkpointJson struct {
	Epoch string `json:"epoch"`
	Root  string `json:"root" hex:"true"`
}

// blockRootContainerJson is a JSON representation of a block root container.
type blockRootContainerJson struct {
	Root string `json:"root" hex:"true"`
}

// beaconBlockContainerJson is a JSON representation of a beacon block container.
type beaconBlockContainerJson struct {
	Message   *beaconBlockJson `json:"message"`
	Signature string           `json:"signature" hex:"true"`
}

// beaconBlockJson is a JSON representation of a beacon block.
type beaconBlockJson struct {
	Slot          string               `json:"slot"`
	ProposerIndex string               `json:"proposer_index"`
	ParentRoot    string               `json:"parent_root" hex:"true"`
	StateRoot     string               `json:"state_root" hex:"true"`
	Body          *beaconBlockBodyJson `json:"body"`
}

// beaconBlockBodyJson is a JSON representation of a beacon block body.
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

// blockHeaderContainerJson is a JSON representation of a block header container.
type blockHeaderContainerJson struct {
	Root      string                          `json:"root" hex:"true"`
	Canonical bool                            `json:"canonical"`
	Header    *beaconBlockHeaderContainerJson `json:"header"`
}

// beaconBlockHeaderContainerJson is a JSON representation of a beacon block header container.
type beaconBlockHeaderContainerJson struct {
	Message   *beaconBlockHeaderJson `json:"message"`
	Signature string                 `json:"signature" hex:"true"`
}

// signedBeaconBlockHeaderJson is a JSON representation of a signed beacon block header.
type signedBeaconBlockHeaderJson struct {
	Header    *beaconBlockHeaderJson `json:"message"`
	Signature string                 `json:"signature" hex:"true"`
}

// beaconBlockHeaderJson is a JSON representation of a beacon block header.
type beaconBlockHeaderJson struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root" hex:"true"`
	StateRoot     string `json:"state_root" hex:"true"`
	BodyRoot      string `json:"body_root" hex:"true"`
}

// eth1DataJson is a JSON representation of eth1data.
type eth1DataJson struct {
	DepositRoot  string `json:"deposit_root" hex:"true"`
	DepositCount string `json:"deposit_count"`
	BlockHash    string `json:"block_hash" hex:"true"`
}

// proposerSlashingJson is a JSON representation of a proposer slashing.
type proposerSlashingJson struct {
	Header_1 *signedBeaconBlockHeaderJson `json:"signed_header_1"`
	Header_2 *signedBeaconBlockHeaderJson `json:"signed_header_2"`
}

// attesterSlashingJson is a JSON representation of an attester slashing.
type attesterSlashingJson struct {
	Attestation_1 *indexedAttestationJson `json:"attestation_1"`
	Attestation_2 *indexedAttestationJson `json:"attestation_2"`
}

// indexedAttestationJson is a JSON representation of an indexed attestation.
type indexedAttestationJson struct {
	AttestingIndices []string             `json:"attesting_indices"`
	Data             *attestationDataJson `json:"data"`
	Signature        string               `json:"signature" hex:"true"`
}

// attestationJson is a JSON representation of an attestation.
type attestationJson struct {
	AggregationBits string               `json:"aggregation_bits" hex:"true"`
	Data            *attestationDataJson `json:"data"`
	Signature       string               `json:"signature" hex:"true"`
}

// attestationDataJson is a JSON representation of attestation data.
type attestationDataJson struct {
	Slot            string          `json:"slot"`
	CommitteeIndex  string          `json:"index"`
	BeaconBlockRoot string          `json:"beacon_block_root" hex:"true"`
	Source          *checkpointJson `json:"source"`
	Target          *checkpointJson `json:"target"`
}

// depositJson is a JSON representation of a deposit.
type depositJson struct {
	Proof []string          `json:"proof" hex:"true"`
	Data  *deposit_DataJson `json:"data"`
}

// deposit_DataJson is a JSON representation of deposit data.
type deposit_DataJson struct {
	PublicKey             string `json:"pubkey" hex:"true"`
	WithdrawalCredentials string `json:"withdrawal_credentials" hex:"true"`
	Amount                string `json:"amount"`
	Signature             string `json:"signature" hex:"true"`
}

// signedVoluntaryExitJson is a JSON representation of a signed voluntary exit.
type signedVoluntaryExitJson struct {
	Exit      *voluntaryExitJson `json:"message"`
	Signature string             `json:"signature" hex:"true"`
}

// voluntaryExitJson is a JSON representation of a voluntary exit.
type voluntaryExitJson struct {
	Epoch          string `json:"epoch"`
	ValidatorIndex string `json:"validator_index"`
}

// identityJson is a JSON representation of a peer's identity.
type identityJson struct {
	PeerId             string        `json:"peer_id"`
	Enr                string        `json:"enr"`
	P2PAddresses       []string      `json:"p2p_addresses"`
	DiscoveryAddresses []string      `json:"discovery_addresses"`
	Metadata           *metadataJson `json:"metadata"`
}

// metadataJson is a JSON representation of p2p metadata.
type metadataJson struct {
	SeqNumber string `json:"seq_number"`
	Attnets   string `json:"attnets" hex:"true"`
}

// peerJson is a JSON representation of a peer.
type peerJson struct {
	PeerId    string `json:"peer_id"`
	Enr       string `json:"enr"`
	Address   string `json:"last_seen_p2p_address"`
	State     string `json:"state" enum:"true"`
	Direction string `json:"direction" enum:"true"`
}

// versionJson is a JSON representation of the system's version.
type versionJson struct {
	Version string `json:"version"`
}

// beaconStateJson is a JSON representation of the beacon state.
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

// forkJson is a JSON representation of a fork.
type forkJson struct {
	PreviousVersion string `json:"previous_version" hex:"true"`
	CurrentVersion  string `json:"current_version" hex:"true"`
	Epoch           string `json:"epoch"`
}

// validatorContainerJson is a JSON representation of a validator container.
type validatorContainerJson struct {
	Index     string         `json:"index"`
	Balance   string         `json:"balance"`
	Status    string         `json:"status" enum:"true"`
	Validator *validatorJson `json:"validator"`
}

// validatorJson is a JSON representation of a validator.
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

// validatorBalanceJson is a JSON representation of a validator's balance.
type validatorBalanceJson struct {
	Index   string `json:"index"`
	Balance string `json:"balance"`
}

// committeeJson is a JSON representation of a committee
type committeeJson struct {
	Index      string   `json:"index"`
	Slot       string   `json:"slot"`
	Validators []string `json:"validators"`
}

// pendingAttestationJson is a JSON representation of a pending attestation.
type pendingAttestationJson struct {
	AggregationBits string               `json:"aggregation_bits" hex:"true"`
	Data            *attestationDataJson `json:"data"`
	InclusionDelay  string               `json:"inclusion_delay"`
	ProposerIndex   string               `json:"proposer_index"`
}

// forkChoiceHeadJson is a JSON representation of a fork choice head.
type forkChoiceHeadJson struct {
	Root string `json:"root" hex:"true"`
	Slot string `json:"slot"`
}

// depositContractJson is a JSON representation of the deposit contract.
type depositContractJson struct {
	ChainId string `json:"chain_id"`
	Address string `json:"address"`
}

// syncInfoJson is a JSON representation of the sync info.
type syncInfoJson struct {
	HeadSlot     string `json:"head_slot"`
	SyncDistance string `json:"sync_distance"`
	IsSyncing    bool   `json:"is_syncing"`
}

// attesterDutyJson is a JSON representation of an attester duty.
type attesterDutyJson struct {
	Pubkey                  string `json:"pubkey" hex:"true"`
	ValidatorIndex          string `json:"validator_index"`
	CommitteeIndex          string `json:"committee_index"`
	CommitteeLength         string `json:"committee_length"`
	CommitteesAtSlot        string `json:"committees_at_slot"`
	ValidatorCommitteeIndex string `json:"validator_committee_index"`
	Slot                    string `json:"slot"`
}

//----------------
// SSZ
// ---------------

// sszResponseJson is a common abstraction over all SSZ responses.
type sszResponseJson interface {
	SSZData() string
}

// blockSSZResponseJson is used in /beacon/blocks/{block_id} API endpoint.
type blockSSZResponseJson struct {
	Data string `json:"data"`
}

func (ssz *blockSSZResponseJson) SSZData() string {
	return ssz.Data
}

// beaconStateSSZResponseJson is used in /debug/beacon/states/{state_id} API endpoint.
type beaconStateSSZResponseJson struct {
	Data string `json:"data"`
}

func (ssz *beaconStateSSZResponseJson) SSZData() string {
	return ssz.Data
}

// TODO: Documentation
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

// submitAttestationsErrorJson is a JSON representation of the error returned when submitting attestations.
type submitAttestationsErrorJson struct {
	gateway.DefaultErrorJson
	Failures []*singleAttestationVerificationFailureJson `json:"failures"`
}

// singleAttestationVerificationFailureJson is a JSON representation of a failure when verifying a single submitted attestation.
type singleAttestationVerificationFailureJson struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

type eventErrorJson struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}
