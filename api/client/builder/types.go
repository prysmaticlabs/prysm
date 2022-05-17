package builder

import (
	"encoding/json"
	"fmt"
	v1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

type SignedValidatorRegistration struct {
	*eth.SignedValidatorRegistrationV1
}

type ValidatorRegistration struct {
	*eth.ValidatorRegistrationV1
}

func (r *SignedValidatorRegistration) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
		Message *ValidatorRegistration `json:"message,omitempty"`
		Signature hexSlice `json:"signature,omitempty"`
	}{
		Message: &ValidatorRegistration{r.Message},
		Signature: r.SignedValidatorRegistrationV1.Signature,
	})
}

func (r *ValidatorRegistration) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
		FeeRecipient hexSlice `json:"fee_recipient,omitempty"`
		GasLimit string `json:"gas_limit"`
		Timestamp string `json:"timestamp"`
		Pubkey hexSlice `json:"pubkey,omitempty"`
		*eth.ValidatorRegistrationV1
	}{
		FeeRecipient: r.FeeRecipient,
		GasLimit: fmt.Sprintf("%d", r.GasLimit),
		Timestamp: fmt.Sprintf("%d", r.Timestamp),
		Pubkey: r.Pubkey,
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
	Version string       `json:"version"`
	Data struct {
		Signature hexSlice `json:"signature"`
		Message *BuilderBid `json:"message"`
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
	// TODO: it looks like Value should probably be a uint
	valueHack := []byte(strconv.FormatUint(uint64(bb.Value), 10))
	header, err := bb.Header.ToProto()
	if err != nil {
		return nil, err
	}
	return &eth.BuilderBid{
		Header: header,
		Value:  valueHack,
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
	Header *ExecutionPayloadHeader
	Value Uint64String
	Pubkey hexSlice
}

type ExecutionPayloadHeader struct {
	ParentHash hexSlice `json:"parent_hash"`
	FeeRecipient hexSlice `json:"fee_recipient"`
	StateRoot hexSlice `json:"state_root"`
	ReceiptsRoot hexSlice `json:"receipts_root"`
	LogsBloom hexSlice `json:"logs_bloom"`
	PrevRandao hexSlice `json:"prev_randao"`
	BlockNumber Uint64String`json:"block_number"`
	GasLimit Uint64String`json:"gas_limit"`
	GasUsed Uint64String`json:"gas_used"`
	Timestamp Uint64String`json:"timestamp"`
	ExtraData hexSlice `json:"extra_data"`
	BaseFeePerGas Uint64String `json:"base_fee_per_gas"`
	BlockHash hexSlice `json:"block_hash"`
	TransactionsRoot hexSlice `json:"transactions_root"`
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
		ParentHash: h.ExecutionPayloadHeader.ParentHash,
		FeeRecipient: h.ExecutionPayloadHeader.FeeRecipient,
		StateRoot: h.ExecutionPayloadHeader.StateRoot,
		ReceiptsRoot: h.ExecutionPayloadHeader.ReceiptsRoot,
		LogsBloom: h.ExecutionPayloadHeader.LogsBloom,
		PrevRandao: h.ExecutionPayloadHeader.PrevRandao,
		BlockNumber: Uint64String(h.ExecutionPayloadHeader.BlockNumber),
		GasLimit: Uint64String(h.ExecutionPayloadHeader.GasLimit),
		GasUsed: Uint64String(h.ExecutionPayloadHeader.GasUsed),
		Timestamp: Uint64String(h.ExecutionPayloadHeader.Timestamp),
		ExtraData: h.ExecutionPayloadHeader.ExtraData,
		BaseFeePerGas: Uint64String(bfpg),
		BlockHash: h.ExecutionPayloadHeader.BlockHash,
		TransactionsRoot: h.ExecutionPayloadHeader.TransactionsRoot,
	})
}

type ExecPayloadResponse struct {
	Version string       `json:"version"`
	Data ExecutionPayload `json:"data"`
}

type ExecutionPayload struct {
	ParentHash hexSlice `json:"parent_hash"`
	FeeRecipient hexSlice `json:"fee_recipient"`
	StateRoot hexSlice `json:"state_root"`
	ReceiptsRoot hexSlice `json:"receipts_root"`
	LogsBloom hexSlice `json:"logs_bloom"`
	PrevRandao hexSlice `json:"prev_randao"`
	BlockNumber Uint64String`json:"block_number"`
	GasLimit Uint64String`json:"gas_limit"`
	GasUsed Uint64String`json:"gas_used"`
	Timestamp Uint64String`json:"timestamp"`
	ExtraData hexSlice `json:"extra_data"`
	BaseFeePerGas Uint64String `json:"base_fee_per_gas"`
	BlockHash hexSlice `json:"block_hash"`
	Transactions []hexSlice `json:"transactions"`
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
		ParentHash:       p.ParentHash,
		FeeRecipient:     p.FeeRecipient,
		StateRoot:        p.StateRoot,
		ReceiptsRoot:     p.ReceiptsRoot,
		LogsBloom:        p.LogsBloom,
		PrevRandao:       p.PrevRandao,
		BlockNumber:      uint64(p.BlockNumber),
		GasLimit:         uint64(p.GasLimit),
		GasUsed:          uint64(p.GasUsed),
		Timestamp:        uint64(p.Timestamp),
		ExtraData:        p.ExtraData,
		BaseFeePerGas:    baseFeeHack,
		BlockHash:        p.BlockHash,
		Transactions:     txs,
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
	return json.Marshal(struct{
		Message *BlindedBeaconBlockBellatrix `json:"message,omitempty"`
		Signature hexSlice `json:"signature,omitempty"`
	}{
		Message: &BlindedBeaconBlockBellatrix{r.SignedBlindedBeaconBlockBellatrix.Block},
		Signature: r.SignedBlindedBeaconBlockBellatrix.Signature,
	})
}

func (b *BlindedBeaconBlockBellatrix) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
		Slot string `json:"slot"`
		ProposerIndex string `json:"proposer_index"`
		ParentRoot hexSlice `json:"parent_root"`
		StateRoot hexSlice `json:"state_root"`
		Body *BlindedBeaconBlockBodyBellatrix `json:"body"`
	}{
		Slot: fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot: b.ParentRoot,
		StateRoot: b.StateRoot,
		Body: &BlindedBeaconBlockBodyBellatrix{b.BlindedBeaconBlockBellatrix.Body},
	})
}

type ProposerSlashing struct {
	*eth.ProposerSlashing
}

func (s *ProposerSlashing) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
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
	return json.Marshal(struct{
		Header *BeaconBlockHeader `json:"message"`
		Signature hexSlice `json:"signature"`
	}{
		Header: &BeaconBlockHeader{h.SignedBeaconBlockHeader.Header},
		Signature: h.SignedBeaconBlockHeader.Signature,
	})
}

type BeaconBlockHeader struct {
	*eth.BeaconBlockHeader
}

func (h *BeaconBlockHeader) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
		Slot string `json:"slot"`
		ProposerIndex string `json:"proposer_index"`
		ParentRoot hexSlice `json:"parent_root"`
		StateRoot hexSlice `json:"state_root"`
		BodyRoot hexSlice `json:"body_root"`
	}{
		Slot: fmt.Sprintf("%d", h.BeaconBlockHeader.Slot),
		ProposerIndex: fmt.Sprintf("%d", h.BeaconBlockHeader.ProposerIndex),
		ParentRoot: h.BeaconBlockHeader.ParentRoot,
		StateRoot: h.BeaconBlockHeader.StateRoot,
		BodyRoot: h.BeaconBlockHeader.BodyRoot,
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
	return json.Marshal(struct{
		AttestingIndices []string `json:"attesting_indices"`
		Data *AttestationData
		Signature hexSlice `json:"signature"`
	}{
		AttestingIndices: indices,
		Data: &AttestationData{a.IndexedAttestation.Data},
		Signature: a.IndexedAttestation.Signature,
	})
}

type AttesterSlashing struct {
	*eth.AttesterSlashing
}

func (s *AttesterSlashing) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
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
	return json.Marshal(struct{
		Epoch string `json:"epoch"`
		Root hexSlice `json:"root"`
	}{
		Epoch: fmt.Sprintf("%d", c.Checkpoint.Epoch),
		Root: c.Checkpoint.Root,
	})
}

type AttestationData struct {
	*eth.AttestationData
}

func (a *AttestationData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
		Slot string `json:"slot"`
		Index string `json:"index"`
		BeaconBlockRoot hexSlice `json:"beacon_block_root"`
		Source *Checkpoint `json:"source"`
		Target *Checkpoint `json:"target"`
	}{
		Slot: fmt.Sprintf("%d", a.AttestationData.Slot),
		Index: fmt.Sprintf("%d", a.AttestationData.CommitteeIndex),
		BeaconBlockRoot: a.AttestationData.BeaconBlockRoot,
		Source: &Checkpoint{a.AttestationData.Source},
		Target: &Checkpoint{a.AttestationData.Target},
	})
}

type Attestation struct {
	*eth.Attestation
}

func (a *Attestation) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
		AggregationBits hexSlice `json:"aggregation_bits,omitempty"`
		Data            *AttestationData `json:"data,omitempty"`
		Signature       hexSlice           `json:"signature,omitempty" ssz-size:"96"`
	}{
		AggregationBits: hexSlice(a.Attestation.AggregationBits),
		Data: &AttestationData{a.Attestation.Data},
		Signature: a.Attestation.Signature,
	})
}

type DepositData struct {
	*eth.Deposit_Data
}

func (d *DepositData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
		PublicKey             hexSlice `json:"pubkey,omitempty"`
		WithdrawalCredentials hexSlice `json:"withdrawal_credentials,omitempty"`
		Amount                string `json:"amount,omitempty"`
		Signature             hexSlice `json:"signature,omitempty"`
	}{
		PublicKey: d.PublicKey,
		WithdrawalCredentials: d.WithdrawalCredentials,
		Amount: fmt.Sprintf("%d", d.Amount),
		Signature: d.Signature,
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
	return json.Marshal(struct{
		Proof []hexSlice `json:"proof"`
		Data *DepositData `json:"data"`
	}{
		Proof: proof,
		Data: &DepositData{Deposit_Data: d.Deposit.Data},
	})
}

type SignedVoluntaryExit struct {
	*eth.SignedVoluntaryExit
}

func (sve *SignedVoluntaryExit) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
		Message *VoluntaryExit `json:"message,omitempty"`
		Signature hexSlice `json:"signature,omitempty"`
	}{
		Signature: sve.SignedVoluntaryExit.Signature,
		Message: &VoluntaryExit{sve.SignedVoluntaryExit.Exit},
	})
}

type VoluntaryExit struct {
	*eth.VoluntaryExit
}

func (ve *VoluntaryExit) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
		Epoch string `json:"epoch"`
		ValidatorIndex string `json:"validator_index"`
	}{
		Epoch: fmt.Sprintf("%d", ve.Epoch),
		ValidatorIndex: fmt.Sprintf("%d", ve.ValidatorIndex),
	})
}

type SyncAggregate struct {
	*eth.SyncAggregate
}

func (s *SyncAggregate) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
		SyncCommitteeBits hexSlice `json:"sync_committee_bits"`
		SyncCommitteeSignature hexSlice `json:"sync_committee_signature"`
	}{
		SyncCommitteeBits: hexSlice(s.SyncAggregate.SyncCommitteeBits),
		SyncCommitteeSignature: s.SyncAggregate.SyncCommitteeSignature,
	})
}

type Eth1Data struct {
	*eth.Eth1Data
}

func (e *Eth1Data) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
		DepositRoot hexSlice `json:"deposit_root,omitempty"`
		DepositCount string `json:"deposit_count,omitempty"`
		BlockHash hexSlice `json:"block_hash,omitempty"`
	}{
		DepositRoot: e.DepositRoot,
		DepositCount: fmt.Sprintf("%d", e.DepositCount),
		BlockHash: e.BlockHash,
	})
}

func (b *BlindedBeaconBlockBodyBellatrix) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
		RandaoReveal hexSlice `json:"randao_reveal,omitempty"`
		Eth1Data *Eth1Data    `json:"eth1_data,omitempty"`
		Graffiti hexSlice     `json:"graffiti,omitempty"`
		ProposerSlashings []*ProposerSlashing `json:"proposer_slashings,omitempty"`
		AttesterSlashings []*AttesterSlashing `json:"attester_slashings,omitempty"`
		Attestations []*Attestation `json:"attestations"`
		Deposits []*Deposit                   `json:"deposits"`
		VoluntaryExits []*SignedVoluntaryExit `json:"voluntary_exits"`
		SyncAggregates *SyncAggregate         `json:"sync_aggregate"`
		ExecutionPayloadHeader *ExecutionPayloadHeader `json:"execution_payload_header"`
	}{
		RandaoReveal: b.RandaoReveal,
		Eth1Data: &Eth1Data{b.BlindedBeaconBlockBodyBellatrix.Eth1Data},
		Graffiti: b.BlindedBeaconBlockBodyBellatrix.Graffiti,
		ProposerSlashings: []*ProposerSlashing{},
		AttesterSlashings: []*AttesterSlashing{},
		Attestations: []*Attestation{},
		Deposits: []*Deposit{},
		VoluntaryExits: []*SignedVoluntaryExit{},
		ExecutionPayloadHeader: &ExecutionPayloadHeader{ExecutionPayloadHeader: b.BlindedBeaconBlockBodyBellatrix.ExecutionPayloadHeader},
	})
}