package collator

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/client"
)

// joinCollatorPool checks if the account is a collator in the SMC. If
// the account is not in the set, it will deposit 100ETH into contract.
func joinCollatorPool(c *client.ShardingClient) error {

	if c.Ctx.GlobalBool(utils.DepositFlag.Name) {

		log.Info("Joining collator pool")
		txOps, err := c.CreateTXOps(sharding.DepositSize)
		if err != nil {
			return fmt.Errorf("unable to intiate the deposit transaction: %v", err)
		}

		tx, err := c.Smc.SMCTransactor.Deposit(txOps)
		if err != nil {
			return fmt.Errorf("unable to deposit eth and become a collator: %v", err)
		}
		log.Info(fmt.Sprintf("Deposited %dETH into contract with transaction hash: %s", new(big.Int).Div(sharding.DepositSize, big.NewInt(params.Ether)), tx.Hash().String()))

	} else {
		log.Info("Not joining collator pool")

	}
	return nil
}
