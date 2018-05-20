package client

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/contracts"
)

// dialRPC endpoint to node.
func dialRPC(endpoint string) (*rpc.Client, error) {
	if endpoint == "" {
		endpoint = node.DefaultIPCEndpoint(clientIdentifier)
	}
	return rpc.Dial(endpoint)
}

// initSMC initializes the sharding manager contract bindings.
// If the SMC does not exist, it will be deployed.
func initSMC(c *shardingClient) (*contracts.SMC, error) {
	b, err := c.client.CodeAt(context.Background(), sharding.ShardingManagerAddress, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to get contract code at %s: %v", sharding.ShardingManagerAddress, err)
	}

	// Deploy SMC for development only.
	// TODO: Separate contract deployment from the sharding client. It would only need to be deployed
	// once on the mainnet, so this code would not need to ship with the client.
	if len(b) == 0 {
		log.Info(fmt.Sprintf("No sharding manager contract found at %s. Deploying new contract.", sharding.ShardingManagerAddress.String()))

		txOps, err := c.CreateTXOpts(big.NewInt(0))
		if err != nil {
			return nil, fmt.Errorf("unable to intiate the transaction: %v", err)
		}

		addr, tx, contract, err := contracts.DeploySMC(txOps, c.client)
		if err != nil {
			return nil, fmt.Errorf("unable to deploy sharding manager contract: %v", err)
		}

		for pending := true; pending; _, pending, err = c.client.TransactionByHash(context.Background(), tx.Hash()) {
			if err != nil {
				return nil, fmt.Errorf("unable to get transaction by hash: %v", err)
			}
			time.Sleep(1 * time.Second)
		}

		log.Info(fmt.Sprintf("New contract deployed at %s", addr.String()))
		return contract, nil
	}

	contract, err := contracts.NewSMC(sharding.ShardingManagerAddress, c.client)
	if err != nil {

	}
	return contract, nil
}

// unlockAccount will unlock the specified account using utils.PasswordFileFlag
// or empty string if unset.
func unlockAccount(c *shardingClient, account accounts.Account) error {
	pass := ""

	if c.ctx.GlobalIsSet(utils.PasswordFileFlag.Name) {
		file, err := os.Open(c.ctx.GlobalString(utils.PasswordFileFlag.Name))
		if err != nil {
			return fmt.Errorf("unable to open file containing account password %s. %v", utils.PasswordFileFlag.Value, err)
		}
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanWords)
		if !scanner.Scan() {
			err = scanner.Err()
			if err != nil {
				return fmt.Errorf("unable to read contents of file %v", err)
			}
			return errors.New("password not found in file")
		}

		pass = scanner.Text()
	}

	return c.keystore.Unlock(account, pass)
}
