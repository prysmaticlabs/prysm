package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
)

var (
	blockByHash = "eth_getBlockByHash"
	blockByNum  = "eth_getBlockByNumber"
)

func main() {
	endpoint := "http://10.0.0.161:8545"
	r, err := rpc.Dial(endpoint)
	if err != nil {
		panic(err)
	}

	client := ethclient.NewClient(r)
	ctx := context.Background()
	blk, err := client.BlockByNumber(ctx, nil)
	if err != nil {
		panic(err)
	}
	blkHash := blk.Hash()
	fmt.Printf("%#x\n", blkHash)

	itemMapping := make(map[string]interface{})
	if err := r.CallContext(
		ctx,
		&itemMapping,
		blockByHash,
		blkHash,
		false,
	); err != nil {
		panic(err)
	}

	fmt.Printf("MAPPING: %+v\n", itemMapping)
	result := &pb.ExecutionBlock{}
	encoded, err := json.Marshal(itemMapping)
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(encoded, &result); err != nil {
		panic(err)
	}
	fmt.Println("")
	fmt.Printf("RESULT BLOCK: %+v\n", result)
	//fmt.Printf("%#x and %d", result.Hash(), result.Number().Uint64())
	//encoded, err := json.Marshal(result)
	//if err != nil {
	//	panic(err)
	//}
	//if err := json.Unmarshal(encoded, &itemMapping); err != nil {
	//	panic(err)
	//}
	//fmt.Printf("%+v\n", itemMapping)
}
