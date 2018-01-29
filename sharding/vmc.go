package sharding

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/contracts"
)

// initVMC initializes the validator management contract bindings.
// If the VMC does not exist, it will be deployed.
func initVMC(c *Client) error {
	b, err := c.client.CodeAt(context.Background(), validatorManagerAddress, nil)
	if err != nil {
		return fmt.Errorf("unable to get contract code at %s. %v", validatorManagerAddress, err)
	}

	if len(b) == 0 {
		log.Info(fmt.Sprintf("No validator management contract found at %s. Deploying new contract.", validatorManagerAddress.String()))

		accounts := c.keystore.Accounts()
		if len(accounts) == 0 {
			return fmt.Errorf("no accounts found")
		}

		if err := c.unlockAccount(accounts[0]); err != nil {
			return fmt.Errorf("unable to unlock account 0: %v", err)
		}
		ops := bind.TransactOpts{
			From: accounts[0].Address,
			Signer: func(signer types.Signer, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
				networkID, err := c.client.NetworkID(context.Background())
				if err != nil {
					return nil, fmt.Errorf("unable to fetch networkID: %v", err)
				}
				return c.keystore.SignTx(accounts[0], tx, networkID /* chainID */)
			},
		}

		addr, tx, contract, err := contracts.DeployVMC(&ops, c.client)
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

// initVMCValidator checks if the account is a validator in the VMC. If
// the account is not in the set, it will deposit 100ETH into contract.
func initVMCValidator(c *Client) error {

	// TODO: Check if account is already in validator set

	accounts := c.keystore.Accounts()
	if len(accounts) == 0 {
		return fmt.Errorf("no accounts found")
	}

	if err := c.unlockAccount(accounts[0]); err != nil {
		return fmt.Errorf("unable to unlock account 0: %v", err)
	}

	// Deposits 100ETH into the VMC from the current account
	ops := bind.TransactOpts{
		From:  accounts[0].Address,
		Value: depositSize,
		Signer: func(signer types.Signer, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
			networkID, err := c.client.NetworkID(context.Background())
			if err != nil {
				return nil, fmt.Errorf("unable to fetch networkID: %v", err)
			}
			return c.keystore.SignTx(accounts[0], tx, networkID /* chainID */)
		},
	}

	tx, err := c.vmc.VMCTransactor.Deposit(&ops, /* validatorcodeaddr */, accounts[0].Address)
	if err != nil {
		return fmt.Errorf("unable to deposit eth and become a validator: %v", err)
	}
	return nil

}
