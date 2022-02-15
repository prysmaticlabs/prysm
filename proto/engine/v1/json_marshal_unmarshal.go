package enginev1

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
)

// PayloadIDBytes defines a custom type for Payload IDs used by the engine API
// client with proper JSON Marshal and Unmarshal methods to hex.
type PayloadIDBytes [8]byte

// MarshalJSON --
func (b PayloadIDBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(hexutil.Bytes(b[:]))
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

type executionBlockJSON struct {
	Number           string          `json:"number"`
	Hash             hexutil.Bytes   `json:"hash"`
	ParentHash       hexutil.Bytes   `json:"parentHash"`
	Sha3Uncles       hexutil.Bytes   `json:"sha3Uncles"`
	Miner            hexutil.Bytes   `json:"miner"`
	StateRoot        hexutil.Bytes   `json:"stateRoot"`
	TransactionsRoot hexutil.Bytes   `json:"transactionsRoot"`
	ReceiptsRoot     hexutil.Bytes   `json:"receiptsRoot"`
	LogsBloom        hexutil.Bytes   `json:"logsBloom"`
	Difficulty       string          `json:"difficulty"`
	TotalDifficulty  string          `json:"totalDifficulty"`
	GasLimit         hexutil.Uint64  `json:"gasLimit"`
	GasUsed          hexutil.Uint64  `json:"gasUsed"`
	Timestamp        hexutil.Uint64  `json:"timestamp"`
	BaseFeePerGas    string          `json:"baseFeePerGas"`
	ExtraData        hexutil.Bytes   `json:"extraData"`
	MixHash          hexutil.Bytes   `json:"mixHash"`
	Nonce            hexutil.Bytes   `json:"nonce"`
	Size             string          `json:"size"`
	Transactions     []hexutil.Bytes `json:"transactions"`
	Uncles           []hexutil.Bytes `json:"uncles"`
}

// MarshalJSON defines a custom json.Marshaler interface implementation
// that uses custom json.Marshalers for the hexutil.Bytes and hexutil.Uint64 types.
func (e *ExecutionBlock) MarshalJSON() ([]byte, error) {
	transactions := make([]hexutil.Bytes, len(e.Transactions))
	for i, tx := range e.Transactions {
		transactions[i] = tx
	}
	uncles := make([]hexutil.Bytes, len(e.Uncles))
	for i, ucl := range e.Uncles {
		uncles[i] = ucl
	}
	num := new(big.Int).SetBytes(e.Number)
	numHex := hexutil.EncodeBig(num)

	diff := new(big.Int).SetBytes(e.Difficulty)
	diffHex := hexutil.EncodeBig(diff)

	totalDiff := new(big.Int).SetBytes(e.TotalDifficulty)
	totalDiffHex := hexutil.EncodeBig(totalDiff)

	size := new(big.Int).SetBytes(e.Size)
	sizeHex := hexutil.EncodeBig(size)

	baseFee := new(big.Int).SetBytes(e.BaseFeePerGas)
	baseFeeHex := hexutil.EncodeBig(baseFee)
	return json.Marshal(executionBlockJSON{
		Number:           numHex,
		Hash:             e.Hash,
		ParentHash:       e.ParentHash,
		Sha3Uncles:       e.Sha3Uncles,
		Miner:            e.Miner,
		StateRoot:        e.StateRoot,
		TransactionsRoot: e.TransactionsRoot,
		ReceiptsRoot:     e.ReceiptsRoot,
		LogsBloom:        e.LogsBloom,
		Difficulty:       diffHex,
		TotalDifficulty:  totalDiffHex,
		GasLimit:         hexutil.Uint64(e.GasLimit),
		GasUsed:          hexutil.Uint64(e.GasUsed),
		Timestamp:        hexutil.Uint64(e.Timestamp),
		ExtraData:        e.ExtraData,
		MixHash:          e.MixHash,
		Nonce:            e.Nonce,
		Size:             sizeHex,
		BaseFeePerGas:    baseFeeHex,
		Transactions:     transactions,
		Uncles:           uncles,
	})
}

// UnmarshalJSON defines a custom json.Unmarshaler interface implementation
// that uses custom json.Unmarshalers for the hexutil.Bytes and hexutil.Uint64 types.
func (e *ExecutionBlock) UnmarshalJSON(enc []byte) error {
	dec := executionBlockJSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}
	*e = ExecutionBlock{}
	num, err := hexutil.DecodeBig(dec.Number)
	if err != nil {
		return err
	}
	e.Number = num.Bytes()
	e.Hash = dec.Hash
	e.ParentHash = dec.ParentHash
	e.Sha3Uncles = dec.Sha3Uncles
	e.Miner = dec.Miner
	e.StateRoot = dec.StateRoot
	e.TransactionsRoot = dec.TransactionsRoot
	e.ReceiptsRoot = dec.ReceiptsRoot
	e.LogsBloom = dec.LogsBloom
	diff, err := hexutil.DecodeBig(dec.Difficulty)
	if err != nil {
		return err
	}
	e.Difficulty = diff.Bytes()
	totalDiff, err := hexutil.DecodeBig(dec.TotalDifficulty)
	if err != nil {
		return err
	}
	e.TotalDifficulty = totalDiff.Bytes()
	e.GasLimit = uint64(dec.GasLimit)
	e.GasUsed = uint64(dec.GasUsed)
	e.Timestamp = uint64(dec.Timestamp)
	e.ExtraData = dec.ExtraData
	e.MixHash = dec.MixHash
	e.Nonce = dec.Nonce
	size, err := hexutil.DecodeBig(dec.Size)
	if err != nil {
		return err
	}
	e.Size = size.Bytes()
	baseFee, err := hexutil.DecodeBig(dec.BaseFeePerGas)
	if err != nil {
		return err
	}
	e.BaseFeePerGas = baseFee.Bytes()
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
	ParentHash    hexutil.Bytes   `json:"parentHash"`
	FeeRecipient  hexutil.Bytes   `json:"feeRecipient"`
	StateRoot     hexutil.Bytes   `json:"stateRoot"`
	ReceiptsRoot  hexutil.Bytes   `json:"receiptsRoot"`
	LogsBloom     hexutil.Bytes   `json:"logsBloom"`
	Random        hexutil.Bytes   `json:"random"`
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
	baseFee := new(big.Int).SetBytes(e.BaseFeePerGas)
	baseFeeHex := hexutil.EncodeBig(baseFee)
	return json.Marshal(executionPayloadJSON{
		ParentHash:    e.ParentHash,
		FeeRecipient:  e.FeeRecipient,
		StateRoot:     e.StateRoot,
		ReceiptsRoot:  e.ReceiptsRoot,
		LogsBloom:     e.LogsBloom,
		Random:        e.Random,
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
	e.Random = bytesutil.PadTo(dec.Random, fieldparams.RootLength)
	e.BlockNumber = uint64(dec.BlockNumber)
	e.GasLimit = uint64(dec.GasLimit)
	e.GasUsed = uint64(dec.GasUsed)
	e.Timestamp = uint64(dec.Timestamp)
	e.ExtraData = dec.ExtraData
	baseFee, err := hexutil.DecodeBig(dec.BaseFeePerGas)
	if err != nil {
		return err
	}
	e.BaseFeePerGas = bytesutil.PadTo(baseFee.Bytes(), fieldparams.RootLength)
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
	Random                hexutil.Bytes  `json:"random"`
	SuggestedFeeRecipient hexutil.Bytes  `json:"suggestedFeeRecipient"`
}

// MarshalJSON --
func (p *PayloadAttributes) MarshalJSON() ([]byte, error) {
	return json.Marshal(payloadAttributesJSON{
		Timestamp:             hexutil.Uint64(p.Timestamp),
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
	LatestValidHash hexutil.Bytes `json:"latestValidHash"`
	Status          string        `json:"status"`
	ValidationError string        `json:"validationError"`
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

type transitionConfigurationJSON struct {
	TerminalTotalDifficulty string        `json:"terminalTotalDifficulty"`
	TerminalBlockHash       hexutil.Bytes `json:"terminalBlockHash"`
	TerminalBlockNumber     string        `json:"terminalBlockNumber"`
}

// MarshalJSON --
func (t *TransitionConfiguration) MarshalJSON() ([]byte, error) {
	num := new(big.Int).SetBytes(t.TerminalBlockNumber)
	numHex := hexutil.EncodeBig(num)
	diff, ok := new(big.Int).SetString(params.BeaconConfig().TerminalTotalDifficulty, 10)
	if !ok {
		return nil, nil
	}
	diffItem, overflows := uint256.FromBig(diff)
	if overflows {
		return nil, errors.New("terminal total difficulty should not overflow")
	}
	fmt.Printf("Got %s", diffItem.String())
	return json.Marshal(transitionConfigurationJSON{
		TerminalTotalDifficulty: diffItem.String(),
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
	fmt.Printf("Got %s\n", dec.TerminalTotalDifficulty)
	num, err := hexutil.DecodeBig(dec.TerminalBlockNumber)
	if err != nil {
		return err
	}
	diff, err := hexutil.DecodeBig(dec.TerminalTotalDifficulty)
	if err != nil {
		return err
	}
	t.TerminalTotalDifficulty = diff.Bytes()
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
