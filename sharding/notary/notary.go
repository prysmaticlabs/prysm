package notary

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	shardparams "github.com/ethereum/go-ethereum/sharding/params"
)

// checkSMCForNotary checks if we are an eligible notary for
// collation for the available shards in the SMC. The function calls
// getEligibleNotary from the SMC and notary a collation if
// conditions are met.
func checkSMCForNotary(caller mainchain.ContractCaller, head *types.Header) error {
	log.Info("Checking if we are an eligible collation notary for a shard...")
	shardCount, err := caller.GetShardCount()
	if err != nil {
		return fmt.Errorf("can't get shard count from smc: %v", err)
	}
	for s := int64(0); s < shardCount; s++ {
		// Checks if we are an eligible notary according to the SMC.
		addr, err := caller.SMCCaller().GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(s))

		if err != nil {
			return err
		}

		// If output is non-empty and the addr == coinbase.
		if addr == caller.Account().Address {
			log.Info(fmt.Sprintf("Selected as notary on shard: %d", s))
			err := submitCollation(s)
			if err != nil {
				return err
			}

			// If the account is selected as notary, submit collation.
			if addr == caller.Account().Address {
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
func isAccountInNotaryPool(caller mainchain.ContractCaller) (bool, error) {
	account := caller.Account()
	// Checks if our deposit has gone through according to the SMC.
	nreg, err := caller.SMCCaller().NotaryRegistry(&bind.CallOpts{}, account.Address)
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
func joinNotaryPool(manager mainchain.ContractManager, config *shardparams.Config) error {
	if b, err := isAccountInNotaryPool(manager); b || err != nil {
		return err
	}

	log.Info("Joining notary pool")
	txOps, err := manager.CreateTXOpts(shardparams.DefaultConfig.NotaryDeposit)
	if err != nil {
		return fmt.Errorf("unable to initiate the deposit transaction: %v", err)
	}

	tx, err := manager.SMCTransactor().RegisterNotary(txOps)
	if err != nil {
		return fmt.Errorf("unable to deposit eth and become a notary: %v", err)
	}
	log.Info(fmt.Sprintf("Deposited %dETH into contract with transaction hash: %s", new(big.Int).Div(config.NotaryDeposit, big.NewInt(params.Ether)), tx.Hash().String()))

	return nil
}
