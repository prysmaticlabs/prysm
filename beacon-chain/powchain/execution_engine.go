package powchain

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/pkg/errors"
)

var errNoExecutionEngineConnection = errors.New("can't connect to execution engine")

// ExecutionEngineCaller defines methods that wraps around execution engine API calls to enable other prysm services to interact with.
type ExecutionEngineCaller interface {
	// PreparePayload is a wrapper on top of `CatalystClient` to abstract out `types.AssembleBlockParams`.
	PreparePayload(ctx context.Context, parentHash []byte, timeStamp uint64, random []byte, feeRecipient []byte) (uint64, error)
	// GetPayload is a wrapper on top of `CatalystClient`.
	GetPayload(ctx context.Context, payloadID uint64) (*catalyst.ExecutableDataV1, error)
	// NotifyConsensusValidated is the wrapper on top of `CatalystClient` to abstract out `types.ConsensusValidatedParams`.
	NotifyConsensusValidated(ctx context.Context, blockHash []byte, valid bool) error
	// NotifyForkChoiceValidated is the wrapper on top of `CatalystClient` to abstract out `types.ConsensusValidatedParams`.
	NotifyForkChoiceValidated(ctx context.Context, forkchoiceState catalyst.ForkchoiceStateV1, payloadAttributes catalyst.PayloadAttributesV1) (uint64, error)
	// ExecutePayload is the wrapper on top of `CatalystClient` to abstract out `types.ForkChoiceParams`.
	ExecutePayload(ctx context.Context, data *catalyst.ExecutableDataV1) error
	// LatestExecutionBlock returns the latest execution block of the pow chain.
	LatestExecutionBlock() (*ExecutionBlock, error)
	// ExecutionBlockByHash returns the execution block of a given block hash.
	ExecutionBlockByHash(blockHash common.Hash) (*ExecutionBlock, error)
}

type GetPayloadRespond struct {
	JsonRPC        string                     `json:"jsonrpc"`
	ExecutableData *catalyst.ExecutableDataV1 `json:"result"`
	Id             int                        `json:"id"`
}

type EngineRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}

type PayloadIDRespond struct {
	PayloadID string `json:"payloadId"`
}

type PreparePayloadRespond struct {
	JsonRPC string           `json:"jsonrpc"`
	Result  PayloadIDRespond `json:"result"`
	Id      int              `json:"id"`
}

type ForkchoiceUpdatedRespond struct {
	JsonRPC string                  `json:"jsonrpc"`
	Result  ForkchoiceUpdatedResult `json:"result"`
	Id      int                     `json:"id"`
}

type ForkchoiceUpdatedResult struct {
	Status    string `json:"status"`
	PayloadID string `json:"payloadId"`
}

type ExecutePayloadRespond struct {
	JsonRPC string               `json:"jsonrpc"`
	Result  ExecutePayloadResult `json:"result"`
	Id      int                  `json:"id"`
}

type ExecutePayloadResult struct {
	Status string `json:"status"`
}

type ConsensusValidatedRespond struct {
	JsonRPC string `json:"jsonrpc"`
	Result  string `json:"status"`
	Id      int    `json:"id"`
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

// PreparePayload initiates a process of building an execution payload on top of the execution chain tip by parent hash.
// it returns an uint64 payload id that is used to obtain the execution payload in a subsequent `GetPayload` call.
// Engine API definition:
//  https://github.com/ethereum/execution-apis/blob/main/src/engine/interop/specification.md#engine_preparepayload
func (s *Service) PreparePayload(ctx context.Context, parentHash []byte, timeStamp uint64, random []byte, feeRecipient []byte) (uint64, error) {
	return 0, nil
	//reqBody := &EngineRequest{
	//	JsonRPC: "2.0",
	//	Method:  "engine_preparePayload",
	//	Params: []interface{}{catalyst.AssembleBlockParams{
	//		ParentHash:   common.BytesToHash(parentHash),
	//		Timestamp:    timeStamp,
	//		Random:       common.BytesToHash(random),
	//		FeeRecipient: common.BytesToAddress(feeRecipient),
	//	}},
	//}
	//enc, err := json.Marshal(reqBody)
	//if err != nil {
	//	return 0, err
	//}
	//req, err := http.NewRequest("POST", s.currHttpEndpoint.Url, bytes.NewBuffer(enc))
	//if err != nil {
	//	return 0, err
	//}
	//req.Header.Set("Content-Type", "application/json")
	//client := &http.Client{}
	//res, err := client.Do(req)
	//if err != nil {
	//	return 0, err
	//}
	//defer func() {
	//	if err := res.Body.Close(); err != nil {
	//		panic(err)
	//	}
	//}()
	//body, err := ioutil.ReadAll(res.Body)
	//if err != nil {
	//	return 0, err
	//}
	//var respond PreparePayloadRespond
	//if err := json.Unmarshal(body, &respond); err != nil {
	//	return 0, err
	//}
	//id, ok := math.ParseUint64(respond.Result.PayloadID)
	//if !ok {
	//	return 0, errors.New("could not parse hex to uint64")
	//}
	//return id, nil
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
	req, err := http.NewRequest("POST", s.currHttpEndpoint.Url, bytes.NewBuffer(enc))
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

	return respond.ExecutableData, nil
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
	req, err := http.NewRequest("GET", s.currHttpEndpoint.Url, bytes.NewBuffer(enc))
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
	req, err := http.NewRequest("GET", s.currHttpEndpoint.Url, bytes.NewBuffer(enc))
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

// ExecutePayload executes execution payload by calling execution engine.
// Engine API definition:
// 	https://github.com/ethereum/execution-apis/blob/main/src/engine/interop/specification.md#engine_executepayload
func (s *Service) ExecutePayload(ctx context.Context, data *catalyst.ExecutableDataV1) error {
	reqBody := &EngineRequest{
		JsonRPC: "2.0",
		Method:  "engine_executePayload",
		Params:  []interface{}{data},
	}
	enc, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", s.currHttpEndpoint.Url, bytes.NewBuffer(enc))
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
	var respond ExecutePayloadRespond
	if err := json.Unmarshal(body, &respond); err != nil {
		return err
	}
	if respond.Result.Status == catalyst.INVALID.Status {
		return errors.New("invalid execute payload status respond")
	}
	if respond.Result.Status == catalyst.SYNCING.Status {
		return errors.New("execution engine is syncing")
	}

	return nil
}

// NotifyConsensusValidated notifies execution engine on the result of beacon state transition.
// Per definition, consensus engine must notify execution engine after `state_transition` function finishes.
// The value of valid parameters must be set as follows:
// -True if state_transition function call succeeds
// -False if state_transition function call fails
// Engine API definition:
// 	https://github.com/ethereum/consensus-specs/blob/dev/specs/merge/beacon-chain.md#notify_consensus_validated
func (s *Service) NotifyConsensusValidated(ctx context.Context, blockHash []byte, valid bool) error {
	validString := "INVALID"
	if valid {
		validString = "VALID"
	}
	reqBody := &EngineRequest{
		JsonRPC: "2.0",
		Method:  "engine_consensusValidated",
		Params: []interface{}{catalyst.ConsensusValidatedParams{
			BlockHash: common.BytesToHash(blockHash),
			Status:    validString,
		}},
	}
	enc, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", s.currHttpEndpoint.Url, bytes.NewBuffer(enc))
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
	return nil
}

// NotifyForkChoiceValidated notifies execution engine on fork choice updates.
// Engine API definition:
// https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#engine_forkchoiceupdatedv1
func (s *Service) NotifyForkChoiceValidated(ctx context.Context, forkchoiceState catalyst.ForkchoiceStateV1, payloadAttributes catalyst.PayloadAttributesV1) (uint64, error) {
	reqBody := &EngineRequest{
		JsonRPC: "2.0",
		Method:  "engine_forkchoiceUpdatedV1",
		Params:  []interface{}{forkchoiceState, payloadAttributes},
	}
	enc, err := json.Marshal(reqBody)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest("POST", s.currHttpEndpoint.Url, bytes.NewBuffer(enc))
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

	id, ok := math.ParseUint64(respond.Result.PayloadID)
	if !ok {
		return 0, errors.New("could not parse hex to uint64")
	}
	return id, nil
}
