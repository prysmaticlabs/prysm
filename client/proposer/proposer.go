package proposer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/client/mainchain"
	"github.com/prysmaticlabs/prysm/client/types"
	"github.com/sirupsen/logrus"
)

// AddHeader adds the collation header to the main chain by sending
// an addHeader transaction to the sharding manager contract.
// There can only exist one header per period per shard, it is the proposer's
// responsibility to check if a header has been added.
func AddHeader(client mainchain.EthClient, transactor mainchain.ContractTransactor, collation *types.Collation) error {
	log.Info("Adding header to SMC")

	txOps, err := transactor.CreateTXOpts(big.NewInt(0))
	if err != nil {
		return fmt.Errorf("unable to initiate add header transaction: %v", err)
	}

	tx, err := transactor.SMCTransactor().AddHeader(txOps, collation.Header().ShardID(), collation.Header().Period(), [32]byte(*collation.Header().ChunkRoot()), collation.Header().Sig())
	if err != nil {
		return fmt.Errorf("unable to add header to SMC: %v", err)
	}

	// TODO: wait 5 mins for addHeader to be mined, hard coded duration for Ruby release, this will be changed with beacon chain.
	err = client.WaitForTransaction(context.Background(), tx.Hash(), 300)
	if err != nil {
		return err
	}

	receipt, err := client.TransactionReceipt(tx.Hash())
	if err != nil {
		return err
	}
	if receipt.Status != gethTypes.ReceiptStatusSuccessful {
		return fmt.Errorf("add header transaction failed with receipt status %v", receipt.Status)
	}

	log.WithFields(logrus.Fields{
		"txHash": tx.Hash().Hex(),
	}).Info("Add header submitted")
	return nil
}

// createCollation creates collation base struct with header
// and body. Header consists of shardID, ChunkRoot, period,
// proposer addr and signatures. Body contains serialized blob
// of a collations transactions.
func createCollation(caller mainchain.ContractCaller, account *accounts.Account, signer mainchain.Signer, shardID *big.Int, period *big.Int, txs []*gethTypes.Transaction) (*types.Collation, error) {
	// shardId has to be within range
	shardCount, err := caller.GetShardCount()
	if err != nil {
		return nil, fmt.Errorf("can't get shard count from smc: %v", err)
	}
	if shardID.Cmp(big.NewInt(0)) < 0 || shardID.Cmp(big.NewInt(shardCount)) > 0 {
		return nil, fmt.Errorf("can't create collation for shard %v. Must be between 0 and %v", shardID, shardCount)
	}

	// check with SMC to see if we can add the header.
	if a, _ := checkHeaderAdded(caller, shardID, period); !a {
		return nil, fmt.Errorf("can't create collation, collation with same period has already been added")
	}

	// serialized tx to blob for collation body.
	blobs, err := types.SerializeTxToBlob(txs)
	if err != nil {
		return nil, fmt.Errorf("can't create collation, serialization to blob failed: %v", err)
	}

	// construct the header, leave chunkRoot and signature fields empty, to be filled later.
	addr := account.Address
	header := types.NewCollationHeader(shardID, nil, period, &addr, [32]byte{})

	// construct the body with header, blobs(serialized txs) and txs.
	collation := types.NewCollation(header, blobs, txs)
	collation.CalculateChunkRoot()
	sig, err := signer.Sign(collation.Header().Hash())
	if err != nil {
		return nil, fmt.Errorf("can't create collation, sign collationHeader failed: %v", err)
	}

	// add proposer signature to collation header.
	var sig32 [32]byte
	copy(sig32[:], sig)
	collation.Header().AddSig(sig32)
	log.WithFields(logrus.Fields{
		"hash":    collation.Header().Hash().Hex(),
		"shardID": collation.Header().ShardID(),
		"period":  collation.Header().Period(),
	}).Info("Collation created")
	return collation, nil
}

// checkHeaderAdded checks if a collation header has already
// submitted to the main chain. There can only be one header per shard
// per period, proposer should check if a header's already submitted,
// checkHeaderAdded returns true if it is available, false if it is unavailable.
func checkHeaderAdded(caller mainchain.ContractCaller, shardID *big.Int, period *big.Int) (bool, error) {
	// Get the period of the last header.
	lastPeriod, err := caller.SMCCaller().LastSubmittedCollation(&bind.CallOpts{}, shardID)
	if err != nil {
		return false, fmt.Errorf("unable to get the period of last submitted collation: %v", err)
	}
	// True if current period is greater than last added period.
	return period.Cmp(lastPeriod) > 0, nil
}
