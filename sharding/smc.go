package sharding

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/sharding/contracts"
)

// initSMC initializes the sharding manager contract bindings.
// If the SMC does not exist, it will be deployed.
func initSMC(c *collatorClient) error {
	b, err := c.client.CodeAt(context.Background(), shardingManagerAddress, nil)
	if err != nil {
		return fmt.Errorf("unable to get contract code at %s: %v", shardingManagerAddress, err)
	}

	if len(b) == 0 {
		log.Info(fmt.Sprintf("No sharding manager contract found at %s. Deploying new contract.", shardingManagerAddress.String()))

		txOps, err := c.createTXOps(big.NewInt(0))
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

		c.smc = contract
		log.Info(fmt.Sprintf("New contract deployed at %s", addr.String()))
	} else {
		contract, err := contracts.NewSMC(shardingManagerAddress, c.client)
		if err != nil {
			return fmt.Errorf("failed to create sharding contract: %v", err)
		}
		c.smc = contract
	}

	return nil
}

// joinCollatorPool checks if the account is a collator in the SMC. If
// the account is not in the set, it will deposit 100ETH into contract.
func joinCollatorPool(c *collatorClient) error {

	if c.ctx.GlobalBool(utils.DepositFlag.Name) {

		log.Info("Joining collator pool")
		txOps, err := c.createTXOps(depositSize)
		if err != nil {
			return fmt.Errorf("unable to intiate the deposit transaction: %v", err)
		}

		tx, err := c.smc.SMCTransactor.Deposit(txOps)
		if err != nil {
			return fmt.Errorf("unable to deposit eth and become a collator: %v", err)
		}
		log.Info(fmt.Sprintf("Deposited %dETH into contract with transaction hash: %s", new(big.Int).Div(depositSize, big.NewInt(params.Ether)), tx.Hash().String()))

	} else {
		log.Info("Not joining collator pool")

	}
	return nil
}
