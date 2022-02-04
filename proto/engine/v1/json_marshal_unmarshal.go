package enginev1

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
)

type executionPayloadAlias struct {
	*ExecutionPayload
}

func (e *executionPayloadAlias) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(e)
}

// MarshalJSON defines a custom json.Marshaler interface implementation
// that uses protojson underneath the hood, as protojson will respect
// proper struct tag naming conventions required for the JSON-RPC engine API to work.
func (e *ExecutionPayload) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ParentHash    hexBytes `json:"parentHash"`
		FeeRecipient  hexBytes `json:"feeRecipient"`
		StateRoot     hexBytes `json:"stateRoot"`
		ReceiptsRoot  hexBytes `json:"receiptsRoot"`
		LogsBloom     hexBytes `json:"logsBloom"`
		Random        hexBytes `json:"random"`
		BlockNumber   quantity `json:"blockNumber"`
		GasLimit      quantity `json:"gasLimit"`
		GasUsed       quantity `json:"gasUsed"`
		Timestamp     quantity `json:"timestamp"`
		ExtraData     hexBytes `json:"extraData"`
		BaseFeePerGas hexBytes `json:"baseFeePerGas"`
		BlockHash     hexBytes `json:"blockHash"`
		Transactions  [][]byte `json:"transactions"`
		*executionPayloadAlias
	}{
		ParentHash:    hexBytes(e.ParentHash),
		FeeRecipient:  hexBytes(e.FeeRecipient),
		StateRoot:     hexBytes(e.StateRoot),
		ReceiptsRoot:  hexBytes(e.ReceiptsRoot),
		LogsBloom:     hexBytes(e.LogsBloom),
		Random:        hexBytes(e.Random),
		BlockNumber:   quantity(e.BlockNumber),
		GasLimit:      quantity(e.GasLimit),
		GasUsed:       quantity(e.GasUsed),
		Timestamp:     quantity(e.Timestamp),
		ExtraData:     hexBytes(e.ExtraData),
		BaseFeePerGas: hexBytes(e.ExtraData),
		BlockHash:     hexBytes(e.ExtraData),
		Transactions:  e.Transactions,
		executionPayloadAlias: (*executionPayloadAlias)(&executionPayloadAlias{
			ExecutionPayload: e,
		}),
	})
}

// UnmarshalJSON defines a custom json.Unmarshaler interface implementation
// that uses protojson underneath the hood, as protojson will respect
// proper struct tag naming conventions required for the JSON-RPC engine API to work.
func (e *ExecutionPayload) UnmarshalJSON(enc []byte) error {
	return protojson.Unmarshal(enc, e)
}

type payloadAttributesAlias struct {
	*PayloadAttributes
}

func (e *payloadAttributesAlias) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(e)
}

// MarshalJSON --
func (p *PayloadAttributes) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Timestamp             quantity `json:"parentHash"`
		Random                hexBytes `json:"random"`
		SuggestedFeeRecipient hexBytes `json:"suggestedFeeRecipient"`
		*payloadAttributesAlias
	}{
		Timestamp:             quantity(p.Timestamp),
		Random:                hexBytes(p.Random),
		SuggestedFeeRecipient: hexBytes(p.SuggestedFeeRecipient),
		payloadAttributesAlias: (*payloadAttributesAlias)(&payloadAttributesAlias{
			PayloadAttributes: p,
		}),
	})
}

// UnmarshalJSON --
func (p *PayloadAttributes) UnmarshalJSON(enc []byte) error {
	return protojson.Unmarshal(enc, p)
}

type payloadStatusAlias struct {
	*PayloadStatus
}

func (e *payloadStatusAlias) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(e)
}

// MarshalJSON --
func (p *PayloadStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		LatestValidHash hexBytes `json:"latestValidHash"`
		*payloadStatusAlias
	}{
		LatestValidHash: hexBytes(p.LatestValidHash),
		payloadStatusAlias: (*payloadStatusAlias)(&payloadStatusAlias{
			PayloadStatus: p,
		}),
	})
}

// UnmarshalJSON --
func (p *PayloadStatus) UnmarshalJSON(enc []byte) error {
	return protojson.Unmarshal(enc, p)
}

type forkchoiceStateAlias struct {
	*ForkchoiceState
}

func (e *forkchoiceStateAlias) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(e)
}

// MarshalJSON --
func (f *ForkchoiceState) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		HeadBlockHash      hexBytes `json:"headBlockHash"`
		SafeBlockHash      hexBytes `json:"safeBlockHash"`
		FinalizedBlockHash hexBytes `json:"finalizedBlockHash"`
		*forkchoiceStateAlias
	}{
		HeadBlockHash:      hexBytes(f.HeadBlockHash),
		SafeBlockHash:      hexBytes(f.SafeBlockHash),
		FinalizedBlockHash: hexBytes(f.FinalizedBlockHash),
		forkchoiceStateAlias: (*forkchoiceStateAlias)(&forkchoiceStateAlias{
			ForkchoiceState: f,
		}),
	})
}

// UnmarshalJSON --
func (f *ForkchoiceState) UnmarshalJSON(enc []byte) error {
	return protojson.Unmarshal(enc, f)
}

type hexBytes []byte
type quantity uint64

func (b hexBytes) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%#x", b)), nil
}

func (b hexBytes) UnmarshalJSON(enc []byte) error {
	decoded, err := hex.DecodeString(strings.TrimPrefix(string(enc), "0x"))
	if err != nil {
		return err
	}
	b = decoded
	return nil
}

func (q quantity) MarshalJSON() ([]byte, error) {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, uint64(q))
	return enc, nil
}

func (q quantity) UnmarshalJSON(enc []byte) error {
	decoded, err := hex.DecodeString(strings.TrimPrefix(string(enc), "0x"))
	if err != nil {
		return err
	}
	q = quantity(binary.BigEndian.Uint64(decoded))
	return nil
}
