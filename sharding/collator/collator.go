package collator

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/client"
)

// SubscribeBlockHeaders checks incoming block headers and determines if
// we are an eligible collator for collations. Then, it finds the pending tx's
// from the running geth node and sorts them by descending order of gas price,
// eliminates those that ask for too much gas, and routes them over
// to the SMC to create a collation
func subscribeBlockHeaders(c client.Client) error {
	headerChan := make(chan *types.Header, 16)

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
		}

	}
}

// checkSMCForCollator checks if we are an eligible collator for
// collation for the available shards in the SMC. The function calls
// getEligibleCollator from the SMC and collator a collation if
// conditions are met
func checkSMCForCollator(c client.Client, head *types.Header) error {
	log.Info("Checking if we are an eligible collation collator for a shard...")
	period := big.NewInt(0).Div(head.Number, big.NewInt(sharding.PeriodLength))
	for s := int64(0); s < sharding.ShardCount; s++ {
		// Checks if we are an eligible collator according to the SMC
		addr, err := c.SMCCaller().GetEligibleCollator(&bind.CallOpts{}, big.NewInt(s), period)

		if err != nil {
			return err
		}

		// If output is non-empty and the addr == coinbase
		if addr == c.Account().Address {
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
func isAccountInCollatorPool(c client.Client) (bool, error) {
	account := c.Account()
	// Checks if our deposit has gone through according to the SMC
	b, err := c.SMCCaller().IsCollatorDeposited(&bind.CallOpts{}, account.Address)
	if !b && err != nil {
		log.Warn(fmt.Sprintf("Account %s not in collator pool.", account.Address.String()))
	}
	return b, err
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

// joinCollatorPool checks if the account is a collator in the SMC. If
// the account is not in the set, it will deposit 100ETH into contract.
func joinCollatorPool(c client.Client) error {

	log.Info("Joining collator pool")
	txOps, err := c.CreateTXOps(sharding.DepositSize)
	if err != nil {
		return fmt.Errorf("unable to intiate the deposit transaction: %v", err)
	}

	tx, err := c.SMCTransactor().Deposit(txOps)
	if err != nil {
		return fmt.Errorf("unable to deposit eth and become a collator: %v", err)
	}
	log.Info(fmt.Sprintf("Deposited %dETH into contract with transaction hash: %s", new(big.Int).Div(sharding.DepositSize, big.NewInt(params.Ether)), tx.Hash().String()))

	return nil
}
