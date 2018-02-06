package sharding

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/contracts"
)

// initVMC initializes the validator management contract bindings.
// If the VMC does not exist, it will be deployed.
func initVMC(c *Client) error {
	b, err := c.client.CodeAt(context.Background(), validatorManagerAddress, nil)
	if err != nil {
		return fmt.Errorf("unable to get contract code at %s: %v", validatorManagerAddress, err)
	}

	if len(b) == 0 {
		log.Info(fmt.Sprintf("No validator management contract found at %s. Deploying new contract.", validatorManagerAddress.String()))

		txOps, err := c.createTXOps(big.NewInt(0))
		if err != nil {
			return fmt.Errorf("unable to intiate the transaction: %v", err)
		}

		addr, tx, contract, err := contracts.DeployVMC(&txOps, c.client)
		if err != nil {
			return fmt.Errorf("unable to deploy validator management contract: %v", err)
		}

		for pending := true; pending; _, pending, err = c.client.TransactionByHash(context.Background(), tx.Hash()) {
			if err != nil {
				return fmt.Errorf("unable to get transaction by hash: %v", err)
			}
			time.Sleep(1 * time.Second)
		}

		c.vmc = contract
		log.Info(fmt.Sprintf("New contract deployed at %s", addr.String()))
	} else {
		contract, err := contracts.NewVMC(validatorManagerAddress, c.client)
		if err != nil {
			return fmt.Errorf("failed to create validator contract: %v", err)
		}
		c.vmc = contract
	}

	return nil
}

// joinValidatorSet checks if the account is a validator in the VMC. If
// the account is not in the set, it will deposit 100ETH into contract.
func joinValidatorSet(c *Client) error {

	// TODO: Check if account is already in validator set. Fetch this From
	// the VMC contract's validator set
	txOps, err := c.createTXOps(depositSize)
	if err != nil {
		return fmt.Errorf("unable to intiate the deposit transaction: %v", err)
	}

	tx, err := c.vmc.VMCTransactor.Deposit(&txOps)
	if err != nil {
		return fmt.Errorf("unable to deposit eth and become a validator: %v", err)
	}
	log.Info(fmt.Sprintf("Deposited 100ETH into contract with transaction hash: %s", tx.Hash().String()))
	return nil

}
