package beacon

import (
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type SignedBeaconBlock struct {
	Message   BeaconBlock `json:"message" validate:"required" validate:"required"`
	Signature string      `json:"signature" validate:"required" validate:"required"`
}

type BeaconBlock struct {
	Slot          string          `json:"slot" validate:"required" validate:"required"`
	ProposerIndex string          `json:"proposer_index" validate:"required" validate:"required"`
	ParentRoot    string          `json:"parent_root" validate:"required" validate:"required"`
	StateRoot     string          `json:"state_root" validate:"required" validate:"required"`
	Body          BeaconBlockBody `json:"body" validate:"required" validate:"required"`
}

type BeaconBlockBody struct {
	RandaoReveal      string                `json:"randao_reveal" validate:"required" validate:"required"`
	Eth1Data          Eth1Data              `json:"eth1_data" validate:"required"`
	Graffiti          string                `json:"graffiti" validate:"required"`
	ProposerSlashings []ProposerSlashing    `json:"proposer_slashings" validate:"required"`
	AttesterSlashings []AttesterSlashing    `json:"attester_slashings" validate:"required"`
	Attestations      []Attestation         `json:"attestations" validate:"required"`
	Deposits          []Deposit             `json:"deposits" validate:"required"`
	VoluntaryExits    []SignedVoluntaryExit `json:"voluntary_exits" validate:"required"`
}

type SignedBeaconBlockAltair struct {
	Message   BeaconBlockAltair `json:"message" validate:"required"`
	Signature string            `json:"signature" validate:"required"`
}

type BeaconBlockAltair struct {
	Slot          string                `json:"slot" validate:"required"`
	ProposerIndex string                `json:"proposer_index" validate:"required"`
	ParentRoot    string                `json:"parent_root" validate:"required"`
	StateRoot     string                `json:"state_root" validate:"required"`
	Body          BeaconBlockBodyAltair `json:"body" validate:"required"`
}

type BeaconBlockBodyAltair struct {
	RandaoReveal      string                `json:"randao_reveal" validate:"required"`
	Eth1Data          Eth1Data              `json:"eth1_data" validate:"required"`
	Graffiti          string                `json:"graffiti" validate:"required"`
	ProposerSlashings []ProposerSlashing    `json:"proposer_slashings" validate:"required"`
	AttesterSlashings []AttesterSlashing    `json:"attester_slashings" validate:"required"`
	Attestations      []Attestation         `json:"attestations" validate:"required"`
	Deposits          []Deposit             `json:"deposits" validate:"required"`
	VoluntaryExits    []SignedVoluntaryExit `json:"voluntary_exits" validate:"required"`
	SyncAggregate     SyncAggregate         `json:"sync_aggregate" validate:"required"`
}

type SignedBeaconBlockBellatrix struct {
	Message   BeaconBlockBellatrix `json:"message" validate:"required"`
	Signature string               `json:"signature" validate:"required"`
}

type BeaconBlockBellatrix struct {
	Slot          string                   `json:"slot" validate:"required"`
	ProposerIndex string                   `json:"proposer_index" validate:"required"`
	ParentRoot    string                   `json:"parent_root" validate:"required"`
	StateRoot     string                   `json:"state_root" validate:"required"`
	Body          BeaconBlockBodyBellatrix `json:"body" validate:"required"`
}

type BeaconBlockBodyBellatrix struct {
	RandaoReveal      string                `json:"randao_reveal" validate:"required"`
	Eth1Data          Eth1Data              `json:"eth1_data" validate:"required"`
	Graffiti          string                `json:"graffiti" validate:"required"`
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
	Signature string                      `json:"signature" validate:"required"`
}

type BlindedBeaconBlockBellatrix struct {
	Slot          string                          `json:"slot" validate:"required"`
	ProposerIndex string                          `json:"proposer_index" validate:"required"`
	ParentRoot    string                          `json:"parent_root" validate:"required"`
	StateRoot     string                          `json:"state_root" validate:"required"`
	Body          BlindedBeaconBlockBodyBellatrix `json:"body" validate:"required"`
}

type BlindedBeaconBlockBodyBellatrix struct {
	RandaoReveal      string                 `json:"randao_reveal" validate:"required"`
	Eth1Data          Eth1Data               `json:"eth1_data" validate:"required"`
	Graffiti          string                 `json:"graffiti" validate:"required"`
	ProposerSlashings []ProposerSlashing     `json:"proposer_slashings" validate:"required"`
	AttesterSlashings []AttesterSlashing     `json:"attester_slashings" validate:"required"`
	Attestations      []Attestation          `json:"attestations" validate:"required"`
	Deposits          []Deposit              `json:"deposits" validate:"required"`
	VoluntaryExits    []SignedVoluntaryExit  `json:"voluntary_exits" validate:"required"`
	SyncAggregate     SyncAggregate          `json:"sync_aggregate" validate:"required"`
	ExecutionPayload  ExecutionPayloadHeader `json:"execution_payload_header" validate:"required"`
}

type SignedBeaconBlockCapella struct {
	Message   BeaconBlockCapella `json:"message" validate:"required"`
	Signature string             `json:"signature" validate:"required"`
}

type BeaconBlockCapella struct {
	Slot          string                 `json:"slot" validate:"required"`
	ProposerIndex string                 `json:"proposer_index" validate:"required"`
	ParentRoot    string                 `json:"parent_root" validate:"required"`
	StateRoot     string                 `json:"state_root" validate:"required"`
	Body          BeaconBlockBodyCapella `json:"body" validate:"required"`
}

type BeaconBlockBodyCapella struct {
	RandaoReveal      string                  `json:"randao_reveal" validate:"required"`
	Eth1Data          Eth1Data                `json:"eth1_data" validate:"required"`
	Graffiti          string                  `json:"graffiti" validate:"required"`
	ProposerSlashings []ProposerSlashing      `json:"proposer_slashings" validate:"required"`
	AttesterSlashings []AttesterSlashing      `json:"attester_slashings" validate:"required"`
	Attestations      []Attestation           `json:"attestations" validate:"required"`
	Deposits          []Deposit               `json:"deposits" validate:"required"`
	VoluntaryExits    []SignedVoluntaryExit   `json:"voluntary_exits" validate:"required"`
	SyncAggregate     SyncAggregate           `json:"sync_aggregate" validate:"required"`
	ExecutionPayload  ExecutionPayloadCapella `json:"execution_payload" validate:"required"`
}

type SignedBlindedBeaconBlockCapella struct {
	Message   BlindedBeaconBlockCapella `json:"message" validate:"required"`
	Signature string                    `json:"signature" validate:"required"`
}

type BlindedBeaconBlockCapella struct {
	Slot          string                        `json:"slot" validate:"required"`
	ProposerIndex string                        `json:"proposer_index" validate:"required"`
	ParentRoot    string                        `json:"parent_root" validate:"required"`
	StateRoot     string                        `json:"state_root" validate:"required"`
	Body          BlindedBeaconBlockBodyCapella `json:"body" validate:"required"`
}

type BlindedBeaconBlockBodyCapella struct {
	RandaoReveal      string                        `json:"randao_reveal" validate:"required"`
	Eth1Data          Eth1Data                      `json:"eth1_data" validate:"required"`
	Graffiti          string                        `json:"graffiti" validate:"required"`
	ProposerSlashings []ProposerSlashing            `json:"proposer_slashings" validate:"required"`
	AttesterSlashings []AttesterSlashing            `json:"attester_slashings" validate:"required"`
	Attestations      []Attestation                 `json:"attestations" validate:"required"`
	Deposits          []Deposit                     `json:"deposits" validate:"required"`
	VoluntaryExits    []SignedVoluntaryExit         `json:"voluntary_exits" validate:"required"`
	SyncAggregate     SyncAggregate                 `json:"sync_aggregate" validate:"required"`
	ExecutionPayload  ExecutionPayloadHeaderCapella `json:"execution_payload_header" validate:"required"`
}

type Eth1Data struct {
	DepositRoot  string `json:"deposit_root" validate:"required"`
	DepositCount string `json:"deposit_count" validate:"required"`
	BlockHash    string `json:"block_hash" validate:"required"`
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
	AggregationBits string          `json:"aggregation_bits" validate:"required"`
	Data            AttestationData `json:"data" validate:"required"`
	Signature       string          `json:"signature" validate:"required"`
}

type Deposit struct {
	Proof []string    `json:"proof" validate:"required"`
	Data  DepositData `json:"data" validate:"required"`
}

type DepositData struct {
	Pubkey                string `json:"pubkey" validate:"required"`
	WithdrawalCredentials string `json:"withdrawal_credentials" validate:"required"`
	Amount                string `json:"amount" validate:"required"`
	Signature             string `json:"signature" validate:"required"`
}

type SignedVoluntaryExit struct {
	Message   VoluntaryExit `json:"message" validate:"required"`
	Signature string        `json:"signature" validate:"required"`
}

type VoluntaryExit struct {
	Epoch          string `json:"epoch" validate:"required"`
	ValidatorIndex string `json:"validator_index" validate:"required"`
}

type SignedBeaconBlockHeader struct {
	Message   BeaconBlockHeader `json:"message" validate:"required"`
	Signature string            `json:"signature" validate:"required"`
}

type BeaconBlockHeader struct {
	Slot          string `json:"slot" validate:"required"`
	ProposerIndex string `json:"proposer_index" validate:"required"`
	ParentRoot    string `json:"parent_root" validate:"required"`
	StateRoot     string `json:"state_root" validate:"required"`
	BodyRoot      string `json:"body_root" validate:"required"`
}

type IndexedAttestation struct {
	AttestingIndices []string        `json:"attesting_indices" validate:"required"`
	Data             AttestationData `json:"data" validate:"required"`
	Signature        string          `json:"signature" validate:"required"`
}

type AttestationData struct {
	Slot            string     `json:"slot" validate:"required"`
	Index           string     `json:"index" validate:"required"`
	BeaconBlockRoot string     `json:"beacon_block_root" validate:"required"`
	Source          Checkpoint `json:"source" validate:"required"`
	Target          Checkpoint `json:"target" validate:"required"`
}

type Checkpoint struct {
	Epoch string `json:"epoch" validate:"required"`
	Root  string `json:"root" validate:"required"`
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
	TimeStamp     string   `json:"timestamp" validate:"required"`
	ExtraData     string   `json:"extra_data" validate:"required"`
	BaseFeePerGas string   `json:"base_fee_per_gas" validate:"required"` // TODO: Uint256
	BlockHash     string   `json:"block_hash" validate:"required"`
	Transactions  []string `json:"transactions" validate:"required"`
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
	TimeStamp        string `json:"timestamp" validate:"required"`
	ExtraData        string `json:"extra_data" validate:"required"`
	BaseFeePerGas    string `json:"base_fee_per_gas" validate:"required"`
	BlockHash        string `json:"block_hash" validate:"required"`
	TransactionsRoot string `json:"transactions_root" validate:"required"`
}

type ExecutionPayloadCapella struct {
	ParentHash    string       `json:"parent_hash" validate:"required"`
	FeeRecipient  string       `json:"fee_recipient" validate:"required"`
	StateRoot     string       `json:"state_root" validate:"required"`
	ReceiptsRoot  string       `json:"receipts_root" validate:"required"`
	LogsBloom     string       `json:"logs_bloom" validate:"required"`
	PrevRandao    string       `json:"prev_randao" validate:"required"`
	BlockNumber   string       `json:"block_number" validate:"required"`
	GasLimit      string       `json:"gas_limit" validate:"required"`
	GasUsed       string       `json:"gas_used" validate:"required"`
	TimeStamp     string       `json:"timestamp" validate:"required"`
	ExtraData     string       `json:"extra_data" validate:"required"`
	BaseFeePerGas string       `json:"base_fee_per_gas" validate:"required"`
	BlockHash     string       `json:"block_hash" validate:"required"`
	Transactions  []string     `json:"transactions" validate:"required"`
	Withdrawals   []Withdrawal `json:"withdrawals" validate:"required"`
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
	TimeStamp        string `json:"timestamp" validate:"required"`
	ExtraData        string `json:"extra_data" validate:"required"`
	BaseFeePerGas    string `json:"base_fee_per_gas" validate:"required"`
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

func (b *SignedBeaconBlock) ToConsensusReadOnly() (interfaces.ReadOnlySignedBeaconBlock, error) {
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
	proposerSlashings := make([]*eth.ProposerSlashing, len(b.Message.Body.ProposerSlashings))
	for i, s := range b.Message.Body.ProposerSlashings {
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
	attesterSlashings := make([]*eth.AttesterSlashing, len(b.Message.Body.AttesterSlashings))
	for i, s := range b.Message.Body.AttesterSlashings {
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
	atts := make([]*eth.Attestation, len(b.Message.Body.Attestations))
	for i, a := range b.Message.Body.Attestations {

		sig, err = hexutil.Decode(a.Signature)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.Attestations[%d].Signature", i)
		}
		slot, err = strconv.ParseUint(a.Data.Slot, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.Attestations[%d].Data.Slot", i)
		}
		committeeIndex, err := strconv.ParseUint(a.Data.Index, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode b.Message.Body.Attestations[%d].Data.Index", i)
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
				BeaconBlockRoot: nil,
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
	deposits := make([]*eth.Deposit, len(b.Message.Body.Deposits))
	for i, d := range b.Message.Body.Deposits {
		proof := make([][]byte, len(d.Proof))
		for j, p := range d.Proof {
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
		sig, err = hexutil.Decode(d.Data.Signature)
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
	exits := make([]*eth.SignedVoluntaryExit, len(b.Message.Body.VoluntaryExits))
	for i, e := range b.Message.Body.VoluntaryExits {
		sig, err = hexutil.Decode(e.Signature)
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
	return blocks.NewSignedBeaconBlock(block)
}
