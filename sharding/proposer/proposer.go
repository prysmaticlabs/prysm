package proposer

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/node"
)

// createCollation creates collation base struct with header
// and body. Header consists of shardID, ChunkRoot, period,
// proposer addr and signatures. Body contains serialized blob
// of a collations transactions.
func createCollation(n node.Node, shardId *big.Int, period *big.Int, txs []*types.Transaction) (*sharding.Collation, error) {
	addr := n.Account().Address

	// serialized tx to blob for collation body
	blobs, err := sharding.SerializeTxToBlob(txs)
	if err != nil {
		return nil, fmt.Errorf("Can not create collation. Serialization to blob failed: %v", err)
	}

	// construct the header, leave chunkRoot and signature fields empty, to be filled later.
	header := sharding.NewCollationHeader(shardId, nil, period, &addr, nil)

	// construct the body with header, blobs(serialized txs) and txs.
	collation := sharding.NewCollation(header, blobs, txs)
	collation.CalculateChunkRoot()
	sig, err := n.Sign(collation.Header().Hash())
	if err != nil {
		return nil, fmt.Errorf("Can not create collation. Sign collationHeader failed: %v", err)
	}

	// add proposer signature to collation header
	collation.Header().AddSig(sig)

	return collation, nil
}

// addHeader adds the collation header to the main chain by sending
// an addHeader transaction to the sharding manager contract.
// There can only exist one header per period per shard, it's proposer's
// responsibility to check if a header has been added.
func addHeader(n node.Node, collation sharding.Collation) error {
	log.Info("Adding header to SMC")

	txOps, err := n.CreateTXOpts(big.NewInt(0))
	if err != nil {
		return fmt.Errorf("unable to initiate add header transaction: %v", err)
	}

	var churnkRoot [32]byte
	copy(churnkRoot[:], collation.Header().ChunkRoot().String())

	_, err = n.SMCTransactor().AddHeader(txOps, collation.Header().ShardID(), collation.Header().Period(), churnkRoot)
	if err != nil {
		return fmt.Errorf("unable to add header to SMC: %v", err)
	}
	return nil
}

// checkHeaderAvailability checks if a collation header has already
// added to the main chain. There can only be one header per shard
// per period, proposer should check if anyone else has added the header.
// checkHeaderAvailability returns true if it's available, false if it's unavailable.
func checkHeaderAvailability(n node.Node, shardId big.Int, period big.Int) (bool, error) {
	log.Info("Checking header in shard: %d, period: %d", shardId, period)

	// Get the period of the last header.
	lastPeriod, err := n.SMCCaller().LastSubmittedCollation(&bind.CallOpts{}, &shardId)
	if err != nil {
		return false, fmt.Errorf("unable to get the period of last submitted collation: %v", err)
	}

	// True if current period is greater than last added period.
	return period.Cmp(lastPeriod) > 0, nil
}
