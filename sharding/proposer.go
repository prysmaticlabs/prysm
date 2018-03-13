package sharding

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// Subscribes to pending sharding transactions. For now, it listens to tx's from the Geth node
// through JSON-RPC
func subscribePendingTransactions(c shardingClient) error {
	txChan := make(chan *types.Transaction, 16)

	_, err := c.PendingStateEventer().SubscribePendingTransactions(context.Background(), txChan)
	if err != nil {
		return fmt.Errorf("unable to subscribe to pending tx's. %v", err)
	}

	log.Info("Listening for pending tx's...")

	for {
		// TODO: Error handling for getting disconnected from the client
		tx := <-txChan
		// Query the current state to see if we are an eligible collator
		log.Info(fmt.Sprintf("Received new pending tx: %v", tx.Hash().Str()))
	}
}
