package sharding

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	// Number of network shards
	shardCount = 100
	// Address of the validator management contract
	validatorManagerAddress = common.StringToAddress("0x315e4c3c34e27Dd28c7dCC27eC233618e76adcAc") // TODO
	// Gas limit for verifying signatures
	sigGasLimit = 40000
	// Number of blocks in a period
	periodLength = 5
	// Number of periods to lookahead for ??? TODO(prestonvanloon) finish this comment.
	lookaheadPeriods = 4
	// Required deposit size in wei
	depositSize = new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil) // 100 ETH
	// Gas limit to create contract
	contractGasLimit = uint64(4700000) // Max is 4712388
	// Gas price for contract creation
	contractGasPrice = new(big.Int).SetInt64(18000000000)
)
