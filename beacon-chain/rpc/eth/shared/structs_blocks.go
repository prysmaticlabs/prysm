package shared

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type SignedBeaconBlock struct {
	Message   *BeaconBlock `json:"message" validate:"required"`
	Signature string       `json:"signature" validate:"required"`
}

type BeaconBlock struct {
	Slot          string           `json:"slot" validate:"required"`
	ProposerIndex string           `json:"proposer_index" validate:"required"`
	ParentRoot    string           `json:"parent_root" validate:"required"`
	StateRoot     string           `json:"state_root" validate:"required"`
	Body          *BeaconBlockBody `json:"body" validate:"required"`
}

type BeaconBlockBody struct {
	RandaoReveal      string                 `json:"randao_reveal" validate:"required"`
	Eth1Data          *Eth1Data              `json:"eth1_data" validate:"required"`
	Graffiti          string                 `json:"graffiti" validate:"required"`
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings" validate:"required"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings" validate:"required"`
	Attestations      []*Attestation         `json:"attestations" validate:"required"`
	Deposits          []*Deposit             `json:"deposits" validate:"required"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits" validate:"required"`
}

type SignedBeaconBlockAltair struct {
	Message   *BeaconBlockAltair `json:"message" validate:"required"`
	Signature string             `json:"signature" validate:"required"`
}

type BeaconBlockAltair struct {
	Slot          string                 `json:"slot" validate:"required"`
	ProposerIndex string                 `json:"proposer_index" validate:"required"`
	ParentRoot    string                 `json:"parent_root" validate:"required"`
	StateRoot     string                 `json:"state_root" validate:"required"`
	Body          *BeaconBlockBodyAltair `json:"body" validate:"required"`
}

type BeaconBlockBodyAltair struct {
	RandaoReveal      string                 `json:"randao_reveal" validate:"required"`
	Eth1Data          *Eth1Data              `json:"eth1_data" validate:"required"`
	Graffiti          string                 `json:"graffiti" validate:"required"`
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings" validate:"required,dive"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings" validate:"required,dive"`
	Attestations      []*Attestation         `json:"attestations" validate:"required,dive"`
	Deposits          []*Deposit             `json:"deposits" validate:"required,dive"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits" validate:"required,dive"`
	SyncAggregate     *SyncAggregate         `json:"sync_aggregate" validate:"required"`
}

type SignedBeaconBlockBellatrix struct {
	Message   *BeaconBlockBellatrix `json:"message" validate:"required"`
	Signature string                `json:"signature" validate:"required"`
}

type BeaconBlockBellatrix struct {
	Slot          string                    `json:"slot" validate:"required"`
	ProposerIndex string                    `json:"proposer_index" validate:"required"`
	ParentRoot    string                    `json:"parent_root" validate:"required"`
	StateRoot     string                    `json:"state_root" validate:"required"`
	Body          *BeaconBlockBodyBellatrix `json:"body" validate:"required"`
}

type BeaconBlockBodyBellatrix struct {
	RandaoReveal      string                 `json:"randao_reveal" validate:"required"`
	Eth1Data          *Eth1Data              `json:"eth1_data" validate:"required"`
	Graffiti          string                 `json:"graffiti" validate:"required"`
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings" validate:"required,dive"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings" validate:"required,dive"`
	Attestations      []*Attestation         `json:"attestations" validate:"required,dive"`
	Deposits          []*Deposit             `json:"deposits" validate:"required,dive"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits" validate:"required,dive"`
	SyncAggregate     *SyncAggregate         `json:"sync_aggregate" validate:"required"`
	ExecutionPayload  *ExecutionPayload      `json:"execution_payload" validate:"required"`
}

type SignedBlindedBeaconBlockBellatrix struct {
	Message   *BlindedBeaconBlockBellatrix `json:"message" validate:"required"`
	Signature string                       `json:"signature" validate:"required"`
}

type BlindedBeaconBlockBellatrix struct {
	Slot          string                           `json:"slot" validate:"required"`
	ProposerIndex string                           `json:"proposer_index" validate:"required"`
	ParentRoot    string                           `json:"parent_root" validate:"required"`
	StateRoot     string                           `json:"state_root" validate:"required"`
	Body          *BlindedBeaconBlockBodyBellatrix `json:"body" validate:"required"`
}

type BlindedBeaconBlockBodyBellatrix struct {
	RandaoReveal           string                  `json:"randao_reveal" validate:"required"`
	Eth1Data               *Eth1Data               `json:"eth1_data" validate:"required"`
	Graffiti               string                  `json:"graffiti" validate:"required"`
	ProposerSlashings      []*ProposerSlashing     `json:"proposer_slashings" validate:"required,dive"`
	AttesterSlashings      []*AttesterSlashing     `json:"attester_slashings" validate:"required,dive"`
	Attestations           []*Attestation          `json:"attestations" validate:"required,dive"`
	Deposits               []*Deposit              `json:"deposits" validate:"required,dive"`
	VoluntaryExits         []*SignedVoluntaryExit  `json:"voluntary_exits" validate:"required,dive"`
	SyncAggregate          *SyncAggregate          `json:"sync_aggregate" validate:"required"`
	ExecutionPayloadHeader *ExecutionPayloadHeader `json:"execution_payload_header" validate:"required"`
}

type SignedBeaconBlockCapella struct {
	Message   *BeaconBlockCapella `json:"message" validate:"required"`
	Signature string              `json:"signature" validate:"required"`
}

type BeaconBlockCapella struct {
	Slot          string                  `json:"slot" validate:"required"`
	ProposerIndex string                  `json:"proposer_index" validate:"required"`
	ParentRoot    string                  `json:"parent_root" validate:"required"`
	StateRoot     string                  `json:"state_root" validate:"required"`
	Body          *BeaconBlockBodyCapella `json:"body" validate:"required"`
}

type BeaconBlockBodyCapella struct {
	RandaoReveal          string                        `json:"randao_reveal" validate:"required"`
	Eth1Data              *Eth1Data                     `json:"eth1_data" validate:"required"`
	Graffiti              string                        `json:"graffiti" validate:"required"`
	ProposerSlashings     []*ProposerSlashing           `json:"proposer_slashings" validate:"required,dive"`
	AttesterSlashings     []*AttesterSlashing           `json:"attester_slashings" validate:"required,dive"`
	Attestations          []*Attestation                `json:"attestations" validate:"required,dive"`
	Deposits              []*Deposit                    `json:"deposits" validate:"required,dive"`
	VoluntaryExits        []*SignedVoluntaryExit        `json:"voluntary_exits" validate:"required,dive"`
	SyncAggregate         *SyncAggregate                `json:"sync_aggregate" validate:"required"`
	ExecutionPayload      *ExecutionPayloadCapella      `json:"execution_payload" validate:"required"`
	BlsToExecutionChanges []*SignedBlsToExecutionChange `json:"bls_to_execution_changes" validate:"required,dive"`
}

type SignedBlindedBeaconBlockCapella struct {
	Message   *BlindedBeaconBlockCapella `json:"message" validate:"required"`
	Signature string                     `json:"signature" validate:"required"`
}

type BlindedBeaconBlockCapella struct {
	Slot          string                         `json:"slot" validate:"required"`
	ProposerIndex string                         `json:"proposer_index" validate:"required"`
	ParentRoot    string                         `json:"parent_root" validate:"required"`
	StateRoot     string                         `json:"state_root" validate:"required"`
	Body          *BlindedBeaconBlockBodyCapella `json:"body" validate:"required"`
}

type BlindedBeaconBlockBodyCapella struct {
	RandaoReveal           string                         `json:"randao_reveal" validate:"required"`
	Eth1Data               *Eth1Data                      `json:"eth1_data" validate:"required"`
	Graffiti               string                         `json:"graffiti" validate:"required"`
	ProposerSlashings      []*ProposerSlashing            `json:"proposer_slashings" validate:"required,dive"`
	AttesterSlashings      []*AttesterSlashing            `json:"attester_slashings" validate:"required,dive"`
	Attestations           []*Attestation                 `json:"attestations" validate:"required,dive"`
	Deposits               []*Deposit                     `json:"deposits" validate:"required,dive"`
	VoluntaryExits         []*SignedVoluntaryExit         `json:"voluntary_exits" validate:"required,dive"`
	SyncAggregate          *SyncAggregate                 `json:"sync_aggregate" validate:"required"`
	ExecutionPayloadHeader *ExecutionPayloadHeaderCapella `json:"execution_payload_header" validate:"required"`
	BlsToExecutionChanges  []*SignedBlsToExecutionChange  `json:"bls_to_execution_changes" validate:"required,dive"`
}

type SignedBeaconBlockContentsDeneb struct {
	SignedBlock        *SignedBeaconBlockDeneb `json:"signed_block" validate:"required"`
	SignedBlobSidecars []*SignedBlobSidecar    `json:"signed_blob_sidecars"  validate:"dive"`
}

type BeaconBlockContentsDeneb struct {
	Block        *BeaconBlockDeneb `json:"block" validate:"required"`
	BlobSidecars []*BlobSidecar    `json:"blob_sidecars" validate:"dive"`
}

type SignedBeaconBlockDeneb struct {
	Message   *BeaconBlockDeneb `json:"message" validate:"required"`
	Signature string            `json:"signature" validate:"required"`
}

type BeaconBlockDeneb struct {
	Slot          string                `json:"slot" validate:"required"`
	ProposerIndex string                `json:"proposer_index" validate:"required"`
	ParentRoot    string                `json:"parent_root" validate:"required"`
	StateRoot     string                `json:"state_root" validate:"required"`
	Body          *BeaconBlockBodyDeneb `json:"body" validate:"required"`
}

type BeaconBlockBodyDeneb struct {
	RandaoReveal          string                        `json:"randao_reveal" validate:"required"`
	Eth1Data              *Eth1Data                     `json:"eth1_data" validate:"required"`
	Graffiti              string                        `json:"graffiti" validate:"required"`
	ProposerSlashings     []*ProposerSlashing           `json:"proposer_slashings" validate:"required,dive"`
	AttesterSlashings     []*AttesterSlashing           `json:"attester_slashings" validate:"required,dive"`
	Attestations          []*Attestation                `json:"attestations" validate:"required,dive"`
	Deposits              []*Deposit                    `json:"deposits" validate:"required,dive"`
	VoluntaryExits        []*SignedVoluntaryExit        `json:"voluntary_exits" validate:"required,dive"`
	SyncAggregate         *SyncAggregate                `json:"sync_aggregate" validate:"required"`
	ExecutionPayload      *ExecutionPayloadDeneb        `json:"execution_payload" validate:"required"`
	BlsToExecutionChanges []*SignedBlsToExecutionChange `json:"bls_to_execution_changes" validate:"required,dive"`
	BlobKzgCommitments    []string                      `json:"blob_kzg_commitments" validate:"required,dive"`
}

type ExecutionPayloadDeneb struct {
	ParentHash    string        `json:"parent_hash" validate:"required"`
	FeeRecipient  string        `json:"fee_recipient" validate:"required"`
	StateRoot     string        `json:"state_root" validate:"required"`
	ReceiptsRoot  string        `json:"receipts_root" validate:"required"`
	LogsBloom     string        `json:"logs_bloom" validate:"required"`
	PrevRandao    string        `json:"prev_randao" validate:"required"`
	BlockNumber   string        `json:"block_number" validate:"required"`
	GasLimit      string        `json:"gas_limit" validate:"required"`
	GasUsed       string        `json:"gas_used" validate:"required"`
	Timestamp     string        `json:"timestamp" validate:"required"`
	ExtraData     string        `json:"extra_data" validate:"required"`
	BaseFeePerGas string        `json:"base_fee_per_gas" validate:"required"`
	BlobGasUsed   string        `json:"blob_gas_used" validate:"required"`   // new in deneb
	ExcessBlobGas string        `json:"excess_blob_gas" validate:"required"` // new in deneb
	BlockHash     string        `json:"block_hash" validate:"required"`
	Transactions  []string      `json:"transactions" validate:"required,dive,hexadecimal"`
	Withdrawals   []*Withdrawal `json:"withdrawals" validate:"required,dive"`
}

type SignedBlindedBeaconBlockContentsDeneb struct {
	SignedBlindedBlock        *SignedBlindedBeaconBlockDeneb `json:"signed_blinded_block" validate:"required"`
	SignedBlindedBlobSidecars []*SignedBlindedBlobSidecar    `json:"signed_blinded_blob_sidecars" validate:"dive"`
}

type BlindedBeaconBlockContentsDeneb struct {
	BlindedBlock        *BlindedBeaconBlockDeneb `json:"blinded_block" validate:"required"`
	BlindedBlobSidecars []*BlindedBlobSidecar    `json:"blinded_blob_sidecars" validate:"dive"`
}

type BlindedBeaconBlockDeneb struct {
	Slot          string                       `json:"slot" validate:"required"`
	ProposerIndex string                       `json:"proposer_index" validate:"required"`
	ParentRoot    string                       `json:"parent_root" validate:"required"`
	StateRoot     string                       `json:"state_root" validate:"required"`
	Body          *BlindedBeaconBlockBodyDeneb `json:"body" validate:"required"`
}

type SignedBlindedBeaconBlockDeneb struct {
	Message   *BlindedBeaconBlockDeneb `json:"message" validate:"required"`
	Signature string                   `json:"signature" validate:"required"`
}

type BlindedBeaconBlockBodyDeneb struct {
	RandaoReveal           string                        `json:"randao_reveal" validate:"required"`
	Eth1Data               *Eth1Data                     `json:"eth1_data" validate:"required"`
	Graffiti               string                        `json:"graffiti" validate:"required"`
	ProposerSlashings      []*ProposerSlashing           `json:"proposer_slashings" validate:"required,dive"`
	AttesterSlashings      []*AttesterSlashing           `json:"attester_slashings" validate:"required,dive"`
	Attestations           []*Attestation                `json:"attestations" validate:"required,dive"`
	Deposits               []*Deposit                    `json:"deposits" validate:"required,dive"`
	VoluntaryExits         []*SignedVoluntaryExit        `json:"voluntary_exits" validate:"required,dive"`
	SyncAggregate          *SyncAggregate                `json:"sync_aggregate" validate:"required"`
	ExecutionPayloadHeader *ExecutionPayloadHeaderDeneb  `json:"execution_payload_header" validate:"required"`
	BlsToExecutionChanges  []*SignedBlsToExecutionChange `json:"bls_to_execution_changes" validate:"required,dive"`
	BlobKzgCommitments     []string                      `json:"blob_kzg_commitments" validate:"required,dive,hexadecimal"`
}

type SignedBlindedBlobSidecar struct {
	Message   *BlindedBlobSidecar `json:"message" validate:"required"`
	Signature string              `json:"signature" validate:"required"`
}

type SignedBlobSidecar struct {
	Message   *BlobSidecar `json:"message" validate:"required"`
	Signature string       `json:"signature" validate:"required"`
}

type BlindedBlobSidecar struct {
	BlockRoot       string `json:"block_root" validate:"required"`
	Index           string `json:"index" validate:"required"`
	Slot            string `json:"slot" validate:"required"`
	BlockParentRoot string `json:"block_parent_root" validate:"required"`
	ProposerIndex   string `json:"proposer_index" validate:"required"`
	BlobRoot        string `json:"blob_root" validate:"required"`
	KzgCommitment   string `json:"kzg_commitment" validate:"required"` // pattern: "^0x[a-fA-F0-9]{96}$" ssz-size:"48"
	KzgProof        string `json:"kzg_proof" validate:"required"`      // pattern: "^0x[a-fA-F0-9]{96}$" ssz-size:"48"
}

type BlobSidecar struct {
	BlockRoot       string `json:"block_root" validate:"required"`
	Index           string `json:"index" validate:"required"`
	Slot            string `json:"slot" validate:"required"`
	BlockParentRoot string `json:"block_parent_root" validate:"required"`
	ProposerIndex   string `json:"proposer_index" validate:"required"`
	Blob            string `json:"blob" validate:"required"`           // pattern: "^0x[a-fA-F0-9]{262144}$"
	KzgCommitment   string `json:"kzg_commitment" validate:"required"` // pattern: "^0x[a-fA-F0-9]{96}$" ssz-size:"48"
	KzgProof        string `json:"kzg_proof" validate:"required"`      // pattern: "^0x[a-fA-F0-9]{96}$" ssz-size:"48"
}

type Eth1Data struct {
	DepositRoot  string `json:"deposit_root" validate:"required"`
	DepositCount string `json:"deposit_count" validate:"required"`
	BlockHash    string `json:"block_hash" validate:"required"`
}

func Eth1DataFromConsensus(e1d *eth.Eth1Data) (*Eth1Data, error) {
	if e1d == nil {
		return nil, errors.New("eth1data is nil")
	}

	return &Eth1Data{
		DepositRoot:  hexutil.Encode(e1d.DepositRoot),
		DepositCount: fmt.Sprintf("%d", e1d.DepositCount),
		BlockHash:    hexutil.Encode(e1d.BlockHash),
	}, nil
}

type ProposerSlashing struct {
	SignedHeader1 *SignedBeaconBlockHeader `json:"signed_header_1" validate:"required"`
	SignedHeader2 *SignedBeaconBlockHeader `json:"signed_header_2" validate:"required"`
}

type AttesterSlashing struct {
	Attestation1 *IndexedAttestation `json:"attestation_1" validate:"required"`
	Attestation2 *IndexedAttestation `json:"attestation_2" validate:"required"`
}

type Deposit struct {
	Proof []string     `json:"proof" validate:"required,dive,hexadecimal"`
	Data  *DepositData `json:"data" validate:"required"`
}

type DepositData struct {
	Pubkey                string `json:"pubkey" validate:"required"`
	WithdrawalCredentials string `json:"withdrawal_credentials" validate:"required"`
	Amount                string `json:"amount" validate:"required"`
	Signature             string `json:"signature" validate:"required"`
}

type SignedBeaconBlockHeaderContainer struct {
	Header    *SignedBeaconBlockHeader `json:"header"`
	Root      string                   `json:"root"`
	Canonical bool                     `json:"canonical"`
}

type SignedBeaconBlockHeader struct {
	Message   *BeaconBlockHeader `json:"message" validate:"required"`
	Signature string             `json:"signature" validate:"required"`
}

type BeaconBlockHeader struct {
	Slot          string `json:"slot" validate:"required"`
	ProposerIndex string `json:"proposer_index" validate:"required"`
	ParentRoot    string `json:"parent_root" validate:"required"`
	StateRoot     string `json:"state_root" validate:"required"`
	BodyRoot      string `json:"body_root" validate:"required"`
}

type IndexedAttestation struct {
	AttestingIndices []string         `json:"attesting_indices" validate:"required,dive"`
	Data             *AttestationData `json:"data" validate:"required"`
	Signature        string           `json:"signature" validate:"required"`
}

type SyncAggregate struct {
	SyncCommitteeBits      string `json:"sync_committee_bits" validate:"required"`
	SyncCommitteeSignature string `json:"sync_committee_signature" validate:"required"`
}

type ExecutionPayload struct {
	ParentHash    string   `json:"parent_hash" validate:"required"`
	FeeRecipient  string   `json:"fee_recipient" validate:"required"`
	StateRoot     string   `json:"state_root" validate:"required"`
	ReceiptsRoot  string   `json:"receipts_root" validate:"required"`
	LogsBloom     string   `json:"logs_bloom" validate:"required"`
	PrevRandao    string   `json:"prev_randao" validate:"required"`
	BlockNumber   string   `json:"block_number" validate:"required"`
	GasLimit      string   `json:"gas_limit" validate:"required"`
	GasUsed       string   `json:"gas_used" validate:"required"`
	Timestamp     string   `json:"timestamp" validate:"required"`
	ExtraData     string   `json:"extra_data" validate:"required"`
	BaseFeePerGas string   `json:"base_fee_per_gas" validate:"required"`
	BlockHash     string   `json:"block_hash" validate:"required"`
	Transactions  []string `json:"transactions" validate:"required,dive,hexadecimal"`
}

type ExecutionPayloadHeader struct {
	ParentHash       string `json:"parent_hash" validate:"required"`
	FeeRecipient     string `json:"fee_recipient" validate:"required"`
	StateRoot        string `json:"state_root" validate:"required"`
	ReceiptsRoot     string `json:"receipts_root" validate:"required"`
	LogsBloom        string `json:"logs_bloom" validate:"required"`
	PrevRandao       string `json:"prev_randao" validate:"required"`
	BlockNumber      string `json:"block_number" validate:"required"`
	GasLimit         string `json:"gas_limit" validate:"required"`
	GasUsed          string `json:"gas_used" validate:"required"`
	Timestamp        string `json:"timestamp" validate:"required"`
	ExtraData        string `json:"extra_data" validate:"required"`
	BaseFeePerGas    string `json:"base_fee_per_gas" validate:"required"`
	BlockHash        string `json:"block_hash" validate:"required"`
	TransactionsRoot string `json:"transactions_root" validate:"required"`
}

type ExecutionPayloadCapella struct {
	ParentHash    string        `json:"parent_hash" validate:"required"`
	FeeRecipient  string        `json:"fee_recipient" validate:"required"`
	StateRoot     string        `json:"state_root" validate:"required"`
	ReceiptsRoot  string        `json:"receipts_root" validate:"required"`
	LogsBloom     string        `json:"logs_bloom" validate:"required"`
	PrevRandao    string        `json:"prev_randao" validate:"required"`
	BlockNumber   string        `json:"block_number" validate:"required"`
	GasLimit      string        `json:"gas_limit" validate:"required"`
	GasUsed       string        `json:"gas_used" validate:"required"`
	Timestamp     string        `json:"timestamp" validate:"required"`
	ExtraData     string        `json:"extra_data" validate:"required"`
	BaseFeePerGas string        `json:"base_fee_per_gas" validate:"required"`
	BlockHash     string        `json:"block_hash" validate:"required"`
	Transactions  []string      `json:"transactions" validate:"required,dive"`
	Withdrawals   []*Withdrawal `json:"withdrawals" validate:"required,dive"`
}

type ExecutionPayloadHeaderCapella struct {
	ParentHash       string `json:"parent_hash" validate:"required"`
	FeeRecipient     string `json:"fee_recipient" validate:"required"`
	StateRoot        string `json:"state_root" validate:"required"`
	ReceiptsRoot     string `json:"receipts_root" validate:"required"`
	LogsBloom        string `json:"logs_bloom" validate:"required"`
	PrevRandao       string `json:"prev_randao" validate:"required"`
	BlockNumber      string `json:"block_number" validate:"required"`
	GasLimit         string `json:"gas_limit" validate:"required"`
	GasUsed          string `json:"gas_used" validate:"required"`
	Timestamp        string `json:"timestamp" validate:"required"`
	ExtraData        string `json:"extra_data" validate:"required"`
	BaseFeePerGas    string `json:"base_fee_per_gas" validate:"required"`
	BlockHash        string `json:"block_hash" validate:"required"`
	TransactionsRoot string `json:"transactions_root" validate:"required"`
	WithdrawalsRoot  string `json:"withdrawals_root" validate:"required"`
}

type ExecutionPayloadHeaderDeneb struct {
	ParentHash       string `json:"parent_hash" validate:"required"`
	FeeRecipient     string `json:"fee_recipient" validate:"required"`
	StateRoot        string `json:"state_root" validate:"required"`
	ReceiptsRoot     string `json:"receipts_root" validate:"required"`
	LogsBloom        string `json:"logs_bloom" validate:"required"`
	PrevRandao       string `json:"prev_randao" validate:"required"`
	BlockNumber      string `json:"block_number" validate:"required"`
	GasLimit         string `json:"gas_limit" validate:"required"`
	GasUsed          string `json:"gas_used" validate:"required"`
	Timestamp        string `json:"timestamp" validate:"required"`
	ExtraData        string `json:"extra_data" validate:"required"`
	BaseFeePerGas    string `json:"base_fee_per_gas" validate:"required"`
	BlobGasUsed      string `json:"blob_gas_used" validate:"required"`   // new in deneb
	ExcessBlobGas    string `json:"excess_blob_gas" validate:"required"` // new in deneb
	BlockHash        string `json:"block_hash" validate:"required"`
	TransactionsRoot string `json:"transactions_root" validate:"required"`
	WithdrawalsRoot  string `json:"withdrawals_root" validate:"required"`
}

type Withdrawal struct {
	WithdrawalIndex  string `json:"index" validate:"required"`
	ValidatorIndex   string `json:"validator_index" validate:"required"`
	ExecutionAddress string `json:"address" validate:"required"`
	Amount           string `json:"amount" validate:"required"`
}

type SignedBlsToExecutionChange struct {
	Message   *BlsToExecutionChange `json:"message" validate:"required"`
	Signature string                `json:"signature" validate:"required"`
}

type BlsToExecutionChange struct {
	ValidatorIndex     string `json:"validator_index" validate:"required"`
	FromBlsPubkey      string `json:"from_bls_pubkey" validate:"required"`
	ToExecutionAddress string `json:"to_execution_address" validate:"required"`
}
