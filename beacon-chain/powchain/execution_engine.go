package powchain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/pkg/errors"
)

var errNoExecutionEngineConnection = errors.New("can't connect to execution engine")
var errInvalidPayload = errors.New("invalid payload")
var errSyncing = errors.New("syncing")

// ExecutionEngineCaller defines methods that wraps around execution engine API calls to enable other prysm services to interact with.
type ExecutionEngineCaller interface {
	// PreparePayload is a wrapper on top of `CatalystClient` to abstract out `types.AssembleBlockParams`.
	PreparePayload(ctx context.Context, forkchoiceState catalyst.ForkchoiceStateV1, payloadAttributes catalyst.PayloadAttributesV1) (uint64, error)
	// GetPayload is a wrapper on top of `CatalystClient`.
	GetPayload(ctx context.Context, payloadID uint64) (*catalyst.ExecutableDataV1, error)
	// NotifyForkChoiceValidated is the wrapper on top of `CatalystClient` to abstract out `types.ConsensusValidatedParams`.
	NotifyForkChoiceValidated(ctx context.Context, forkchoiceState catalyst.ForkchoiceStateV1) error
	// ExecutePayload is the wrapper on top of `CatalystClient` to abstract out `types.ForkChoiceParams`.
	ExecutePayload(ctx context.Context, data *catalyst.ExecutableDataV1) ([]byte, error)
	// LatestExecutionBlock returns the latest execution block of the pow chain.
	LatestExecutionBlock() (*ExecutionBlock, error)
	// ExecutionBlockByHash returns the execution block of a given block hash.
	ExecutionBlockByHash(blockHash common.Hash) (*ExecutionBlock, error)
}

type EngineRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}

type ErrorRespond struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type GetPayloadRespond struct {
	JsonRPC        string                     `json:"jsonrpc"`
	ExecutableData *catalyst.ExecutableDataV1 `json:"result"`
	Id             int                        `json:"id"`
	Error          ErrorRespond               `json:"error"`
}

type PreparePayloadRespond struct {
	JsonRPC string           `json:"jsonrpc"`
	Result  PayloadIDRespond `json:"result"`
	Id      int              `json:"id"`
}

type PayloadIDRespond struct {
	PayloadID string `json:"payloadId"`
}

type ForkchoiceUpdatedRespond struct {
	JsonRPC string                  `json:"jsonrpc"`
	Result  ForkchoiceUpdatedResult `json:"result"`
	Id      int                     `json:"id"`
	Error   ErrorRespond            `json:"error"`
}

type ForkchoiceUpdatedResult struct {
	Status    string `json:"status"`
	PayloadID string `json:"payloadId"`
}

type ExecutePayloadRespond struct {
	JsonRPC string               `json:"jsonrpc"`
	Result  ExecutePayloadResult `json:"result"`
	Id      int                  `json:"id"`
	Error   ErrorRespond         `json:"error"`
}

type ExecutePayloadResult struct {
	Status          string `json:"status"`
	LatestValidHash string `json:"latestValidHash"`
	Message         string `json:"message"`
}

type ExecutionBlockRespond struct {
	JsonRPC string          `json:"jsonrpc"`
	Result  *ExecutionBlock `json:"result"`
	Id      int             `json:"id"`
}

type ExecutionBlock struct {
	ParentHash       string   `json:"parentHash"`
	Sha3Uncles       string   `json:"sha3Uncles"`
	Miner            string   `json:"miner"`
	StateRoot        string   `json:"stateRoot"`
	TransactionsRoot string   `json:"transactionsRoot"`
	ReceiptsRoot     string   `json:"receiptsRoot"`
	LogsBloom        string   `json:"logsBloom"`
	Difficulty       string   `json:"difficulty"`
	Number           string   `json:"number"`
	GasLimit         string   `json:"gasLimit"`
	GasUsed          string   `json:"gasUsed"`
	Timestamp        string   `json:"timestamp"`
	ExtraData        string   `json:"extraData"`
	MixHash          string   `json:"mixHash"`
	Nonce            string   `json:"nonce"`
	TotalDifficulty  string   `json:"totalDifficulty"`
	BaseFeePerGas    string   `json:"baseFeePerGas"`
	Size             string   `json:"size"`
	Hash             string   `json:"hash"`
	Transactions     []string `json:"transactions"`
	Uncles           []string `json:"uncles"`
}

//GetPayload returns the most recent version of the execution payload that has been built since the corresponding
//call to `PreparePayload` method. It returns the `ExecutionPayload` object.
//Engine API definition:
// https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#engine_getpayloadv1
func (s *Service) GetPayload(ctx context.Context, payloadID uint64) (*catalyst.ExecutableDataV1, error) {
	reqBody := &EngineRequest{
		JsonRPC: "2.0",
		Method:  "engine_getPayloadV1",
		Params:  []interface{}{hexutil.EncodeUint64(payloadID)},
	}
	enc, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", s.cfg.currHttpEndpoint.Url, bytes.NewBuffer(enc))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			panic(err)
		}
	}()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var respond GetPayloadRespond
	if err := json.Unmarshal(body, &respond); err != nil {
		return nil, err
	}
	if respond.Error.Code != 0 {
		return nil, fmt.Errorf("could not call engine_getPayloadV1, code: %d, message: %s", respond.Error.Code, respond.Error.Message)
	}

	return respond.ExecutableData, nil
}

// ExecutePayload executes execution payload by calling execution engine.
// Engine API definition:
// 	https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#engine_executepayloadv1
func (s *Service) ExecutePayload(ctx context.Context, data *catalyst.ExecutableDataV1) ([]byte, error) {
	// TODO: Fix this. Somehow transactions becomes nil with grpc call server->client
	if data.Transactions == nil {
		data.Transactions = [][]byte{}
	}
	reqBody := &EngineRequest{
		JsonRPC: "2.0",
		Method:  "engine_executePayloadV1",
		Params:  []interface{}{data},
	}
	enc, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", s.cfg.currHttpEndpoint.Url, bytes.NewBuffer(enc))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			panic(err)
		}
	}()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var respond ExecutePayloadRespond
	if err := json.Unmarshal(body, &respond); err != nil {
		return nil, err
	}
	if respond.Error.Code != 0 {
		return nil, fmt.Errorf("could not call engine_executePayloadV1, code: %d, message: %s", respond.Error.Code, respond.Error.Message)
	}

	if respond.Result.Status == catalyst.INVALID.Status {
		return common.FromHex(respond.Result.LatestValidHash), errInvalidPayload
	}
	if respond.Result.Status == catalyst.SYNCING.Status {
		return common.FromHex(respond.Result.LatestValidHash), errSyncing
	}

	return common.FromHex(respond.Result.LatestValidHash), nil
}

// NotifyForkChoiceValidated notifies execution engine on fork choice updates.
// Engine API definition:
// https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#engine_forkchoiceupdatedv1
func (s *Service) NotifyForkChoiceValidated(ctx context.Context, forkchoiceState catalyst.ForkchoiceStateV1) error {
	reqBody := &EngineRequest{
		JsonRPC: "2.0",
		Method:  "engine_forkchoiceUpdatedV1",
		Params:  []interface{}{forkchoiceState, nil},
	}
	enc, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", s.cfg.currHttpEndpoint.Url, bytes.NewBuffer(enc))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			panic(err)
		}
	}()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var respond ForkchoiceUpdatedRespond
	if err := json.Unmarshal(body, &respond); err != nil {
		return err
	}
	if respond.Error.Code != 0 {
		return fmt.Errorf("could not call engine_forkchoiceUpdatedV1, code: %d, message: %s", respond.Error.Code, respond.Error.Message)
	}
	if respond.Result.Status == catalyst.SYNCING.Status {
		return errSyncing
	}

	return nil
}

// PreparePayload signals execution engine to prepare execution payload along with the latest fork choice state.
// It reuses `engine_forkchoiceUpdatedV1` end point to prepare payload.
// Engine API definition:
// https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#engine_forkchoiceupdatedv1
func (s *Service) PreparePayload(ctx context.Context, forkchoiceState catalyst.ForkchoiceStateV1, payloadAttributes catalyst.PayloadAttributesV1) (uint64, error) {
	reqBody := &EngineRequest{
		JsonRPC: "2.0",
		Method:  "engine_forkchoiceUpdatedV1",
		Params:  []interface{}{forkchoiceState, payloadAttributes},
	}
	enc, err := json.Marshal(reqBody)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest("POST", s.cfg.currHttpEndpoint.Url, bytes.NewBuffer(enc))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			panic(err)
		}
	}()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	var respond ForkchoiceUpdatedRespond
	if err := json.Unmarshal(body, &respond); err != nil {
		return 0, err
	}
	if respond.Error.Code != 0 {
		return 0, fmt.Errorf("could not call engine_forkchoiceUpdatedV1, code: %d, message: %s", respond.Error.Code, respond.Error.Message)
	}
	if respond.Result.Status == catalyst.SYNCING.Status {
		return 0, errSyncing
	}
	id, ok := math.ParseUint64(respond.Result.PayloadID)
	if !ok {
		return 0, errors.New("could not parse hex to uint64")
	}

	return id, nil
}

func (s *Service) LatestExecutionBlock() (*ExecutionBlock, error) {
	reqBody := &EngineRequest{
		JsonRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{"latest", false},
	}
	enc, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", s.cfg.currHttpEndpoint.Url, bytes.NewBuffer(enc))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			panic(err)
		}
	}()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var data ExecutionBlockRespond
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return data.Result, nil
}

func (s *Service) ExecutionBlockByHash(blockHash common.Hash) (*ExecutionBlock, error) {
	reqBody := &EngineRequest{
		JsonRPC: "2.0",
		Method:  "eth_getBlockByHash",
		Params:  []interface{}{blockHash.String(), false},
	}
	enc, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", s.cfg.currHttpEndpoint.Url, bytes.NewBuffer(enc))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			panic(err)
		}
	}()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var data ExecutionBlockRespond
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return data.Result, nil
}
