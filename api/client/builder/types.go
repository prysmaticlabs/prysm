package builder

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

type SignedValidatorRegistration struct {
	*eth.SignedValidatorRegistrationV1
}

type ValidatorRegistration struct {
	*eth.ValidatorRegistrationV1
}

func (r *SignedValidatorRegistration) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Message   *ValidatorRegistration `json:"message"`
		Signature hexutil.Bytes          `json:"signature"`
	}{
		Message:   &ValidatorRegistration{r.Message},
		Signature: r.SignedValidatorRegistrationV1.Signature,
	})
}

func (r *SignedValidatorRegistration) UnmarshalJSON(b []byte) error {
	if r.SignedValidatorRegistrationV1 == nil {
		r.SignedValidatorRegistrationV1 = &eth.SignedValidatorRegistrationV1{}
	}
	o := struct {
		Message   *ValidatorRegistration `json:"message"`
		Signature hexutil.Bytes          `json:"signature"`
	}{}
	if err := json.Unmarshal(b, &o); err != nil {
		return err
	}
	r.Message = o.Message.ValidatorRegistrationV1
	r.Signature = o.Signature
	return nil
}

func (r *ValidatorRegistration) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		FeeRecipient hexutil.Bytes `json:"fee_recipient"`
		GasLimit     string        `json:"gas_limit"`
		Timestamp    string        `json:"timestamp"`
		Pubkey       hexutil.Bytes `json:"pubkey"`
	}{
		FeeRecipient: r.FeeRecipient,
		GasLimit:     fmt.Sprintf("%d", r.GasLimit),
		Timestamp:    fmt.Sprintf("%d", r.Timestamp),
		Pubkey:       r.Pubkey,
	})
}

func (r *ValidatorRegistration) UnmarshalJSON(b []byte) error {
	if r.ValidatorRegistrationV1 == nil {
		r.ValidatorRegistrationV1 = &eth.ValidatorRegistrationV1{}
	}
	o := struct {
		FeeRecipient hexutil.Bytes `json:"fee_recipient"`
		GasLimit     string        `json:"gas_limit"`
		Timestamp    string        `json:"timestamp"`
		Pubkey       hexutil.Bytes `json:"pubkey"`
	}{}
	if err := json.Unmarshal(b, &o); err != nil {
		return err
	}

	r.FeeRecipient = o.FeeRecipient
	r.Pubkey = o.Pubkey
	var err error
	if r.GasLimit, err = strconv.ParseUint(o.GasLimit, 10, 64); err != nil {
		return errors.Wrap(err, "failed to parse gas limit")
	}
	if r.Timestamp, err = strconv.ParseUint(o.Timestamp, 10, 64); err != nil {
		return errors.Wrap(err, "failed to parse timestamp")
	}

	return nil
}

var errInvalidUint256 = errors.New("invalid Uint256")
var errDecodeUint256 = errors.New("unable to decode into Uint256")

type Uint256 struct {
	*big.Int
}

func isValidUint256(bi *big.Int) bool {
	return bi.Cmp(big.NewInt(0)) >= 0 && bi.BitLen() <= 256
}

func stringToUint256(s string) (Uint256, error) {
	bi := new(big.Int)
	_, ok := bi.SetString(s, 10)
	if !ok || !isValidUint256(bi) {
		return Uint256{}, errors.Wrapf(errDecodeUint256, "value=%s", s)
	}
	return Uint256{Int: bi}, nil
}

// sszBytesToUint256 creates a Uint256 from a ssz-style (little-endian byte slice) representation.
func sszBytesToUint256(b []byte) (Uint256, error) {
	bi := bytesutil.LittleEndianBytesToBigInt(b)
	if !isValidUint256(bi) {
		return Uint256{}, errors.Wrapf(errDecodeUint256, "value=%s", b)
	}
	return Uint256{Int: bi}, nil
}

// SSZBytes creates an ssz-style (little-endian byte slice) representation of the Uint256
func (s Uint256) SSZBytes() []byte {
	if !isValidUint256(s.Int) {
		return []byte{}
	}
	return bytesutil.PadTo(bytesutil.ReverseByteOrder(s.Int.Bytes()), 32)
}

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
		return errors.Wrapf(errDecodeUint256, "value=%s", t)
	}
	if !isValidUint256(z) {
		return errors.Wrapf(errDecodeUint256, "value=%s", t)
	}
	s.Int = z
	return nil
}

func (s Uint256) MarshalJSON() ([]byte, error) {
	t, err := s.MarshalText()
	if err != nil {
		return nil, err
	}
	t = append([]byte{'"'}, t...)
	t = append(t, '"')
	return t, nil
}

func (s Uint256) MarshalText() ([]byte, error) {
	if !isValidUint256(s.Int) {
		return nil, errors.Wrapf(errInvalidUint256, "value=%s", s.Int)
	}
	return []byte(s.String()), nil
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

type VersionResponse struct {
	Version string `json:"version"`
}

type ExecHeaderResponse struct {
	Version string `json:"version"`
	Data    struct {
		Signature hexutil.Bytes `json:"signature"`
		Message   *BuilderBid   `json:"message"`
	} `json:"data"`
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
		Value:  bb.Value.SSZBytes(),
		Pubkey: bb.Pubkey,
	}, nil
}

func (h *ExecutionPayloadHeader) ToProto() (*v1.ExecutionPayloadHeader, error) {
	return &v1.ExecutionPayloadHeader{
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
		BaseFeePerGas:    h.BaseFeePerGas.SSZBytes(),
		BlockHash:        h.BlockHash,
		TransactionsRoot: h.TransactionsRoot,
	}, nil
}

type BuilderBid struct {
	Header *ExecutionPayloadHeader `json:"header"`
	Value  Uint256                 `json:"value"`
	Pubkey hexutil.Bytes           `json:"pubkey"`
}

type ExecutionPayloadHeader struct {
	ParentHash       hexutil.Bytes `json:"parent_hash"`
	FeeRecipient     hexutil.Bytes `json:"fee_recipient"`
	StateRoot        hexutil.Bytes `json:"state_root"`
	ReceiptsRoot     hexutil.Bytes `json:"receipts_root"`
	LogsBloom        hexutil.Bytes `json:"logs_bloom"`
	PrevRandao       hexutil.Bytes `json:"prev_randao"`
	BlockNumber      Uint64String  `json:"block_number"`
	GasLimit         Uint64String  `json:"gas_limit"`
	GasUsed          Uint64String  `json:"gas_used"`
	Timestamp        Uint64String  `json:"timestamp"`
	ExtraData        hexutil.Bytes `json:"extra_data"`
	BaseFeePerGas    Uint256       `json:"base_fee_per_gas"`
	BlockHash        hexutil.Bytes `json:"block_hash"`
	TransactionsRoot hexutil.Bytes `json:"transactions_root"`
	*v1.ExecutionPayloadHeader
}

func (h *ExecutionPayloadHeader) MarshalJSON() ([]byte, error) {
	type MarshalCaller ExecutionPayloadHeader
	baseFeePerGas, err := sszBytesToUint256(h.ExecutionPayloadHeader.BaseFeePerGas)
	if err != nil {
		return []byte{}, errors.Wrapf(err, "invalid BaseFeePerGas")
	}
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
		BaseFeePerGas:    baseFeePerGas,
		BlockHash:        h.ExecutionPayloadHeader.BlockHash,
		TransactionsRoot: h.ExecutionPayloadHeader.TransactionsRoot,
	})
}

func (h *ExecutionPayloadHeader) UnmarshalJSON(b []byte) error {
	type UnmarshalCaller ExecutionPayloadHeader
	uc := &UnmarshalCaller{}
	if err := json.Unmarshal(b, uc); err != nil {
		return err
	}
	ep := ExecutionPayloadHeader(*uc)
	*h = ep
	var err error
	h.ExecutionPayloadHeader, err = h.ToProto()
	return err
}

type ExecPayloadResponse struct {
	Version string           `json:"version"`
	Data    ExecutionPayload `json:"data"`
}

type ExecutionPayload struct {
	ParentHash    hexutil.Bytes   `json:"parent_hash"`
	FeeRecipient  hexutil.Bytes   `json:"fee_recipient"`
	StateRoot     hexutil.Bytes   `json:"state_root"`
	ReceiptsRoot  hexutil.Bytes   `json:"receipts_root"`
	LogsBloom     hexutil.Bytes   `json:"logs_bloom"`
	PrevRandao    hexutil.Bytes   `json:"prev_randao"`
	BlockNumber   Uint64String    `json:"block_number"`
	GasLimit      Uint64String    `json:"gas_limit"`
	GasUsed       Uint64String    `json:"gas_used"`
	Timestamp     Uint64String    `json:"timestamp"`
	ExtraData     hexutil.Bytes   `json:"extra_data"`
	BaseFeePerGas Uint256         `json:"base_fee_per_gas"`
	BlockHash     hexutil.Bytes   `json:"block_hash"`
	Transactions  []hexutil.Bytes `json:"transactions"`
}

func (r *ExecPayloadResponse) ToProto() (*v1.ExecutionPayload, error) {
	return r.Data.ToProto()
}

func (p *ExecutionPayload) ToProto() (*v1.ExecutionPayload, error) {
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
		BaseFeePerGas: p.BaseFeePerGas.SSZBytes(),
		BlockHash:     p.BlockHash,
		Transactions:  txs,
	}, nil
}

type ExecHeaderResponseCapella struct {
	Data struct {
		Signature hexutil.Bytes      `json:"signature"`
		Message   *BuilderBidCapella `json:"message"`
	} `json:"data"`
}

func (ehr *ExecHeaderResponseCapella) ToProto() (*eth.SignedBuilderBidCapella, error) {
	bb, err := ehr.Data.Message.ToProto()
	if err != nil {
		return nil, err
	}
	return &eth.SignedBuilderBidCapella{
		Message:   bb,
		Signature: ehr.Data.Signature,
	}, nil
}

func (bb *BuilderBidCapella) ToProto() (*eth.BuilderBidCapella, error) {
	header, err := bb.Header.ToProto()
	if err != nil {
		return nil, err
	}
	return &eth.BuilderBidCapella{
		Header: header,
		Value:  bb.Value.SSZBytes(),
		Pubkey: bb.Pubkey,
	}, nil
}

func (h *ExecutionPayloadHeaderCapella) ToProto() (*v1.ExecutionPayloadHeaderCapella, error) {
	return &v1.ExecutionPayloadHeaderCapella{
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
		BaseFeePerGas:    h.BaseFeePerGas.SSZBytes(),
		BlockHash:        h.BlockHash,
		TransactionsRoot: h.TransactionsRoot,
		WithdrawalsRoot:  h.WithdrawalsRoot,
	}, nil
}

type BuilderBidCapella struct {
	Header *ExecutionPayloadHeaderCapella `json:"header"`
	Value  Uint256                        `json:"value"`
	Pubkey hexutil.Bytes                  `json:"pubkey"`
}

type ExecutionPayloadHeaderCapella struct {
	ParentHash       hexutil.Bytes `json:"parent_hash"`
	FeeRecipient     hexutil.Bytes `json:"fee_recipient"`
	StateRoot        hexutil.Bytes `json:"state_root"`
	ReceiptsRoot     hexutil.Bytes `json:"receipts_root"`
	LogsBloom        hexutil.Bytes `json:"logs_bloom"`
	PrevRandao       hexutil.Bytes `json:"prev_randao"`
	BlockNumber      Uint64String  `json:"block_number"`
	GasLimit         Uint64String  `json:"gas_limit"`
	GasUsed          Uint64String  `json:"gas_used"`
	Timestamp        Uint64String  `json:"timestamp"`
	ExtraData        hexutil.Bytes `json:"extra_data"`
	BaseFeePerGas    Uint256       `json:"base_fee_per_gas"`
	BlockHash        hexutil.Bytes `json:"block_hash"`
	TransactionsRoot hexutil.Bytes `json:"transactions_root"`
	WithdrawalsRoot  hexutil.Bytes `json:"withdrawals_root"`
	*v1.ExecutionPayloadHeaderCapella
}

func (h *ExecutionPayloadHeaderCapella) MarshalJSON() ([]byte, error) {
	type MarshalCaller ExecutionPayloadHeaderCapella
	baseFeePerGas, err := sszBytesToUint256(h.ExecutionPayloadHeaderCapella.BaseFeePerGas)
	if err != nil {
		return []byte{}, errors.Wrapf(err, "invalid BaseFeePerGas")
	}
	return json.Marshal(&MarshalCaller{
		ParentHash:       h.ExecutionPayloadHeaderCapella.ParentHash,
		FeeRecipient:     h.ExecutionPayloadHeaderCapella.FeeRecipient,
		StateRoot:        h.ExecutionPayloadHeaderCapella.StateRoot,
		ReceiptsRoot:     h.ExecutionPayloadHeaderCapella.ReceiptsRoot,
		LogsBloom:        h.ExecutionPayloadHeaderCapella.LogsBloom,
		PrevRandao:       h.ExecutionPayloadHeaderCapella.PrevRandao,
		BlockNumber:      Uint64String(h.ExecutionPayloadHeaderCapella.BlockNumber),
		GasLimit:         Uint64String(h.ExecutionPayloadHeaderCapella.GasLimit),
		GasUsed:          Uint64String(h.ExecutionPayloadHeaderCapella.GasUsed),
		Timestamp:        Uint64String(h.ExecutionPayloadHeaderCapella.Timestamp),
		ExtraData:        h.ExecutionPayloadHeaderCapella.ExtraData,
		BaseFeePerGas:    baseFeePerGas,
		BlockHash:        h.ExecutionPayloadHeaderCapella.BlockHash,
		TransactionsRoot: h.ExecutionPayloadHeaderCapella.TransactionsRoot,
		WithdrawalsRoot:  h.ExecutionPayloadHeaderCapella.WithdrawalsRoot,
	})
}

func (h *ExecutionPayloadHeaderCapella) UnmarshalJSON(b []byte) error {
	type UnmarshalCaller ExecutionPayloadHeaderCapella
	uc := &UnmarshalCaller{}
	if err := json.Unmarshal(b, uc); err != nil {
		return err
	}
	ep := ExecutionPayloadHeaderCapella(*uc)
	*h = ep
	var err error
	h.ExecutionPayloadHeaderCapella, err = h.ToProto()
	return err
}

type ExecPayloadResponseCapella struct {
	Version string                  `json:"version"`
	Data    ExecutionPayloadCapella `json:"data"`
}

type ExecutionPayloadCapella struct {
	ParentHash    hexutil.Bytes   `json:"parent_hash"`
	FeeRecipient  hexutil.Bytes   `json:"fee_recipient"`
	StateRoot     hexutil.Bytes   `json:"state_root"`
	ReceiptsRoot  hexutil.Bytes   `json:"receipts_root"`
	LogsBloom     hexutil.Bytes   `json:"logs_bloom"`
	PrevRandao    hexutil.Bytes   `json:"prev_randao"`
	BlockNumber   Uint64String    `json:"block_number"`
	GasLimit      Uint64String    `json:"gas_limit"`
	GasUsed       Uint64String    `json:"gas_used"`
	Timestamp     Uint64String    `json:"timestamp"`
	ExtraData     hexutil.Bytes   `json:"extra_data"`
	BaseFeePerGas Uint256         `json:"base_fee_per_gas"`
	BlockHash     hexutil.Bytes   `json:"block_hash"`
	Transactions  []hexutil.Bytes `json:"transactions"`
	Withdrawals   []Withdrawal    `json:"withdrawals"`
}

func (r *ExecPayloadResponseCapella) ToProto() (*v1.ExecutionPayloadCapella, error) {
	return r.Data.ToProto()
}

func (p *ExecutionPayloadCapella) ToProto() (*v1.ExecutionPayloadCapella, error) {
	txs := make([][]byte, len(p.Transactions))
	for i := range p.Transactions {
		txs[i] = p.Transactions[i]
	}
	withdrawals := make([]*v1.Withdrawal, len(p.Withdrawals))
	for i, w := range p.Withdrawals {
		withdrawals[i] = &v1.Withdrawal{
			Index:          w.Index.Uint64(),
			ValidatorIndex: types.ValidatorIndex(w.ValidatorIndex.Uint64()),
			Address:        w.Address,
			Amount:         w.Amount.Uint64(),
		}
	}
	return &v1.ExecutionPayloadCapella{
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
		BaseFeePerGas: p.BaseFeePerGas.SSZBytes(),
		BlockHash:     p.BlockHash,
		Transactions:  txs,
		Withdrawals:   withdrawals,
	}, nil
}

type Withdrawal struct {
	Index          Uint256       `json:"index"`
	ValidatorIndex Uint256       `json:"validator_index"`
	Address        hexutil.Bytes `json:"address"`
	Amount         Uint256       `json:"amount"`
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
		Message   *BlindedBeaconBlockBellatrix `json:"message"`
		Signature hexutil.Bytes                `json:"signature"`
	}{
		Message:   &BlindedBeaconBlockBellatrix{r.SignedBlindedBeaconBlockBellatrix.Block},
		Signature: r.SignedBlindedBeaconBlockBellatrix.Signature,
	})
}

func (b *BlindedBeaconBlockBellatrix) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Slot          string                           `json:"slot"`
		ProposerIndex string                           `json:"proposer_index"`
		ParentRoot    hexutil.Bytes                    `json:"parent_root"`
		StateRoot     hexutil.Bytes                    `json:"state_root"`
		Body          *BlindedBeaconBlockBodyBellatrix `json:"body"`
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
		SignedHeader1 *SignedBeaconBlockHeader `json:"signed_header_1"`
		SignedHeader2 *SignedBeaconBlockHeader `json:"signed_header_2"`
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
		Header    *BeaconBlockHeader `json:"message"`
		Signature hexutil.Bytes      `json:"signature"`
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
		Slot          string        `json:"slot"`
		ProposerIndex string        `json:"proposer_index"`
		ParentRoot    hexutil.Bytes `json:"parent_root"`
		StateRoot     hexutil.Bytes `json:"state_root"`
		BodyRoot      hexutil.Bytes `json:"body_root"`
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
		AttestingIndices []string         `json:"attesting_indices"`
		Data             *AttestationData `json:"data"`
		Signature        hexutil.Bytes    `json:"signature"`
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
		Attestation1 *IndexedAttestation `json:"attestation_1"`
		Attestation2 *IndexedAttestation `json:"attestation_2"`
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
		Epoch string        `json:"epoch"`
		Root  hexutil.Bytes `json:"root"`
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
		Slot            string        `json:"slot"`
		Index           string        `json:"index"`
		BeaconBlockRoot hexutil.Bytes `json:"beacon_block_root"`
		Source          *Checkpoint   `json:"source"`
		Target          *Checkpoint   `json:"target"`
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
		AggregationBits hexutil.Bytes    `json:"aggregation_bits"`
		Data            *AttestationData `json:"data"`
		Signature       hexutil.Bytes    `json:"signature" ssz-size:"96"`
	}{
		AggregationBits: hexutil.Bytes(a.Attestation.AggregationBits),
		Data:            &AttestationData{a.Attestation.Data},
		Signature:       a.Attestation.Signature,
	})
}

type DepositData struct {
	*eth.Deposit_Data
}

func (d *DepositData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PublicKey             hexutil.Bytes `json:"pubkey"`
		WithdrawalCredentials hexutil.Bytes `json:"withdrawal_credentials"`
		Amount                string        `json:"amount"`
		Signature             hexutil.Bytes `json:"signature"`
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
	proof := make([]hexutil.Bytes, len(d.Proof))
	for i := range d.Proof {
		proof[i] = d.Proof[i]
	}
	return json.Marshal(struct {
		Proof []hexutil.Bytes `json:"proof"`
		Data  *DepositData    `json:"data"`
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
		Message   *VoluntaryExit `json:"message"`
		Signature hexutil.Bytes  `json:"signature"`
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
		Epoch          string `json:"epoch"`
		ValidatorIndex string `json:"validator_index"`
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
		SyncCommitteeBits      hexutil.Bytes `json:"sync_committee_bits"`
		SyncCommitteeSignature hexutil.Bytes `json:"sync_committee_signature"`
	}{
		SyncCommitteeBits:      hexutil.Bytes(s.SyncAggregate.SyncCommitteeBits),
		SyncCommitteeSignature: s.SyncAggregate.SyncCommitteeSignature,
	})
}

type Eth1Data struct {
	*eth.Eth1Data
}

func (e *Eth1Data) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		DepositRoot  hexutil.Bytes `json:"deposit_root"`
		DepositCount string        `json:"deposit_count"`
		BlockHash    hexutil.Bytes `json:"block_hash"`
	}{
		DepositRoot:  e.DepositRoot,
		DepositCount: fmt.Sprintf("%d", e.DepositCount),
		BlockHash:    e.BlockHash,
	})
}

func (b *BlindedBeaconBlockBodyBellatrix) MarshalJSON() ([]byte, error) {
	sve := make([]*SignedVoluntaryExit, len(b.BlindedBeaconBlockBodyBellatrix.VoluntaryExits))
	for i := range b.BlindedBeaconBlockBodyBellatrix.VoluntaryExits {
		sve[i] = &SignedVoluntaryExit{SignedVoluntaryExit: b.BlindedBeaconBlockBodyBellatrix.VoluntaryExits[i]}
	}
	deps := make([]*Deposit, len(b.BlindedBeaconBlockBodyBellatrix.Deposits))
	for i := range b.BlindedBeaconBlockBodyBellatrix.Deposits {
		deps[i] = &Deposit{Deposit: b.BlindedBeaconBlockBodyBellatrix.Deposits[i]}
	}
	atts := make([]*Attestation, len(b.BlindedBeaconBlockBodyBellatrix.Attestations))
	for i := range b.BlindedBeaconBlockBodyBellatrix.Attestations {
		atts[i] = &Attestation{Attestation: b.BlindedBeaconBlockBodyBellatrix.Attestations[i]}
	}
	atsl := make([]*AttesterSlashing, len(b.BlindedBeaconBlockBodyBellatrix.AttesterSlashings))
	for i := range b.BlindedBeaconBlockBodyBellatrix.AttesterSlashings {
		atsl[i] = &AttesterSlashing{AttesterSlashing: b.BlindedBeaconBlockBodyBellatrix.AttesterSlashings[i]}
	}
	pros := make([]*ProposerSlashing, len(b.BlindedBeaconBlockBodyBellatrix.ProposerSlashings))
	for i := range b.BlindedBeaconBlockBodyBellatrix.ProposerSlashings {
		pros[i] = &ProposerSlashing{ProposerSlashing: b.BlindedBeaconBlockBodyBellatrix.ProposerSlashings[i]}
	}
	return json.Marshal(struct {
		RandaoReveal           hexutil.Bytes           `json:"randao_reveal"`
		Eth1Data               *Eth1Data               `json:"eth1_data"`
		Graffiti               hexutil.Bytes           `json:"graffiti"`
		ProposerSlashings      []*ProposerSlashing     `json:"proposer_slashings"`
		AttesterSlashings      []*AttesterSlashing     `json:"attester_slashings"`
		Attestations           []*Attestation          `json:"attestations"`
		Deposits               []*Deposit              `json:"deposits"`
		VoluntaryExits         []*SignedVoluntaryExit  `json:"voluntary_exits"`
		SyncAggregate          *SyncAggregate          `json:"sync_aggregate"`
		ExecutionPayloadHeader *ExecutionPayloadHeader `json:"execution_payload_header"`
	}{
		RandaoReveal:           b.RandaoReveal,
		Eth1Data:               &Eth1Data{b.BlindedBeaconBlockBodyBellatrix.Eth1Data},
		Graffiti:               b.BlindedBeaconBlockBodyBellatrix.Graffiti,
		ProposerSlashings:      pros,
		AttesterSlashings:      atsl,
		Attestations:           atts,
		Deposits:               deps,
		VoluntaryExits:         sve,
		SyncAggregate:          &SyncAggregate{b.BlindedBeaconBlockBodyBellatrix.SyncAggregate},
		ExecutionPayloadHeader: &ExecutionPayloadHeader{ExecutionPayloadHeader: b.BlindedBeaconBlockBodyBellatrix.ExecutionPayloadHeader},
	})
}

type SignedBLSToExecutionChange struct {
	*eth.SignedBLSToExecutionChange
}

func (ch *SignedBLSToExecutionChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Message   *BLSToExecutionChange `json:"message"`
		Signature hexutil.Bytes         `json:"signature"`
	}{
		Signature: ch.Signature,
		Message:   &BLSToExecutionChange{ch.Message},
	})
}

type BLSToExecutionChange struct {
	*eth.BLSToExecutionChange
}

func (ch *BLSToExecutionChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ValidatorIndex     string        `json:"validator_index"`
		FromBlsPubkey      hexutil.Bytes `json:"from_bls_pubkey"`
		ToExecutionAddress hexutil.Bytes `json:"to_execution_address"`
	}{
		ValidatorIndex:     fmt.Sprintf("%d", ch.ValidatorIndex),
		FromBlsPubkey:      ch.FromBlsPubkey,
		ToExecutionAddress: ch.ToExecutionAddress,
	})
}

type SignedBlindedBeaconBlockCapella struct {
	*eth.SignedBlindedBeaconBlockCapella
}

type BlindedBeaconBlockCapella struct {
	*eth.BlindedBeaconBlockCapella
}

type BlindedBeaconBlockBodyCapella struct {
	*eth.BlindedBeaconBlockBodyCapella
}

func (b *SignedBlindedBeaconBlockCapella) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Message   *BlindedBeaconBlockCapella `json:"message"`
		Signature hexutil.Bytes              `json:"signature"`
	}{
		Message:   &BlindedBeaconBlockCapella{b.Block},
		Signature: b.Signature,
	})
}

func (b *BlindedBeaconBlockCapella) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Slot          string                         `json:"slot"`
		ProposerIndex string                         `json:"proposer_index"`
		ParentRoot    hexutil.Bytes                  `json:"parent_root"`
		StateRoot     hexutil.Bytes                  `json:"state_root"`
		Body          *BlindedBeaconBlockBodyCapella `json:"body"`
	}{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    b.ParentRoot,
		StateRoot:     b.StateRoot,
		Body:          &BlindedBeaconBlockBodyCapella{b.Body},
	})
}

func (b *BlindedBeaconBlockBodyCapella) MarshalJSON() ([]byte, error) {
	sve := make([]*SignedVoluntaryExit, len(b.VoluntaryExits))
	for i := range b.VoluntaryExits {
		sve[i] = &SignedVoluntaryExit{SignedVoluntaryExit: b.VoluntaryExits[i]}
	}
	deps := make([]*Deposit, len(b.Deposits))
	for i := range b.Deposits {
		deps[i] = &Deposit{Deposit: b.Deposits[i]}
	}
	atts := make([]*Attestation, len(b.Attestations))
	for i := range b.Attestations {
		atts[i] = &Attestation{Attestation: b.Attestations[i]}
	}
	atsl := make([]*AttesterSlashing, len(b.AttesterSlashings))
	for i := range b.AttesterSlashings {
		atsl[i] = &AttesterSlashing{AttesterSlashing: b.AttesterSlashings[i]}
	}
	pros := make([]*ProposerSlashing, len(b.ProposerSlashings))
	for i := range b.ProposerSlashings {
		pros[i] = &ProposerSlashing{ProposerSlashing: b.ProposerSlashings[i]}
	}
	chs := make([]*SignedBLSToExecutionChange, len(b.BlsToExecutionChanges))
	for i := range b.BlsToExecutionChanges {
		chs[i] = &SignedBLSToExecutionChange{SignedBLSToExecutionChange: b.BlsToExecutionChanges[i]}
	}
	return json.Marshal(struct {
		RandaoReveal           hexutil.Bytes                  `json:"randao_reveal"`
		Eth1Data               *Eth1Data                      `json:"eth1_data"`
		Graffiti               hexutil.Bytes                  `json:"graffiti"`
		ProposerSlashings      []*ProposerSlashing            `json:"proposer_slashings"`
		AttesterSlashings      []*AttesterSlashing            `json:"attester_slashings"`
		Attestations           []*Attestation                 `json:"attestations"`
		Deposits               []*Deposit                     `json:"deposits"`
		VoluntaryExits         []*SignedVoluntaryExit         `json:"voluntary_exits"`
		BLSToExecutionChanges  []*SignedBLSToExecutionChange  `json:"bls_to_execution_changes"`
		SyncAggregate          *SyncAggregate                 `json:"sync_aggregate"`
		ExecutionPayloadHeader *ExecutionPayloadHeaderCapella `json:"execution_payload_header"`
	}{
		RandaoReveal:           b.RandaoReveal,
		Eth1Data:               &Eth1Data{b.Eth1Data},
		Graffiti:               b.Graffiti,
		ProposerSlashings:      pros,
		AttesterSlashings:      atsl,
		Attestations:           atts,
		Deposits:               deps,
		VoluntaryExits:         sve,
		BLSToExecutionChanges:  chs,
		SyncAggregate:          &SyncAggregate{b.SyncAggregate},
		ExecutionPayloadHeader: &ExecutionPayloadHeaderCapella{ExecutionPayloadHeaderCapella: b.ExecutionPayloadHeader},
	})
}

type ErrorMessage struct {
	Code        int      `json:"code"`
	Message     string   `json:"message"`
	Stacktraces []string `json:"stacktraces,omitempty"`
}
