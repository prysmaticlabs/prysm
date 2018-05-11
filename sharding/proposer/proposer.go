package proposer

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/client"
)

// addHeader adds the collation header to the main chain by sending
// an addHeader transaction to the sharding manager contract.
// There can only exist one header per period per shard, it's proposer's
// responsibility to check if a header has been added.
func addHeader(c client.Client, header sharding.Collation) error {
	log.Info("Adding header to SMC")

	txOps, err := c.CreateTXOpts(big.NewInt(0))
	if err != nil {
		return fmt.Errorf("unable to initiate add header transaction: %v", err)
	}

	_, err = c.SMCTransactor().AddHeader(txOps, header.ShardID(), header.Period(), header.ChunkRoot())
	if err != nil {
		return fmt.Errorf("unable to add header to SMC: %v", err)
	}
	return nil
}

// checkHeaderAvailability checks if a collation header has already
// added to the main chain. There can only be one header per shard
// per period, proposer should check if anyone else has added the header.
// checkHeaderAvailability returns true if it's available, false if it's unavailable.
func checkHeaderAvailability(c client.Client, shardId big.Int, period big.Int) (bool, error) {
	log.Info("Checking header in shard: %d, period: %d", shardId, period)

	// Get the period of the last header.
	lastPeriod, err := c.SMCCaller().LastSubmittedCollation(&bind.CallOpts{}, &shardId)
	if err != nil {
		return false, fmt.Errorf("unable to get the period of last submitted collation: %v", err)
	}

	// True if current period is greater than last added period.
	return period.Cmp(lastPeriod) > 0, nil
}
