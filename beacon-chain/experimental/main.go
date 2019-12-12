package main

import (
	"context"
	"log"

	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
)

func main() {
	store, err := kv.NewKVStore("/Users/zypherx/Desktop/some5/beaconchaindata")
	if err != nil {
		panic(err)
	}
	blocks, err := store.Blocks(context.Background(), filters.NewFilter().SetStartSlot(0).SetEndSlot(0))
	if err != nil {
		panic(err)
	}
	for i := 0; i < len(blocks); i++ {
		log.Println(blocks[i])
	}
}

