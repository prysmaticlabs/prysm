package sharding

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	// Number of network shards
	shardCount = int64(1)
	// Address of the sharding manager contract
	shardingManagerAddress = common.HexToAddress("0x0") // TODO
	// Gas limit for verifying signatures
	sigGasLimit = 40000
	// Number of blocks in a period
	periodLength = int64(5)
	// Number of periods ahead of current period which the contract is able to return the collator of that period.
	lookaheadPeriods = 4
	// Required deposit size in wei
	depositSize = new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil) // 100 ETH
	// Gas limit to create contract
	contractGasLimit = uint64(4700000) // Max is 4712388
)
