package sharding

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// Subscribes to pending sharding transactions. For now, it listens to tx's from the Geth node
// through JSON-RPC
func subscribePendingTransactions(c shardingClient) error {
	txChan := make(chan *common.Hash, 16)

	_, err := c.PendingStateEventer().SubscribePendingTransactions(context.Background(), txChan)
	if err != nil {
		return fmt.Errorf("unable to subscribe to pending tx's. %v", err)
	}

	log.Info("Listening for pending tx's...")

	for {
		// TODO: Error handling for getting disconnected from the client
		hash := <-txChan
		// Query the current state to see if we are an eligible collator
		log.Info("Received new pending tx")

		pending, err := c.Client().TransactionByHash(context.Background(), hash)
		if err != nil {
			return fmt.Errorf("Could not fetch tx by hash. %v", err)
		}
		log.Info(fmt.Sprintf("TX Nonce: %v", pending.Nonce()))
	}
}
