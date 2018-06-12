package notary

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/contracts"
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

		if addr == client.Account().Address {
			log.Info(fmt.Sprintf("Selected as notary on shard: %d", s))
		}

	}

	return nil
}

// getNotaryRegistry retrieves the registry of the registered account.
func getNotaryRegistry(client mainchain.Client) (*contracts.Registry, error) {

	var nreg contracts.Registry

	account := client.Account()

	nreg, err := client.SMCCaller().NotaryRegistry(&bind.CallOpts{}, account.Address)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve notary registry: %v", err)
	}

	return &nreg, nil

}

// isAccountInNotaryPool checks if the user is in the notary pool because
// we can't guarantee our tx for deposit will be in the next block header we receive.
// The function calls IsNotaryDeposited from the SMC and returns true if
// the user is in the notary pool.
func isAccountInNotaryPool(client mainchain.Client) (bool, error) {

	nreg, err := getNotaryRegistry(client)
	if err != nil {
		return false, err
	}

	if !nreg.Deposited {
		log.Warn(fmt.Sprintf("Account %s not in notary pool.", client.Account().Address.Hex()))
	}

	return nreg.Deposited, nil
}

// hasAccountBeenDeregistered checks if the account has been deregistered from the notary pool.
func hasAccountBeenDeregistered(client mainchain.Client) (bool, error) {
	nreg, err := getNotaryRegistry(client)

	if err != nil {
		return false, err
	}

	return nreg.DeregisteredPeriod.Cmp(big.NewInt(0)) == 0, err
}

// isLockUpOver checks if the lock up period is over
// which will allow the notary to call the releaseNotary function
// in the SMC and get their deposit back.
func isLockUpOver(client mainchain.Client) (bool, error) {
	block, err := client.ChainReader().BlockByNumber(context.Background(), nil)
	if err != nil {
		return false, fmt.Errorf("unable to retrieve last block: %v", err)
	}
	nreg, err := getNotaryRegistry(client)
	if err != nil {
		return false, err
	}

	return (block.Number().Int64() / sharding.PeriodLength) > nreg.DeregisteredPeriod.Int64()+sharding.NotaryLockupLength, nil

}

// joinNotaryPool checks if the deposit flag is true and the account is a
// notary in the SMC. If the account is not in the set, it will deposit ETH
// into contract.
func joinNotaryPool(client mainchain.Client) error {
	if !client.DepositFlag() {
		return errors.New("joinNotaryPool called when deposit flag was not set")
	}

	if b, err := isAccountInNotaryPool(client); b || err != nil {
		if b {

			log.Info(fmt.Sprint("Already joined notary pool"))
			return nil

		}
		return err
	}

	log.Info("Joining notary pool")
	txOps, err := client.CreateTXOpts(sharding.NotaryDeposit)
	if err != nil {
		return fmt.Errorf("unable to intiate the deposit transaction: %v", err)
	}

	tx, err := client.SMCTransactor().RegisterNotary(txOps)
	if err != nil {
		return fmt.Errorf("unable to deposit eth and become a notary: %v", err)
	}
	receipt, err := client.TransactionReceipt(tx.Hash())
	if err != nil {
		return err
	}
	if receipt.Status == types.ReceiptStatusFailed {

		return errors.New("transaction was not successful, unable to deposit ETH and become a notary")

	}

	if inPool, err := isAccountInNotaryPool(client); !inPool || err != nil {
		if err != nil {
			return err
		}
		return errors.New("account has not been able to be deposited in notary pool")
	}

	log.Info(fmt.Sprintf("Deposited %dETH into contract with transaction hash: %s", new(big.Int).Div(sharding.NotaryDeposit, big.NewInt(params.Ether)), tx.Hash().Hex()))

	return nil
}

// leaveNotaryPool causes the notary to deregister and leave the notary pool
// by calling the DeregisterNotary function in the SMC.
func leaveNotaryPool(client mainchain.Client) error {

	if inPool, err := isAccountInNotaryPool(client); !inPool || err != nil {
		if err != nil {
			return err
		}
		return errors.New("account has not been able to be deposited in notary pool")
	}

	txOps, err := client.CreateTXOpts(nil)
	if err != nil {
		return fmt.Errorf("unable to create txOpts: %v", err)
	}
	tx, err := client.SMCTransactor().DeregisterNotary(txOps)

	if err != nil {
		return fmt.Errorf("unable to deregister notary: %v", err)
	}

	receipt, err := client.TransactionReceipt(tx.Hash())
	if err != nil {
		return err
	}

	if receipt.Status == types.ReceiptStatusFailed {
		return errors.New("transaction was not successful, unable to deregister notary")
	}

	if dreg, err := hasAccountBeenDeregistered(client); !dreg || err != nil {
		if err != nil {
			return err
		}
		return errors.New("notary unable to be deregistered successfully from pool")
	}

	log.Info(fmt.Sprintf("Notary deregistered from the pool with hash: %s", tx.Hash().Hex()))
	return nil

}

// releaseNotary releases the Notary from the pool by deleting the notary from
// the registry and transferring back the deposit
func releaseNotary(client mainchain.Client) error {
	nreg, err := getNotaryRegistry(client)
	if err != nil {
		return err
	}
	if !nreg.Deposited {
		return errors.New("account has not deposited in the Notary Pool")
	}

	if nreg.DeregisteredPeriod.Cmp(big.NewInt(0)) == 0 {

		return errors.New("notary is still registered to the pool")

	}

	if lockup, err := isLockUpOver(client); !lockup || err != nil {
		if err != nil {
			return err
		}
		return errors.New("lockup period is not over")
	}

	txOps, err := client.CreateTXOpts(nil)
	if err != nil {
		return fmt.Errorf("unable to create txOpts: %v", err)
	}
	tx, err := client.SMCTransactor().ReleaseNotary(txOps)

	if err != nil {
		return fmt.Errorf("unable to Release Notary: %v", err)
	}

	receipt, err := client.TransactionReceipt(tx.Hash())
	if err != nil {
		return err
	}

	if receipt.Status == types.ReceiptStatusFailed {

		return errors.New("transaction was not successful, unable to release Notary")

	}

	nreg, err = getNotaryRegistry(client)

	if err != nil {

		return err

	}

	if nreg.Deposited || nreg.Balance.Cmp(big.NewInt(0)) != 0 || nreg.DeregisteredPeriod.Cmp(big.NewInt(0)) != 0 || nreg.PoolIndex.Cmp(big.NewInt(0)) != 0 {

		return errors.New("notary unable to be released from the pool")

	}

	log.Info(fmt.Sprintf("notary with address: %s released from pool", client.Account().Address.Hex()))

	return nil

}

// submitVote votes for a collation on the shard
// by taking in the shard and the hash of the collation header
func submitVote(shard sharding.Shard, client mainchain.Client, headerHash *common.Hash) error {

	shardID := shard.ShardID()
	// checks if the shardID is valid
	if shardID.Int64() <= int64(0) || shardID.Int64() > sharding.ShardCount {
		return fmt.Errorf("shardId is invalid, it has to be between %d and %d, instead it is %v", 0, sharding.ShardCount, shardID)
	}

	currentblock, err := client.ChainReader().BlockByNumber(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("unable to retrieve last block: %v", err)
	}

	period := big.NewInt(0).Div(currentblock.Number(), big.NewInt(sharding.PeriodLength))

	collPeriod, err := client.SMCCaller().LastSubmittedCollation(&bind.CallOpts{}, shardID)

	if err != nil {
		return fmt.Errorf("unable to get period from last submitted collation: %v", err)
	}

	// Checks if the current period is valid in order to vote for the collation on the shard
	if period.Int64() != collPeriod.Int64() {
		return fmt.Errorf("period in collation is not equal to current period : %d , %d", collPeriod, period)
	}

	nreg, err := getNotaryRegistry(client)

	if err != nil {
		return err
	}

	if !nreg.Deposited {
		return fmt.Errorf("notary has not deposited to the SMC")
	}

	// Checking if the pool index is valid
	if nreg.PoolIndex.Int64() >= sharding.NotaryCommitSize {
		return fmt.Errorf("invalid pool index %d as it is more than the committee size of %d", nreg.PoolIndex, sharding.NotaryCommitSize)
	}

	collationRecords, err := client.SMCCaller().CollationRecords(&bind.CallOpts{}, shardID, period)

	if err != nil {
		return fmt.Errorf("unable to get collation record: %v", err)
	}

	chunkroot, err := shard.ChunkRootfromHeaderHash(headerHash)

	if err != nil {
		return fmt.Errorf("unable to get chunk root: %v", err)
	}

	// Checking if the chunkroot is valid
	if !bytes.Equal(collationRecords.ChunkRoot[:], chunkroot.Bytes()) {
		return fmt.Errorf("submmitted collation header has a different chunkroot to the one saved in the SMC")

	}

	hasVoted, err := client.SMCCaller().HasVoted(&bind.CallOpts{}, shardID, nreg.PoolIndex)

	if err != nil {
		return fmt.Errorf("unable to know if notary voted: %v", err)
	}

	if hasVoted {
		return errors.New("notary has already voted")
	}

	inCommitee, err := client.SMCCaller().GetNotaryInCommittee(&bind.CallOpts{}, shardID)

	if err != nil {
		return fmt.Errorf("unable to know if notary is in committee: %v", err)
	}

	if inCommitee != client.Account().Address {
		return errors.New("notary is not eligible to vote in this shard at the current period")
	}

	txOps, err := client.CreateTXOpts(nil)
	if err != nil {
		return fmt.Errorf("unable to create txOpts: %v", err)
	}

	tx, err := client.SMCTransactor().SubmitVote(txOps, shardID, period, nreg.PoolIndex, collationRecords.ChunkRoot)

	if err != nil {

		return fmt.Errorf("unable to submit Vote: %v", err)
	}

	receipt, err := client.TransactionReceipt(tx.Hash())
	if err != nil {
		return err
	}

	if receipt.Status == types.ReceiptStatusFailed {

		return errors.New("transaction was not successful, unable to vote for collation")

	}

	hasVoted, err = client.SMCCaller().HasVoted(&bind.CallOpts{}, shardID, nreg.PoolIndex)

	if err != nil {
		return fmt.Errorf("unable to know if notary voted: %v", err)
	}

	if !hasVoted {
		return errors.New("notary has not voted")
	}

	log.Info(fmt.Sprintf("Notary has voted for shard: %v in the %v period", shardID, period))

	collationRecords, err = client.SMCCaller().CollationRecords(&bind.CallOpts{}, shardID, period)

	if err != nil {
		return fmt.Errorf("unable to get collation record: %v", err)
	}

	// Logs if quorum has been reached and collation is added to the canonical shard chain
	if collationRecords.IsElected {

		log.Info(fmt.Sprintf(
			"Shard %v in period %v has chosen the collation with its header hash %v to be added to the canonical shard chain",
			shardID, period, headerHash))

		// Setting collation header as canonical in the shard chain
		header, err := shard.HeaderByHash(headerHash)

		if err != nil {
			return fmt.Errorf("unable to set Header from hash: %v", err)
		}

		err = shard.SetCanonical(header)

		if err != nil {
			return fmt.Errorf("unable to add collation to canonical shard chain: %v", err)
		}

	}

	return nil

}
