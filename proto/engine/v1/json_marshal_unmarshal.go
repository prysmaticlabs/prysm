package enginev1

import (
	"encoding/json"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// PayloadIDBytes defines a custom type for Payload IDs used by the engine API
// client with proper JSON Marshal and Unmarshal methods to hex.
type PayloadIDBytes [8]byte

// MarshalJSON --
func (b PayloadIDBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(hexutil.Bytes(b[:]))
}

type ExecutionBlock interface {
	Version() int
	GetHeader() gethtypes.Header
	GetHash() common.Hash
	GetTransactions() []*gethtypes.Transaction
	GetTotalDifficulty() string
	GetWithdrawals() ([]*Withdrawal, error)
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(enc []byte) error
}

// ExecutionBlockBellatrix is the response kind received by the eth_getBlockByHash and
// eth_getBlockByNumber endpoints via JSON-RPC.
type ExecutionBlockBellatrix struct {
	gethtypes.Header
	Hash            common.Hash              `json:"hash"`
	Transactions    []*gethtypes.Transaction `json:"transactions"`
	TotalDifficulty string                   `json:"totalDifficulty"`
}

// ExecutionBlockCapella is the response kind received by the eth_getBlockByHash and
// eth_getBlockByNumber endpoints via JSON-RPC.
type ExecutionBlockCapella struct {
	gethtypes.Header
	Hash            common.Hash              `json:"hash"`
	Transactions    []*gethtypes.Transaction `json:"transactions"`
	TotalDifficulty string                   `json:"totalDifficulty"`
	Withdrawals     []*Withdrawal            `json:"withdrawals"`
}

func (e *ExecutionBlockBellatrix) Version() int {
	return version.Bellatrix
}

func (e *ExecutionBlockBellatrix) GetHeader() gethtypes.Header {
	return e.Header
}

func (e *ExecutionBlockBellatrix) GetHash() common.Hash {
	return e.Hash
}

func (e *ExecutionBlockBellatrix) GetTransactions() []*gethtypes.Transaction {
	return e.Transactions
}

func (e *ExecutionBlockBellatrix) GetTotalDifficulty() string {
	return e.TotalDifficulty
}

func (e *ExecutionBlockBellatrix) GetWithdrawals() ([]*Withdrawal, error) {
	return nil, errors.New("unsupported getter")
}

func (e *ExecutionBlockCapella) Version() int {
	return version.Capella
}

func (e *ExecutionBlockCapella) GetHeader() gethtypes.Header {
	return e.Header
}

func (e *ExecutionBlockCapella) GetHash() common.Hash {
	return e.Hash
}

func (e *ExecutionBlockCapella) GetTransactions() []*gethtypes.Transaction {
	return e.Transactions
}

func (e *ExecutionBlockCapella) GetTotalDifficulty() string {
	return e.TotalDifficulty
}

func (e *ExecutionBlockCapella) GetWithdrawals() ([]*Withdrawal, error) {
	return e.Withdrawals, nil
}

func (e *ExecutionBlockBellatrix) MarshalJSON() ([]byte, error) {
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

func (e *ExecutionBlockCapella) MarshalJSON() ([]byte, error) {
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
	ws := make([]*withdrawalJSON, len(e.Withdrawals))
	for i, w := range e.Withdrawals {
		ws[i], err = w.toWithdrawalJSON()
		if err != nil {
			return nil, err
		}
	}
	decoded["withdrawals"] = ws
	return json.Marshal(decoded)
}

func (e *ExecutionBlockBellatrix) UnmarshalJSON(enc []byte) error {
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

func (e *ExecutionBlockCapella) UnmarshalJSON(enc []byte) error {
	type transactionsJson struct {
		Transactions []*gethtypes.Transaction `json:"transactions"`
	}
	type withdrawalsJson struct {
		Withdrawals []*withdrawalJSON `json:"withdrawals"`
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

	rawWithdrawals, ok := decoded["withdrawals"]
	if !ok || rawWithdrawals == nil {
		e.Withdrawals = []*Withdrawal{}
	} else {
		j := &withdrawalsJson{}
		if err := json.Unmarshal(enc, j); err != nil {
			return err
		}
		ws := make([]*Withdrawal, len(j.Withdrawals))
		for i, wj := range j.Withdrawals {
			ws[i], err = wj.ToWithdrawal()
			if err != nil {
				return err
			}
		}
		e.Withdrawals = ws
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
	for _, tx := range txsList {
		// If the transaction is just a hex string, do not attempt to
		// unmarshal into a full transaction object.
		if txItem, ok := tx.(string); ok && strings.HasPrefix(txItem, "0x") {
			return nil
		}
	}
	// If the block contains a list of transactions, we JSON unmarshal
	// them into a list of geth transaction objects.
	txJson := &transactionsJson{}
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

type withdrawalJSON struct {
	Index     *hexutil.Uint64 `json:"index"`
	Validator *hexutil.Uint64 `json:"validatorIndex"`
	Address   *common.Address `json:"address"`
	Amount    string          `json:"amount"`
}

func (j *withdrawalJSON) ToWithdrawal() (*Withdrawal, error) {
	w := &Withdrawal{}
	b, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	if err := w.UnmarshalJSON(b); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Withdrawal) toWithdrawalJSON() (*withdrawalJSON, error) {
	b, err := w.MarshalJSON()
	if err != nil {
		return nil, err
	}
	j := &withdrawalJSON{}
	if err = json.Unmarshal(b, j); err != nil {
		return nil, err
	}
	return j, nil
}

func (w *Withdrawal) MarshalJSON() ([]byte, error) {
	index := hexutil.Uint64(w.Index)
	validatorIndex := hexutil.Uint64(w.ValidatorIndex)
	address := common.BytesToAddress(w.Address)
	wei := new(big.Int).SetUint64(1000000000)
	amountWei := new(big.Int).Mul(new(big.Int).SetUint64(w.Amount), wei)
	return json.Marshal(withdrawalJSON{
		Index:     &index,
		Validator: &validatorIndex,
		Address:   &address,
		Amount:    hexutil.EncodeBig(amountWei),
	})
}

func (w *Withdrawal) UnmarshalJSON(enc []byte) error {
	dec := withdrawalJSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}
	if dec.Index == nil {
		return errors.New("missing withdrawal index")
	}
	if dec.Validator == nil {
		return errors.New("missing validator index")
	}
	if dec.Address == nil {
		return errors.New("missing execution address")
	}
	*w = Withdrawal{}
	w.Index = uint64(*dec.Index)
	w.ValidatorIndex = types.ValidatorIndex(*dec.Validator)
	w.Address = dec.Address.Bytes()
	wei := new(big.Int).SetUint64(1000000000)
	amountWei, err := hexutil.DecodeBig(dec.Amount)
	if err != nil {
		return err
	}
	amount := new(big.Int).Div(amountWei, wei)
	if !amount.IsUint64() {
		return errors.New("withdrawal amount overflow")
	}
	w.Amount = amount.Uint64()
	return nil
}

type executionPayloadJSON struct {
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
}

type executionPayloadCapellaJSON struct {
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
	Withdrawals   []*Withdrawal   `json:"withdrawals"`
}

type executionPayload4844JSON struct {
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
	ExcessDataGas string          `json:"excessDataGas"`
	BlockHash     *common.Hash    `json:"blockHash"`
	Transactions  []hexutil.Bytes `json:"transactions"`
	Withdrawals   []*Withdrawal   `json:"withdrawals"`
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
	return json.Marshal(executionPayloadJSON{
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

// MarshalJSON --
func (e *ExecutionPayloadCapella) MarshalJSON() ([]byte, error) {
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
	if e.Withdrawals == nil {
		e.Withdrawals = make([]*Withdrawal, 0)
	}
	return json.Marshal(executionPayloadCapellaJSON{
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
		Withdrawals:   e.Withdrawals,
	})
}

// MarshalJSON --
func (e *ExecutionPayload4844) MarshalJSON() ([]byte, error) {
	transactions := make([]hexutil.Bytes, len(e.Transactions))
	for i, tx := range e.Transactions {
		transactions[i] = tx
	}
	baseFee := new(big.Int).SetBytes(bytesutil.ReverseByteOrder(e.BaseFeePerGas))
	baseFeeHex := hexutil.EncodeBig(baseFee)
	dataGas := new(big.Int).SetBytes(bytesutil.ReverseByteOrder(e.ExcessDataGas))
	dataGasHex := hexutil.EncodeBig(dataGas)
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
	if e.Withdrawals == nil {
		e.Withdrawals = make([]*Withdrawal, 0)
	}

	return json.Marshal(executionPayload4844JSON{
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
		ExcessDataGas: dataGasHex,
		BlockHash:     &bHash,
		Transactions:  transactions,
		Withdrawals:   e.Withdrawals,
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
	e.LogsBloom = *dec.LogsBloom
	e.PrevRandao = dec.PrevRandao.Bytes()
	e.BlockNumber = uint64(*dec.BlockNumber)
	e.GasLimit = uint64(*dec.GasLimit)
	e.GasUsed = uint64(*dec.GasUsed)
	e.Timestamp = uint64(*dec.Timestamp)
	e.ExtraData = dec.ExtraData
	baseFee, err := hexutil.DecodeBig(dec.BaseFeePerGas)
	if err != nil {
		return err
	}
	e.BaseFeePerGas = bytesutil.PadTo(bytesutil.ReverseByteOrder(baseFee.Bytes()), fieldparams.RootLength)
	e.BlockHash = dec.BlockHash.Bytes()
	transactions := make([][]byte, len(dec.Transactions))
	for i, tx := range dec.Transactions {
		transactions[i] = tx
	}
	e.Transactions = transactions
	return nil
}

// UnmarshalJSON --
func (e *ExecutionPayloadCapella) UnmarshalJSON(enc []byte) error {
	dec := executionPayloadCapellaJSON{}
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
	*e = ExecutionPayloadCapella{}
	e.ParentHash = dec.ParentHash.Bytes()
	e.FeeRecipient = dec.FeeRecipient.Bytes()
	e.StateRoot = dec.StateRoot.Bytes()
	e.ReceiptsRoot = dec.ReceiptsRoot.Bytes()
	e.LogsBloom = *dec.LogsBloom
	e.PrevRandao = dec.PrevRandao.Bytes()
	e.BlockNumber = uint64(*dec.BlockNumber)
	e.GasLimit = uint64(*dec.GasLimit)
	e.GasUsed = uint64(*dec.GasUsed)
	e.Timestamp = uint64(*dec.Timestamp)
	e.ExtraData = dec.ExtraData

	baseFee, err := hexutil.DecodeBig(dec.BaseFeePerGas)
	if err != nil {
		return err
	}
	e.BaseFeePerGas = bytesutil.PadTo(bytesutil.ReverseByteOrder(baseFee.Bytes()), fieldparams.RootLength)

	e.BlockHash = dec.BlockHash.Bytes()
	transactions := make([][]byte, len(dec.Transactions))
	for i, tx := range dec.Transactions {
		transactions[i] = tx
	}
	e.Transactions = transactions
	if dec.Withdrawals == nil {
		dec.Withdrawals = make([]*Withdrawal, 0)
	}
	e.Withdrawals = dec.Withdrawals
	return nil
}

// UnmarshalJSON --
func (e *ExecutionPayload4844) UnmarshalJSON(enc []byte) error {
	dec := executionPayload4844JSON{}
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
	*e = ExecutionPayload4844{}
	e.ParentHash = dec.ParentHash.Bytes()
	e.FeeRecipient = dec.FeeRecipient.Bytes()
	e.StateRoot = dec.StateRoot.Bytes()
	e.ReceiptsRoot = dec.ReceiptsRoot.Bytes()
	e.LogsBloom = *dec.LogsBloom
	e.PrevRandao = dec.PrevRandao.Bytes()
	e.BlockNumber = uint64(*dec.BlockNumber)
	e.GasLimit = uint64(*dec.GasLimit)
	e.GasUsed = uint64(*dec.GasUsed)
	e.Timestamp = uint64(*dec.Timestamp)
	e.ExtraData = dec.ExtraData

	baseFee, err := hexutil.DecodeBig(dec.BaseFeePerGas)
	if err != nil {
		return err
	}
	e.BaseFeePerGas = bytesutil.PadTo(bytesutil.ReverseByteOrder(baseFee.Bytes()), fieldparams.RootLength)

	dataGas, err := hexutil.DecodeBig(dec.ExcessDataGas)
	if err != nil {
		return err
	}
	e.ExcessDataGas = bytesutil.PadTo(bytesutil.ReverseByteOrder(dataGas.Bytes()), fieldparams.RootLength)

	e.BlockHash = dec.BlockHash.Bytes()
	transactions := make([][]byte, len(dec.Transactions))
	for i, tx := range dec.Transactions {
		transactions[i] = tx
	}
	e.Transactions = transactions
	if dec.Withdrawals == nil {
		dec.Withdrawals = make([]*Withdrawal, 0)
	}
	e.Withdrawals = dec.Withdrawals
	return nil
}

type payloadAttributesJSON struct {
	Timestamp             hexutil.Uint64 `json:"timestamp"`
	PrevRandao            hexutil.Bytes  `json:"prevRandao"`
	SuggestedFeeRecipient hexutil.Bytes  `json:"suggestedFeeRecipient"`
}

type payloadAttributesV2JSON struct {
	Timestamp             hexutil.Uint64 `json:"timestamp"`
	PrevRandao            hexutil.Bytes  `json:"prevRandao"`
	SuggestedFeeRecipient hexutil.Bytes  `json:"suggestedFeeRecipient"`
	Withdrawals           []*Withdrawal  `json:"withdrawals"`
}

// MarshalJSON --
func (p *PayloadAttributes) MarshalJSON() ([]byte, error) {
	return json.Marshal(payloadAttributesJSON{
		Timestamp:             hexutil.Uint64(p.Timestamp),
		PrevRandao:            p.PrevRandao,
		SuggestedFeeRecipient: p.SuggestedFeeRecipient,
	})
}

// MarshalJSON --
func (p *PayloadAttributesV2) MarshalJSON() ([]byte, error) {
	return json.Marshal(payloadAttributesV2JSON{
		Timestamp:             hexutil.Uint64(p.Timestamp),
		PrevRandao:            p.PrevRandao,
		SuggestedFeeRecipient: p.SuggestedFeeRecipient,
		Withdrawals:           p.Withdrawals,
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

func (p *PayloadAttributesV2) UnmarshalJSON(enc []byte) error {
	dec := payloadAttributesV2JSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}
	*p = PayloadAttributesV2{}
	p.Timestamp = uint64(dec.Timestamp)
	p.PrevRandao = dec.PrevRandao
	p.SuggestedFeeRecipient = dec.SuggestedFeeRecipient
	p.Withdrawals = dec.Withdrawals
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
	BlockHash       common.Hash               `json:"blockHash"`
	Kzgs            []gethTypes.KZGCommitment `json:"kzgs"`
	Blobs           []gethTypes.Blob          `json:"blobs"`
	AggregatedProof gethTypes.KZGProof        `json:"aggregatedProof"`
}

// MarshalJSON --
func (b *BlobsBundle) MarshalJSON() ([]byte, error) {
	kzgs := make([]gethTypes.KZGCommitment, len(b.KzgCommitments))
	for i, kzg := range b.KzgCommitments {
		kzgs[i] = bytesutil.ToBytes48(kzg)
	}
	blobs := make([]gethTypes.Blob, len(b.Blobs))
	for i, b1 := range b.Blobs {
		var blob [params.FieldElementsPerBlob]gethTypes.BLSFieldElement
		for j := 0; j < params.FieldElementsPerBlob; j++ {
			blob[j] = bytesutil.ToBytes32(b1.Data[j*32 : j*32+31])
		}
		blobs[i] = blob
	}

	return json.Marshal(blobBundleJSON{
		BlockHash: bytesutil.ToBytes32(b.BlockHash),
		Kzgs:      kzgs,
		Blobs:     blobs,
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
	e.KzgCommitments = kzgs
	blobs := make([]*Blob, len(dec.Blobs))
	for i, blob := range dec.Blobs {
		b := make([]byte, 0, params.FieldElementsPerBlob*32)
		for _, fe := range blob {
			b = append(b, fe[:]...)
		}
		blobs[i] = &Blob{Data: bytesutil.SafeCopyBytes(b)}
	}
	e.Blobs = blobs
	return nil
}
