package sharding

import (
	"context"
	"fmt"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/contracts"
)

type collatorClient interface {
	Account() (*accounts.Account, error)
	ChainReader() ethereum.ChainReader
	VMCCaller() *contracts.VMCCaller
}

// SubscribeBlockHeaders checks incoming block headers and determines if
// we are an eligible proposer for collations. Then, it finds the pending tx's
// from the running geth node and sorts them by descending order of gas price,
// eliminates those that ask for too much gas, and routes them over
// to the VMC to create a collation
func subscribeBlockHeaders(c collatorClient) error {
	headerChan := make(chan *types.Header, 16)

	account, err := c.Account()
	if err != nil {
		return err
	}

	_, err = c.ChainReader().SubscribeNewHead(context.Background(), headerChan)
	if err != nil {
		return fmt.Errorf("unable to subscribe to incoming headers. %v", err)
	}

	log.Info("Listening for new headers...")

	for {
		// TODO: Error handling for getting disconnected from the client
		head := <-headerChan
		// Query the current state to see if we are an eligible proposer
		log.Info(fmt.Sprintf("Received new header: %v", head.Number.String()))
		// TODO: Only run this code on certain periods?

		// Check if we are in the validator pool before checking if we are an eligible proposer
		v, err := isAccountInValidatorSet(c)
		if err != nil {
			return fmt.Errorf("unable to verify client in validator pool. %v", err)
		}

		if v {
			if err := checkShardsForProposal(c, head); err != nil {
				return fmt.Errorf("unable to watch shards. %v", err)
			}
		} else {
			log.Warn(fmt.Sprintf("Account %s not in validator pool.", account.Address.String()))
		}

	}
}

// checkShardsForProposal checks if we are an eligible proposer for
// collation for the available shards in the VMC. The function calls
// getEligibleProposer from the VMC and proposes a collation if
// conditions are met
func checkShardsForProposal(c collatorClient, head *types.Header) error {
	account, err := c.Account()
	if err != nil {
		return err
	}

	log.Info("Checking if we are an eligible collation proposer for a shard...")
	period := big.NewInt(0).Div(head.Number, big.NewInt(periodLength))
	for s := int64(0); s < shardCount; s++ {
		// Checks if we are an eligible proposer according to the VMC
		addr, err := c.VMCCaller().GetEligibleProposer(&bind.CallOpts{}, big.NewInt(s), period)

		// TODO: When we are not a proposer, we get the error of being unable to
		// unmarshal empty output. Open issue to deal with this.
		if err != nil {
			return err
		}

		// If output is non-empty and the addr == coinbase
		if addr == account.Address {
			log.Info(fmt.Sprintf("Selected as collator on shard: %d", s))
			err := proposeCollation(s)
			if err != nil {
				return fmt.Errorf("could not propose collation. %v", err)
			}
		}
	}

	return nil
}

// isAccountInValidatorSet checks if the client is in the validator pool because
// we can't guarantee our tx for deposit will be in the next block header we receive.
// The function calls IsValidatorDeposited from the VMC and returns true if
// the client is in the validator pool
func isAccountInValidatorSet(c collatorClient) (bool, error) {
	account, err := c.Account()
	if err != nil {
		return false, err
	}

	// Checks if our deposit has gone through according to the VMC
	b, err := c.VMCCaller().IsValidatorDeposited(&bind.CallOpts{}, account.Address)
	if err != nil {
		return false, err
	}

	return b, nil
}

// proposeCollation interacts with the VMC directly to add a collation header
func proposeCollation(shardID int64) error {
	// TODO: Adds a collation header to the VMC with the following fields:
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
	// This function will call FetchCandidateHead() of the VMC to obtain
	// more necessary information.
	//
	// This functions will fetch the transactions in the txpool and and apply
	// them to finish up the collation. It will then need to broadcast the
	// collation to the main chain using JSON-RPC.
	log.Info("Propose collation function called")
	return nil
}
