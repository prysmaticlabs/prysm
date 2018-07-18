package attester

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/prysmaticlabs/geth-sharding/sharding/contracts"
	"github.com/prysmaticlabs/geth-sharding/sharding/mainchain"
	shardparams "github.com/prysmaticlabs/geth-sharding/sharding/params"
	log "github.com/sirupsen/logrus"
)

// subscribeBlockHeaders checks incoming block headers and determines if
// we are an eligible attester for collations. Then, it finds the pending tx's
// from the running geth node and sorts them by descending order of gas price,
// eliminates those that ask for too much gas, and routes them over
// to the SMC to create a collation.
func subscribeBlockHeaders(reader mainchain.Reader, caller mainchain.ContractCaller, account *accounts.Account) error {
	headerChan := make(chan *gethTypes.Header, 16)

	_, err := reader.SubscribeNewHead(context.Background(), headerChan)
	if err != nil {
		return fmt.Errorf("unable to subscribe to incoming headers. %v", err)
	}

	log.Info("Listening for new headers...")

	for {
		// TODO: Error handling for getting disconnected from the client.
		head := <-headerChan
		// Query the current state to see if we are an eligible attester.
		log.Infof("Received new header: %v", head.Number.String())

		// Check if we are in the attester pool before checking if we are an eligible attester.
		v, err := isAccountInAttesterPool(caller, account)
		if err != nil {
			return fmt.Errorf("unable to verify client in attester pool. %v", err)
		}

		if v {
<<<<<<< HEAD:sharding/attester/attester.go
			if err := checkSMCForAttester(caller, account, head); err != nil {
=======
			if err := checkSMCForNotary(caller, account); err != nil {
>>>>>>> f2f8850cccf5ff3498aebbce71baa05267bc07cc:sharding/notary/notary.go
				return fmt.Errorf("unable to watch shards. %v", err)
			}
		}
	}
}

// checkSMCForAttester checks if we are an eligible attester for
// collation for the available shards in the SMC. The function calls
// getEligibleAttester from the SMC and attester a collation if
// conditions are met.
<<<<<<< HEAD:sharding/attester/attester.go
func checkSMCForAttester(caller mainchain.ContractCaller, account *accounts.Account, head *gethTypes.Header) error {
	log.Info("Checking if we are an eligible collation attester for a shard...")
=======
func checkSMCForNotary(caller mainchain.ContractCaller, account *accounts.Account) error {
	log.Info("Checking if we are an eligible collation notary for a shard...")
>>>>>>> f2f8850cccf5ff3498aebbce71baa05267bc07cc:sharding/notary/notary.go
	shardCount, err := caller.GetShardCount()
	if err != nil {
		return fmt.Errorf("can't get shard count from smc: %v", err)
	}
	for s := int64(0); s < shardCount; s++ {
		// Checks if we are an eligible attester according to the SMC.
		addr, err := caller.SMCCaller().GetAttesterInCommittee(&bind.CallOpts{}, big.NewInt(s))

		if err != nil {
			return err
		}

		if addr == account.Address {
			log.Infof("Selected as attester on shard: %d", s)
		}

	}

	return nil
}

// getAttesterRegistry retrieves the registry of the registered account.
func getAttesterRegistry(caller mainchain.ContractCaller, account *accounts.Account) (*contracts.Registry, error) {

	var nreg contracts.Registry
	nreg, err := caller.SMCCaller().AttesterRegistry(&bind.CallOpts{}, account.Address)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve attester registry: %v", err)
	}

	return &nreg, nil
}

// isAccountInAttesterPool checks if the user is in the attester pool because
// we can't guarantee our tx for deposit will be in the next block header we receive.
// The function calls IsAttesterDeposited from the SMC and returns true if
// the user is in the attester pool.
func isAccountInAttesterPool(caller mainchain.ContractCaller, account *accounts.Account) (bool, error) {

	nreg, err := getAttesterRegistry(caller, account)
	if err != nil {
		return false, err
	}

	if !nreg.Deposited {
		log.Warnf("Account %s not in attester pool.", account.Address.Hex())
	}

	return nreg.Deposited, nil
}

<<<<<<< HEAD:sharding/attester/attester.go
// hasAccountBeenDeregistered checks if the account has been deregistered from the attester pool.
func hasAccountBeenDeregistered(caller mainchain.ContractCaller, account *accounts.Account) (bool, error) {
	nreg, err := getAttesterRegistry(caller, account)

	if err != nil {
		return false, err
	}

	return nreg.DeregisteredPeriod.Cmp(big.NewInt(0)) == 1, err
}

// isLockUpOver checks if the lock up period is over
// which will allow the attester to call the releaseAttester function
// in the SMC and get their deposit back.
func isLockUpOver(caller mainchain.ContractCaller, reader mainchain.Reader, account *accounts.Account) (bool, error) {

	//TODO: When chainreader for tests is implemented, get block using the method
	//get BlockByNumber instead of passing as an argument to this function.
	nreg, err := getAttesterRegistry(caller, account)
	if err != nil {
		return false, err
	}
	block, err := reader.BlockByNumber(context.Background(), nil)
	if err != nil {
		return false, err
	}

	return (block.Number().Int64() / shardparams.DefaultConfig.PeriodLength) > nreg.DeregisteredPeriod.Int64()+shardparams.DefaultConfig.AttesterLockupLength, nil

}

func transactionWaiting(client mainchain.EthClient, tx *gethTypes.Transaction, duration time.Duration) error {

	err := client.WaitForTransaction(context.Background(), tx.Hash(), duration)
	if err != nil {
		return err
	}

	receipt, err := client.TransactionReceipt(tx.Hash())
	if err != nil {
		return err
	}

	if receipt.Status == gethTypes.ReceiptStatusFailed {
		return errors.New("transaction was not successful, unable to release Attester")
	}
	return nil

}

func settingCanonicalShardChain(shard types.Shard, manager mainchain.ContractManager, period *big.Int, headerHash *common.Hash) error {

	shardID := shard.ShardID()
	collationRecords, err := manager.SMCCaller().CollationRecords(&bind.CallOpts{}, shardID, period)

	if err != nil {
		return fmt.Errorf("unable to get collation record: %v", err)
	}

	// Logs if quorum has been reached and collation is added to the canonical shard chain
	if collationRecords.IsElected {
		log.Infof("Shard %v in period %v has chosen the collation with its header hash %v to be added to the canonical shard chain",
			shardID, period, headerHash)

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

func getCurrentNetworkState(manager mainchain.ContractManager, shard types.Shard, reader mainchain.Reader) (int64, *big.Int, *gethTypes.Block, error) {

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

func checkCollationPeriod(manager mainchain.ContractManager, block *gethTypes.Block, shardID *big.Int) (*big.Int, *big.Int, error) {

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

func hasAttesterVoted(manager mainchain.ContractManager, shardID *big.Int, poolIndex *big.Int) (bool, error) {
	hasVoted, err := manager.SMCCaller().HasVoted(&bind.CallOpts{}, shardID, poolIndex)

	if err != nil {
		return false, fmt.Errorf("unable to know if attester voted: %v", err)
	}

	return hasVoted, nil
}

func verifyAttester(manager mainchain.ContractManager, client mainchain.EthClient) (*contracts.Registry, error) {
	nreg, err := getAttesterRegistry(manager, client.Account())

	if err != nil {
		return nil, err
	}

	if !nreg.Deposited {
		return nil, fmt.Errorf("attester has not deposited to the SMC")
	}

	// Checking if the pool index is valid
	if nreg.PoolIndex.Int64() >= shardparams.DefaultConfig.AttesterCommitteeSize {
		return nil, fmt.Errorf("invalid pool index %d as it is more than the committee size of %d", nreg.PoolIndex, shardparams.DefaultConfig.AttesterCommitteeSize)
	}

	return nreg, nil
}

// joinAttesterPool checks if the deposit flag is true and the account is a
// attester in the SMC. If the account is not in the set, it will deposit ETH
// into contract.
func joinAttesterPool(manager mainchain.ContractManager, client mainchain.EthClient, config *shardparams.Config) error {
=======
// joinNotaryPool checks if the deposit flag is true and the account is a
// notary in the SMC. If the account is not in the set, it will deposit ETH
// into contract.
func joinNotaryPool(manager mainchain.ContractManager, client mainchain.EthClient) error {
>>>>>>> f2f8850cccf5ff3498aebbce71baa05267bc07cc:sharding/notary/notary.go
	if !client.DepositFlag() {
		return errors.New("joinAttesterPool called when deposit flag was not set")
	}

	if b, err := isAccountInAttesterPool(manager, client.Account()); b || err != nil {
		if b {
			log.Info("Already joined attester pool")
			return nil
		}
		return err
	}

	log.Info("Joining attester pool")
	txOps, err := manager.CreateTXOpts(shardparams.DefaultConfig.AttesterDeposit)
	if err != nil {
		return fmt.Errorf("unable to initiate the deposit transaction: %v", err)
	}

	tx, err := manager.SMCTransactor().RegisterAttester(txOps)
	if err != nil {
		return fmt.Errorf("unable to deposit eth and become an attester: %v", err)
	}

	err = client.WaitForTransaction(context.Background(), tx.Hash(), 400)
	if err != nil {
		return err
	}

	receipt, err := client.TransactionReceipt(tx.Hash())
	if err != nil {
		return err
	}
	if receipt.Status == gethTypes.ReceiptStatusFailed {
		return errors.New("transaction was not successful, unable to deposit ETH and become an attester")
	}

	if inPool, err := isAccountInAttesterPool(manager, client.Account()); !inPool || err != nil {
		if err != nil {
			return err
		}
		return errors.New("account has not been able to be deposited in attester pool")
	}

	log.Infof("Deposited %dETH into contract with transaction hash: %s", new(big.Int).Div(shardparams.DefaultConfig.AttesterDeposit, big.NewInt(params.Ether)), tx.Hash().Hex())

	return nil
}
<<<<<<< HEAD:sharding/attester/attester.go

// leaveAttesterPool causes the attester to deregister and leave the attester pool
// by calling the DeregisterAttester function in the SMC.
func leaveAttesterPool(manager mainchain.ContractManager, client mainchain.EthClient) error {

	if inPool, err := isAccountInAttesterPool(manager, client.Account()); !inPool || err != nil {
		if err != nil {
			return err
		}
		return errors.New("account has not been able to be deposited in attester pool")
	}

	txOps, err := manager.CreateTXOpts(nil)
	if err != nil {
		return fmt.Errorf("unable to create txOpts: %v", err)
	}

	tx, err := manager.SMCTransactor().DeregisterAttester(txOps)
	if err != nil {
		return fmt.Errorf("unable to deregister attester: %v", err)
	}

	err = client.WaitForTransaction(context.Background(), tx.Hash(), 400)
	if err != nil {
		return err
	}

	receipt, err := client.TransactionReceipt(tx.Hash())
	if err != nil {
		return err
	}

	if receipt.Status == gethTypes.ReceiptStatusFailed {
		return errors.New("transaction was not successful, unable to deregister attester")
	}

	if dreg, err := hasAccountBeenDeregistered(manager, client.Account()); !dreg || err != nil {
		if err != nil {
			return err
		}
		return errors.New("attester unable to be deregistered successfully from pool")
	}

	log.Infof("Attester deregistered from the pool with hash: %s", tx.Hash().Hex())
	return nil

}

// releaseAttester releases the Attester from the pool by deleting the attester from
// the registry and transferring back the deposit
func releaseAttester(manager mainchain.ContractManager, client mainchain.EthClient, reader mainchain.Reader) error {

	if dreg, err := hasAccountBeenDeregistered(manager, client.Account()); !dreg || err != nil {
		if err != nil {
			return err
		}
		return errors.New("attester has not been deregistered from the pool")
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

	tx, err := manager.SMCTransactor().ReleaseAttester(txOps)
	if err != nil {
		return fmt.Errorf("unable to Release Attester: %v", err)
	}

	err = transactionWaiting(client, tx, 400)
	if err != nil {
		return err
	}

	nreg, err := getAttesterRegistry(manager, client.Account())
	if err != nil {
		return err
	}

	if nreg.Deposited || nreg.Balance.Cmp(big.NewInt(0)) != 0 || nreg.DeregisteredPeriod.Cmp(big.NewInt(0)) != 0 || nreg.PoolIndex.Cmp(big.NewInt(0)) != 0 {
		return errors.New("attester unable to be released from the pool")
	}

	log.Infof("Attester with address: %s released from pool", client.Account().Address.Hex())

	return nil

}

// submitVote votes for a collation on the shard
// by taking in the shard and the hash of the collation header
func submitVote(shard types.Shard, manager mainchain.ContractManager, client mainchain.EthClient, reader mainchain.Reader, headerHash *common.Hash) error {

	_, shardID, block, err := getCurrentNetworkState(manager, shard, reader)
	if err != nil {
		return err
	}

	period, _, err := checkCollationPeriod(manager, block, shardID)
	if err != nil {
		return err
	}

	nreg, err := verifyAttester(manager, client)

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

	hasVoted, err := hasAttesterVoted(manager, shardID, nreg.PoolIndex)
	if err != nil {
		return err
	}
	if hasVoted {
		return errors.New("attester has already voted")
	}

	inCommitee, err := manager.SMCCaller().GetAttesterInCommittee(&bind.CallOpts{}, shardID)
	if err != nil {
		return fmt.Errorf("unable to know if attester is in committee: %v", err)
	}

	if inCommitee != client.Account().Address {
		return errors.New("attester is not eligible to vote in this shard at the current period")
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

	hasVoted, err = hasAttesterVoted(manager, shardID, nreg.PoolIndex)

	if err != nil {
		return fmt.Errorf("unable to know if attester voted: %v", err)
	}

	if !hasVoted {
		return errors.New("attester has not voted")
	}

	log.Infof("Attester has voted for shard: %v in the %v period", shardID, period)

	err = settingCanonicalShardChain(shard, manager, period, headerHash)

	return err
}
=======
>>>>>>> f2f8850cccf5ff3498aebbce71baa05267bc07cc:sharding/notary/notary.go
