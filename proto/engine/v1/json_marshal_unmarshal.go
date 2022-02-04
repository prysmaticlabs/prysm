package enginev1

import (
	"encoding/binary"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"google.golang.org/protobuf/encoding/protojson"
)

type HexBytes []byte
type Quantity uint64

func (b HexBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(hexutil.Encode(b))
}

func (b *HexBytes) UnmarshalJSON(enc []byte) error {
	if len(enc) == 0 {
		*b = make([]byte, 0)
		return nil
	}
	var hexString string
	if err := json.Unmarshal(enc, &hexString); err != nil {
		return err
	}
	dst, err := hexutil.Decode(hexString)
	if err != nil {
		return err
	}
	*b = dst
	return nil
}

func (q Quantity) MarshalJSON() ([]byte, error) {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, uint64(q))
	return json.Marshal(hexutil.Encode(enc))
}

func (q *Quantity) UnmarshalJSON(enc []byte) error {
	if len(enc) == 0 {
		*q = 0
		return nil
	}
	var hexString string
	if err := json.Unmarshal(enc, &hexString); err != nil {
		return err
	}
	dst, err := hexutil.Decode(hexString)
	if err != nil {
		return err
	}
	*q = Quantity(binary.BigEndian.Uint64(dst))
	return nil
}

type executionPayloadJSON struct {
	ParentHash    HexBytes `json:"parentHash"`
	FeeRecipient  HexBytes `json:"feeRecipient"`
	StateRoot     HexBytes `json:"stateRoot"`
	ReceiptsRoot  HexBytes `json:"receiptsRoot"`
	LogsBloom     HexBytes `json:"logsBloom"`
	Random        HexBytes `json:"random"`
	BlockNumber   Quantity `json:"blockNumber"`
	GasLimit      Quantity `json:"gasLimit"`
	GasUsed       Quantity `json:"gasUsed"`
	Timestamp     Quantity `json:"timestamp"`
	ExtraData     HexBytes `json:"extraData"`
	BaseFeePerGas HexBytes `json:"baseFeePerGas"`
	BlockHash     HexBytes `json:"blockHash"`
	Transactions  [][]byte `json:"transactions"`
}

// MarshalJSON defines a custom json.Marshaler interface implementation
// that uses protojson underneath the hood, as protojson will respect
// proper struct tag naming conventions required for the JSON-RPC engine API to work.
func (e *ExecutionPayload) MarshalJSON() ([]byte, error) {
	return json.Marshal(executionPayloadJSON{
		ParentHash:    HexBytes(e.ParentHash),
		FeeRecipient:  HexBytes(e.FeeRecipient),
		StateRoot:     HexBytes(e.StateRoot),
		ReceiptsRoot:  HexBytes(e.ReceiptsRoot),
		LogsBloom:     HexBytes(e.LogsBloom),
		Random:        HexBytes(e.Random),
		BlockNumber:   Quantity(e.BlockNumber),
		GasLimit:      Quantity(e.GasLimit),
		GasUsed:       Quantity(e.GasUsed),
		Timestamp:     Quantity(e.Timestamp),
		ExtraData:     HexBytes(e.ExtraData),
		BaseFeePerGas: HexBytes(e.ExtraData),
		BlockHash:     HexBytes(e.ExtraData),
		Transactions:  e.Transactions,
	})
}

// UnmarshalJSON defines a custom json.Unmarshaler interface implementation
// that uses protojson underneath the hood, as protojson will respect
// proper struct tag naming conventions required for the JSON-RPC engine API to work.
func (e *ExecutionPayload) UnmarshalJSON(enc []byte) error {
	return protojson.Unmarshal(enc, e)
}

type payloadAttributesJSON struct {
	Timestamp             Quantity `json:"timestamp"`
	Random                HexBytes `json:"random"`
	SuggestedFeeRecipient HexBytes `json:"suggestedFeeRecipient"`
}

// MarshalJSON --
func (p *PayloadAttributes) MarshalJSON() ([]byte, error) {
	return json.Marshal(payloadAttributesJSON{
		Timestamp:             Quantity(p.Timestamp),
		Random:                p.Random,
		SuggestedFeeRecipient: p.SuggestedFeeRecipient,
	})
}

// UnmarshalJSON --
func (p *PayloadAttributes) UnmarshalJSON(enc []byte) error {
	dec := payloadAttributesJSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}
	*p = PayloadAttributes{}
	p.Timestamp = uint64(dec.Timestamp)
	p.Random = dec.Random
	p.SuggestedFeeRecipient = dec.SuggestedFeeRecipient
	return nil
}

type payloadStatusJSON struct {
	LatestValidHash HexBytes `json:"latestValidHash"`
	Status          string   `json:"status"`
	ValidationError string   `json:"validationError"`
}

// MarshalJSON --
func (p *PayloadStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(payloadStatusJSON{
		LatestValidHash: p.LatestValidHash,
		Status:          p.Status.String(),
		ValidationError: p.ValidationError,
	})
}

// UnmarshalJSON --
func (p *PayloadStatus) UnmarshalJSON(enc []byte) error {
	dec := payloadStatusJSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}
	*p = PayloadStatus{}
	p.LatestValidHash = dec.LatestValidHash
	p.Status = PayloadStatus_Status(PayloadStatus_Status_value[dec.Status])
	p.ValidationError = dec.ValidationError
	return nil
}

type forkchoiceStateJSON struct {
	HeadBlockHash      HexBytes `json:"headBlockHash"`
	SafeBlockHash      HexBytes `json:"safeBlockHash"`
	FinalizedBlockHash HexBytes `json:"finalizedBlockHash"`
}

// MarshalJSON --
func (f *ForkchoiceState) MarshalJSON() ([]byte, error) {
	return json.Marshal(forkchoiceStateJSON{
		HeadBlockHash:      f.HeadBlockHash,
		SafeBlockHash:      f.SafeBlockHash,
		FinalizedBlockHash: f.FinalizedBlockHash,
	})
}

// UnmarshalJSON --
func (f *ForkchoiceState) UnmarshalJSON(enc []byte) error {
	dec := forkchoiceStateJSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}
	*f = ForkchoiceState{}
	f.HeadBlockHash = dec.HeadBlockHash
	f.SafeBlockHash = dec.SafeBlockHash
	f.FinalizedBlockHash = dec.FinalizedBlockHash
	return nil
}
