package enginev1

import (
	"encoding/json"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
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

func (e *ExecutionBlock) MarshalJSON() ([]byte, error) {
	decoded := make(map[string]interface{})
	encodedHeader, err := e.Header.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(encodedHeader, &decoded); err != nil {
		return nil, err
	}
	decoded["hash"] = e.Hash.String()
	decoded["transactions"] = e.Transactions
	decoded["totalDifficulty"] = e.TotalDifficulty
	return json.Marshal(decoded)
}

func (e *ExecutionBlock) UnmarshalJSON(enc []byte) error {
	type transactionJson struct {
		Transactions []*gethtypes.Transaction `json:"transactions"`
	}
	if err := e.Header.UnmarshalJSON(enc); err != nil {
		return err
	}
	decoded := make(map[string]interface{})
	if err := json.Unmarshal(enc, &decoded); err != nil {
		return err
	}
	blockHashStr, ok := decoded["hash"].(string)
	if !ok {
		return errors.New("expected `hash` field in JSON response")
	}
	decodedHash, err := hexutil.Decode(blockHashStr)
	if err != nil {
		return err
	}
	e.Hash = common.BytesToHash(decodedHash)
	e.TotalDifficulty, ok = decoded["totalDifficulty"].(string)
	if !ok {
		return errors.New("expected `totalDifficulty` field in JSON response")
	}
	rawTxList, ok := decoded["transactions"]
	if !ok || rawTxList == nil {
		// Exit early if there are no transactions stored in the json payload.
		return nil
	}
	txsList, ok := rawTxList.([]interface{})
	if !ok {
		return errors.Errorf("expected transaction list to be of a slice interface type.")
	}

	//
	for _, tx := range txsList {
		// If the transaction is just a hex string, do not attempt to
		// unmarshal into a full transaction object.
		if txItem, ok := tx.(string); ok && strings.HasPrefix(txItem, "0x") {
			return nil
		}
	}
	// If the block contains a list of transactions, we JSON unmarshal
	// them into a list of geth transaction objects.
	txJson := &transactionJson{}
	if err := json.Unmarshal(enc, txJson); err != nil {
		return err
	}
	e.Transactions = txJson.Transactions
	return nil
}

// UnmarshalJSON --
func (b *PayloadIDBytes) UnmarshalJSON(enc []byte) error {
	res := [8]byte{}
	if err := hexutil.UnmarshalFixedJSON(reflect.TypeOf(b), enc, res[:]); err != nil {
		return err
	}
	*b = res
	return nil
}

type ExecutionPayloadJSON struct {
	ParentHash    *common.Hash    `json:"parentHash"`
	FeeRecipient  *common.Address `json:"feeRecipient"`
	StateRoot     *common.Hash    `json:"stateRoot"`
	ReceiptsRoot  *common.Hash    `json:"receiptsRoot"`
	LogsBloom     *hexutil.Bytes  `json:"logsBloom"`
	PrevRandao    *common.Hash    `json:"prevRandao"`
	BlockNumber   *hexutil.Uint64 `json:"blockNumber"`
	GasLimit      *hexutil.Uint64 `json:"gasLimit"`
	GasUsed       *hexutil.Uint64 `json:"gasUsed"`
	Timestamp     *hexutil.Uint64 `json:"timestamp"`
	ExtraData     hexutil.Bytes   `json:"extraData"`
	BaseFeePerGas string          `json:"baseFeePerGas"`
	BlockHash     *common.Hash    `json:"blockHash"`
	Transactions  []hexutil.Bytes `json:"transactions"`
	ExcessDataGas *hexutil.Uint64 `json:"excessDataGas"`
}

func (e *ExecutionPayloadJSON) Pre4844() (*ExecutionPayload, error) {
	if e.ParentHash == nil {
		return nil, errors.New("missing required field 'parentHash' for ExecutionPayload")
	}
	if e.FeeRecipient == nil {
		return nil, errors.New("missing required field 'feeRecipient' for ExecutionPayload")
	}
	if e.StateRoot == nil {
		return nil, errors.New("missing required field 'stateRoot' for ExecutionPayload")
	}
	if e.ReceiptsRoot == nil {
		return nil, errors.New("missing required field 'receiptsRoot' for ExecutableDataV1")
	}

	if e.LogsBloom == nil {
		return nil, errors.New("missing required field 'logsBloom' for ExecutionPayload")
	}
	if e.PrevRandao == nil {
		return nil, errors.New("missing required field 'prevRandao' for ExecutionPayload")
	}
	if e.ExtraData == nil {
		return nil, errors.New("missing required field 'extraData' for ExecutionPayload")
	}
	if e.BlockHash == nil {
		return nil, errors.New("missing required field 'blockHash' for ExecutionPayload")
	}
	if e.Transactions == nil {
		return nil, errors.New("missing required field 'transactions' for ExecutionPayload")
	}
	if e.BlockNumber == nil {
		return nil, errors.New("missing required field 'blockNumber' for ExecutionPayload")
	}
	if e.Timestamp == nil {
		return nil, errors.New("missing required field 'timestamp' for ExecutionPayload")
	}
	if e.GasUsed == nil {
		return nil, errors.New("missing required field 'gasUsed' for ExecutionPayload")
	}
	if e.GasLimit == nil {
		return nil, errors.New("missing required field 'gasLimit' for ExecutionPayload")
	}
	payload := ExecutionPayload{}
	payload.ParentHash = e.ParentHash.Bytes()
	payload.FeeRecipient = e.FeeRecipient.Bytes()
	payload.StateRoot = e.StateRoot.Bytes()
	payload.ReceiptsRoot = e.ReceiptsRoot.Bytes()
	payload.LogsBloom = *e.LogsBloom
	payload.PrevRandao = e.PrevRandao.Bytes()
	payload.BlockNumber = uint64(*e.BlockNumber)
	payload.GasLimit = uint64(*e.GasLimit)
	payload.GasUsed = uint64(*e.GasUsed)
	payload.Timestamp = uint64(*e.Timestamp)
	payload.ExtraData = e.ExtraData
	baseFee, err := hexutil.DecodeBig(e.BaseFeePerGas)
	if err != nil {
		return nil, err
	}
	payload.BaseFeePerGas = bytesutil.PadTo(bytesutil.ReverseByteOrder(baseFee.Bytes()), fieldparams.RootLength)
	payload.BlockHash = e.BlockHash.Bytes()
	transactions := make([][]byte, len(e.Transactions))
	for i, tx := range e.Transactions {
		transactions[i] = tx
	}
	payload.Transactions = transactions
	return &payload, nil
}

func (e *ExecutionPayloadJSON) Post4844() (*ExecutionPayload4844, error) {
	if e.ParentHash == nil {
		return nil, errors.New("missing required field 'parentHash' for ExecutionPayload")
	}
	if e.FeeRecipient == nil {
		return nil, errors.New("missing required field 'feeRecipient' for ExecutionPayload")
	}
	if e.StateRoot == nil {
		return nil, errors.New("missing required field 'stateRoot' for ExecutionPayload")
	}
	if e.ReceiptsRoot == nil {
		return nil, errors.New("missing required field 'receiptsRoot' for ExecutableDataV1")
	}

	if e.LogsBloom == nil {
		return nil, errors.New("missing required field 'logsBloom' for ExecutionPayload")
	}
	if e.PrevRandao == nil {
		return nil, errors.New("missing required field 'prevRandao' for ExecutionPayload")
	}
	if e.ExtraData == nil {
		return nil, errors.New("missing required field 'extraData' for ExecutionPayload")
	}
	if e.BlockHash == nil {
		return nil, errors.New("missing required field 'blockHash' for ExecutionPayload")
	}
	if e.Transactions == nil {
		return nil, errors.New("missing required field 'transactions' for ExecutionPayload")
	}
	if e.BlockNumber == nil {
		return nil, errors.New("missing required field 'blockNumber' for ExecutionPayload")
	}
	if e.Timestamp == nil {
		return nil, errors.New("missing required field 'timestamp' for ExecutionPayload")
	}
	if e.GasUsed == nil {
		return nil, errors.New("missing required field 'gasUsed' for ExecutionPayload")
	}
	if e.GasLimit == nil {
		return nil, errors.New("missing required field 'gasLimit' for ExecutionPayload")
	}
	if e.ExcessDataGas == nil {
		return nil, errors.New("missing required field 'excessDataGas' for ExecutionPayload")
	}
	payload := ExecutionPayload4844{}
	payload.ParentHash = e.ParentHash.Bytes()
	payload.FeeRecipient = e.FeeRecipient.Bytes()
	payload.StateRoot = e.StateRoot.Bytes()
	payload.ReceiptsRoot = e.ReceiptsRoot.Bytes()
	payload.LogsBloom = *e.LogsBloom
	payload.PrevRandao = e.PrevRandao.Bytes()
	payload.BlockNumber = uint64(*e.BlockNumber)
	payload.GasLimit = uint64(*e.GasLimit)
	payload.GasUsed = uint64(*e.GasUsed)
	payload.Timestamp = uint64(*e.Timestamp)
	payload.ExtraData = e.ExtraData
	baseFee, err := hexutil.DecodeBig(e.BaseFeePerGas)
	if err != nil {
		return nil, err
	}
	payload.BaseFeePerGas = bytesutil.PadTo(bytesutil.ReverseByteOrder(baseFee.Bytes()), fieldparams.RootLength)
	payload.BlockHash = e.BlockHash.Bytes()
	transactions := make([][]byte, len(e.Transactions))
	payload.ExcessDataGas = uint64(*e.ExcessDataGas)
	for i, tx := range e.Transactions {
		transactions[i] = tx
	}
	payload.Transactions = transactions
	return &payload, nil
}

// MarshalJSON --
func (e *ExecutionPayload) MarshalJSON() ([]byte, error) {
	transactions := make([]hexutil.Bytes, len(e.Transactions))
	for i, tx := range e.Transactions {
		transactions[i] = tx
	}
	baseFee := new(big.Int).SetBytes(bytesutil.ReverseByteOrder(e.BaseFeePerGas))
	baseFeeHex := hexutil.EncodeBig(baseFee)
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
	logsBloom := hexutil.Bytes(e.LogsBloom)
	return json.Marshal(ExecutionPayloadJSON{
		ParentHash:    &pHash,
		FeeRecipient:  &recipient,
		StateRoot:     &sRoot,
		ReceiptsRoot:  &recRoot,
		LogsBloom:     &logsBloom,
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

func (e *ExecutionPayload4844) MarshalJSON() ([]byte, error) {
	transactions := make([]hexutil.Bytes, len(e.Transactions))
	for i, tx := range e.Transactions {
		transactions[i] = tx
	}
	baseFee := new(big.Int).SetBytes(bytesutil.ReverseByteOrder(e.BaseFeePerGas))
	baseFeeHex := hexutil.EncodeBig(baseFee)
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
	logsBloom := hexutil.Bytes(e.LogsBloom)
	excessDataGas := hexutil.Uint64(e.ExcessDataGas)
	return json.Marshal(ExecutionPayloadJSON{
		ParentHash:    &pHash,
		FeeRecipient:  &recipient,
		StateRoot:     &sRoot,
		ReceiptsRoot:  &recRoot,
		LogsBloom:     &logsBloom,
		PrevRandao:    &prevRan,
		BlockNumber:   &blockNum,
		GasLimit:      &gasLimit,
		GasUsed:       &gasUsed,
		Timestamp:     &timeStamp,
		ExtraData:     e.ExtraData,
		BaseFeePerGas: baseFeeHex,
		BlockHash:     &bHash,
		Transactions:  transactions,
		ExcessDataGas: &excessDataGas,
	})
}

// UnmarshalJSON --
func (e *ExecutionPayload) UnmarshalJSON(enc []byte) error {
	dec := ExecutionPayloadJSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}
	payload, err := dec.Pre4844()
	if err != nil {
		return err
	}
	*e = ExecutionPayload{
		ParentHash:    bytesutil.SafeCopyBytes(payload.ParentHash),
		FeeRecipient:  bytesutil.SafeCopyBytes(payload.FeeRecipient),
		StateRoot:     bytesutil.SafeCopyBytes(payload.StateRoot),
		ReceiptsRoot:  bytesutil.SafeCopyBytes(payload.ReceiptsRoot),
		LogsBloom:     bytesutil.SafeCopyBytes(payload.LogsBloom),
		PrevRandao:    bytesutil.SafeCopyBytes(payload.PrevRandao),
		BlockNumber:   payload.BlockNumber,
		GasLimit:      payload.GasLimit,
		GasUsed:       payload.GasUsed,
		Timestamp:     payload.Timestamp,
		ExtraData:     bytesutil.SafeCopyBytes(payload.ExtraData),
		BaseFeePerGas: bytesutil.SafeCopyBytes(payload.BaseFeePerGas),
		BlockHash:     bytesutil.SafeCopyBytes(payload.BlockHash),
		Transactions:  bytesutil.SafeCopy2dBytes(payload.Transactions),
	}
	return nil
}

// TODO(EIP-4844): Refactor this to not duplicate the entirety of ExecutionPayload's UnmarshalJSON
func (e *ExecutionPayload4844) UnmarshalJSON(enc []byte) error {
	dec := ExecutionPayloadJSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}

	payload, err := dec.Post4844()
	if err != nil {
		return err
	}
	*e = ExecutionPayload4844{
		ParentHash:    bytesutil.SafeCopyBytes(payload.ParentHash),
		FeeRecipient:  bytesutil.SafeCopyBytes(payload.FeeRecipient),
		StateRoot:     bytesutil.SafeCopyBytes(payload.StateRoot),
		ReceiptsRoot:  bytesutil.SafeCopyBytes(payload.ReceiptsRoot),
		LogsBloom:     bytesutil.SafeCopyBytes(payload.LogsBloom),
		PrevRandao:    bytesutil.SafeCopyBytes(payload.PrevRandao),
		BlockNumber:   payload.BlockNumber,
		GasLimit:      payload.GasLimit,
		GasUsed:       payload.GasUsed,
		Timestamp:     payload.Timestamp,
		ExtraData:     bytesutil.SafeCopyBytes(payload.ExtraData),
		BaseFeePerGas: bytesutil.SafeCopyBytes(payload.BaseFeePerGas),
		BlockHash:     bytesutil.SafeCopyBytes(payload.BlockHash),
		Transactions:  bytesutil.SafeCopy2dBytes(payload.Transactions),
		ExcessDataGas: payload.ExcessDataGas,
	}
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
		latestHash = &hash
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
	TerminalTotalDifficulty *hexutil.Big   `json:"terminalTotalDifficulty"`
	TerminalBlockHash       common.Hash    `json:"terminalBlockHash"`
	TerminalBlockNumber     hexutil.Uint64 `json:"terminalBlockNumber"`
}

// MarshalJSON --
func (t *TransitionConfiguration) MarshalJSON() ([]byte, error) {
	num := new(big.Int).SetBytes(t.TerminalBlockNumber)
	var hexNum *hexutil.Big
	if t.TerminalTotalDifficulty != "" {
		ttdNum, err := hexutil.DecodeBig(t.TerminalTotalDifficulty)
		if err != nil {
			return nil, err
		}
		bHex := hexutil.Big(*ttdNum)
		hexNum = &bHex
	}
	if len(t.TerminalBlockHash) != fieldparams.RootLength {
		return nil, errors.Errorf("terminal block hash is of the wrong length: %d", len(t.TerminalBlockHash))
	}
	return json.Marshal(transitionConfigurationJSON{
		TerminalTotalDifficulty: hexNum,
		TerminalBlockHash:       *(*[32]byte)(t.TerminalBlockHash),
		TerminalBlockNumber:     hexutil.Uint64(num.Uint64()),
	})
}

// UnmarshalJSON --
func (t *TransitionConfiguration) UnmarshalJSON(enc []byte) error {
	dec := transitionConfigurationJSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}
	*t = TransitionConfiguration{}
	num := big.NewInt(int64(dec.TerminalBlockNumber))
	if dec.TerminalTotalDifficulty != nil {
		t.TerminalTotalDifficulty = dec.TerminalTotalDifficulty.String()
	}
	t.TerminalBlockHash = dec.TerminalBlockHash[:]
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

type blobBundleJSON struct {
	BlockHash       common.Hash           `json:"blockHash"`
	Kzgs            []types.KZGCommitment `json:"kzgs"`
	Blobs           []types.Blob          `json:"blobs"`
	AggregatedProof types.KZGProof        `json:"aggregatedProof"`
}

// MarshalJSON --
func (b *BlobsBundle) MarshalJSON() ([]byte, error) {
	kzgs := make([]types.KZGCommitment, len(b.Kzgs))
	for i, kzg := range b.Kzgs {
		kzgs[i] = bytesutil.ToBytes48(kzg)
	}
	blobs := make([]types.Blob, len(b.Blobs))
	for i, b1 := range b.Blobs {
		var blob [params.FieldElementsPerBlob]types.BLSFieldElement
		for j, b2 := range b1.Blob {
			blob[j] = bytesutil.ToBytes32(b2)
		}
		blobs[i] = blob
	}

	return json.Marshal(blobBundleJSON{
		BlockHash:       bytesutil.ToBytes32(b.BlockHash),
		Kzgs:            kzgs,
		Blobs:           blobs,
		AggregatedProof: bytesutil.ToBytes48(b.AggregatedProof),
	})
}

// UnmarshalJSON --
func (e *BlobsBundle) UnmarshalJSON(enc []byte) error {
	dec := blobBundleJSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}
	*e = BlobsBundle{}
	e.BlockHash = bytesutil.PadTo(dec.BlockHash.Bytes(), fieldparams.RootLength)
	kzgs := make([][]byte, len(dec.Kzgs))
	for i, kzg := range dec.Kzgs {
		kzgs[i] = bytesutil.PadTo(kzg[:], fieldparams.BLSPubkeyLength)
	}
	e.Kzgs = kzgs
	blobs := make([]*Blob, len(dec.Blobs))
	for i, blob := range dec.Blobs {
		blobs[i] = &Blob{
			Blob: make([][]byte, len(blob)),
		}
		for j, b := range blob {
			blobs[i].Blob[j] = bytesutil.SafeCopyBytes(b[:])
		}
	}
	e.Blobs = blobs
	e.AggregatedProof = bytesutil.PadTo(dec.AggregatedProof[:], fieldparams.BLSPubkeyLength)
	return nil
}
