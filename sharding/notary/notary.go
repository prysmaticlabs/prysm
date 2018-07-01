package notary

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/contracts"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	shardparams "github.com/ethereum/go-ethereum/sharding/params"
)

// subscribeBlockHeaders checks incoming block headers and determines if
// we are an eligible notary for collations. Then, it finds the pending tx's
// from the running geth node and sorts them by descending order of gas price,
// eliminates those that ask for too much gas, and routes them over
// to the SMC to create a collation.
func subscribeBlockHeaders(reader mainchain.Reader, caller mainchain.ContractCaller, account *accounts.Account) error {
	headerChan := make(chan *types.Header, 16)

	_, err := reader.SubscribeNewHead(context.Background(), headerChan)
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
		v, err := isAccountInNotaryPool(caller, account)
		if err != nil {
			return fmt.Errorf("unable to verify client in notary pool. %v", err)
		}

		if v {
			if err := checkSMCForNotary(caller, account, head); err != nil {
				return fmt.Errorf("unable to watch shards. %v", err)
			}
		}
	}
}

// checkSMCForNotary checks if we are an eligible notary for
// collation for the available shards in the SMC. The function calls
// getEligibleNotary from the SMC and notary a collation if
// conditions are met.
func checkSMCForNotary(caller mainchain.ContractCaller, account *accounts.Account, head *types.Header) error {
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

		if addr == account.Address {
			log.Info(fmt.Sprintf("Selected as notary on shard: %d", s))
		}

	}

	return nil
}

// getNotaryRegistry retrieves the registry of the registered account.
func getNotaryRegistry(caller mainchain.ContractCaller, account *accounts.Account) (*contracts.Registry, error) {

	var nreg contracts.Registry
	nreg, err := caller.SMCCaller().NotaryRegistry(&bind.CallOpts{}, account.Address)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve notary registry: %v", err)
	}

	return &nreg, nil
}

// isAccountInNotaryPool checks if the user is in the notary pool because
// we can't guarantee our tx for deposit will be in the next block header we receive.
// The function calls IsNotaryDeposited from the SMC and returns true if
// the user is in the notary pool.
func isAccountInNotaryPool(caller mainchain.ContractCaller, account *accounts.Account) (bool, error) {

	nreg, err := getNotaryRegistry(caller, account)
	if err != nil {
		return false, err
	}

	if !nreg.Deposited {
		log.Warn(fmt.Sprintf("Account %s not in notary pool.", account.Address.Hex()))
	}

	return nreg.Deposited, nil
}

// hasAccountBeenDeregistered checks if the account has been deregistered from the notary pool.
func hasAccountBeenDeregistered(caller mainchain.ContractCaller, account *accounts.Account) (bool, error) {
	nreg, err := getNotaryRegistry(caller, account)

	if err != nil {
		return false, err
	}

	return nreg.DeregisteredPeriod.Cmp(big.NewInt(0)) == 1, err
}

// isLockUpOver checks if the lock up period is over
// which will allow the notary to call the releaseNotary function
// in the SMC and get their deposit back.
func isLockUpOver(caller mainchain.ContractCaller, reader mainchain.Reader, account *accounts.Account) (bool, error) {

	//TODO: When chainreader for tests is implemented, get block using the method
	//get BlockByNumber instead of passing as an argument to this function.
	nreg, err := getNotaryRegistry(caller, account)
	if err != nil {
		return false, err
	}
	block, err := reader.BlockByNumber(context.Background(), nil)
	if err != nil {
		return false, err
	}

	return (block.Number().Int64() / shardparams.DefaultConfig.PeriodLength) > nreg.DeregisteredPeriod.Int64()+shardparams.DefaultConfig.NotaryLockupLength, nil

}

func transactionWaiting(client mainchain.EthClient, tx *types.Transaction, duration time.Duration) error {

	err := client.WaitForTransaction(context.Background(), tx.Hash(), duration)
	if err != nil {
		return err
	}

	receipt, err := client.TransactionReceipt(tx.Hash())
	if err != nil {
		return err
	}

	if receipt.Status == types.ReceiptStatusFailed {
		return errors.New("transaction was not successful, unable to release Notary")
	}
	return nil

}

func settingCanonicalShardChain(shard sharding.Shard, manager mainchain.ContractManager, period *big.Int, headerHash *common.Hash) error {

	shardID := shard.ShardID()
	collationRecords, err := manager.SMCCaller().CollationRecords(&bind.CallOpts{}, shardID, period)

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

func getCurrentNetworkState(manager mainchain.ContractManager, shard sharding.Shard, reader mainchain.Reader) (int64, *big.Int, *types.Block, error) {

	shardcount, err := manager.GetShardCount()
	if err != nil {
		return 0, nil, nil, fmt.Errorf("could not get shard count: %v", err)
	}

	shardID := shard.ShardID()
	// checks if the shardID is valid
	if shardID.Int64() <= int64(0) || shardID.Int64() > shardcount {
		return 0, nil, nil, fmt.Errorf("shardId is invalid, it has to be between %d and %d, instead it is %v", 0, shardcount, shardID)
	}

	block, err := reader.BlockByNumber(context.Background(), nil)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("unable to retrieve block: %v", err)
	}

	return shardcount, shardID, block, nil

}

func checkCollationPeriod(manager mainchain.ContractManager, block *types.Block, shardID *big.Int) (*big.Int, *big.Int, error) {

	period := big.NewInt(0).Div(block.Number(), big.NewInt(shardparams.DefaultConfig.PeriodLength))
	collPeriod, err := manager.SMCCaller().LastSubmittedCollation(&bind.CallOpts{}, shardID)

	if err != nil {
		return nil, nil, fmt.Errorf("unable to get period from last submitted collation: %v", err)
	}

	// Checks if the current period is valid in order to vote for the collation on the shard
	if period.Int64() != collPeriod.Int64() {
		return nil, nil, fmt.Errorf("period in collation is not equal to current period : %d , %d", collPeriod, period)
	}
	return period, collPeriod, nil

}

func hasNotaryVoted(manager mainchain.ContractManager, shardID *big.Int, poolIndex *big.Int) (bool, error) {
	hasVoted, err := manager.SMCCaller().HasVoted(&bind.CallOpts{}, shardID, poolIndex)

	if err != nil {
		return false, fmt.Errorf("unable to know if notary voted: %v", err)
	}

	return hasVoted, nil
}

func verifyNotary(manager mainchain.ContractManager, client mainchain.EthClient) (*contracts.Registry, error) {
	nreg, err := getNotaryRegistry(manager, client.Account())

	if err != nil {
		return nil, err
	}

	if !nreg.Deposited {
		return nil, fmt.Errorf("notary has not deposited to the SMC")
	}

	// Checking if the pool index is valid
	if nreg.PoolIndex.Int64() >= shardparams.DefaultConfig.NotaryCommitteeSize {
		return nil, fmt.Errorf("invalid pool index %d as it is more than the committee size of %d", nreg.PoolIndex, shardparams.DefaultConfig.NotaryCommitteeSize)
	}

	return nreg, nil
}

// joinNotaryPool checks if the deposit flag is true and the account is a
// notary in the SMC. If the account is not in the set, it will deposit ETH
// into contract.
func joinNotaryPool(manager mainchain.ContractManager, client mainchain.EthClient, config *shardparams.Config) error {
	if !client.DepositFlag() {
		return errors.New("joinNotaryPool called when deposit flag was not set")
	}

	if b, err := isAccountInNotaryPool(manager, client.Account()); b || err != nil {
		if b {
			log.Info(fmt.Sprint("Already joined notary pool"))
			return nil
		}
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

	err = client.WaitForTransaction(context.Background(), tx.Hash(), 400)
	if err != nil {
		return err
	}

	receipt, err := client.TransactionReceipt(tx.Hash())
	if err != nil {
		return err
	}
	if receipt.Status == types.ReceiptStatusFailed {
		return errors.New("transaction was not successful, unable to deposit ETH and become a notary")
	}

	if inPool, err := isAccountInNotaryPool(manager, client.Account()); !inPool || err != nil {
		if err != nil {
			return err
		}
		return errors.New("account has not been able to be deposited in notary pool")
	}

	log.Info(fmt.Sprintf("Deposited %dETH into contract with transaction hash: %s", new(big.Int).Div(shardparams.DefaultConfig.NotaryDeposit, big.NewInt(params.Ether)), tx.Hash().Hex()))

	return nil
}

// leaveNotaryPool causes the notary to deregister and leave the notary pool
// by calling the DeregisterNotary function in the SMC.
func leaveNotaryPool(manager mainchain.ContractManager, client mainchain.EthClient) error {

	if inPool, err := isAccountInNotaryPool(manager, client.Account()); !inPool || err != nil {
		if err != nil {
			return err
		}
		return errors.New("account has not been able to be deposited in notary pool")
	}

	txOps, err := manager.CreateTXOpts(nil)
	if err != nil {
		return fmt.Errorf("unable to create txOpts: %v", err)
	}

	tx, err := manager.SMCTransactor().DeregisterNotary(txOps)
	if err != nil {
		return fmt.Errorf("unable to deregister notary: %v", err)
	}

	err = client.WaitForTransaction(context.Background(), tx.Hash(), 400)
	if err != nil {
		return err
	}

	receipt, err := client.TransactionReceipt(tx.Hash())
	if err != nil {
		return err
	}

	if receipt.Status == types.ReceiptStatusFailed {
		return errors.New("transaction was not successful, unable to deregister notary")
	}

	if dreg, err := hasAccountBeenDeregistered(manager, client.Account()); !dreg || err != nil {
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
func releaseNotary(manager mainchain.ContractManager, client mainchain.EthClient, reader mainchain.Reader) error {

	if dreg, err := hasAccountBeenDeregistered(manager, client.Account()); !dreg || err != nil {
		if err != nil {
			return err
		}
		return errors.New("notary has not been deregistered from the pool")
	}

	if lockup, err := isLockUpOver(manager, reader, client.Account()); !lockup || err != nil {
		if err != nil {
			return err
		}
		return errors.New("lockup period is not over")
	}

	txOps, err := manager.CreateTXOpts(nil)
	if err != nil {
		return fmt.Errorf("unable to create txOpts: %v", err)
	}

	tx, err := manager.SMCTransactor().ReleaseNotary(txOps)
	if err != nil {
		return fmt.Errorf("unable to Release Notary: %v", err)
	}

	err = transactionWaiting(client, tx, 400)
	if err != nil {
		return err
	}

	nreg, err := getNotaryRegistry(manager, client.Account())
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
func submitVote(shard sharding.Shard, manager mainchain.ContractManager, client mainchain.EthClient, reader mainchain.Reader, headerHash *common.Hash) error {

	_, shardID, block, err := getCurrentNetworkState(manager, shard, reader)
	if err != nil {
		return err
	}

	period, _, err := checkCollationPeriod(manager, block, shardID)
	if err != nil {
		return err
	}

	nreg, err := verifyNotary(manager, client)

	if err != nil {
		return err
	}

	collationRecords, err := manager.SMCCaller().CollationRecords(&bind.CallOpts{}, shardID, period)
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

	hasVoted, err := hasNotaryVoted(manager, shardID, nreg.PoolIndex)
	if err != nil {
		return err
	}
	if hasVoted {
		return errors.New("notary has already voted")
	}

	inCommitee, err := manager.SMCCaller().GetNotaryInCommittee(&bind.CallOpts{}, shardID)
	if err != nil {
		return fmt.Errorf("unable to know if notary is in committee: %v", err)
	}

	if inCommitee != client.Account().Address {
		return errors.New("notary is not eligible to vote in this shard at the current period")
	}

	txOps, err := manager.CreateTXOpts(nil)
	if err != nil {
		return fmt.Errorf("unable to create txOpts: %v", err)
	}

	tx, err := manager.SMCTransactor().SubmitVote(txOps, shardID, period, nreg.PoolIndex, collationRecords.ChunkRoot)
	if err != nil {

		return fmt.Errorf("unable to submit Vote: %v", err)
	}

	err = transactionWaiting(client, tx, 400)

	if err != nil {
		return err
	}

	hasVoted, err = hasNotaryVoted(manager, shardID, nreg.PoolIndex)

	if err != nil {
		return fmt.Errorf("unable to know if notary voted: %v", err)
	}

	if !hasVoted {
		return errors.New("notary has not voted")
	}

	log.Info(fmt.Sprintf("Notary has voted for shard: %v in the %v period", shardID, period))

	err = settingCanonicalShardChain(shard, manager, period, headerHash)

	return err
}

func RequestCollation(shard sharding.Shard, Proposer *common.Address, Period *big.Int, ChunkRoot *common.Hash) (*sharding.Collation, error) {

}

func CalculatePOCAndVote(c *sharding.Collation, s *sharding.Shard) error {

}

func AddCanonicalCollation(s *sharding.Shard) error {

}

func CollationStore(c *sharding.Collation, s *sharding.Shard) error {

}
