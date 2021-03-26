package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	gethRPC "github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()
	httpRPCClient, err := gethRPC.Dial("http://localhost:8545")
	if err != nil {
		panic(err)
	}
	httpClient := ethclient.NewClient(httpRPCClient)
	defer httpClient.Close()

	genesisRoot := common.HexToHash("0x3a3fdfc9ab6e17ff530b57bc21494da3848ebbeaf9343545fded7a18d221ffec")
	randaoMix := params.BeaconConfig().ZeroHash
	recentBlockRoots := make([]common.Hash, 8)
	for i := 0; i < len(recentBlockRoots); i++ {
		recentBlockRoots[i] = params.BeaconConfig().ZeroHash
	}
	req := eth.ProduceBlockParams{
		ParentHash:             genesisRoot,
		RandaoMix:              randaoMix,
		Slot:                   0,
		Timestamp:              uint64(time.Now().Unix()),
		RecentBeaconBlockRoots: recentBlockRoots,
	}
	log.Info("Prysm asking eth1 node to produce executable data...")
	resp, err := httpClient.ProduceBlock(ctx, req)
	if err != nil {
		panic(err)
	}
	log.WithFields(log.Fields{
		"coinbase":        fmt.Sprintf("%#x", resp.Coinbase),
		"blockHash":       fmt.Sprintf("%#x", resp.BlockHash),
		"difficulty":      resp.Difficulty,
		"gasLimit":        resp.GasLimit,
		"gasUsed":         resp.GasUsed,
		"logsBloom":       fmt.Sprintf("%#x", resp.BlockHash),
		"parentHash":      fmt.Sprintf("%#x", resp.ParentHash),
		"receiptRoot":     fmt.Sprintf("%#x", resp.ReceiptRoot),
		"stateRoot":       fmt.Sprintf("%#x", resp.StateRoot),
		"numTransactions": len(resp.Transactions),
	}).Info("Received response, executable data is ready to be put into a beacon block")
}
