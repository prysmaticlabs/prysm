package sharding

import (
	"context"
	"fmt"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/sharding/contracts"
	cli "gopkg.in/urfave/cli.v1"
)

type collator interface {
	Account() *accounts.Account
	UnlockAccount(account accounts.Account) error
	Context() *cli.Context
	ChainReader() ethereum.ChainReader
	SMCCaller() *contracts.SMCCaller
	SMCTransactor() *contracts.SMCTransactor
	InitSMC() error
	AttachClient(ec *ethclient.Client) *ethclient.Client
	CreateTXOps(value *big.Int) (*bind.TransactOpts, error)
	Endpoint() string
}

// StartCollatorClient inits the collator that connects to Geth node
func StartCollatorClient(c collator) error {
	log.Info("Starting collator client")
	rpcClient, err := dialRPC(c.Endpoint())
	if err != nil {
		return err
	}
	c.AttachClient(ethclient.NewClient(rpcClient))
	defer rpcClient.Close()

	account := c.Account()
	if err := c.UnlockAccount(*account); err != nil {
		return fmt.Errorf("cannot unlock account: %v", err)
	}

	if err := c.InitSMC(); err != nil {
		return err
	}

	// Deposit 100ETH into the collator set in the SMC. Checks if account
	// is already a collator in the SMC (in the case the client restarted)
	if err := joinCollatorPool(c); err != nil {
		return err
	}

	// Listens to block headers from the Geth node and if we are an eligible
	// collator, we fetch pending transactions and collator a collation
	if err := subscribeBlockHeaders(c); err != nil {
		return err
	}
	return nil
}

// WaitCollatorClient on shutdown
func WaitCollatorClient(c collator) {
	log.Info("Sharding client has been shutdown...")
}

// SubscribeBlockHeaders checks incoming block headers and determines if
// we are an eligible collator for collations. Then, it finds the pending tx's
// from the running geth node and sorts them by descending order of gas price,
// eliminates those that ask for too much gas, and routes them over
// to the SMC to create a collation
func subscribeBlockHeaders(c collator) error {
	headerChan := make(chan *types.Header, 16)

	account := c.Account()

	_, err := c.ChainReader().SubscribeNewHead(context.Background(), headerChan)
	if err != nil {
		return fmt.Errorf("unable to subscribe to incoming headers. %v", err)
	}

	log.Info("Listening for new headers...")

	for {
		// TODO: Error handling for getting disconnected from the client
		head := <-headerChan
		// Query the current state to see if we are an eligible collator
		log.Info(fmt.Sprintf("Received new header: %v", head.Number.String()))

		// Check if we are in the collator pool before checking if we are an eligible collator
		v, err := isAccountInCollatorPool(c)
		if err != nil {
			return fmt.Errorf("unable to verify client in collator pool. %v", err)
		}

		if v {
			if err := checkSMCForCollator(c, head); err != nil {
				return fmt.Errorf("unable to watch shards. %v", err)
			}
		} else {
			log.Warn(fmt.Sprintf("Account %s not in collator pool.", account.Address.String()))
		}

	}
}

// joinCollatorPool checks if the account is a collator in the SMC. If
// the account is not in the set, it will deposit 100ETH into contract.
func joinCollatorPool(c collator) error {

	if c.Context().GlobalBool(utils.DepositFlag.Name) {

		log.Info("Joining collator pool")
		txOps, err := c.CreateTXOps(depositSize)
		if err != nil {
			return fmt.Errorf("unable to intiate the deposit transaction: %v", err)
		}

		tx, err := c.SMCTransactor().Deposit(txOps)
		if err != nil {
			return fmt.Errorf("unable to deposit eth and become a collator: %v", err)
		}
		log.Info(fmt.Sprintf("Deposited %dETH into contract with transaction hash: %s", new(big.Int).Div(depositSize, big.NewInt(params.Ether)), tx.Hash().String()))

	} else {
		log.Info("Not joining collator pool")

	}
	return nil
}

// checkSMCForCollator checks if we are an eligible collator for
// collation for the available shards in the SMC. The function calls
// getEligibleCollator from the SMC and collator a collation if
// conditions are met
func checkSMCForCollator(c collator, head *types.Header) error {
	account := c.Account()

	log.Info("Checking if we are an eligible collation collator for a shard...")
	period := big.NewInt(0).Div(head.Number, big.NewInt(periodLength))
	for s := int64(0); s < shardCount; s++ {
		// Checks if we are an eligible collator according to the SMC
		addr, err := c.SMCCaller().GetEligibleCollator(&bind.CallOpts{}, big.NewInt(s), period)

		if err != nil {
			return err
		}

		// If output is non-empty and the addr == coinbase
		if addr == account.Address {
			log.Info(fmt.Sprintf("Selected as collator on shard: %d", s))
			err := submitCollation(s)
			if err != nil {
				return fmt.Errorf("could not add collation. %v", err)
			}
		}
	}

	return nil
}

// isAccountInCollatorPool checks if the client is in the collator pool because
// we can't guarantee our tx for deposit will be in the next block header we receive.
// The function calls IsCollatorDeposited from the SMC and returns true if
// the client is in the collator pool
func isAccountInCollatorPool(c collator) (bool, error) {
	account := c.Account()
	// Checks if our deposit has gone through according to the SMC
	return c.SMCCaller().IsCollatorDeposited(&bind.CallOpts{}, account.Address)
}

// submitCollation interacts with the SMC directly to add a collation header
func submitCollation(shardID int64) error {
	// TODO: Adds a collation header to the SMC with the following fields:
	// [
	//  shard_id: uint256,
	//  expected_period_number: uint256,
	//  period_start_prevhash: bytes32,
	//  parent_hash: bytes32,
	//  transactions_root: bytes32,
	//  coinbase: address,
	//  state_root: bytes32,
	//  receipts_root: bytes32,
	//  number: uint256,
	//  sig: bytes
	// ]
	//
	// Before calling this, we would need to have access to the state of
	// the period_start_prevhash. Refer to the comments in:
	// https://github.com/ethereum/py-evm/issues/258#issuecomment-359879350
	//
	// This function will call FetchCandidateHead() of the SMC to obtain
	// more necessary information.
	//
	// This functions will fetch the transactions in the proposer tx pool and and apply
	// them to finish up the collation. It will then need to broadcast the
	// collation to the main chain using JSON-RPC.
	log.Info("Submit collation function called")
	return nil
}
