package sharding

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// subscribeBlockHeaders checks incoming block headers and determines if
// we are an eligible proposer for collations. Then, it finds the pending tx's
// from the running geth node and sorts them by descending order of gas price,
// eliminates those that ask for too much gas, and routes them over
// to the VMC to create a collation
func subscribeBlockHeaders(c *Client) error {
	headerChan := make(chan *types.Header, 16)

	_, err := c.client.SubscribeNewHead(context.Background(), headerChan)
	if err != nil {
		return err
	}

	log.Info("listening for new headers...")

	for {
		select {
		case head := <-headerChan:
			// Query the current state to see if we are an eligible proposer
			log.Info(fmt.Sprintf("received new header %v", head.Number.String()))
		}
	}
}
