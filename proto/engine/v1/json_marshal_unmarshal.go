package enginev1

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
)

// PayloadIDBytes defines a custom type for Payload IDs used by the engine API
// client with proper JSON Marshal and Unmarshal methods to hex.
type PayloadIDBytes [8]byte

// MarshalJSON --

func (b PayloadIDBytes) MarshalText() ([]byte, error) {
	return hexutil.Bytes(b[:]).MarshalText()
}

func (b *PayloadIDBytes) UnmarshalText(input []byte) error {
	err := hexutil.UnmarshalFixedText("PayloadID", input, b[:])
	if err != nil {
		return fmt.Errorf("invalid payload id %q: %w", input, err)
	}
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
		TotalDifficulty:  e.TotalDifficulty,
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
	e.TotalDifficulty = dec.TotalDifficulty
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
	ParentHash    *common.Hash    `json:"parentHash"`
	FeeRecipient  *common.Address `json:"feeRecipient"`
	StateRoot     *common.Hash    `json:"stateRoot"`
	ReceiptsRoot  *common.Hash    `json:"receiptsRoot"`
	LogsBloom     hexutil.Bytes   `json:"logsBloom"`
	PrevRandao    *common.Hash    `json:"prevRandao"`
	BlockNumber   *hexutil.Uint64 `json:"blockNumber"`
	GasLimit      *hexutil.Uint64 `json:"gasLimit"`
	GasUsed       *hexutil.Uint64 `json:"gasUsed"`
	Timestamp     *hexutil.Uint64 `json:"timestamp"`
	ExtraData     hexutil.Bytes   `json:"extraData"`
	BaseFeePerGas string          `json:"baseFeePerGas"`
	BlockHash     *common.Hash    `json:"blockHash"`
	Transactions  []hexutil.Bytes `json:"transactions"`
}

// MarshalJSON --
func (e *ExecutionPayload) MarshalJSON() ([]byte, error) {
	transactions := make([]hexutil.Bytes, len(e.Transactions))
	for i, tx := range e.Transactions {
		transactions[i] = tx
	}
	baseFeeHex := hexutil.Encode(e.BaseFeePerGas)
	pHash := common.BytesToHash(e.ParentHash)
	sRoot := common.BytesToHash(e.StateRoot)
	recRoot := common.BytesToHash(e.ReceiptsRoot)
	prevRan := common.BytesToHash(e.PrevRandao)
	bHash := common.BytesToHash(e.BlockHash)
	blockNum := hexutil.Uint64(e.BlockNumber)
	gasLimit := hexutil.Uint64(e.GasLimit)
	gasUsed := hexutil.Uint64(e.GasUsed)
	timeStamp := hexutil.Uint64(e.Timestamp)
	recipient := common.BytesToAddress(e.FeeRecipient)
	return json.Marshal(executionPayloadJSON{
		ParentHash:    &pHash,
		FeeRecipient:  &recipient,
		StateRoot:     &sRoot,
		ReceiptsRoot:  &recRoot,
		LogsBloom:     e.LogsBloom,
		PrevRandao:    &prevRan,
		BlockNumber:   &blockNum,
		GasLimit:      &gasLimit,
		GasUsed:       &gasUsed,
		Timestamp:     &timeStamp,
		ExtraData:     e.ExtraData,
		BaseFeePerGas: baseFeeHex,
		BlockHash:     &bHash,
		Transactions:  transactions,
	})
}

// UnmarshalJSON --
func (e *ExecutionPayload) UnmarshalJSON(enc []byte) error {
	dec := executionPayloadJSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}

	if dec.ParentHash == nil {
		return errors.New("missing required field 'parentHash' for ExecutionPayload")
	}
	if dec.FeeRecipient == nil {
		return errors.New("missing required field 'feeRecipient' for ExecutionPayload")
	}
	if dec.StateRoot == nil {
		return errors.New("missing required field 'stateRoot' for ExecutionPayload")
	}
	if dec.ReceiptsRoot == nil {
		return errors.New("missing required field 'receiptsRoot' for ExecutableDataV1")
	}

	if dec.LogsBloom == nil {
		return errors.New("missing required field 'logsBloom' for ExecutionPayload")
	}
	if dec.PrevRandao == nil {
		return errors.New("missing required field 'prevRandao' for ExecutionPayload")
	}
	if dec.ExtraData == nil {
		return errors.New("missing required field 'extraData' for ExecutionPayload")
	}
	if dec.BlockHash == nil {
		return errors.New("missing required field 'blockHash' for ExecutionPayload")
	}
	if dec.Transactions == nil {
		return errors.New("missing required field 'transactions' for ExecutionPayload")
	}
	if dec.BlockNumber == nil {
		return errors.New("missing required field 'blockNumber' for ExecutionPayload")
	}
	if dec.Timestamp == nil {
		return errors.New("missing required field 'timestamp' for ExecutionPayload")
	}
	if dec.GasUsed == nil {
		return errors.New("missing required field 'gasUsed' for ExecutionPayload")
	}
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for ExecutionPayload")
	}
	*e = ExecutionPayload{}
	e.ParentHash = dec.ParentHash.Bytes()
	e.FeeRecipient = dec.FeeRecipient.Bytes()
	e.StateRoot = dec.StateRoot.Bytes()
	e.ReceiptsRoot = dec.ReceiptsRoot.Bytes()
	e.LogsBloom = bytesutil.PadTo(dec.LogsBloom, fieldparams.LogsBloomLength)
	e.PrevRandao = dec.PrevRandao.Bytes()
	e.BlockNumber = uint64(*dec.BlockNumber)
	e.GasLimit = uint64(*dec.GasLimit)
	e.GasUsed = uint64(*dec.GasUsed)
	e.Timestamp = uint64(*dec.Timestamp)
	e.ExtraData = dec.ExtraData
	val, err := hexutil.Decode(dec.BaseFeePerGas)
	if err != nil {
		return err
	}
	e.BaseFeePerGas = val
	e.BlockHash = dec.BlockHash.Bytes()
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
	LatestValidHash *common.Hash `json:"latestValidHash"`
	Status          string       `json:"status"`
	ValidationError *string      `json:"validationError"`
}

// MarshalJSON --
func (p *PayloadStatus) MarshalJSON() ([]byte, error) {
	var latestHash *common.Hash
	if p.LatestValidHash != nil {
		hash := common.Hash(bytesutil.ToBytes32(p.LatestValidHash))
		latestHash = (*common.Hash)(&hash)
	}
	return json.Marshal(payloadStatusJSON{
		LatestValidHash: latestHash,
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
		p.LatestValidHash = dec.LatestValidHash[:]
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
