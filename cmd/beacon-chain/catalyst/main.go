package main

import (
	"github.com/ethereum/go-ethereum/ethclient"
	gethRPC "github.com/ethereum/go-ethereum/rpc"
	log "github.com/sirupsen/logrus"
)

func main() {
	httpRPCClient, err := gethRPC.Dial("http://localhost:8545")
	if err != nil {
		panic(err)
	}
	httpClient := ethclient.NewClient(httpRPCClient)
	_ = httpClient
	log.Infof("Connected %T", httpClient.ProduceBlock)
}
