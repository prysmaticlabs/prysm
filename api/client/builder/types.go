package builder

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	v1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

type SignedValidatorRegistration struct {
	*eth.SignedValidatorRegistrationV1
}

type ValidatorRegistration struct {
	*eth.ValidatorRegistrationV1
}

func (r *SignedValidatorRegistration) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Message   *ValidatorRegistration `json:"message,omitempty"`
		Signature hexSlice               `json:"signature,omitempty"`
	}{
		Message:   &ValidatorRegistration{r.Message},
		Signature: r.SignedValidatorRegistrationV1.Signature,
	})
}

func (r *ValidatorRegistration) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		FeeRecipient hexSlice `json:"fee_recipient,omitempty"`
		GasLimit     string   `json:"gas_limit,omitempty"`
		Timestamp    string   `json:"timestamp,omitempty"`
		Pubkey       hexSlice `json:"pubkey,omitempty"`
		*eth.ValidatorRegistrationV1
	}{
		FeeRecipient:            r.FeeRecipient,
		GasLimit:                fmt.Sprintf("%d", r.GasLimit),
		Timestamp:               fmt.Sprintf("%d", r.Timestamp),
		Pubkey:                  r.Pubkey,
		ValidatorRegistrationV1: r.ValidatorRegistrationV1,
	})
}

type hexSlice []byte

func (hs hexSlice) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%#x", hs)), nil
}

func (hs *hexSlice) UnmarshalText(t []byte) error {
	decoded, err := hexutil.Decode(string(t))
	if err != nil {
		return errors.Wrapf(err, "error unmarshaling text value %s", string(t))
	}
	*hs = decoded
	return nil
}

type Uint256 struct {
	*big.Int
}

var errUnmarshalUint256Failed = errors.New("unable to UnmarshalText into a Uint256 value")

func (s *Uint256) UnmarshalJSON(t []byte) error {
	start := 0
	end := len(t)
	if t[0] == '"' {
		start += 1
	}
	if t[end-1] == '"' {
		end -= 1
	}
	return s.UnmarshalText(t[start:end])
}

func (s *Uint256) UnmarshalText(t []byte) error {
	if s.Int == nil {
		s.Int = big.NewInt(0)
	}
	z, ok := s.SetString(string(t), 10)
	if !ok {
		return errors.Wrapf(errUnmarshalUint256Failed, "value=%s", string(t))
	}
	s.Int = z
	return nil
}

func (s Uint256) MarshalText() ([]byte, error) {
	return s.Bytes(), nil
}

type Uint64String uint64

func (s *Uint64String) UnmarshalText(t []byte) error {
	u, err := strconv.ParseUint(string(t), 10, 64)
	*s = Uint64String(u)
	return err
}

func (s Uint64String) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", s)), nil
}

type ExecHeaderResponse struct {
	Version string `json:"version,omitempty"`
	Data    struct {
		Signature hexSlice    `json:"signature,omitempty"`
		Message   *BuilderBid `json:"message,omitempty"`
	} `json:"data,omitempty"`
}

func (ehr *ExecHeaderResponse) ToProto() (*eth.SignedBuilderBid, error) {
	bb, err := ehr.Data.Message.ToProto()
	if err != nil {
		return nil, err
	}
	return &eth.SignedBuilderBid{
		Message:   bb,
		Signature: ehr.Data.Signature,
	}, nil
}

func (bb *BuilderBid) ToProto() (*eth.BuilderBid, error) {
	header, err := bb.Header.ToProto()
	if err != nil {
		return nil, err
	}
	return &eth.BuilderBid{
		Header: header,
		Value:  bb.Value.Bytes(),
		Pubkey: bb.Pubkey,
	}, nil
}

func (h *ExecutionPayloadHeader) ToProto() (*eth.ExecutionPayloadHeader, error) {
	// TODO: it looks like BaseFeePerGas should probably be a uint
	baseFeeHack := []byte(strconv.FormatUint(uint64(h.BaseFeePerGas), 10))
	return &eth.ExecutionPayloadHeader{
		ParentHash:       h.ParentHash,
		FeeRecipient:     h.FeeRecipient,
		StateRoot:        h.StateRoot,
		ReceiptsRoot:     h.ReceiptsRoot,
		LogsBloom:        h.LogsBloom,
		PrevRandao:       h.PrevRandao,
		BlockNumber:      uint64(h.BlockNumber),
		GasLimit:         uint64(h.GasLimit),
		GasUsed:          uint64(h.GasUsed),
		Timestamp:        uint64(h.Timestamp),
		ExtraData:        h.ExtraData,
		BaseFeePerGas:    baseFeeHack,
		BlockHash:        h.BlockHash,
		TransactionsRoot: h.TransactionsRoot,
	}, nil
}

type BuilderBid struct {
	Header *ExecutionPayloadHeader `json:"header,omitempty"`
	Value  Uint256                 `json:"value,omitempty"`
	Pubkey hexSlice                `json:"pubkey,omitempty"`
}

type ExecutionPayloadHeader struct {
	ParentHash       hexSlice     `json:"parent_hash,omitempty"`
	FeeRecipient     hexSlice     `json:"fee_recipient,omitempty"`
	StateRoot        hexSlice     `json:"state_root,omitempty"`
	ReceiptsRoot     hexSlice     `json:"receipts_root,omitempty"`
	LogsBloom        hexSlice     `json:"logs_bloom,omitempty"`
	PrevRandao       hexSlice     `json:"prev_randao,omitempty"`
	BlockNumber      Uint64String `json:"block_number,omitempty"`
	GasLimit         Uint64String `json:"gas_limit,omitempty"`
	GasUsed          Uint64String `json:"gas_used,omitempty"`
	Timestamp        Uint64String `json:"timestamp,omitempty"`
	ExtraData        hexSlice     `json:"extra_data,omitempty"`
	BaseFeePerGas    Uint64String `json:"base_fee_per_gas,omitempty"`
	BlockHash        hexSlice     `json:"block_hash,omitempty"`
	TransactionsRoot hexSlice     `json:"transactions_root,omitempty"`
	*eth.ExecutionPayloadHeader
}

func (h *ExecutionPayloadHeader) MarshalJSON() ([]byte, error) {
	// TODO check this encoding and confirm bounds of values
	bfpg, err := strconv.ParseUint(string(h.ExecutionPayloadHeader.BaseFeePerGas), 10, 64)
	if err != nil {
		return nil, err
	}
	type MarshalCaller ExecutionPayloadHeader
	return json.Marshal(&MarshalCaller{
		ParentHash:       h.ExecutionPayloadHeader.ParentHash,
		FeeRecipient:     h.ExecutionPayloadHeader.FeeRecipient,
		StateRoot:        h.ExecutionPayloadHeader.StateRoot,
		ReceiptsRoot:     h.ExecutionPayloadHeader.ReceiptsRoot,
		LogsBloom:        h.ExecutionPayloadHeader.LogsBloom,
		PrevRandao:       h.ExecutionPayloadHeader.PrevRandao,
		BlockNumber:      Uint64String(h.ExecutionPayloadHeader.BlockNumber),
		GasLimit:         Uint64String(h.ExecutionPayloadHeader.GasLimit),
		GasUsed:          Uint64String(h.ExecutionPayloadHeader.GasUsed),
		Timestamp:        Uint64String(h.ExecutionPayloadHeader.Timestamp),
		ExtraData:        h.ExecutionPayloadHeader.ExtraData,
		BaseFeePerGas:    Uint64String(bfpg),
		BlockHash:        h.ExecutionPayloadHeader.BlockHash,
		TransactionsRoot: h.ExecutionPayloadHeader.TransactionsRoot,
	})
}

type ExecPayloadResponse struct {
	Version string           `json:"version,omitempty"`
	Data    ExecutionPayload `json:"data,omitempty"`
}

type ExecutionPayload struct {
	ParentHash    hexSlice     `json:"parent_hash,omitempty"`
	FeeRecipient  hexSlice     `json:"fee_recipient,omitempty"`
	StateRoot     hexSlice     `json:"state_root,omitempty"`
	ReceiptsRoot  hexSlice     `json:"receipts_root,omitempty"`
	LogsBloom     hexSlice     `json:"logs_bloom,omitempty"`
	PrevRandao    hexSlice     `json:"prev_randao,omitempty"`
	BlockNumber   Uint64String `json:"block_number,omitempty"`
	GasLimit      Uint64String `json:"gas_limit,omitempty"`
	GasUsed       Uint64String `json:"gas_used,omitempty"`
	Timestamp     Uint64String `json:"timestamp,omitempty"`
	ExtraData     hexSlice     `json:"extra_data,omitempty"`
	BaseFeePerGas Uint64String `json:"base_fee_per_gas,omitempty"`
	BlockHash     hexSlice     `json:"block_hash,omitempty"`
	Transactions  []hexSlice   `json:"transactions,omitempty"`
}

func (r *ExecPayloadResponse) ToProto() (*v1.ExecutionPayload, error) {
	return r.Data.ToProto()
}

func (p *ExecutionPayload) ToProto() (*v1.ExecutionPayload, error) {
	// TODO: it looks like BaseFeePerGas should probably be a uint
	baseFeeHack := []byte(strconv.FormatUint(uint64(p.BaseFeePerGas), 10))
	txs := make([][]byte, len(p.Transactions))
	for i := range p.Transactions {
		txs[i] = p.Transactions[i]
	}
	return &v1.ExecutionPayload{
		ParentHash:    p.ParentHash,
		FeeRecipient:  p.FeeRecipient,
		StateRoot:     p.StateRoot,
		ReceiptsRoot:  p.ReceiptsRoot,
		LogsBloom:     p.LogsBloom,
		PrevRandao:    p.PrevRandao,
		BlockNumber:   uint64(p.BlockNumber),
		GasLimit:      uint64(p.GasLimit),
		GasUsed:       uint64(p.GasUsed),
		Timestamp:     uint64(p.Timestamp),
		ExtraData:     p.ExtraData,
		BaseFeePerGas: baseFeeHack,
		BlockHash:     p.BlockHash,
		Transactions:  txs,
	}, nil
}

type SignedBlindedBeaconBlockBellatrix struct {
	*eth.SignedBlindedBeaconBlockBellatrix
}

type BlindedBeaconBlockBellatrix struct {
	*eth.BlindedBeaconBlockBellatrix
}

type BlindedBeaconBlockBodyBellatrix struct {
	*eth.BlindedBeaconBlockBodyBellatrix
}

func (r *SignedBlindedBeaconBlockBellatrix) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Message   *BlindedBeaconBlockBellatrix `json:"message,omitempty"`
		Signature hexSlice                     `json:"signature,omitempty"`
	}{
		Message:   &BlindedBeaconBlockBellatrix{r.SignedBlindedBeaconBlockBellatrix.Block},
		Signature: r.SignedBlindedBeaconBlockBellatrix.Signature,
	})
}

func (b *BlindedBeaconBlockBellatrix) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Slot          string                           `json:"slot"`
		ProposerIndex string                           `json:"proposer_index,omitempty"`
		ParentRoot    hexSlice                         `json:"parent_root,omitempty"`
		StateRoot     hexSlice                         `json:"state_root,omitempty"`
		Body          *BlindedBeaconBlockBodyBellatrix `json:"body,omitempty"`
	}{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    b.ParentRoot,
		StateRoot:     b.StateRoot,
		Body:          &BlindedBeaconBlockBodyBellatrix{b.BlindedBeaconBlockBellatrix.Body},
	})
}

type ProposerSlashing struct {
	*eth.ProposerSlashing
}

func (s *ProposerSlashing) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		SignedHeader1 *SignedBeaconBlockHeader `json:"signed_header_1,omitempty"`
		SignedHeader2 *SignedBeaconBlockHeader `json:"signed_header_2,omitempty"`
	}{
		SignedHeader1: &SignedBeaconBlockHeader{s.ProposerSlashing.Header_1},
		SignedHeader2: &SignedBeaconBlockHeader{s.ProposerSlashing.Header_2},
	})
}

type SignedBeaconBlockHeader struct {
	*eth.SignedBeaconBlockHeader
}

func (h *SignedBeaconBlockHeader) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Header    *BeaconBlockHeader `json:"message,omitempty"`
		Signature hexSlice           `json:"signature,omitempty"`
	}{
		Header:    &BeaconBlockHeader{h.SignedBeaconBlockHeader.Header},
		Signature: h.SignedBeaconBlockHeader.Signature,
	})
}

type BeaconBlockHeader struct {
	*eth.BeaconBlockHeader
}

func (h *BeaconBlockHeader) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Slot          string   `json:"slot,omitempty"`
		ProposerIndex string   `json:"proposer_index,omitempty"`
		ParentRoot    hexSlice `json:"parent_root,omitempty"`
		StateRoot     hexSlice `json:"state_root,omitempty"`
		BodyRoot      hexSlice `json:"body_root,omitempty"`
	}{
		Slot:          fmt.Sprintf("%d", h.BeaconBlockHeader.Slot),
		ProposerIndex: fmt.Sprintf("%d", h.BeaconBlockHeader.ProposerIndex),
		ParentRoot:    h.BeaconBlockHeader.ParentRoot,
		StateRoot:     h.BeaconBlockHeader.StateRoot,
		BodyRoot:      h.BeaconBlockHeader.BodyRoot,
	})
}

type IndexedAttestation struct {
	*eth.IndexedAttestation
}

func (a *IndexedAttestation) MarshalJSON() ([]byte, error) {
	indices := make([]string, len(a.IndexedAttestation.AttestingIndices))
	for i := range a.IndexedAttestation.AttestingIndices {
		indices[i] = fmt.Sprintf("%d", a.AttestingIndices[i])
	}
	return json.Marshal(struct {
		AttestingIndices []string `json:"attesting_indices,omitempty"`
		Data             *AttestationData
		Signature        hexSlice `json:"signature,omitempty"`
	}{
		AttestingIndices: indices,
		Data:             &AttestationData{a.IndexedAttestation.Data},
		Signature:        a.IndexedAttestation.Signature,
	})
}

type AttesterSlashing struct {
	*eth.AttesterSlashing
}

func (s *AttesterSlashing) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Attestation1 *IndexedAttestation `json:"attestation_1,omitempty"`
		Attestation2 *IndexedAttestation `json:"attestation_2,omitempty"`
	}{
		Attestation1: &IndexedAttestation{s.Attestation_1},
		Attestation2: &IndexedAttestation{s.Attestation_2},
	})
}

type Checkpoint struct {
	*eth.Checkpoint
}

func (c *Checkpoint) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Epoch string   `json:"epoch,omitempty"`
		Root  hexSlice `json:"root,omitempty"`
	}{
		Epoch: fmt.Sprintf("%d", c.Checkpoint.Epoch),
		Root:  c.Checkpoint.Root,
	})
}

type AttestationData struct {
	*eth.AttestationData
}

func (a *AttestationData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Slot            string      `json:"slot,omitempty"`
		Index           string      `json:"index,omitempty"`
		BeaconBlockRoot hexSlice    `json:"beacon_block_root,omitempty"`
		Source          *Checkpoint `json:"source,omitempty"`
		Target          *Checkpoint `json:"target,omitempty"`
	}{
		Slot:            fmt.Sprintf("%d", a.AttestationData.Slot),
		Index:           fmt.Sprintf("%d", a.AttestationData.CommitteeIndex),
		BeaconBlockRoot: a.AttestationData.BeaconBlockRoot,
		Source:          &Checkpoint{a.AttestationData.Source},
		Target:          &Checkpoint{a.AttestationData.Target},
	})
}

type Attestation struct {
	*eth.Attestation
}

func (a *Attestation) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		AggregationBits hexSlice         `json:"aggregation_bits,omitempty"`
		Data            *AttestationData `json:"data,omitempty"`
		Signature       hexSlice         `json:"signature,omitempty" ssz-size:"96"`
	}{
		AggregationBits: hexSlice(a.Attestation.AggregationBits),
		Data:            &AttestationData{a.Attestation.Data},
		Signature:       a.Attestation.Signature,
	})
}

type DepositData struct {
	*eth.Deposit_Data
}

func (d *DepositData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PublicKey             hexSlice `json:"pubkey,omitempty"`
		WithdrawalCredentials hexSlice `json:"withdrawal_credentials,omitempty"`
		Amount                string   `json:"amount,omitempty"`
		Signature             hexSlice `json:"signature,omitempty"`
	}{
		PublicKey:             d.PublicKey,
		WithdrawalCredentials: d.WithdrawalCredentials,
		Amount:                fmt.Sprintf("%d", d.Amount),
		Signature:             d.Signature,
	})
}

type Deposit struct {
	*eth.Deposit
}

func (d *Deposit) MarshalJSON() ([]byte, error) {
	proof := make([]hexSlice, len(d.Proof))
	for i := range d.Proof {
		proof[i] = d.Proof[i]
	}
	return json.Marshal(struct {
		Proof []hexSlice   `json:"proof"`
		Data  *DepositData `json:"data"`
	}{
		Proof: proof,
		Data:  &DepositData{Deposit_Data: d.Deposit.Data},
	})
}

type SignedVoluntaryExit struct {
	*eth.SignedVoluntaryExit
}

func (sve *SignedVoluntaryExit) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Message   *VoluntaryExit `json:"message,omitempty"`
		Signature hexSlice       `json:"signature,omitempty"`
	}{
		Signature: sve.SignedVoluntaryExit.Signature,
		Message:   &VoluntaryExit{sve.SignedVoluntaryExit.Exit},
	})
}

type VoluntaryExit struct {
	*eth.VoluntaryExit
}

func (ve *VoluntaryExit) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Epoch          string `json:"epoch,omitempty"`
		ValidatorIndex string `json:"validator_index,omitempty"`
	}{
		Epoch:          fmt.Sprintf("%d", ve.Epoch),
		ValidatorIndex: fmt.Sprintf("%d", ve.ValidatorIndex),
	})
}

type SyncAggregate struct {
	*eth.SyncAggregate
}

func (s *SyncAggregate) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		SyncCommitteeBits      hexSlice `json:"sync_committee_bits,omitempty"`
		SyncCommitteeSignature hexSlice `json:"sync_committee_signature,omitempty"`
	}{
		SyncCommitteeBits:      hexSlice(s.SyncAggregate.SyncCommitteeBits),
		SyncCommitteeSignature: s.SyncAggregate.SyncCommitteeSignature,
	})
}

type Eth1Data struct {
	*eth.Eth1Data
}

func (e *Eth1Data) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		DepositRoot  hexSlice `json:"deposit_root,omitempty"`
		DepositCount string   `json:"deposit_count,omitempty"`
		BlockHash    hexSlice `json:"block_hash,omitempty"`
	}{
		DepositRoot:  e.DepositRoot,
		DepositCount: fmt.Sprintf("%d", e.DepositCount),
		BlockHash:    e.BlockHash,
	})
}

func (b *BlindedBeaconBlockBodyBellatrix) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RandaoReveal           hexSlice                `json:"randao_reveal,omitempty"`
		Eth1Data               *Eth1Data               `json:"eth1_data,omitempty"`
		Graffiti               hexSlice                `json:"graffiti,omitempty"`
		ProposerSlashings      []*ProposerSlashing     `json:"proposer_slashings,omitempty"`
		AttesterSlashings      []*AttesterSlashing     `json:"attester_slashings,omitempty"`
		Attestations           []*Attestation          `json:"attestations,omitempty"`
		Deposits               []*Deposit              `json:"deposits,omitempty"`
		VoluntaryExits         []*SignedVoluntaryExit  `json:"voluntary_exits,omitempty"`
		SyncAggregates         *SyncAggregate          `json:"sync_aggregate,omitempty"`
		ExecutionPayloadHeader *ExecutionPayloadHeader `json:"execution_payload_header,omitempty"`
	}{
		RandaoReveal:           b.RandaoReveal,
		Eth1Data:               &Eth1Data{b.BlindedBeaconBlockBodyBellatrix.Eth1Data},
		Graffiti:               b.BlindedBeaconBlockBodyBellatrix.Graffiti,
		ProposerSlashings:      []*ProposerSlashing{},
		AttesterSlashings:      []*AttesterSlashing{},
		Attestations:           []*Attestation{},
		Deposits:               []*Deposit{},
		VoluntaryExits:         []*SignedVoluntaryExit{},
		ExecutionPayloadHeader: &ExecutionPayloadHeader{ExecutionPayloadHeader: b.BlindedBeaconBlockBodyBellatrix.ExecutionPayloadHeader},
	})
}
