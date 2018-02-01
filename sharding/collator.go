package sharding

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
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
			// TODO: Only run this code on certain periods?
			err := watchShards(c, head)
			if err != nil {
				return err
			}
		}
	}
}

// watchShards checks if we are an eligible proposer for collation for
// the available shards in the VMC. The function calls getEligibleProposer from
// the VMC and proposes a collation if conditions are met
func watchShards(c *Client, head *types.Header) error {

	accounts := c.keystore.Accounts()
	if len(accounts) == 0 {
		return fmt.Errorf("no accounts found")
	}

	if err := c.unlockAccount(accounts[0]); err != nil {
		return err
	}

	ops := bind.CallOpts{}
	count, err := c.vmc.VMCCaller.ShardCount(&ops)
	if err != nil {
		return err
	}

	s := 0
	for s < int(count.Int64()) {
		// Checks if we are an eligible proposer according to the VMC
		log.Info(fmt.Sprintf("shard %d", s))
		s++
	}

	return nil
}
