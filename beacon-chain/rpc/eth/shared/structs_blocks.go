package shared

import (
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

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
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings" validate:"required,dive"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings" validate:"required,dive"`
	Attestations      []*Attestation         `json:"attestations" validate:"required,dive"`
	Deposits          []*Deposit             `json:"deposits" validate:"required,dive"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits" validate:"required,dive"`
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
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings" validate:"required,dive"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings" validate:"required,dive"`
	Attestations      []*Attestation         `json:"attestations" validate:"required,dive"`
	Deposits          []*Deposit             `json:"deposits" validate:"required,dive"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits" validate:"required,dive"`
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
	ProposerSlashings      []*ProposerSlashing     `json:"proposer_slashings" validate:"required,dive"`
	AttesterSlashings      []*AttesterSlashing     `json:"attester_slashings" validate:"required,dive"`
	Attestations           []*Attestation          `json:"attestations" validate:"required,dive"`
	Deposits               []*Deposit              `json:"deposits" validate:"required,dive"`
	VoluntaryExits         []*SignedVoluntaryExit  `json:"voluntary_exits" validate:"required,dive"`
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
	ProposerSlashings     []*ProposerSlashing           `json:"proposer_slashings" validate:"required,dive"`
	AttesterSlashings     []*AttesterSlashing           `json:"attester_slashings" validate:"required,dive"`
	Attestations          []*Attestation                `json:"attestations" validate:"required,dive"`
	Deposits              []*Deposit                    `json:"deposits" validate:"required,dive"`
	VoluntaryExits        []*SignedVoluntaryExit        `json:"voluntary_exits" validate:"required,dive"`
	SyncAggregate         *SyncAggregate                `json:"sync_aggregate"`
	ExecutionPayload      *ExecutionPayloadCapella      `json:"execution_payload"`
	BlsToExecutionChanges []*SignedBlsToExecutionChange `json:"bls_to_execution_changes" validate:"required,dive"`
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
	ProposerSlashings      []*ProposerSlashing            `json:"proposer_slashings" validate:"required,dive"`
	AttesterSlashings      []*AttesterSlashing            `json:"attester_slashings" validate:"required,dive"`
	Attestations           []*Attestation                 `json:"attestations" validate:"required,dive"`
	Deposits               []*Deposit                     `json:"deposits" validate:"required,dive"`
	VoluntaryExits         []*SignedVoluntaryExit         `json:"voluntary_exits" validate:"required,dive"`
	SyncAggregate          *SyncAggregate                 `json:"sync_aggregate"`
	ExecutionPayloadHeader *ExecutionPayloadHeaderCapella `json:"execution_payload_header"`
	BlsToExecutionChanges  []*SignedBlsToExecutionChange  `json:"bls_to_execution_changes" validate:"required,dive"`
}

type SignedBeaconBlockContentsDeneb struct {
	SignedBlock        *SignedBeaconBlockDeneb `json:"signed_block"`
	SignedBlobSidecars []*SignedBlobSidecar    `json:"signed_blob_sidecars"  validate:"dive"`
}

type BeaconBlockContentsDeneb struct {
	Block        *BeaconBlockDeneb `json:"block"`
	BlobSidecars []*BlobSidecar    `json:"blob_sidecars" validate:"dive"`
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
	ProposerSlashings     []*ProposerSlashing           `json:"proposer_slashings" validate:"required,dive"`
	AttesterSlashings     []*AttesterSlashing           `json:"attester_slashings" validate:"required,dive"`
	Attestations          []*Attestation                `json:"attestations" validate:"required,dive"`
	Deposits              []*Deposit                    `json:"deposits" validate:"required,dive"`
	VoluntaryExits        []*SignedVoluntaryExit        `json:"voluntary_exits" validate:"required,dive"`
	SyncAggregate         *SyncAggregate                `json:"sync_aggregate"`
	ExecutionPayload      *ExecutionPayloadDeneb        `json:"execution_payload"`
	BlsToExecutionChanges []*SignedBlsToExecutionChange `json:"bls_to_execution_changes" validate:"required,dive"`
	BlobKzgCommitments    []string                      `json:"blob_kzg_commitments" validate:"required,dive"`
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
	BlobGasUsed   string        `json:"blob_gas_used"`   // new in deneb
	ExcessBlobGas string        `json:"excess_blob_gas"` // new in deneb
	BlockHash     string        `json:"block_hash"`
	Transactions  []string      `json:"transactions" validate:"required,dive,hexadecimal"`
	Withdrawals   []*Withdrawal `json:"withdrawals" validate:"required,dive"`
}

type SignedBlindedBeaconBlockContentsDeneb struct {
	SignedBlindedBlock        *SignedBlindedBeaconBlockDeneb `json:"signed_blinded_block"`
	SignedBlindedBlobSidecars []*SignedBlindedBlobSidecar    `json:"signed_blinded_blob_sidecars" validate:"dive"`
}

type BlindedBeaconBlockContentsDeneb struct {
	BlindedBlock        *BlindedBeaconBlockDeneb `json:"blinded_block"`
	BlindedBlobSidecars []*BlindedBlobSidecar    `json:"blinded_blob_sidecars" validate:"dive"`
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
	ProposerSlashings      []*ProposerSlashing           `json:"proposer_slashings" validate:"required,dive"`
	AttesterSlashings      []*AttesterSlashing           `json:"attester_slashings" validate:"required,dive"`
	Attestations           []*Attestation                `json:"attestations" validate:"required,dive"`
	Deposits               []*Deposit                    `json:"deposits" validate:"required,dive"`
	VoluntaryExits         []*SignedVoluntaryExit        `json:"voluntary_exits" validate:"required,dive"`
	SyncAggregate          *SyncAggregate                `json:"sync_aggregate"`
	ExecutionPayloadHeader *ExecutionPayloadHeaderDeneb  `json:"execution_payload_header"`
	BlsToExecutionChanges  []*SignedBlsToExecutionChange `json:"bls_to_execution_changes" validate:"required,dive"`
	BlobKzgCommitments     []string                      `json:"blob_kzg_commitments" validate:"required,dive,hexadecimal"`
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
	SignedHeader1 *SignedBeaconBlockHeader `json:"signed_header_1"`
	SignedHeader2 *SignedBeaconBlockHeader `json:"signed_header_2"`
}

func (s *ProposerSlashing) ToConsensus() (*eth.ProposerSlashing, error) {
	h1, err := s.SignedHeader1.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "SignedHeader1")
	}
	h2, err := s.SignedHeader2.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "SignedHeader2")
	}

	return &eth.ProposerSlashing{
		Header_1: h1,
		Header_2: h2,
	}, nil
}

type AttesterSlashing struct {
	Attestation1 *IndexedAttestation `json:"attestation_1"`
	Attestation2 *IndexedAttestation `json:"attestation_2"`
}

func (s *AttesterSlashing) ToConsensus() (*eth.AttesterSlashing, error) {
	att1, err := s.Attestation1.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Attestation1")
	}
	att2, err := s.Attestation2.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Attestation2")
	}
	return &eth.AttesterSlashing{Attestation_1: att1, Attestation_2: att2}, nil
}

type Deposit struct {
	Proof []string     `json:"proof" validate:"required,dive,hexadecimal"`
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

func (h *SignedBeaconBlockHeader) ToConsensus() (*eth.SignedBeaconBlockHeader, error) {
	msg, err := h.Message.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Message")
	}
	sig, err := DecodeHexWithLength(h.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}

	return &eth.SignedBeaconBlockHeader{
		Header:    msg,
		Signature: sig,
	}, nil
}

type BeaconBlockHeader struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root"`
	StateRoot     string `json:"state_root"`
	BodyRoot      string `json:"body_root"`
}

func (h *BeaconBlockHeader) ToConsensus() (*eth.BeaconBlockHeader, error) {
	s, err := strconv.ParseUint(h.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	pi, err := strconv.ParseUint(h.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	pr, err := DecodeHexWithLength(h.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	sr, err := DecodeHexWithLength(h.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	br, err := DecodeHexWithLength(h.BodyRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "BodyRoot")
	}

	return &eth.BeaconBlockHeader{
		Slot:          primitives.Slot(s),
		ProposerIndex: primitives.ValidatorIndex(pi),
		ParentRoot:    pr,
		StateRoot:     sr,
		BodyRoot:      br,
	}, nil
}

type IndexedAttestation struct {
	AttestingIndices []string         `json:"attesting_indices" validate:"required,dive"`
	Data             *AttestationData `json:"data"`
	Signature        string           `json:"signature"`
}

func (a *IndexedAttestation) ToConsensus() (*eth.IndexedAttestation, error) {
	indices := make([]uint64, len(a.AttestingIndices))
	var err error
	for i, ix := range a.AttestingIndices {
		indices[i], err = strconv.ParseUint(ix, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("AttestingIndices[%d]", i))
		}
	}
	data, err := a.Data.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Data")
	}
	sig, err := DecodeHexWithLength(a.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}

	return &eth.IndexedAttestation{
		AttestingIndices: indices,
		Data:             data,
		Signature:        sig,
	}, nil
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
	Transactions  []string `json:"transactions" validate:"required,dive,hexadecimal"`
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
	Transactions  []string      `json:"transactions" validate:"required,dive"`
	Withdrawals   []*Withdrawal `json:"withdrawals" validate:"required,dive"`
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
	BlobGasUsed      string `json:"blob_gas_used"`   // new in deneb
	ExcessBlobGas    string `json:"excess_blob_gas"` // new in deneb
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

func WithdrawalsFromConsensus(ws []*enginev1.Withdrawal) []*Withdrawal {
	result := make([]*Withdrawal, len(ws))
	for i, w := range ws {
		result[i] = WithdrawalFromConsensus(w)
	}
	return result
}

func WithdrawalFromConsensus(w *enginev1.Withdrawal) *Withdrawal {
	return &Withdrawal{
		WithdrawalIndex:  fmt.Sprintf("%d", w.Index),
		ValidatorIndex:   fmt.Sprintf("%d", w.ValidatorIndex),
		ExecutionAddress: hexutil.Encode(w.Address),
		Amount:           fmt.Sprintf("%d", w.Amount),
	}
}

type SignedBlsToExecutionChange struct {
	Message   *BlsToExecutionChange `json:"message"`
	Signature string                `json:"signature"`
}

type BlsToExecutionChange struct {
	ValidatorIndex     string `json:"validator_index"`
	FromBlsPubkey      string `json:"from_bls_pubkey"`
	ToExecutionAddress string `json:"to_execution_address"`
}
