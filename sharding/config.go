package sharding

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

//go:generate abigen --sol contracts/sharding_manager.sol --pkg contracts --out contracts/sharding_manager.go

var (
	// Number of network shards
	ShardCount = int64(1)
	// Address of the sharding manager contract
	ShardingManagerAddress = common.HexToAddress("0x0") // TODO
	// Gas limit for verifying signatures
	SigGasLimit = 40000
	// Number of blocks in a period
	PeriodLength = int64(5)
	// Number of periods ahead of current period which the contract is able to return the collator of that period.
	LookaheadPeriods = 4
	// Required deposit size in wei for collator
	CollatorDeposit = new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil) // 1000 ETH
	// Required deposit size in wei for proposer
	ProposerDeposit = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // 1 ETH
	// Minimum Balance of proposer where bids are decuted
	MinProposerBalance = new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil) // 0.1 ETH
	// Gas limit to create contract
	ContractGasLimit = uint64(4700000) // Max is 4712388
	// Number of collations collators need to check to apply fork choice rule
	WindbackLength = int64(25)
	// Number of periods to lockup collator deposit from time of deregistration
	CollatorLockupLength = int64(16128)
	// Number of periods to lockup proposer deposit from time of deregistration
	ProposerLockupLength = int64(48)
	// Number of vETH awarded to collators after collation included in canonical chain
	CollatorSubsidy = new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil) // 0.001 ETH
)
