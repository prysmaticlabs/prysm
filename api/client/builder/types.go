package builder

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
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
		Message   *ValidatorRegistration `json:"message,omitempty"`
		Signature hexutil.Bytes          `json:"signature,omitempty"`
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
		Message   *ValidatorRegistration `json:"message,omitempty"`
		Signature hexutil.Bytes          `json:"signature,omitempty"`
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
		FeeRecipient hexutil.Bytes `json:"fee_recipient,omitempty"`
		GasLimit     string        `json:"gas_limit,omitempty"`
		Timestamp    string        `json:"timestamp,omitempty"`
		Pubkey       hexutil.Bytes `json:"pubkey,omitempty"`
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
		FeeRecipient hexutil.Bytes `json:"fee_recipient,omitempty"`
		GasLimit     string        `json:"gas_limit,omitempty"`
		Timestamp    string        `json:"timestamp,omitempty"`
		Pubkey       hexutil.Bytes `json:"pubkey,omitempty"`
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
	bi := new(big.Int)
	bi.SetBytes(bytesutil.ReverseByteOrder(b))
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

type ExecHeaderResponse struct {
	Version string `json:"version,omitempty"`
	Data    struct {
		Signature hexutil.Bytes `json:"signature,omitempty"`
		Message   *BuilderBid   `json:"message,omitempty"`
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
	Header *ExecutionPayloadHeader `json:"header,omitempty"`
	Value  Uint256                 `json:"value,omitempty"`
	Pubkey hexutil.Bytes           `json:"pubkey,omitempty"`
}

type ExecutionPayloadHeader struct {
	ParentHash       hexutil.Bytes `json:"parent_hash,omitempty"`
	FeeRecipient     hexutil.Bytes `json:"fee_recipient,omitempty"`
	StateRoot        hexutil.Bytes `json:"state_root,omitempty"`
	ReceiptsRoot     hexutil.Bytes `json:"receipts_root,omitempty"`
	LogsBloom        hexutil.Bytes `json:"logs_bloom,omitempty"`
	PrevRandao       hexutil.Bytes `json:"prev_randao,omitempty"`
	BlockNumber      Uint64String  `json:"block_number,omitempty"`
	GasLimit         Uint64String  `json:"gas_limit,omitempty"`
	GasUsed          Uint64String  `json:"gas_used,omitempty"`
	Timestamp        Uint64String  `json:"timestamp,omitempty"`
	ExtraData        hexutil.Bytes `json:"extra_data,omitempty"`
	BaseFeePerGas    Uint256       `json:"base_fee_per_gas,omitempty"`
	BlockHash        hexutil.Bytes `json:"block_hash,omitempty"`
	TransactionsRoot hexutil.Bytes `json:"transactions_root,omitempty"`
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
	Version string           `json:"version,omitempty"`
	Data    ExecutionPayload `json:"data,omitempty"`
}

type ExecutionPayload struct {
	ParentHash    hexutil.Bytes   `json:"parent_hash,omitempty"`
	FeeRecipient  hexutil.Bytes   `json:"fee_recipient,omitempty"`
	StateRoot     hexutil.Bytes   `json:"state_root,omitempty"`
	ReceiptsRoot  hexutil.Bytes   `json:"receipts_root,omitempty"`
	LogsBloom     hexutil.Bytes   `json:"logs_bloom,omitempty"`
	PrevRandao    hexutil.Bytes   `json:"prev_randao,omitempty"`
	BlockNumber   Uint64String    `json:"block_number,omitempty"`
	GasLimit      Uint64String    `json:"gas_limit,omitempty"`
	GasUsed       Uint64String    `json:"gas_used,omitempty"`
	Timestamp     Uint64String    `json:"timestamp,omitempty"`
	ExtraData     hexutil.Bytes   `json:"extra_data,omitempty"`
	BaseFeePerGas Uint256         `json:"base_fee_per_gas,omitempty"`
	BlockHash     hexutil.Bytes   `json:"block_hash,omitempty"`
	Transactions  []hexutil.Bytes `json:"transactions,omitempty"`
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
		Signature hexutil.Bytes                `json:"signature,omitempty"`
	}{
		Message:   &BlindedBeaconBlockBellatrix{r.SignedBlindedBeaconBlockBellatrix.Block},
		Signature: r.SignedBlindedBeaconBlockBellatrix.Signature,
	})
}

func (b *BlindedBeaconBlockBellatrix) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Slot          string                           `json:"slot"`
		ProposerIndex string                           `json:"proposer_index,omitempty"`
		ParentRoot    hexutil.Bytes                    `json:"parent_root,omitempty"`
		StateRoot     hexutil.Bytes                    `json:"state_root,omitempty"`
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
		Signature hexutil.Bytes      `json:"signature,omitempty"`
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
		Slot          string        `json:"slot,omitempty"`
		ProposerIndex string        `json:"proposer_index,omitempty"`
		ParentRoot    hexutil.Bytes `json:"parent_root,omitempty"`
		StateRoot     hexutil.Bytes `json:"state_root,omitempty"`
		BodyRoot      hexutil.Bytes `json:"body_root,omitempty"`
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
		AttestingIndices []string         `json:"attesting_indices,omitempty"`
		Data             *AttestationData `json:"data,omitempty"`
		Signature        hexutil.Bytes    `json:"signature,omitempty"`
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
		Epoch string        `json:"epoch,omitempty"`
		Root  hexutil.Bytes `json:"root,omitempty"`
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
		Slot            string        `json:"slot,omitempty"`
		Index           string        `json:"index,omitempty"`
		BeaconBlockRoot hexutil.Bytes `json:"beacon_block_root,omitempty"`
		Source          *Checkpoint   `json:"source,omitempty"`
		Target          *Checkpoint   `json:"target,omitempty"`
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
		AggregationBits hexutil.Bytes    `json:"aggregation_bits,omitempty"`
		Data            *AttestationData `json:"data,omitempty"`
		Signature       hexutil.Bytes    `json:"signature,omitempty" ssz-size:"96"`
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
		PublicKey             hexutil.Bytes `json:"pubkey,omitempty"`
		WithdrawalCredentials hexutil.Bytes `json:"withdrawal_credentials,omitempty"`
		Amount                string        `json:"amount,omitempty"`
		Signature             hexutil.Bytes `json:"signature,omitempty"`
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
		Message   *VoluntaryExit `json:"message,omitempty"`
		Signature hexutil.Bytes  `json:"signature,omitempty"`
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
		SyncCommitteeBits      hexutil.Bytes `json:"sync_committee_bits,omitempty"`
		SyncCommitteeSignature hexutil.Bytes `json:"sync_committee_signature,omitempty"`
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
		DepositRoot  hexutil.Bytes `json:"deposit_root,omitempty"`
		DepositCount string        `json:"deposit_count,omitempty"`
		BlockHash    hexutil.Bytes `json:"block_hash,omitempty"`
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
		RandaoReveal           hexutil.Bytes           `json:"randao_reveal,omitempty"`
		Eth1Data               *Eth1Data               `json:"eth1_data,omitempty"`
		Graffiti               hexutil.Bytes           `json:"graffiti,omitempty"`
		ProposerSlashings      []*ProposerSlashing     `json:"proposer_slashings,omitempty"`
		AttesterSlashings      []*AttesterSlashing     `json:"attester_slashings,omitempty"`
		Attestations           []*Attestation          `json:"attestations,omitempty"`
		Deposits               []*Deposit              `json:"deposits,omitempty"`
		VoluntaryExits         []*SignedVoluntaryExit  `json:"voluntary_exits,omitempty"`
		SyncAggregate          *SyncAggregate          `json:"sync_aggregate,omitempty"`
		ExecutionPayloadHeader *ExecutionPayloadHeader `json:"execution_payload_header,omitempty"`
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

type ErrorMessage struct {
	Code        int      `json:"code"`
	Message     string   `json:"message"`
	Stacktraces []string `json:"stacktraces,omitempty"`
}
