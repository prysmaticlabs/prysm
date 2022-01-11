package v1

/* Web3Signer Specs are found on conensys website: google Consensys' Web3Signer API specification*/
// All Types represent version 1.0 of the Web3Signer specs i.e. /api/v1/eth2

// AggregationSlotSignRequest is a request object for web3signer sign api.
type AggregationSlotSignRequest struct {
	Type            string           `json:"type"`
	ForkInfo        *ForkInfo        `json:"fork_info"`
	SigningRoot     string           `json:"signingRoot"`
	AggregationSlot *AggregationSlot `json:"aggregation_slot"`
}

// AggregationSlotSignRequest is a request object for web3signer sign api.
type AggregateAndProofSignRequest struct {
	Type              string             `json:"type"`
	ForkInfo          *ForkInfo          `json:"fork_info"`
	SigningRoot       string             `json:"signingRoot"`
	AggregateAndProof *AggregateAndProof `json:"aggregation_and_proof"`
}

// AttestationSignRequest is a request object for web3signer sign api.
type AttestationSignRequest struct {
	Type        string           `json:"type"`
	ForkInfo    *ForkInfo        `json:"fork_info"`
	SigningRoot string           `json:"signingRoot"`
	Attestation *AttestationData `json:"attestation"`
}

// BlockSignRequest is a request object for web3signer sign api.
type BlockSignRequest struct {
	Type        string       `json:"type"`
	ForkInfo    *ForkInfo    `json:"fork_info"`
	SigningRoot string       `json:"signingRoot"`
	Block       *BeaconBlock `json:"block"`
}

// BlockV2AltairSignRequest is a request object for web3signer sign api.
type BlockV2AltairSignRequest struct {
	Type        string                    `json:"type"`
	ForkInfo    *ForkInfo                 `json:"fork_info"`
	SigningRoot string                    `json:"signingRoot"`
	BeaconBlock *BeaconBlockAltairBlockV2 `json:"beacon_block"`
}

// BlockV2SignRequest is a request object for web3signer sign api.
type BlockV2SignRequest struct {
	Type        string              `json:"type"`
	ForkInfo    *ForkInfo           `json:"fork_info"`
	SigningRoot string              `json:"signingRoot"`
	BeaconBlock *BeaconBlockBlockV2 `json:"beacon_block"`
}

// DepositSignRequest Not currently supported by Prysm.
// DepositSignRequest is a request object for web3signer sign api.

// RandaoRevealSignRequest is a request object for web3signer sign api.
type RandaoRevealSignRequest struct {
	Type         string        `json:"type"`
	ForkInfo     *ForkInfo     `json:"fork_info"`
	SigningRoot  string        `json:"signingRoot"`
	RandaoReveal *RandaoReveal `json:"randao_reveal"`
}

// VoluntaryExitSignRequest is a request object for web3signer sign api.
type VoluntaryExitSignRequest struct {
	Type          string         `json:"type"`
	ForkInfo      *ForkInfo      `json:"fork_info"`
	SigningRoot   string         `json:"signingRoot"`
	VoluntaryExit *VoluntaryExit `json:"voluntary_exit"`
}

// SyncCommitteeMessageSignRequest is a request object for web3signer sign api.
type SyncCommitteeMessageSignRequest struct {
	Type          string                `json:"type"`
	ForkInfo      *ForkInfo             `json:"fork_info"`
	SigningRoot   string                `json:"signingRoot"`
	SyncCommittee *SyncCommitteeMessage `json:"sync_committee_message"`
}

// SyncCommitteeSelectionProofSignRequest is a request object for web3signer sign api.
type SyncCommitteeSelectionProofSignRequest struct {
	Type          string                       `json:"type"`
	ForkInfo      *ForkInfo                    `json:"fork_info"`
	SigningRoot   string                       `json:"signingRoot"`
	SyncCommittee *SyncAggregatorSelectionData `json:"sync_committee_selection_proof"`
}

// SyncCommitteeContributionAndProofSignRequest is a request object for web3signer sign api.
type SyncCommitteeContributionAndProofSignRequest struct {
	Type          string                `json:"type"`
	ForkInfo      *ForkInfo             `json:"fork_info"`
	SigningRoot   string                `json:"signingRoot"`
	SyncCommittee *ContributionAndProof `json:"sync_committee_contribution_and_proof"`
}

////////////////////////////////////////////////////////////////////////////////
// sub properties of Sign Requests /////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

// ForkInfo a sub property object of the Sign request
type ForkInfo struct {
	Fork                  *Fork  `json:"fork"`
	GenesisValidatorsRoot string `json:"genesis_validators_root"`
}

// Fork a sub property of ForkInfo.
type Fork struct {
	PreviousVersion string `json:"previous_version"`
	CurrentVersion  string `json:"current_version"`
	Epoch           string `json:"epoch"`
}

// AggregationSlot a sub property of AggregationSlotSignRequest.
type AggregationSlot struct {
	Slot string `json:"slot"`
}

// AggregateAndProof a sub property of AggregateAndProofSignRequest.
type AggregateAndProof struct {
	AggregatorIndex string       `json:"aggregator_index"` /* uint64 */
	Aggregate       *Attestation `json:"aggregate"`
	SelectionProof  string       `json:"selection_proof"` /* 96 bytes */
}

// Attestation a sub property of AggregateAndProofSignRequest.
type Attestation struct {
	AggregationBits string           `json:"aggregation_bits"`
	Data            *AttestationData `json:"data"`
	Signature       string           `json:"signature"`
}

// AttestationData a sub property of Attestation.
type AttestationData struct {
	Slot            string      `json:"slot"`  /* uint64 */
	Index           string      `json:"index"` /* uint64 */ // Prysm uses CommitteeIndex but web3signer uses index.
	BeaconBlockRoot string      `json:"beacon_block_root"`
	Source          *Checkpoint `json:"source"`
	Target          *Checkpoint `json:"target"`
}

// Checkpoint a sub property of AttestationData.
type Checkpoint struct {
	Epoch string `json:"epoch"`
	Root  string `json:"root"`
}

// BeaconBlock a sub property of BeaconBlockBlockV2.
type BeaconBlock struct {
	Slot          string           `json:"slot"`           /* uint64 */
	ProposerIndex string           `json:"proposer_index"` /* uint64 */
	ParentRoot    string           `json:"parent_root"`
	StateRoot     string           `json:"state_root"`
	Body          *BeaconBlockBody `json:"body"`
}

// BeaconBlockBody a sub property of BeaconBlock.
type BeaconBlockBody struct {
	RandaoReveal      string                 `json:"randao_reveal"`
	Eth1Data          *Eth1Data              `json:"eth1_data"`
	Graffiti          string                 `json:"graffiti"` // 32 bytes
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings"`
	Attestations      []*Attestation         `json:"attestations"`
	Deposits          []*Deposit             `json:"deposits"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits"`
}

// Eth1Data a sub property of BeaconBlockBody.
type Eth1Data struct {
	DepositRoot  string `json:"deposit_root"`
	DepositCount string `json:"deposit_count"` /* uint64 */
	BlockHash    string `json:"block_hash"`
}

// ProposerSlashing a sub property of BeaconBlockBody.
type ProposerSlashing struct {
	// Prysm uses Header_1 but web3signer uses signed_header_1.
	SignedHeader_1 *SignedBeaconBlockHeader `json:"signed_header_1"`
	// Prysm uses Header_2 but web3signer uses signed_header_2.
	SignedHeader_2 *SignedBeaconBlockHeader `json:"signed_header_2"`
}

// SignedBeaconBlockHeader is a sub property of ProposerSlashing.
type SignedBeaconBlockHeader struct {
	Message   *BeaconBlockHeader `json:"message"`
	Signature string             `json:"signature"`
}

// BeaconBlockHeader is a sub property of SignedBeaconBlockHeader.
type BeaconBlockHeader struct {
	Slot          string `json:"slot"`           /* uint64 */
	ProposerIndex string `json:"proposer_index"` /* uint64 */
	ParentRoot    string `json:"parent_root"`    /* Hash32 */
	StateRoot     string `json:"state_root"`     /* Hash32 */
	BodyRoot      string `json:"body_root"`      /* Hash32 */
}

// AttesterSlashing a sub property of BeaconBlockBody.
type AttesterSlashing struct {
	Attestation_1 *IndexedAttestation `json:"attestation_1"`
	Attestation_2 *IndexedAttestation `json:"attestation_2"`
}

// IndexedAttestation a sub property of AttesterSlashing.
type IndexedAttestation struct {
	AttestingIndices []string         `json:"attesting_indices"` /* uint64[] */
	Data             *AttestationData `json:"data"`
	Signature        string           `json:"signature"`
}

// Deposit a sub property of DepositSignRequest.
type Deposit struct {
	Proof []string     `json:"proof"`
	Data  *DepositData `json:"data"`
}

// DepositData a sub property of Deposit.
// DepositData :Prysm uses Deposit_data instead of DepositData which is inconsistent naming
type DepositData struct {
	PublicKey             string `json:"public_key"`
	WithdrawalCredentials string `json:"withdrawal_credentials"`
	Amount                string `json:"amount"` /* uint64 */
	Signature             string `json:"signature"`
}

// SignedVoluntaryExit is a sub property of BeaconBlockBody.
type SignedVoluntaryExit struct {
	// Prysm uses Exit instead of Message
	Message   *VoluntaryExit `json:"message"`
	Signature string         `json:"signature"`
}

// VoluntaryExit a sub property of SignedVoluntaryExit.
type VoluntaryExit struct {
	Epoch          string `json:"epoch"`           /* uint64 */
	ValidatorIndex string `json:"validator_index"` /* uint64 */
}

// BeaconBlockAltairBlockV2 a sub property of BlockV2AltairSignRequest.
type BeaconBlockAltairBlockV2 struct {
	Version string             `json:"version"`
	Block   *BeaconBlockAltair `json:"beacon_block"`
}

// BeaconBlockAltair a sub property of BeaconBlockAltairBlockV2.
type BeaconBlockAltair struct {
	Slot          string                 `json:"slot"`
	ProposerIndex string                 `json:"proposer_index"`
	ParentRoot    string                 `json:"parent_root"`
	StateRoot     string                 `json:"state_root"`
	Body          *BeaconBlockBodyAltair `json:"body"`
}

// BeaconBlockBodyAltair a sub property of BeaconBlockAltair.
type BeaconBlockBodyAltair struct {
	RandaoReveal      string                 `json:"randao_reveal"`
	Eth1Data          *Eth1Data              `json:"eth1_data"`
	Graffiti          string                 `json:"graffiti"` /* Hash32 */
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings"`
	Attestations      []*Attestation         `json:"attestations"`
	Deposits          []*Deposit             `json:"deposits"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits"`
	SyncAggregate     *SyncAggregate         `json:"sync_aggregate"`
}

// SyncAggregate is a sub property of BeaconBlockBodyAltair.
type SyncAggregate struct {
	SyncCommitteeBits      string `json:"sync_committee_bits"`      /* SSZ hexadecimal string */
	SyncCommitteeSignature string `json:"sync_committee_signature"` /* 96 byte hexadecimal string */
}

// BeaconBlockBlockV2 a sub property of BlockV2SignRequest.
type BeaconBlockBlockV2 struct {
	Version string       `json:"version"`
	Block   *BeaconBlock `json:"beacon_block"`
}

// RandaoReveal a sub property of RandaoRevealSignRequest.
type RandaoReveal struct {
	Epoch string `json:"epoch"` /* uint64 */
}

// SyncCommitteeMessage a sub property of SyncCommitteeSignRequest.
type SyncCommitteeMessage struct {
	BeaconBlockRoot string `json:"beacon_block_root"` /* Hash32 */
	Slot            string `json:"slot"`              /* uint64 */
	// Prysm uses BlockRoot instead of BeaconBlockRoot and has the following extra properties : ValidatorIndex, Signature
}

// SyncAggregatorSelectionData a sub property of SyncAggregatorSelectionSignRequest.
type SyncAggregatorSelectionData struct {
	Slot              string `json:"slot"`               /* uint64 */
	SubcommitteeIndex string `json:"subcommittee_index"` /* uint64 */
}

// ContributionAndProof a sub property of AggregatorSelectionSignRequest.
type ContributionAndProof struct {
	AggregatorIndex string                     `json:"aggregator_index"` /* uint64 */
	SelectionProof  string                     `json:"selection_proof"`  /* 96 byte hexadecimal */
	Contribution    *SyncCommitteeContribution `json:"contribution"`
}

// SyncCommitteeContribution a sub property of AggregatorSelectionSignRequest.
type SyncCommitteeContribution struct {
	Slot              string `json:"slot"`               /* uint64 */
	BeaconBlockRoot   string `json:"block_root"`         /* Hash32 */ // Prysm uses BlockRoot instead of BeaconBlockRoot
	SubcommitteeIndex string `json:"subcommittee_index"` /* uint64 */
	AggregationBits   string `json:"aggregation_bits"`   /* SSZ hexadecimal string */
	Signature         string `json:"signature"`          /* 96 byte hexadecimal string */
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

// SignResponse the response object of the web3signer sign api.
type SignResponse struct {
	Signature string `json:"signature"`
}
