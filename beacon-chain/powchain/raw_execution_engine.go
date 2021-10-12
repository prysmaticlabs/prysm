package powchain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type BlockNumberRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}

type BlockNumberRespond struct {
	ParentHash       string `json:"parentHash"`
	Sha3Uncles       string `json:"sha3Uncles"`
	Miner            string `json:"miner"`
	StateRoot        string `json:"stateRoot"`
	TransactionsRoot string `json:"transactionsRoot"`
	ReceiptsRoot     string `json:"receiptsRoot"`
	LogsBloom        string `json:"logsBloom"`
	Difficulty       string `json:"difficulty"`
	Number           string `json:"number"`
	GasLimit         string `json:"gasLimit"`
	GasUsed          string `json:"gasUsed"`
	Timestamp        string `json:"timestamp"`
	ExtraData        string `json:"extraData"`
	MixHash          string `json:"mixHash"`
	Nonce            string `json:"nonce"`
	TotalDifficulty  string `json:"totalDifficulty"`
	BaseFeePerGas    string `json:"baseFeePerGas"`
	Size             string `json:"size"`
	Transactions interface{} `json:"transactions"`
	Uncles interface{} `json:"uncles"`
}

func (s *Service) RawLatestBlockDifficulty() error {
	reqBody := &BlockNumberRequest{
		JsonRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{"latest", false},
		Id:      1,
	}
	enc, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", s.currHttpEndpoint.Url, bytes.NewBuffer(enc))
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
	fmt.Println("body", string(body))
	var data BlockNumberRespond
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}
	fmt.Println("block number: ", data.Number)
	fmt.Println("total difficulty: ", data.TotalDifficulty)
	return nil
}
