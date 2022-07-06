package enginev1

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
)

// PayloadIDBytes defines a custom type for Payload IDs used by the engine API
// client with proper JSON Marshal and Unmarshal methods to hex.
type PayloadIDBytes [8]byte

// MarshalJSON --
func (b PayloadIDBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(hexutil.Bytes(b[:]))
}

// ExecutionBlock is the response kind received by the eth_getBlockByHash and
// eth_getBlockByNumber endpoints via JSON-RPC.
type ExecutionBlock struct {
	gethtypes.Header
	Hash            common.Hash              `json:"hash"`
	Transactions    []*gethtypes.Transaction `json:"transactions"`
	TotalDifficulty string                   `json:"totalDifficulty"`
}

func (e *ExecutionBlock) UnmarshalJSON(enc []byte) error {
	if err := e.Header.UnmarshalJSON(enc); err != nil {
		return nil
	}
	decoded := make(map[string]interface{})
	if err := json.Unmarshal(enc, &decoded); err != nil {
		return err
	}
	blockHashStr := decoded["hash"].(string)
	e.Hash = common.HexToHash(blockHashStr)
	e.TotalDifficulty = decoded["totalDifficulty"].(string)
	rawTxs, ok := decoded["transactions"]
	if !ok {
		return nil
	}
	txs, ok := rawTxs.([]*gethtypes.Transaction)
	if !ok {
		return nil
	}
	e.Transactions = txs
	return nil
}

// UnmarshalJSON --
func (b *PayloadIDBytes) UnmarshalJSON(enc []byte) error {
	hexBytes := hexutil.Bytes(make([]byte, 0))
	if err := json.Unmarshal(enc, &hexBytes); err != nil {
		return err
	}
	res := [8]byte{}
	copy(res[:], hexBytes)
	*b = res
	return nil
}

type executionPayloadJSON struct {
	ParentHash    hexutil.Bytes   `json:"parentHash"`
	FeeRecipient  hexutil.Bytes   `json:"feeRecipient"`
	StateRoot     hexutil.Bytes   `json:"stateRoot"`
	ReceiptsRoot  hexutil.Bytes   `json:"receiptsRoot"`
	LogsBloom     hexutil.Bytes   `json:"logsBloom"`
	PrevRandao    hexutil.Bytes   `json:"prevRandao"`
	BlockNumber   hexutil.Uint64  `json:"blockNumber"`
	GasLimit      hexutil.Uint64  `json:"gasLimit"`
	GasUsed       hexutil.Uint64  `json:"gasUsed"`
	Timestamp     hexutil.Uint64  `json:"timestamp"`
	ExtraData     hexutil.Bytes   `json:"extraData"`
	BaseFeePerGas string          `json:"baseFeePerGas"`
	BlockHash     hexutil.Bytes   `json:"blockHash"`
	Transactions  []hexutil.Bytes `json:"transactions"`
}

// MarshalJSON --
func (e *ExecutionPayload) MarshalJSON() ([]byte, error) {
	transactions := make([]hexutil.Bytes, len(e.Transactions))
	for i, tx := range e.Transactions {
		transactions[i] = tx
	}
	baseFee := new(big.Int).SetBytes(bytesutil.ReverseByteOrder(e.BaseFeePerGas))
	baseFeeHex := hexutil.EncodeBig(baseFee)
	return json.Marshal(executionPayloadJSON{
		ParentHash:    e.ParentHash,
		FeeRecipient:  e.FeeRecipient,
		StateRoot:     e.StateRoot,
		ReceiptsRoot:  e.ReceiptsRoot,
		LogsBloom:     e.LogsBloom,
		PrevRandao:    e.PrevRandao,
		BlockNumber:   hexutil.Uint64(e.BlockNumber),
		GasLimit:      hexutil.Uint64(e.GasLimit),
		GasUsed:       hexutil.Uint64(e.GasUsed),
		Timestamp:     hexutil.Uint64(e.Timestamp),
		ExtraData:     e.ExtraData,
		BaseFeePerGas: baseFeeHex,
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
	e.ParentHash = bytesutil.PadTo(dec.ParentHash, fieldparams.RootLength)
	e.FeeRecipient = bytesutil.PadTo(dec.FeeRecipient, fieldparams.FeeRecipientLength)
	e.StateRoot = bytesutil.PadTo(dec.StateRoot, fieldparams.RootLength)
	e.ReceiptsRoot = bytesutil.PadTo(dec.ReceiptsRoot, fieldparams.RootLength)
	e.LogsBloom = bytesutil.PadTo(dec.LogsBloom, fieldparams.LogsBloomLength)
	e.PrevRandao = bytesutil.PadTo(dec.PrevRandao, fieldparams.RootLength)
	e.BlockNumber = uint64(dec.BlockNumber)
	e.GasLimit = uint64(dec.GasLimit)
	e.GasUsed = uint64(dec.GasUsed)
	e.Timestamp = uint64(dec.Timestamp)
	e.ExtraData = dec.ExtraData
	baseFee, err := hexutil.DecodeBig(dec.BaseFeePerGas)
	if err != nil {
		return err
	}
	e.BaseFeePerGas = bytesutil.PadTo(bytesutil.ReverseByteOrder(baseFee.Bytes()), fieldparams.RootLength)
	e.BlockHash = bytesutil.PadTo(dec.BlockHash, fieldparams.RootLength)
	transactions := make([][]byte, len(dec.Transactions))
	for i, tx := range dec.Transactions {
		transactions[i] = tx
	}
	e.Transactions = transactions
	return nil
}

type payloadAttributesJSON struct {
	Timestamp             hexutil.Uint64 `json:"timestamp"`
	PrevRandao            hexutil.Bytes  `json:"prevRandao"`
	SuggestedFeeRecipient hexutil.Bytes  `json:"suggestedFeeRecipient"`
}

// MarshalJSON --
func (p *PayloadAttributes) MarshalJSON() ([]byte, error) {
	return json.Marshal(payloadAttributesJSON{
		Timestamp:             hexutil.Uint64(p.Timestamp),
		PrevRandao:            p.PrevRandao,
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
	p.PrevRandao = dec.PrevRandao
	p.SuggestedFeeRecipient = dec.SuggestedFeeRecipient
	return nil
}

type payloadStatusJSON struct {
	LatestValidHash *hexutil.Bytes `json:"latestValidHash"`
	Status          string         `json:"status"`
	ValidationError *string        `json:"validationError"`
}

// MarshalJSON --
func (p *PayloadStatus) MarshalJSON() ([]byte, error) {
	hash := p.LatestValidHash
	return json.Marshal(payloadStatusJSON{
		LatestValidHash: (*hexutil.Bytes)(&hash),
		Status:          p.Status.String(),
		ValidationError: &p.ValidationError,
	})
}

// UnmarshalJSON --
func (p *PayloadStatus) UnmarshalJSON(enc []byte) error {
	dec := payloadStatusJSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}
	*p = PayloadStatus{}
	if dec.LatestValidHash != nil {
		p.LatestValidHash = *dec.LatestValidHash
	}
	p.Status = PayloadStatus_Status(PayloadStatus_Status_value[dec.Status])
	if dec.ValidationError != nil {
		p.ValidationError = *dec.ValidationError
	}
	return nil
}

type transitionConfigurationJSON struct {
	TerminalTotalDifficulty string        `json:"terminalTotalDifficulty"`
	TerminalBlockHash       hexutil.Bytes `json:"terminalBlockHash"`
	TerminalBlockNumber     string        `json:"terminalBlockNumber"`
}

// MarshalJSON --
func (t *TransitionConfiguration) MarshalJSON() ([]byte, error) {
	num := new(big.Int).SetBytes(t.TerminalBlockNumber)
	numHex := hexutil.EncodeBig(num)
	return json.Marshal(transitionConfigurationJSON{
		TerminalTotalDifficulty: t.TerminalTotalDifficulty,
		TerminalBlockHash:       t.TerminalBlockHash,
		TerminalBlockNumber:     numHex,
	})
}

// UnmarshalJSON --
func (t *TransitionConfiguration) UnmarshalJSON(enc []byte) error {
	dec := transitionConfigurationJSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}
	*t = TransitionConfiguration{}
	num, err := hexutil.DecodeBig(dec.TerminalBlockNumber)
	if err != nil {
		return err
	}
	t.TerminalTotalDifficulty = dec.TerminalTotalDifficulty
	t.TerminalBlockHash = dec.TerminalBlockHash
	t.TerminalBlockNumber = num.Bytes()
	return nil
}

type forkchoiceStateJSON struct {
	HeadBlockHash      hexutil.Bytes `json:"headBlockHash"`
	SafeBlockHash      hexutil.Bytes `json:"safeBlockHash"`
	FinalizedBlockHash hexutil.Bytes `json:"finalizedBlockHash"`
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
