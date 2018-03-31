package client

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/contracts"
)

// initSMC initializes the sharding manager contract bindings.
// If the SMC does not exist, it will be deployed.
func initSMC(c *ShardingClient) error {
	b, err := c.client.CodeAt(context.Background(), sharding.ShardingManagerAddress, nil)
	if err != nil {
		return fmt.Errorf("unable to get contract code at %s: %v", sharding.ShardingManagerAddress, err)
	}

	if len(b) == 0 {
		log.Info(fmt.Sprintf("No sharding manager contract found at %s. Deploying new contract.", sharding.ShardingManagerAddress.String()))

		txOps, err := c.CreateTXOps(big.NewInt(0))
		if err != nil {
			return fmt.Errorf("unable to intiate the transaction: %v", err)
		}

		addr, tx, contract, err := contracts.DeploySMC(txOps, c.client)
		if err != nil {
			return fmt.Errorf("unable to deploy sharding manager contract: %v", err)
		}

		for pending := true; pending; _, pending, err = c.client.TransactionByHash(context.Background(), tx.Hash()) {
			if err != nil {
				return fmt.Errorf("unable to get transaction by hash: %v", err)
			}
			time.Sleep(1 * time.Second)
		}

		c.Smc = contract
		log.Info(fmt.Sprintf("New contract deployed at %s", addr.String()))
	} else {
		contract, err := contracts.NewSMC(sharding.ShardingManagerAddress, c.client)
		if err != nil {
			return fmt.Errorf("failed to create sharding contract: %v", err)
		}
		c.Smc = contract
	}

	return nil
}
