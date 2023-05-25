package beacon

import (
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	bytesutil2 "github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/wealdtech/go-bytesutil"
)

type SignedBeaconBlock struct {
	Message   BeaconBlock `json:"message" validate:"required"`
	Signature string      `json:"signature" validate:"required" hex:"true"`
}

type BeaconBlock struct {
	Slot          string          `json:"slot" validate:"required"`
	ProposerIndex string          `json:"proposer_index" validate:"required"`
	ParentRoot    string          `json:"parent_root" validate:"required" hex:"true"`
	StateRoot     string          `json:"state_root" validate:"required" hex:"true"`
	Body          BeaconBlockBody `json:"body" validate:"required"`
}

type BeaconBlockBody struct {
	RandaoReveal      string                `json:"randao_reveal" validate:"required" hex:"true"`
	Eth1Data          Eth1Data              `json:"eth1_data" validate:"required"`
	Graffiti          string                `json:"graffiti" validate:"required" hex:"true"`
	ProposerSlashings []ProposerSlashing    `json:"proposer_slashings" validate:"required"`
	AttesterSlashings []AttesterSlashing    `json:"attester_slashings" validate:"required"`
	Attestations      []Attestation         `json:"attestations" validate:"required"`
	Deposits          []Deposit             `json:"deposits" validate:"required"`
	VoluntaryExits    []SignedVoluntaryExit `json:"voluntary_exits" validate:"required"`
}

type SignedBeaconBlockAltair struct {
	Message   BeaconBlockAltair `json:"message" validate:"required"`
	Signature string            `json:"signature" validate:"required" hex:"true"`
}

type BeaconBlockAltair struct {
	Slot          string                `json:"slot" validate:"required"`
	ProposerIndex string                `json:"proposer_index" validate:"required"`
	ParentRoot    string                `json:"parent_root" validate:"required" hex:"true"`
	StateRoot     string                `json:"state_root" validate:"required" hex:"true"`
	Body          BeaconBlockBodyAltair `json:"body" validate:"required"`
}

type BeaconBlockBodyAltair struct {
	RandaoReveal      string                `json:"randao_reveal" validate:"required" hex:"true"`
	Eth1Data          Eth1Data              `json:"eth1_data" validate:"required"`
	Graffiti          string                `json:"graffiti" validate:"required" hex:"true"`
	ProposerSlashings []ProposerSlashing    `json:"proposer_slashings" validate:"required"`
	AttesterSlashings []AttesterSlashing    `json:"attester_slashings" validate:"required"`
	Attestations      []Attestation         `json:"attestations" validate:"required"`
	Deposits          []Deposit             `json:"deposits" validate:"required"`
	VoluntaryExits    []SignedVoluntaryExit `json:"voluntary_exits" validate:"required"`
	SyncAggregate     SyncAggregate         `json:"sync_aggregate" validate:"required"`
}

type SignedBeaconBlockBellatrix struct {
	Message   BeaconBlockBellatrix `json:"message" validate:"required"`
	Signature string               `json:"signature" validate:"required" hex:"true"`
}

type BeaconBlockBellatrix struct {
	Slot          string                   `json:"slot" validate:"required"`
	ProposerIndex string                   `json:"proposer_index" validate:"required"`
	ParentRoot    string                   `json:"parent_root" validate:"required" hex:"true"`
	StateRoot     string                   `json:"state_root" validate:"required" hex:"true"`
	Body          BeaconBlockBodyBellatrix `json:"body" validate:"required"`
}

type BeaconBlockBodyBellatrix struct {
	RandaoReveal      string                `json:"randao_reveal" validate:"required" hex:"true"`
	Eth1Data          Eth1Data              `json:"eth1_data" validate:"required"`
	Graffiti          string                `json:"graffiti" validate:"required" hex:"true"`
	ProposerSlashings []ProposerSlashing    `json:"proposer_slashings" validate:"required"`
	AttesterSlashings []AttesterSlashing    `json:"attester_slashings" validate:"required"`
	Attestations      []Attestation         `json:"attestations" validate:"required"`
	Deposits          []Deposit             `json:"deposits" validate:"required"`
	VoluntaryExits    []SignedVoluntaryExit `json:"voluntary_exits" validate:"required"`
	SyncAggregate     SyncAggregate         `json:"sync_aggregate" validate:"required"`
	ExecutionPayload  ExecutionPayload      `json:"execution_payload" validate:"required"`
}

type SignedBlindedBeaconBlockBellatrix struct {
	Message   BlindedBeaconBlockBellatrix `json:"message" validate:"required"`
	Signature string                      `json:"signature" validate:"required" hex:"true"`
}

type BlindedBeaconBlockBellatrix struct {
	Slot          string                          `json:"slot" validate:"required"`
	ProposerIndex string                          `json:"proposer_index" validate:"required"`
	ParentRoot    string                          `json:"parent_root" validate:"required" hex:"true"`
	StateRoot     string                          `json:"state_root" validate:"required" hex:"true"`
	Body          BlindedBeaconBlockBodyBellatrix `json:"body" validate:"required"`
}

type BlindedBeaconBlockBodyBellatrix struct {
	RandaoReveal           string                 `json:"randao_reveal" validate:"required" hex:"true"`
	Eth1Data               Eth1Data               `json:"eth1_data" validate:"required"`
	Graffiti               string                 `json:"graffiti" validate:"required" hex:"true"`
	ProposerSlashings      []ProposerSlashing     `json:"proposer_slashings" validate:"required"`
	AttesterSlashings      []AttesterSlashing     `json:"attester_slashings" validate:"required"`
	Attestations           []Attestation          `json:"attestations" validate:"required"`
	Deposits               []Deposit              `json:"deposits" validate:"required"`
	VoluntaryExits         []SignedVoluntaryExit  `json:"voluntary_exits" validate:"required"`
	SyncAggregate          SyncAggregate          `json:"sync_aggregate" validate:"required"`
	ExecutionPayloadHeader ExecutionPayloadHeader `json:"execution_payload_header" validate:"required"`
}

type SignedBeaconBlockCapella struct {
	Message   BeaconBlockCapella `json:"message" validate:"required"`
	Signature string             `json:"signature" validate:"required" hex:"true"`
}

type BeaconBlockCapella struct {
	Slot          string                 `json:"slot" validate:"required"`
	ProposerIndex string                 `json:"proposer_index" validate:"required"`
	ParentRoot    string                 `json:"parent_root" validate:"required" hex:"true"`
	StateRoot     string                 `json:"state_root" validate:"required" hex:"true"`
	Body          BeaconBlockBodyCapella `json:"body" validate:"required"`
}

type BeaconBlockBodyCapella struct {
	RandaoReveal          string                       `json:"randao_reveal" validate:"required" hex:"true"`
	Eth1Data              Eth1Data                     `json:"eth1_data" validate:"required"`
	Graffiti              string                       `json:"graffiti" validate:"required" hex:"true"`
	ProposerSlashings     []ProposerSlashing           `json:"proposer_slashings" validate:"required"`
	AttesterSlashings     []AttesterSlashing           `json:"attester_slashings" validate:"required"`
	Attestations          []Attestation                `json:"attestations" validate:"required"`
	Deposits              []Deposit                    `json:"deposits" validate:"required"`
	VoluntaryExits        []SignedVoluntaryExit        `json:"voluntary_exits" validate:"required"`
	SyncAggregate         SyncAggregate                `json:"sync_aggregate" validate:"required"`
	ExecutionPayload      ExecutionPayloadCapella      `json:"execution_payload" validate:"required"`
	BlsToExecutionChanges []SignedBlsToExecutionChange `json:"bls_to_execution_changes" validate:"required"`
}

type SignedBlindedBeaconBlockCapella struct {
	Message   BlindedBeaconBlockCapella `json:"message" validate:"required"`
	Signature string                    `json:"signature" validate:"required" hex:"true"`
}

type BlindedBeaconBlockCapella struct {
	Slot          string                        `json:"slot" validate:"required"`
	ProposerIndex string                        `json:"proposer_index" validate:"required"`
	ParentRoot    string                        `json:"parent_root" validate:"required" hex:"true"`
	StateRoot     string                        `json:"state_root" validate:"required" hex:"true"`
	Body          BlindedBeaconBlockBodyCapella `json:"body" validate:"required"`
}

type BlindedBeaconBlockBodyCapella struct {
	RandaoReveal           string                        `json:"randao_reveal" validate:"required" hex:"true"`
	Eth1Data               Eth1Data                      `json:"eth1_data" validate:"required"`
	Graffiti               string                        `json:"graffiti" validate:"required" hex:"true"`
	ProposerSlashings      []ProposerSlashing            `json:"proposer_slashings" validate:"required"`
	AttesterSlashings      []AttesterSlashing            `json:"attester_slashings" validate:"required"`
	Attestations           []Attestation                 `json:"attestations" validate:"required"`
	Deposits               []Deposit                     `json:"deposits" validate:"required"`
	VoluntaryExits         []SignedVoluntaryExit         `json:"voluntary_exits" validate:"required"`
	SyncAggregate          SyncAggregate                 `json:"sync_aggregate" validate:"required"`
	ExecutionPayloadHeader ExecutionPayloadHeaderCapella `json:"execution_payload_header" validate:"required"`
	BlsToExecutionChanges  []SignedBlsToExecutionChange  `json:"bls_to_execution_changes" validate:"required"`
}

type Eth1Data struct {
	DepositRoot  string `json:"deposit_root" validate:"required" hex:"true"`
	DepositCount string `json:"deposit_count" validate:"required"`
	BlockHash    string `json:"block_hash" validate:"required" hex:"true"`
}

type ProposerSlashing struct {
	SignedHeader1 SignedBeaconBlockHeader `json:"signed_header_1" validate:"required"`
	SignedHeader2 SignedBeaconBlockHeader `json:"signed_header_2" validate:"required"`
}

type AttesterSlashing struct {
	Attestation1 IndexedAttestation `json:"attestation_1" validate:"required"`
	Attestation2 IndexedAttestation `json:"attestation_2" validate:"required"`
}

type Attestation struct {
	AggregationBits string          `json:"aggregation_bits" validate:"required" hex:"true"`
	Data            AttestationData `json:"data" validate:"required"`
	Signature       string          `json:"signature" validate:"required" hex:"true"`
}

type Deposit struct {
	Proof []string    `json:"proof" validate:"required" hex:"true"`
	Data  DepositData `json:"data" validate:"required"`
}

type DepositData struct {
	Pubkey                string `json:"pubkey" validate:"required" hex:"true"`
	WithdrawalCredentials string `json:"withdrawal_credentials" validate:"required" hex:"true"`
	Amount                string `json:"amount" validate:"required"`
	Signature             string `json:"signature" validate:"required" hex:"true"`
}

type SignedVoluntaryExit struct {
	Message   VoluntaryExit `json:"message" validate:"required"`
	Signature string        `json:"signature" validate:"required" hex:"true"`
}

type VoluntaryExit struct {
	Epoch          string `json:"epoch" validate:"required"`
	ValidatorIndex string `json:"validator_index" validate:"required"`
}

type SignedBeaconBlockHeader struct {
	Message   BeaconBlockHeader `json:"message" validate:"required"`
	Signature string            `json:"signature" validate:"required" hex:"true"`
}

type BeaconBlockHeader struct {
	Slot          string `json:"slot" validate:"required"`
	ProposerIndex string `json:"proposer_index" validate:"required"`
	ParentRoot    string `json:"parent_root" validate:"required" hex:"true"`
	StateRoot     string `json:"state_root" validate:"required" hex:"true"`
	BodyRoot      string `json:"body_root" validate:"required" hex:"true"`
}

type IndexedAttestation struct {
	AttestingIndices []string        `json:"attesting_indices" validate:"required"`
	Data             AttestationData `json:"data" validate:"required"`
	Signature        string          `json:"signature" validate:"required" hex:"true"`
}

type AttestationData struct {
	Slot            string     `json:"slot" validate:"required"`
	Index           string     `json:"index" validate:"required"`
	BeaconBlockRoot string     `json:"beacon_block_root" validate:"required" hex:"true"`
	Source          Checkpoint `json:"source" validate:"required"`
	Target          Checkpoint `json:"target" validate:"required"`
}

type Checkpoint struct {
	Epoch string `json:"epoch" validate:"required"`
	Root  string `json:"root" validate:"required" hex:"true"`
}

type SyncAggregate struct {
	SyncCommitteeBits      string `json:"sync_committee_bits" validate:"required" hex:"true"`
	SyncCommitteeSignature string `json:"sync_committee_signature" validate:"required" hex:"true"`
}

type ExecutionPayload struct {
	ParentHash    string   `json:"parent_hash" validate:"required" hex:"true"`
	FeeRecipient  string   `json:"fee_recipient" validate:"required" hex:"true"`
	StateRoot     string   `json:"state_root" validate:"required" hex:"true"`
	ReceiptsRoot  string   `json:"receipts_root" validate:"required" hex:"true"`
	LogsBloom     string   `json:"logs_bloom" validate:"required" hex:"true"`
	PrevRandao    string   `json:"prev_randao" validate:"required" hex:"true"`
	BlockNumber   string   `json:"block_number" validate:"required"`
	GasLimit      string   `json:"gas_limit" validate:"required"`
	GasUsed       string   `json:"gas_used" validate:"required"`
	Timestamp     string   `json:"timestamp" validate:"required"`
	ExtraData     string   `json:"extra_data" validate:"required" hex:"true"`
	BaseFeePerGas string   `json:"base_fee_per_gas" validate:"required" uint256:"true"` // TODO: Uint256
	BlockHash     string   `json:"block_hash" validate:"required" hex:"true"`
	Transactions  []string `json:"transactions" validate:"required" hex:"true"`
}

type ExecutionPayloadHeader struct {
	ParentHash       string `json:"parent_hash" validate:"required" hex:"true"`
	FeeRecipient     string `json:"fee_recipient" validate:"required" hex:"true"`
	StateRoot        string `json:"state_root" validate:"required" hex:"true"`
	ReceiptsRoot     string `json:"receipts_root" validate:"required" hex:"true"`
	LogsBloom        string `json:"logs_bloom" validate:"required" hex:"true"`
	PrevRandao       string `json:"prev_randao" validate:"required" hex:"true"`
	BlockNumber      string `json:"block_number" validate:"required"`
	GasLimit         string `json:"gas_limit" validate:"required"`
	GasUsed          string `json:"gas_used" validate:"required"`
	Timestamp        string `json:"timestamp" validate:"required"`
	ExtraData        string `json:"extra_data" validate:"required" hex:"true"`
	BaseFeePerGas    string `json:"base_fee_per_gas" validate:"required" uint256:"true"`
	BlockHash        string `json:"block_hash" validate:"required" hex:"true"`
	TransactionsRoot string `json:"transactions_root" validate:"required" hex:"true"`
}

type ExecutionPayloadCapella struct {
	ParentHash    string       `json:"parent_hash" validate:"required" hex:"true"`
	FeeRecipient  string       `json:"fee_recipient" validate:"required" hex:"true"`
	StateRoot     string       `json:"state_root" validate:"required" hex:"true"`
	ReceiptsRoot  string       `json:"receipts_root" validate:"required" hex:"true"`
	LogsBloom     string       `json:"logs_bloom" validate:"required" hex:"true"`
	PrevRandao    string       `json:"prev_randao" validate:"required" hex:"true"`
	BlockNumber   string       `json:"block_number" validate:"required"`
	GasLimit      string       `json:"gas_limit" validate:"required"`
	GasUsed       string       `json:"gas_used" validate:"required"`
	Timestamp     string       `json:"timestamp" validate:"required"`
	ExtraData     string       `json:"extra_data" validate:"required" hex:"true"`
	BaseFeePerGas string       `json:"base_fee_per_gas" validate:"required" uint256:"true"`
	BlockHash     string       `json:"block_hash" validate:"required" hex:"true"`
	Transactions  []string     `json:"transactions" validate:"required" hex:"true"`
	Withdrawals   []Withdrawal `json:"withdrawals" validate:"required"`
}

type ExecutionPayloadHeaderCapella struct {
	ParentHash       string `json:"parent_hash" validate:"required" hex:"true"`
	FeeRecipient     string `json:"fee_recipient" validate:"required" hex:"true"`
	StateRoot        string `json:"state_root" validate:"required" hex:"true"`
	ReceiptsRoot     string `json:"receipts_root" validate:"required" hex:"true"`
	LogsBloom        string `json:"logs_bloom" validate:"required" hex:"true"`
	PrevRandao       string `json:"prev_randao" validate:"required" hex:"true"`
	BlockNumber      string `json:"block_number" validate:"required"`
	GasLimit         string `json:"gas_limit" validate:"required"`
	GasUsed          string `json:"gas_used" validate:"required"`
	Timestamp        string `json:"timestamp" validate:"required"`
	ExtraData        string `json:"extra_data" validate:"required" hex:"true"`
	BaseFeePerGas    string `json:"base_fee_per_gas" validate:"required" uint256:"true"`
	BlockHash        string `json:"block_hash" validate:"required" hex:"true"`
	TransactionsRoot string `json:"transactions_root" validate:"required" hex:"true"`
	WithdrawalsRoot  string `json:"withdrawals_root" validate:"required" hex:"true"`
}

type Withdrawal struct {
	WithdrawalIndex  string `json:"index" validate:"required"`
	ValidatorIndex   string `json:"validator_index" validate:"required"`
	ExecutionAddress string `json:"address" validate:"required" hex:"true"`
	Amount           string `json:"amount" validate:"required"`
}

type SignedBlsToExecutionChange struct {
	BlsToExecutionChange BlsToExecutionChange `json:"bls_to_execution_change" validate:"required"`
	Signature            string               `json:"signature" validate:"required" hex:"true"`
}

type BlsToExecutionChange struct {
	ValidatorIndex string `json:"validator_index" validate:"required"`
}

func (b *SignedBeaconBlock) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	sig, err := hexutil.Decode(b.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Signature")
	}
	slot, err := strconv.ParseUint(b.Message.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Slot")
	}
	proposerIndex, err := strconv.ParseUint(b.Message.ProposerIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.ProposerIndex")
	}
	parentRoot, err := hexutil.Decode(b.Message.ParentRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.ParentRoot")
	}
	stateRoot, err := hexutil.Decode(b.Message.StateRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.StateRoot")
	}
	randaoReveal, err := hexutil.Decode(b.Message.Body.RandaoReveal)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.RandaoReveal")
	}
	depositRoot, err := hexutil.Decode(b.Message.Body.Eth1Data.DepositRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(b.Message.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.DepositCount")
	}
	blockHash, err := hexutil.Decode(b.Message.Body.Eth1Data.BlockHash)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.BlockHash")
	}
	graffiti, err := hexutil.Decode(b.Message.Body.Graffiti)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Graffiti")
	}
	proposerSlashings, err := convertProposerSlashings(b.Message.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := convertAttesterSlashings(b.Message.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
	atts, err := convertAtts(b.Message.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := convertDeposits(b.Message.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := convertExits(b.Message.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}

	block := &eth.SignedBeaconBlock{
		Block: &eth.BeaconBlock{
			Slot:          primitives.Slot(slot),
			ProposerIndex: primitives.ValidatorIndex(proposerIndex),
			ParentRoot:    parentRoot,
			StateRoot:     stateRoot,
			Body: &eth.BeaconBlockBody{
				RandaoReveal: randaoReveal,
				Eth1Data: &eth.Eth1Data{
					DepositRoot:  depositRoot,
					DepositCount: depositCount,
					BlockHash:    blockHash,
				},
				Graffiti:          graffiti,
				ProposerSlashings: proposerSlashings,
				AttesterSlashings: attesterSlashings,
				Attestations:      atts,
				Deposits:          deposits,
				VoluntaryExits:    exits,
			},
		},
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_Phase0{Phase0: block}}, nil
}

func (b *SignedBeaconBlockAltair) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	sig, err := hexutil.Decode(b.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Signature")
	}
	slot, err := strconv.ParseUint(b.Message.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Slot")
	}
	proposerIndex, err := strconv.ParseUint(b.Message.ProposerIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.ProposerIndex")
	}
	parentRoot, err := hexutil.Decode(b.Message.ParentRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.ParentRoot")
	}
	stateRoot, err := hexutil.Decode(b.Message.StateRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.StateRoot")
	}
	randaoReveal, err := hexutil.Decode(b.Message.Body.RandaoReveal)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.RandaoReveal")
	}
	depositRoot, err := hexutil.Decode(b.Message.Body.Eth1Data.DepositRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(b.Message.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.DepositCount")
	}
	blockHash, err := hexutil.Decode(b.Message.Body.Eth1Data.BlockHash)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.BlockHash")
	}
	graffiti, err := hexutil.Decode(b.Message.Body.Graffiti)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Graffiti")
	}
	proposerSlashings, err := convertProposerSlashings(b.Message.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := convertAttesterSlashings(b.Message.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
	atts, err := convertAtts(b.Message.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := convertDeposits(b.Message.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := convertExits(b.Message.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	syncCommitteeBits, err := bytesutil.FromHexString(b.Message.Body.SyncAggregate.SyncCommitteeBits)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := hexutil.Decode(b.Message.Body.SyncAggregate.SyncCommitteeSignature)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.SyncAggregate.SyncCommitteeSignature")
	}

	block := &eth.SignedBeaconBlockAltair{
		Block: &eth.BeaconBlockAltair{
			Slot:          primitives.Slot(slot),
			ProposerIndex: primitives.ValidatorIndex(proposerIndex),
			ParentRoot:    parentRoot,
			StateRoot:     stateRoot,
			Body: &eth.BeaconBlockBodyAltair{
				RandaoReveal: randaoReveal,
				Eth1Data: &eth.Eth1Data{
					DepositRoot:  depositRoot,
					DepositCount: depositCount,
					BlockHash:    blockHash,
				},
				Graffiti:          graffiti,
				ProposerSlashings: proposerSlashings,
				AttesterSlashings: attesterSlashings,
				Attestations:      atts,
				Deposits:          deposits,
				VoluntaryExits:    exits,
				SyncAggregate: &eth.SyncAggregate{
					SyncCommitteeBits:      syncCommitteeBits,
					SyncCommitteeSignature: syncCommitteeSig,
				},
			},
		},
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_Altair{Altair: block}}, nil
}

func (b *SignedBeaconBlockBellatrix) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	sig, err := hexutil.Decode(b.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Signature")
	}
	slot, err := strconv.ParseUint(b.Message.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Slot")
	}
	proposerIndex, err := strconv.ParseUint(b.Message.ProposerIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.ProposerIndex")
	}
	parentRoot, err := hexutil.Decode(b.Message.ParentRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.ParentRoot")
	}
	stateRoot, err := hexutil.Decode(b.Message.StateRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.StateRoot")
	}
	randaoReveal, err := hexutil.Decode(b.Message.Body.RandaoReveal)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.RandaoReveal")
	}
	depositRoot, err := hexutil.Decode(b.Message.Body.Eth1Data.DepositRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(b.Message.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.DepositCount")
	}
	blockHash, err := hexutil.Decode(b.Message.Body.Eth1Data.BlockHash)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.BlockHash")
	}
	graffiti, err := hexutil.Decode(b.Message.Body.Graffiti)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Graffiti")
	}
	proposerSlashings, err := convertProposerSlashings(b.Message.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := convertAttesterSlashings(b.Message.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
	atts, err := convertAtts(b.Message.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := convertDeposits(b.Message.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := convertExits(b.Message.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	syncCommitteeBits, err := bytesutil.FromHexString(b.Message.Body.SyncAggregate.SyncCommitteeBits)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := hexutil.Decode(b.Message.Body.SyncAggregate.SyncCommitteeSignature)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := hexutil.Decode(b.Message.Body.ExecutionPayload.ParentHash)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.ParentHash")
	}
	payloadFeeRecipient, err := hexutil.Decode(b.Message.Body.ExecutionPayload.FeeRecipient)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.FeeRecipient")
	}
	payloadStateRoot, err := hexutil.Decode(b.Message.Body.ExecutionPayload.StateRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.StateRoot")
	}
	payloadReceiptsRoot, err := hexutil.Decode(b.Message.Body.ExecutionPayload.ReceiptsRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.ReceiptsRoot")
	}
	payloadLogsBloom, err := hexutil.Decode(b.Message.Body.ExecutionPayload.LogsBloom)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.LogsBloom")
	}
	payloadPrevRandao, err := hexutil.Decode(b.Message.Body.ExecutionPayload.PrevRandao)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(b.Message.Body.ExecutionPayload.BlockNumber, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(b.Message.Body.ExecutionPayload.GasLimit, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(b.Message.Body.ExecutionPayload.GasUsed, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(b.Message.Body.ExecutionPayload.Timestamp, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.Timestamp")
	}
	payloadExtraData, err := hexutil.Decode(b.Message.Body.ExecutionPayload.ExtraData)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.ExtraData")
	}
	payloadBaseFeePerGas, err := uint256ToHex(b.Message.Body.ExecutionPayload.BaseFeePerGas)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.BaseFeePerGas")
	}
	payloadBlockHash, err := hexutil.Decode(b.Message.Body.ExecutionPayload.BlockHash)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.BlockHash")
	}
	payloadTxs := make([][]byte, len(b.Message.Body.ExecutionPayload.Transactions))
	for i, tx := range b.Message.Body.ExecutionPayload.Transactions {
		payloadTxs[i], err = hexutil.Decode(tx)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ExecutionPayload.Transactions[%d]", i)
		}
	}

	block := &eth.SignedBeaconBlockBellatrix{
		Block: &eth.BeaconBlockBellatrix{
			Slot:          primitives.Slot(slot),
			ProposerIndex: primitives.ValidatorIndex(proposerIndex),
			ParentRoot:    parentRoot,
			StateRoot:     stateRoot,
			Body: &eth.BeaconBlockBodyBellatrix{
				RandaoReveal: randaoReveal,
				Eth1Data: &eth.Eth1Data{
					DepositRoot:  depositRoot,
					DepositCount: depositCount,
					BlockHash:    blockHash,
				},
				Graffiti:          graffiti,
				ProposerSlashings: proposerSlashings,
				AttesterSlashings: attesterSlashings,
				Attestations:      atts,
				Deposits:          deposits,
				VoluntaryExits:    exits,
				SyncAggregate: &eth.SyncAggregate{
					SyncCommitteeBits:      syncCommitteeBits,
					SyncCommitteeSignature: syncCommitteeSig,
				},
				ExecutionPayload: &enginev1.ExecutionPayload{
					ParentHash:    payloadParentHash,
					FeeRecipient:  payloadFeeRecipient,
					StateRoot:     payloadStateRoot,
					ReceiptsRoot:  payloadReceiptsRoot,
					LogsBloom:     payloadLogsBloom,
					PrevRandao:    payloadPrevRandao,
					BlockNumber:   payloadBlockNumber,
					GasLimit:      payloadGasLimit,
					GasUsed:       payloadGasUsed,
					Timestamp:     payloadTimestamp,
					ExtraData:     payloadExtraData,
					BaseFeePerGas: payloadBaseFeePerGas,
					BlockHash:     payloadBlockHash,
					Transactions:  payloadTxs,
				},
			},
		},
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_Bellatrix{Bellatrix: block}}, nil
}

func (b *SignedBlindedBeaconBlockBellatrix) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	sig, err := hexutil.Decode(b.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Signature")
	}
	slot, err := strconv.ParseUint(b.Message.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Slot")
	}
	proposerIndex, err := strconv.ParseUint(b.Message.ProposerIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.ProposerIndex")
	}
	parentRoot, err := hexutil.Decode(b.Message.ParentRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.ParentRoot")
	}
	stateRoot, err := hexutil.Decode(b.Message.StateRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.StateRoot")
	}
	randaoReveal, err := hexutil.Decode(b.Message.Body.RandaoReveal)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.RandaoReveal")
	}
	depositRoot, err := hexutil.Decode(b.Message.Body.Eth1Data.DepositRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(b.Message.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.DepositCount")
	}
	blockHash, err := hexutil.Decode(b.Message.Body.Eth1Data.BlockHash)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.BlockHash")
	}
	graffiti, err := hexutil.Decode(b.Message.Body.Graffiti)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Graffiti")
	}
	proposerSlashings, err := convertProposerSlashings(b.Message.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := convertAttesterSlashings(b.Message.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
	atts, err := convertAtts(b.Message.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := convertDeposits(b.Message.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := convertExits(b.Message.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	syncCommitteeBits, err := bytesutil.FromHexString(b.Message.Body.SyncAggregate.SyncCommitteeBits)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := hexutil.Decode(b.Message.Body.SyncAggregate.SyncCommitteeSignature)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.ParentHash)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.ParentHash")
	}
	payloadFeeRecipient, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.FeeRecipient)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.FeeRecipient")
	}
	payloadStateRoot, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.StateRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.StateRoot")
	}
	payloadReceiptsRoot, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.ReceiptsRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.ReceiptsRoot")
	}
	payloadLogsBloom, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.LogsBloom)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.LogsBloom")
	}
	payloadPrevRandao, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.PrevRandao)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(b.Message.Body.ExecutionPayloadHeader.BlockNumber, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(b.Message.Body.ExecutionPayloadHeader.GasLimit, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(b.Message.Body.ExecutionPayloadHeader.GasUsed, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(b.Message.Body.ExecutionPayloadHeader.Timestamp, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.Timestamp")
	}
	payloadExtraData, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.ExtraData)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.ExtraData")
	}
	payloadBaseFeePerGas, err := uint256ToHex(b.Message.Body.ExecutionPayloadHeader.BaseFeePerGas)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.BaseFeePerGas")
	}
	payloadBlockHash, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.BlockHash)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.BlockHash")
	}
	payloadTxsRoot, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.TransactionsRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.TransactionsRoot")
	}

	block := &eth.SignedBlindedBeaconBlockBellatrix{
		Block: &eth.BlindedBeaconBlockBellatrix{
			Slot:          primitives.Slot(slot),
			ProposerIndex: primitives.ValidatorIndex(proposerIndex),
			ParentRoot:    parentRoot,
			StateRoot:     stateRoot,
			Body: &eth.BlindedBeaconBlockBodyBellatrix{
				RandaoReveal: randaoReveal,
				Eth1Data: &eth.Eth1Data{
					DepositRoot:  depositRoot,
					DepositCount: depositCount,
					BlockHash:    blockHash,
				},
				Graffiti:          graffiti,
				ProposerSlashings: proposerSlashings,
				AttesterSlashings: attesterSlashings,
				Attestations:      atts,
				Deposits:          deposits,
				VoluntaryExits:    exits,
				SyncAggregate: &eth.SyncAggregate{
					SyncCommitteeBits:      syncCommitteeBits,
					SyncCommitteeSignature: syncCommitteeSig,
				},
				ExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader{
					ParentHash:       payloadParentHash,
					FeeRecipient:     payloadFeeRecipient,
					StateRoot:        payloadStateRoot,
					ReceiptsRoot:     payloadReceiptsRoot,
					LogsBloom:        payloadLogsBloom,
					PrevRandao:       payloadPrevRandao,
					BlockNumber:      payloadBlockNumber,
					GasLimit:         payloadGasLimit,
					GasUsed:          payloadGasUsed,
					Timestamp:        payloadTimestamp,
					ExtraData:        payloadExtraData,
					BaseFeePerGas:    payloadBaseFeePerGas,
					BlockHash:        payloadBlockHash,
					TransactionsRoot: payloadTxsRoot,
				},
			},
		},
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_BlindedBellatrix{BlindedBellatrix: block}}, nil
}

func (b *SignedBeaconBlockCapella) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	sig, err := hexutil.Decode(b.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Signature")
	}
	slot, err := strconv.ParseUint(b.Message.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Slot")
	}
	proposerIndex, err := strconv.ParseUint(b.Message.ProposerIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.ProposerIndex")
	}
	parentRoot, err := hexutil.Decode(b.Message.ParentRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.ParentRoot")
	}
	stateRoot, err := hexutil.Decode(b.Message.StateRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.StateRoot")
	}
	randaoReveal, err := hexutil.Decode(b.Message.Body.RandaoReveal)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.RandaoReveal")
	}
	depositRoot, err := hexutil.Decode(b.Message.Body.Eth1Data.DepositRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(b.Message.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.DepositCount")
	}
	blockHash, err := hexutil.Decode(b.Message.Body.Eth1Data.BlockHash)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.BlockHash")
	}
	graffiti, err := hexutil.Decode(b.Message.Body.Graffiti)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Graffiti")
	}
	proposerSlashings, err := convertProposerSlashings(b.Message.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := convertAttesterSlashings(b.Message.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
	atts, err := convertAtts(b.Message.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := convertDeposits(b.Message.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := convertExits(b.Message.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	syncCommitteeBits, err := bytesutil.FromHexString(b.Message.Body.SyncAggregate.SyncCommitteeBits)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := hexutil.Decode(b.Message.Body.SyncAggregate.SyncCommitteeSignature)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := hexutil.Decode(b.Message.Body.ExecutionPayload.ParentHash)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.ParentHash")
	}
	payloadFeeRecipient, err := hexutil.Decode(b.Message.Body.ExecutionPayload.FeeRecipient)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.FeeRecipient")
	}
	payloadStateRoot, err := hexutil.Decode(b.Message.Body.ExecutionPayload.StateRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.StateRoot")
	}
	payloadReceiptsRoot, err := hexutil.Decode(b.Message.Body.ExecutionPayload.ReceiptsRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.ReceiptsRoot")
	}
	payloadLogsBloom, err := hexutil.Decode(b.Message.Body.ExecutionPayload.LogsBloom)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.LogsBloom")
	}
	payloadPrevRandao, err := hexutil.Decode(b.Message.Body.ExecutionPayload.PrevRandao)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(b.Message.Body.ExecutionPayload.BlockNumber, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(b.Message.Body.ExecutionPayload.GasLimit, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(b.Message.Body.ExecutionPayload.GasUsed, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(b.Message.Body.ExecutionPayload.Timestamp, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.Timestamp")
	}
	payloadExtraData, err := hexutil.Decode(b.Message.Body.ExecutionPayload.ExtraData)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.ExtraData")
	}
	payloadBaseFeePerGas, err := uint256ToHex(b.Message.Body.ExecutionPayload.BaseFeePerGas)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.BaseFeePerGas")
	}
	payloadBlockHash, err := hexutil.Decode(b.Message.Body.ExecutionPayload.BlockHash)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayload.BlockHash")
	}
	txs := make([][]byte, len(b.Message.Body.ExecutionPayload.Transactions))
	for i, tx := range b.Message.Body.ExecutionPayload.Transactions {
		txs[i], err = hexutil.Decode(tx)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ExecutionPayload.Transactions[%d]", i)
		}
	}
	withdrawals := make([]*enginev1.Withdrawal, len(b.Message.Body.ExecutionPayload.Withdrawals))
	for i, w := range b.Message.Body.ExecutionPayload.Withdrawals {
		withdrawalIndex, err := strconv.ParseUint(w.WithdrawalIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ExecutionPayload.Withdrawals[%d].WithdrawalIndex", i)
		}
		validatorIndex, err := strconv.ParseUint(w.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ExecutionPayload.Withdrawals[%d].ValidatorIndex", i)
		}
		address, err := hexutil.Decode(w.ExecutionAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ExecutionPayload.Withdrawals[%d].ExecutionAddress", i)
		}
		amount, err := strconv.ParseUint(w.Amount, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ExecutionPayload.Withdrawals[%d].Amount", i)
		}
		withdrawals[i] = &enginev1.Withdrawal{
			Index:          withdrawalIndex,
			ValidatorIndex: primitives.ValidatorIndex(validatorIndex),
			Address:        address,
			Amount:         amount,
		}
	}
	blsChanges := make([]*eth.SignedBLSToExecutionChange, len(b.Message.Body.BlsToExecutionChanges))

	block := &eth.SignedBeaconBlockCapella{
		Block: &eth.BeaconBlockCapella{
			Slot:          primitives.Slot(slot),
			ProposerIndex: primitives.ValidatorIndex(proposerIndex),
			ParentRoot:    parentRoot,
			StateRoot:     stateRoot,
			Body: &eth.BeaconBlockBodyCapella{
				RandaoReveal: randaoReveal,
				Eth1Data: &eth.Eth1Data{
					DepositRoot:  depositRoot,
					DepositCount: depositCount,
					BlockHash:    blockHash,
				},
				Graffiti:          graffiti,
				ProposerSlashings: proposerSlashings,
				AttesterSlashings: attesterSlashings,
				Attestations:      atts,
				Deposits:          deposits,
				VoluntaryExits:    exits,
				SyncAggregate: &eth.SyncAggregate{
					SyncCommitteeBits:      syncCommitteeBits,
					SyncCommitteeSignature: syncCommitteeSig,
				},
				ExecutionPayload: &enginev1.ExecutionPayloadCapella{
					ParentHash:    payloadParentHash,
					FeeRecipient:  payloadFeeRecipient,
					StateRoot:     payloadStateRoot,
					ReceiptsRoot:  payloadReceiptsRoot,
					LogsBloom:     payloadLogsBloom,
					PrevRandao:    payloadPrevRandao,
					BlockNumber:   payloadBlockNumber,
					GasLimit:      payloadGasLimit,
					GasUsed:       payloadGasUsed,
					Timestamp:     payloadTimestamp,
					ExtraData:     payloadExtraData,
					BaseFeePerGas: payloadBaseFeePerGas,
					BlockHash:     payloadBlockHash,
					Transactions:  txs,
					Withdrawals:   withdrawals,
				},
				BlsToExecutionChanges: blsChanges,
			},
		},
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_Capella{Capella: block}}, nil
}

func (b *SignedBlindedBeaconBlockCapella) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	sig, err := hexutil.Decode(b.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Signature")
	}
	slot, err := strconv.ParseUint(b.Message.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Slot")
	}
	proposerIndex, err := strconv.ParseUint(b.Message.ProposerIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.ProposerIndex")
	}
	parentRoot, err := hexutil.Decode(b.Message.ParentRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.ParentRoot")
	}
	stateRoot, err := hexutil.Decode(b.Message.StateRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.StateRoot")
	}
	randaoReveal, err := hexutil.Decode(b.Message.Body.RandaoReveal)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.RandaoReveal")
	}
	depositRoot, err := hexutil.Decode(b.Message.Body.Eth1Data.DepositRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(b.Message.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.DepositCount")
	}
	blockHash, err := hexutil.Decode(b.Message.Body.Eth1Data.BlockHash)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Eth1Data.BlockHash")
	}
	graffiti, err := hexutil.Decode(b.Message.Body.Graffiti)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.Graffiti")
	}
	proposerSlashings, err := convertProposerSlashings(b.Message.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := convertAttesterSlashings(b.Message.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
	atts, err := convertAtts(b.Message.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := convertDeposits(b.Message.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := convertExits(b.Message.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	syncCommitteeBits, err := bytesutil.FromHexString(b.Message.Body.SyncAggregate.SyncCommitteeBits)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := hexutil.Decode(b.Message.Body.SyncAggregate.SyncCommitteeSignature)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.ParentHash)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.ParentHash")
	}
	payloadFeeRecipient, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.FeeRecipient)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.FeeRecipient")
	}
	payloadStateRoot, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.StateRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.StateRoot")
	}
	payloadReceiptsRoot, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.ReceiptsRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.ReceiptsRoot")
	}
	payloadLogsBloom, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.LogsBloom)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.LogsBloom")
	}
	payloadPrevRandao, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.PrevRandao)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(b.Message.Body.ExecutionPayloadHeader.BlockNumber, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(b.Message.Body.ExecutionPayloadHeader.GasLimit, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(b.Message.Body.ExecutionPayloadHeader.GasUsed, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(b.Message.Body.ExecutionPayloadHeader.Timestamp, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.Timestamp")
	}
	payloadExtraData, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.ExtraData)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.ExtraData")
	}
	payloadBaseFeePerGas, err := uint256ToHex(b.Message.Body.ExecutionPayloadHeader.BaseFeePerGas)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.BaseFeePerGas")
	}
	payloadBlockHash, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.BlockHash)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.BlockHash")
	}
	payloadTxsRoot, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.TransactionsRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.TransactionsRoot")
	}
	payloadWithdrawalsRoot, err := hexutil.Decode(b.Message.Body.ExecutionPayloadHeader.WithdrawalsRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode b.Message.Body.ExecutionPayloadHeader.WithdrawalsRoot")
	}
	blsChanges := make([]*eth.SignedBLSToExecutionChange, len(b.Message.Body.BlsToExecutionChanges))

	block := &eth.SignedBlindedBeaconBlockCapella{
		Block: &eth.BlindedBeaconBlockCapella{
			Slot:          primitives.Slot(slot),
			ProposerIndex: primitives.ValidatorIndex(proposerIndex),
			ParentRoot:    parentRoot,
			StateRoot:     stateRoot,
			Body: &eth.BlindedBeaconBlockBodyCapella{
				RandaoReveal: randaoReveal,
				Eth1Data: &eth.Eth1Data{
					DepositRoot:  depositRoot,
					DepositCount: depositCount,
					BlockHash:    blockHash,
				},
				Graffiti:          graffiti,
				ProposerSlashings: proposerSlashings,
				AttesterSlashings: attesterSlashings,
				Attestations:      atts,
				Deposits:          deposits,
				VoluntaryExits:    exits,
				SyncAggregate: &eth.SyncAggregate{
					SyncCommitteeBits:      syncCommitteeBits,
					SyncCommitteeSignature: syncCommitteeSig,
				},
				ExecutionPayloadHeader: &enginev1.ExecutionPayloadHeaderCapella{
					ParentHash:       payloadParentHash,
					FeeRecipient:     payloadFeeRecipient,
					StateRoot:        payloadStateRoot,
					ReceiptsRoot:     payloadReceiptsRoot,
					LogsBloom:        payloadLogsBloom,
					PrevRandao:       payloadPrevRandao,
					BlockNumber:      payloadBlockNumber,
					GasLimit:         payloadGasLimit,
					GasUsed:          payloadGasUsed,
					Timestamp:        payloadTimestamp,
					ExtraData:        payloadExtraData,
					BaseFeePerGas:    payloadBaseFeePerGas,
					BlockHash:        payloadBlockHash,
					TransactionsRoot: payloadTxsRoot,
					WithdrawalsRoot:  payloadWithdrawalsRoot,
				},
				BlsToExecutionChanges: blsChanges,
			},
		},
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_BlindedCapella{BlindedCapella: block}}, nil
}

func convertProposerSlashings(src []ProposerSlashing) ([]*eth.ProposerSlashing, error) {
	if src == nil {
		return nil, errors.New("nil b.Message.Body.ProposerSlashings")
	}

	proposerSlashings := make([]*eth.ProposerSlashing, len(src))
	for i, s := range src {
		h1Sig, err := hexutil.Decode(s.SignedHeader1.Signature)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ProposerSlashings[%d].SignedHeader1.Signature", i)
		}
		h1Slot, err := strconv.ParseUint(s.SignedHeader1.Message.Slot, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ProposerSlashings[%d].SignedHeader1.Message.Slot", i)
		}
		h1ProposerIndex, err := strconv.ParseUint(s.SignedHeader1.Message.ProposerIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ProposerSlashings[%d].SignedHeader1.Message.ProposerIndex", i)
		}
		h1ParentRoot, err := hexutil.Decode(s.SignedHeader1.Message.ParentRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ProposerSlashings[%d].SignedHeader1.Message.ParentRoot", i)
		}
		h1StateRoot, err := hexutil.Decode(s.SignedHeader1.Message.StateRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ProposerSlashings[%d].SignedHeader1.Message.StateRoot", i)
		}
		h1BodyRoot, err := hexutil.Decode(s.SignedHeader1.Message.BodyRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ProposerSlashings[%d].SignedHeader1.Message.BodyRoot", i)
		}
		h2Sig, err := hexutil.Decode(s.SignedHeader2.Signature)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ProposerSlashings[%d].SignedHeader2.Signature", i)
		}
		h2Slot, err := strconv.ParseUint(s.SignedHeader2.Message.Slot, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ProposerSlashings[%d].SignedHeader2.Message.Slot", i)
		}
		h2ProposerIndex, err := strconv.ParseUint(s.SignedHeader2.Message.ProposerIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ProposerSlashings[%d].SignedHeader2.Message.ProposerIndex", i)
		}
		h2ParentRoot, err := hexutil.Decode(s.SignedHeader2.Message.ParentRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ProposerSlashings[%d].SignedHeader2.Message.ParentRoot", i)
		}
		h2StateRoot, err := hexutil.Decode(s.SignedHeader2.Message.StateRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ProposerSlashings[%d].SignedHeader2.Message.StateRoot", i)
		}
		h2BodyRoot, err := hexutil.Decode(s.SignedHeader2.Message.BodyRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.ProposerSlashings[%d].SignedHeader2.Message.BodyRoot", i)
		}
		proposerSlashings[i] = &eth.ProposerSlashing{
			Header_1: &eth.SignedBeaconBlockHeader{
				Header: &eth.BeaconBlockHeader{
					Slot:          primitives.Slot(h1Slot),
					ProposerIndex: primitives.ValidatorIndex(h1ProposerIndex),
					ParentRoot:    h1ParentRoot,
					StateRoot:     h1StateRoot,
					BodyRoot:      h1BodyRoot,
				},
				Signature: h1Sig,
			},
			Header_2: &eth.SignedBeaconBlockHeader{
				Header: &eth.BeaconBlockHeader{
					Slot:          primitives.Slot(h2Slot),
					ProposerIndex: primitives.ValidatorIndex(h2ProposerIndex),
					ParentRoot:    h2ParentRoot,
					StateRoot:     h2StateRoot,
					BodyRoot:      h2BodyRoot,
				},
				Signature: h2Sig,
			},
		}
	}
	return proposerSlashings, nil
}

func convertAttesterSlashings(src []AttesterSlashing) ([]*eth.AttesterSlashing, error) {
	if src == nil {
		return nil, errors.New("nil b.Message.Body.AttesterSlashings")
	}

	attesterSlashings := make([]*eth.AttesterSlashing, len(src))
	for i, s := range src {
		a1Sig, err := hexutil.Decode(s.Attestation1.Signature)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation1.Signature", i)
		}
		a1AttestingIndices := make([]uint64, len(s.Attestation1.AttestingIndices))
		for j, ix := range s.Attestation1.AttestingIndices {
			attestingIndex, err := strconv.ParseUint(ix, 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation1.AttestingIndices[%d]", i, j)
			}
			a1AttestingIndices[j] = attestingIndex
		}
		a1Slot, err := strconv.ParseUint(s.Attestation1.Data.Slot, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation1.Data.Slot", i)
		}
		a1CommitteeIndex, err := strconv.ParseUint(s.Attestation1.Data.Index, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation1.Data.Index", i)
		}
		a1BeaconBlockRoot, err := hexutil.Decode(s.Attestation1.Data.BeaconBlockRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation1.Data.BeaconBlockRoot", i)
		}
		a1SourceEpoch, err := strconv.ParseUint(s.Attestation1.Data.Source.Epoch, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation1.Data.Source.Epoch", i)
		}
		a1SourceRoot, err := hexutil.Decode(s.Attestation1.Data.Source.Root)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation1.Data.Source.Root", i)
		}
		a1TargetEpoch, err := strconv.ParseUint(s.Attestation1.Data.Target.Epoch, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation1.Data.Target.Epoch", i)
		}
		a1TargetRoot, err := hexutil.Decode(s.Attestation1.Data.Target.Root)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation1.Data.Target.Root", i)
		}
		a2Sig, err := hexutil.Decode(s.Attestation2.Signature)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation2.Signature", i)
		}
		a2AttestingIndices := make([]uint64, len(s.Attestation2.AttestingIndices))
		for j, ix := range s.Attestation2.AttestingIndices {
			attestingIndex, err := strconv.ParseUint(ix, 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation2.AttestingIndices[%d]", i, j)
			}
			a2AttestingIndices[j] = attestingIndex
		}
		a2Slot, err := strconv.ParseUint(s.Attestation2.Data.Slot, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation2.Data.Slot", i)
		}
		a2CommitteeIndex, err := strconv.ParseUint(s.Attestation2.Data.Index, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation2.Data.Index", i)
		}
		a2BeaconBlockRoot, err := hexutil.Decode(s.Attestation2.Data.BeaconBlockRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation2.Data.BeaconBlockRoot", i)
		}
		a2SourceEpoch, err := strconv.ParseUint(s.Attestation2.Data.Source.Epoch, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation2.Data.Source.Epoch", i)
		}
		a2SourceRoot, err := hexutil.Decode(s.Attestation2.Data.Source.Root)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation2.Data.Source.Root", i)
		}
		a2TargetEpoch, err := strconv.ParseUint(s.Attestation2.Data.Target.Epoch, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation2.Data.Target.Epoch", i)
		}
		a2TargetRoot, err := hexutil.Decode(s.Attestation2.Data.Target.Root)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.AttesterSlashings[%d].Attestation2.Data.Target.Root", i)
		}
		attesterSlashings[i] = &eth.AttesterSlashing{
			Attestation_1: &eth.IndexedAttestation{
				AttestingIndices: a1AttestingIndices,
				Data: &eth.AttestationData{
					Slot:            primitives.Slot(a1Slot),
					CommitteeIndex:  primitives.CommitteeIndex(a1CommitteeIndex),
					BeaconBlockRoot: a1BeaconBlockRoot,
					Source: &eth.Checkpoint{
						Epoch: primitives.Epoch(a1SourceEpoch),
						Root:  a1SourceRoot,
					},
					Target: &eth.Checkpoint{
						Epoch: primitives.Epoch(a1TargetEpoch),
						Root:  a1TargetRoot,
					},
				},
				Signature: a1Sig,
			},
			Attestation_2: &eth.IndexedAttestation{
				AttestingIndices: a2AttestingIndices,
				Data: &eth.AttestationData{
					Slot:            primitives.Slot(a2Slot),
					CommitteeIndex:  primitives.CommitteeIndex(a2CommitteeIndex),
					BeaconBlockRoot: a2BeaconBlockRoot,
					Source: &eth.Checkpoint{
						Epoch: primitives.Epoch(a2SourceEpoch),
						Root:  a2SourceRoot,
					},
					Target: &eth.Checkpoint{
						Epoch: primitives.Epoch(a2TargetEpoch),
						Root:  a2TargetRoot,
					},
				},
				Signature: a2Sig,
			},
		}
	}
	return attesterSlashings, nil
}

func convertAtts(src []Attestation) ([]*eth.Attestation, error) {
	if src == nil {
		return nil, errors.New("nil b.Message.Body.Attestations")
	}

	atts := make([]*eth.Attestation, len(src))
	for i, a := range src {
		sig, err := hexutil.Decode(a.Signature)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.Attestations[%d].Signature", i)
		}
		slot, err := strconv.ParseUint(a.Data.Slot, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.Attestations[%d].Data.Slot", i)
		}
		committeeIndex, err := strconv.ParseUint(a.Data.Index, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.Attestations[%d].Data.Index", i)
		}
		beaconBlockRoot, err := hexutil.Decode(a.Data.BeaconBlockRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.Attestations[%d].Data.BeaconBlockRoot", i)
		}
		sourceEpoch, err := strconv.ParseUint(a.Data.Source.Epoch, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.Attestations[%d].Data.Source.Epoch", i)
		}
		sourceRoot, err := hexutil.Decode(a.Data.Source.Root)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.Attestations[%d].Data.Source.Root", i)
		}
		targetEpoch, err := strconv.ParseUint(a.Data.Target.Epoch, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.Attestations[%d].Data.Target.Epoch", i)
		}
		targetRoot, err := hexutil.Decode(a.Data.Target.Root)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.Attestations[%d].Data.Target.Root", i)
		}
		atts[i] = &eth.Attestation{
			AggregationBits: []byte(a.AggregationBits),
			Data: &eth.AttestationData{
				Slot:            primitives.Slot(slot),
				CommitteeIndex:  primitives.CommitteeIndex(committeeIndex),
				BeaconBlockRoot: beaconBlockRoot,
				Source: &eth.Checkpoint{
					Epoch: primitives.Epoch(sourceEpoch),
					Root:  sourceRoot,
				},
				Target: &eth.Checkpoint{
					Epoch: primitives.Epoch(targetEpoch),
					Root:  targetRoot,
				},
			},
			Signature: sig,
		}
	}
	return atts, nil
}

func convertDeposits(src []Deposit) ([]*eth.Deposit, error) {
	if src == nil {
		return nil, errors.New("nil b.Message.Body.Deposits")
	}

	deposits := make([]*eth.Deposit, len(src))
	for i, d := range src {
		proof := make([][]byte, len(d.Proof))
		for j, p := range d.Proof {
			var err error
			proof[j], err = hexutil.Decode(p)
			if err != nil {
				return nil, errors.Wrapf(err, "could not decode b.Message.Body.Deposits[%d].Proof[%d]", i, j)
			}
		}
		pubkey, err := hexutil.Decode(d.Data.Pubkey)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.Deposits[%d].Pubkey", i)
		}
		withdrawalCreds, err := hexutil.Decode(d.Data.WithdrawalCredentials)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.Deposits[%d].WithdrawalCredentials", i)
		}
		amount, err := strconv.ParseUint(d.Data.Amount, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.Deposits[%d].Amount", i)
		}
		sig, err := hexutil.Decode(d.Data.Signature)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.Deposits[%d].Signature", i)
		}
		deposits[i] = &eth.Deposit{
			Proof: proof,
			Data: &eth.Deposit_Data{
				PublicKey:             pubkey,
				WithdrawalCredentials: withdrawalCreds,
				Amount:                amount,
				Signature:             sig,
			},
		}
	}
	return deposits, nil
}

func convertExits(src []SignedVoluntaryExit) ([]*eth.SignedVoluntaryExit, error) {
	if src == nil {
		return nil, errors.New("nil b.Message.Body.VoluntaryExits")
	}

	exits := make([]*eth.SignedVoluntaryExit, len(src))
	for i, e := range src {
		sig, err := hexutil.Decode(e.Signature)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.VoluntaryExits[%d].Signature", i)
		}
		epoch, err := strconv.ParseUint(e.Message.Epoch, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.VoluntaryExits[%d].Epoch", i)
		}
		validatorIndex, err := strconv.ParseUint(e.Message.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.VoluntaryExits[%d].ValidatorIndex", i)
		}
		exits[i] = &eth.SignedVoluntaryExit{
			Exit: &eth.VoluntaryExit{
				Epoch:          primitives.Epoch(epoch),
				ValidatorIndex: primitives.ValidatorIndex(validatorIndex),
			},
			Signature: sig,
		}
	}
	return exits, nil
}

func uint256ToHex(num string) ([]byte, error) {
	uint256, ok := new(big.Int).SetString(num, 10)
	if !ok {
		return nil, errors.New("could not parse Uint256")
	}
	bigEndian := uint256.Bytes()
	if len(bigEndian) > 32 {
		return nil, errors.New("number too big for Uint256")
	}
	return bytesutil2.ReverseByteOrder(bigEndian), nil
}
