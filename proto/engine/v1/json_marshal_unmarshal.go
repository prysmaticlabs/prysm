package enginev1

import (
	"encoding/binary"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// HexBytes implements a custom json.Marshaler/Unmarshaler for byte slices that encodes them as
// hex strings per the Ethereum JSON-RPC specification.
type HexBytes []byte

// Quantity implements a custom json.Marshaler/Unmarshaler for uint64 that encodes them as
// big-endian hex strings per the Ethereum JSON-RPC specification.
type Quantity uint64

// MarshalJSON --
func (b HexBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(hexutil.Encode(b))
}

// UnmarshalJSON --
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

// MarshalJSON --
func (q Quantity) MarshalJSON() ([]byte, error) {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, uint64(q))
	return json.Marshal(hexutil.Encode(enc))
}

// UnmarshalJSON --
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

type executionBlockJSON struct {
	Number           Quantity   `json:"number"`
	Hash             HexBytes   `json:"hash"`
	ParentHash       HexBytes   `json:"parentHash"`
	Sha3Uncles       HexBytes   `json:"sha3Uncles"`
	Miner            HexBytes   `json:"miner"`
	StateRoot        HexBytes   `json:"stateRoot"`
	TransactionsRoot HexBytes   `json:"transactionsRoot"`
	ReceiptsRoot     HexBytes   `json:"receiptsRoot"`
	LogsBloom        HexBytes   `json:"logsBloom"`
	Difficulty       Quantity   `json:"difficulty"`
	TotalDifficulty  Quantity   `json:"totalDifficulty"`
	GasLimit         Quantity   `json:"gasLimit"`
	GasUsed          Quantity   `json:"gasUsed"`
	Timestamp        Quantity   `json:"timestamp"`
	BaseFeePerGas    HexBytes   `json:"baseFeePerGas"`
	ExtraData        HexBytes   `json:"extraData"`
	MixHash          HexBytes   `json:"mixHash"`
	Nonce            HexBytes   `json:"nonce"`
	Transactions     []HexBytes `json:"transactions"`
	Uncles           []HexBytes `json:"uncles"`
}

// MarshalJSON defines a custom json.Marshaler interface implementation
// that uses custom json.Marshalers for the HexBytes and Quantity types.
func (e *ExecutionBlock) MarshalJSON() ([]byte, error) {
	transactions := make([]HexBytes, len(e.Transactions))
	for i, tx := range e.Transactions {
		transactions[i] = tx
	}
	uncles := make([]HexBytes, len(e.Uncles))
	for i, ucl := range e.Uncles {
		uncles[i] = ucl
	}
	return json.Marshal(executionBlockJSON{
		Number:           Quantity(e.Number),
		Hash:             e.Hash,
		ParentHash:       e.ParentHash,
		Sha3Uncles:       e.Sha3Uncles,
		Miner:            e.Miner,
		StateRoot:        e.StateRoot,
		TransactionsRoot: e.TransactionsRoot,
		ReceiptsRoot:     e.ReceiptsRoot,
		LogsBloom:        e.LogsBloom,
		Difficulty:       Quantity(e.Difficulty),
		TotalDifficulty:  Quantity(e.TotalDifficulty),
		GasLimit:         Quantity(e.GasLimit),
		GasUsed:          Quantity(e.GasUsed),
		Timestamp:        Quantity(e.Timestamp),
		ExtraData:        e.ExtraData,
		MixHash:          e.MixHash,
		Nonce:            e.Nonce,
		BaseFeePerGas:    e.BaseFeePerGas,
		Transactions:     transactions,
		Uncles:           uncles,
	})
}

// UnmarshalJSON defines a custom json.Unmarshaler interface implementation
// that uses custom json.Unmarshalers for the HexBytes and Quantity types.
func (e *ExecutionBlock) UnmarshalJSON(enc []byte) error {
	dec := executionBlockJSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}
	*e = ExecutionBlock{}
	e.Number = uint64(dec.Number)
	e.Hash = dec.Hash
	e.ParentHash = dec.ParentHash
	e.Sha3Uncles = dec.Sha3Uncles
	e.Miner = dec.Miner
	e.StateRoot = dec.StateRoot
	e.TransactionsRoot = dec.TransactionsRoot
	e.ReceiptsRoot = dec.ReceiptsRoot
	e.LogsBloom = dec.LogsBloom
	e.Difficulty = uint64(dec.Difficulty)
	e.TotalDifficulty = uint64(dec.TotalDifficulty)
	e.GasLimit = uint64(dec.GasLimit)
	e.GasUsed = uint64(dec.GasUsed)
	e.Timestamp = uint64(dec.Timestamp)
	e.ExtraData = dec.ExtraData
	e.MixHash = dec.MixHash
	e.Nonce = dec.Nonce
	e.BaseFeePerGas = dec.BaseFeePerGas
	transactions := make([][]byte, len(dec.Transactions))
	for i, tx := range dec.Transactions {
		transactions[i] = tx
	}
	e.Transactions = transactions
	uncles := make([][]byte, len(dec.Uncles))
	for i, ucl := range dec.Uncles {
		uncles[i] = ucl
	}
	e.Uncles = uncles
	return nil
}

type executionPayloadJSON struct {
	ParentHash    HexBytes   `json:"parentHash"`
	FeeRecipient  HexBytes   `json:"feeRecipient"`
	StateRoot     HexBytes   `json:"stateRoot"`
	ReceiptsRoot  HexBytes   `json:"receiptsRoot"`
	LogsBloom     HexBytes   `json:"logsBloom"`
	Random        HexBytes   `json:"random"`
	BlockNumber   Quantity   `json:"blockNumber"`
	GasLimit      Quantity   `json:"gasLimit"`
	GasUsed       Quantity   `json:"gasUsed"`
	Timestamp     Quantity   `json:"timestamp"`
	ExtraData     HexBytes   `json:"extraData"`
	BaseFeePerGas HexBytes   `json:"baseFeePerGas"`
	BlockHash     HexBytes   `json:"blockHash"`
	Transactions  []HexBytes `json:"transactions"`
}

// MarshalJSON --
func (e *ExecutionPayload) MarshalJSON() ([]byte, error) {
	transactions := make([]HexBytes, len(e.Transactions))
	for i, tx := range e.Transactions {
		transactions[i] = tx
	}
	return json.Marshal(executionPayloadJSON{
		ParentHash:    e.ParentHash,
		FeeRecipient:  e.FeeRecipient,
		StateRoot:     e.StateRoot,
		ReceiptsRoot:  e.ReceiptsRoot,
		LogsBloom:     e.LogsBloom,
		Random:        e.Random,
		BlockNumber:   Quantity(e.BlockNumber),
		GasLimit:      Quantity(e.GasLimit),
		GasUsed:       Quantity(e.GasUsed),
		Timestamp:     Quantity(e.Timestamp),
		ExtraData:     e.ExtraData,
		BaseFeePerGas: e.BaseFeePerGas,
		BlockHash:     e.BlockHash,
		Transactions:  transactions,
	})
}

// UnmarshalJSON --
func (e *ExecutionPayload) UnmarshalJSON(enc []byte) error {
	dec := executionPayloadJSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}
	*e = ExecutionPayload{}
	e.ParentHash = dec.ParentHash
	e.FeeRecipient = dec.FeeRecipient
	e.StateRoot = dec.StateRoot
	e.ReceiptsRoot = dec.ReceiptsRoot
	e.LogsBloom = dec.LogsBloom
	e.Random = dec.Random
	e.BlockNumber = uint64(dec.BlockNumber)
	e.GasLimit = uint64(dec.GasLimit)
	e.GasUsed = uint64(dec.GasUsed)
	e.Timestamp = uint64(dec.Timestamp)
	e.ExtraData = dec.ExtraData
	e.BaseFeePerGas = dec.BaseFeePerGas
	e.BlockHash = dec.BlockHash
	transactions := make([][]byte, len(dec.Transactions))
	for i, tx := range dec.Transactions {
		transactions[i] = tx
	}
	e.Transactions = transactions
	return nil
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
