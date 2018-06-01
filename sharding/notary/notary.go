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
	"github.com/ethereum/go-ethereum/sharding/node"
)

// SubscribeBlockHeaders checks incoming block headers and determines if
// we are an eligible notary for collations. Then, it finds the pending tx's
// from the running geth node and sorts them by descending order of gas price,
// eliminates those that ask for too much gas, and routes them over
// to the SMC to create a collation.
func subscribeBlockHeaders(n node.Node) error {
	headerChan := make(chan *types.Header, 16)

	_, err := n.ChainReader().SubscribeNewHead(context.Background(), headerChan)
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
		v, err := isAccountInNotaryPool(n)
		if err != nil {
			return fmt.Errorf("unable to verify client in notary pool. %v", err)
		}

		if v {
			if err := checkSMCForNotary(n, head); err != nil {
				return fmt.Errorf("unable to watch shards. %v", err)
			}
		} else {
		}

	}
}

// checkSMCForNotary checks if we are an eligible notary for
// collation for the available shards in the SMC. The function calls
// getEligibleNotary from the SMC and notary a collation if
// conditions are met.
func checkSMCForNotary(n node.Node, head *types.Header) error {
	log.Info("Checking if we are an eligible collation notary for a shard...")
	for s := int64(0); s < sharding.ShardCount; s++ {
		// Checks if we are an eligible notary according to the SMC.
		addr, err := n.SMCCaller().GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(s))

		if err != nil {
			return err
		}

		// If output is non-empty and the addr == coinbase.
		if addr == n.Account().Address {
			log.Info(fmt.Sprintf("Selected as notary on shard: %d", s))
			err := submitCollation(s)
			if err != nil {
				return err
			}

			// If the account is selected as notary, submit collation.
			if addr == n.Account().Address {
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

func getNotaryRegistry(n node.Node) (struct {
	DeregisteredPeriod *big.Int
	PoolIndex          *big.Int
	Balance            *big.Int
	Deposited          bool
}, error) {
	account := n.Account()

	nreg, err := n.SMCCaller().NotaryRegistry(&bind.CallOpts{}, account.Address)
	if err != nil {
		return nreg, fmt.Errorf("Unable to retrieve notary registry: %v", err)
	}

	return nreg, nil

}

// isAccountInNotaryPool checks if the user is in the notary pool because
// we can't guarantee our tx for deposit will be in the next block header we receive.
// The function calls IsNotaryDeposited from the SMC and returns true if
// the user is in the notary pool.
func isAccountInNotaryPool(n node.Node) (bool, error) {
	if nreg, err := getNotaryRegistry(n); err != nil {
		return false, err
	}

	if !nreg.Deposited {
		log.Warn(fmt.Sprintf("Account %s not in notary pool.", account.Address.String()))
	}
	return nreg.Deposited, err
}

func hasAccountBeenDeregistered(n node.Node) (bool, error) {
	account := n.Account()
	// Checks if we have been deregistered through the SMC
	nreg, err := n.SMCCaller().NotaryRegistry(&bind.CallOpts{}, account.Address)

	return nreg.DeregisteredPeriod != big.NewInt(0), err
}

func isLockUpOver(n node.Node) (bool, error) {
	block, err := n.ChainReader().BlockByNumber(context.Background(), nil)
	if err != nil {
		return false, fmt.Errorf("Unable to retrieve last block: %v", err)
	}
	account := n.Account()

	nreg, err := n.SMCCaller().NotaryRegistry(&bind.CallOpts{}, account.Address)

	if err != nil {
		return false, fmt.Errorf("Unable to retrieve notary registry: %v", err)
	}

	if !nreg.Deposited {
		return false, errors.New("Account has not deposited in Notary Pool")
	}

	if nreg.DeregisteredPeriod == big.NewInt(0) {

		return false, errors.New("Lock Up Period has not started")

	}

	return (block.Number().Int64() / sharding.PeriodLength) > nreg.DeregisteredPeriod.Int64()+sharding.NotaryLockupLength, nil

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
func joinNotaryPool(n node.Node) error {
	if !n.DepositFlagSet() {
		return errors.New("joinNotaryPool called when deposit flag was not set")
	}

	if b, err := isAccountInNotaryPool(n); b || err != nil {
		if b {

			return errors.New("Account has already deposited into the Notary Pool")
		}

		return err
	}

	log.Info("Joining notary pool")
	txOps, err := n.CreateTXOpts(sharding.NotaryDeposit)
	if err != nil {
		return fmt.Errorf("unable to intiate the deposit transaction: %v", err)
	}

	tx, err := n.SMCTransactor().RegisterNotary(txOps)
	if err != nil {
		return fmt.Errorf("unable to deposit eth and become a notary: %v", err)
	}
	log.Info(fmt.Sprintf("Deposited %dETH into contract with transaction hash: %s", new(big.Int).Div(sharding.NotaryDeposit, big.NewInt(params.Ether)), tx.Hash().String()))

	return nil
}

// leaveNotaryPool checks if the account is a Notary in  the SMC.
// if it is then it checks if it has deposited , if it has it deregisters
// itself from the pool and the lockup period counts down.
func leaveNotaryPool(n node.Node) error {

	account := n.Account()

	nreg, err := n.SMCCaller().NotaryRegistry(&bind.CallOpts{}, account.Address)

	if err != nil {
		return fmt.Errorf("Account %s not in notary pool", account.Address.String())
	}
	if !nreg.Deposited {
		return errors.New("Account has not deposited in the Notary Pool")
	}

	txOps, err := n.CreateTXOpts(nil)
	if err != nil {
		return fmt.Errorf("unable to create txOpts: %v", err)
	}
	tx, err := n.SMCTransactor().DeregisterNotary(txOps)

	if err != nil {
		return fmt.Errorf("Unable to deregister Notary: %v", err)
	}

	if dreg, err := hasAccountBeenDeregistered(n); dreg || err != nil {
		if !dreg {

			return errors.New("Notary unable to be Deregistered Succesfully from Pool")
		}

		return err
	}

	log.Info(fmt.Sprintf("Notary Deregistered from the pool with hash: %s", tx.Hash().String()))
	return nil

}

func releaseNotary(n node.Node) error {
	account := n.Account()

	nreg, err := n.SMCCaller().NotaryRegistry(&bind.CallOpts{}, account.Address)

	if err != nil {
		return fmt.Errorf("Account %s not in notary pool", account.Address.String())
	}
	if !nreg.Deposited {
		return errors.New("Account has not deposited in the Notary Pool")
	}

}

func VoteforCollation() error {
	return nil
}
