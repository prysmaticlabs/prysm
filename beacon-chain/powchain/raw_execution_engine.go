package powchain

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
)

type BlockNumberRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}

type BlockNumberRespond struct {
	JsonRPC string       `json:"jsonrpc"`
	Result  *BlockResult `json:"result"`
	Id      int          `json:"id"`
}

type BlockResult struct {
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
	Transactions     []string `json:"transactions"`
	Uncles           []string `json:"uncles"`
}

func (s *Service) LatestTotalBlockDifficulty() (*big.Int, error) {
	reqBody := &BlockNumberRequest{
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
	var data BlockNumberRespond
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return common.HexToHash(data.Result.TotalDifficulty).Big(), nil
}

func (s *Service) LatestTotalBlockDifficultyByHash(blockHash common.Hash) (*big.Int, error) {
	reqBody := &BlockNumberRequest{
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
	var data BlockNumberRespond
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return common.HexToHash(data.Result.TotalDifficulty).Big(), nil
}
