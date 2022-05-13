package builder

import (
	"encoding/json"
	"fmt"
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

func (hs *hexSlice) MarshalText() ([]byte, error) {
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

/*
func (ehr *ExecHeaderResponse) UnmarshalJSON(data []byte) error {
	container := struct{
		Version string       `json:"version"`
		Data json.RawMessage `json:"data"`
	}{}
	if err := json.Unmarshal(data, &container); err != nil {
		return err
	}
	signedContainer := struct{
		Signature hexSlice `json:"signature"`
		Message *BuilderBid `json:"message"`
	}{}
	if err := json.Unmarshal(container.Data, &signedContainer); err != nil {
		return err
	}
	ehr.SignedBuilderBid = &eth.SignedBuilderBid{
		//Message:   signedContainer.Message.BuilderBid,
		Signature: signedContainer.Signature,
	}

	return nil
}

 */

/*
func (bb *BuilderBid) UnmarshalJSON(data []byte) error {
	container := struct{
		Header *ExecutionPayloadHeader
		Value hexSlice
		Pubkey hexSlice
	}{}
	if err := json.Unmarshal(data, &container); err != nil {
		return err
	}
	bb.BuilderBid = &eth.BuilderBid{
		Header: container.Header.ExecutionPayloadHeader,
		Value:  container.Value,
		Pubkey: container.Pubkey,
	}
	return nil
}

 */

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