package notary

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
)

// SubscribeBlockHeaders checks incoming block headers and determines if
// we are an eligible notary for collations. Then, it finds the pending tx's
// from the running geth node and sorts them by descending order of gas price,
// eliminates those that ask for too much gas, and routes them over
// to the SMC to create a collation.
func subscribeBlockHeaders(client mainchain.Client) error {
	headerChan := make(chan *types.Header, 16)

	_, err := client.ChainReader().SubscribeNewHead(context.Background(), headerChan)
	if err != nil {
		return fmt.Errorf("unable to subscribe to incoming headers. %v", err)
	}

	log.Info("Listening for new headers...")

	for {
		// TODO: Error handling for getting disconnected from the client.
		head := <-headerChan
		// Query the current state to see if we are an eligible notary.
		log.Info(fmt.Sprintf("Received new header: %v", head.Number.String()))

		// Check if we are in the notary pool before checking if we are an eligible notary.
		v, err := isAccountInNotaryPool(client)
		if err != nil {
			return fmt.Errorf("unable to verify client in notary pool. %v", err)
		}

		if v {
			if err := checkSMCForNotary(client, head); err != nil {
				return fmt.Errorf("unable to watch shards. %v", err)
			}
		}
	}
}

// checkSMCForNotary checks if we are an eligible notary for
// collation for the available shards in the SMC. The function calls
// getEligibleNotary from the SMC and notary a collation if
// conditions are met.
func checkSMCForNotary(client mainchain.Client, head *types.Header) error {
	log.Info("Checking if we are an eligible collation notary for a shard...")
	for s := int64(0); s < sharding.ShardCount; s++ {
		// Checks if we are an eligible notary according to the SMC.
		addr, err := client.SMCCaller().GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(s))

		if err != nil {
			return err
		}

		// If output is non-empty and the addr == coinbase.
		if addr == client.Account().Address {
			log.Info(fmt.Sprintf("Selected as notary on shard: %d", s))
			err := submitCollation(s)
			if err != nil {
				return err
			}

			// If the account is selected as notary, submit collation.
			if addr == client.Account().Address {
				log.Info(fmt.Sprintf("Selected as notary on shard: %d", s))
				err := submitCollation(s)
				if err != nil {
					return fmt.Errorf("could not add collation. %v", err)
				}
			}
		}
	}

	return nil
}

// isAccountInNotaryPool checks if the user is in the notary pool because
// we can't guarantee our tx for deposit will be in the next block header we receive.
// The function calls IsNotaryDeposited from the SMC and returns true if
// the user is in the notary pool.
func isAccountInNotaryPool(client mainchain.Client) (bool, error) {
	account := client.Account()
	// Checks if our deposit has gone through according to the SMC.
	nreg, err := client.SMCCaller().NotaryRegistry(&bind.CallOpts{}, account.Address)
	if !nreg.Deposited && err != nil {
		log.Warn(fmt.Sprintf("Account %s not in notary pool.", account.Address.String()))
	}
	return nreg.Deposited, err
}

// submitCollation interacts with the SMC directly to add a collation header.
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

// joinNotaryPool checks if the deposit flag is true and the account is a
// notary in the SMC. If the account is not in the set, it will deposit ETH
// into contract.
func joinNotaryPool(client mainchain.Client) error {
	if !client.DepositFlag() {
		return errors.New("joinNotaryPool called when deposit flag was not set")
	}

	if b, err := isAccountInNotaryPool(client); b || err != nil {
		return err
	}

	log.Info("Joining notary pool")
	txOps, err := client.CreateTXOpts(sharding.NotaryDeposit)
	if err != nil {
		return fmt.Errorf("unable to initiate the deposit transaction: %v", err)
	}

	tx, err := client.SMCTransactor().RegisterNotary(txOps)
	if err != nil {
		return fmt.Errorf("unable to deposit eth and become a notary: %v", err)
	}
	log.Info(fmt.Sprintf("Deposited %dETH into contract with transaction hash: %s", new(big.Int).Div(sharding.NotaryDeposit, big.NewInt(params.Ether)), tx.Hash().String()))

	return nil
}
