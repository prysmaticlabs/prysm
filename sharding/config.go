package sharding

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	// Number of network shards
	ShardCount = int64(100)
	// Address of the sharding manager contract
	ShardingManagerAddress = common.HexToAddress("0x0") // TODO
	// Gas limit for verifying signatures
	SigGasLimit = 40000
	// Number of blocks in a period
	PeriodLength = int64(5)
	// Number of periods ahead of current period which the contract is able to return the notary of that period.
	LookaheadPeriods = 4
	// Required deposit size in wei for notary
	NotaryDeposit = new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil) // 1000 ETH
	// Required deposit size in wei for proposer
	ProposerDeposit = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // 1 ETH
	// Minimum Balance of proposer where bids are deducted
	MinProposerBalance = new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil) // 0.1 ETH
	// Gas limit to create contract
	ContractGasLimit = uint64(4700000) // Max is 4712388
	// Number of periods to lockup notary deposit from time of deregistration
	NotaryLockupLength = int64(16128)
	// Number of periods to lockup proposer deposit from time of deregistration
	ProposerLockupLength = int64(48)
	// Number of ETH awarded to notary after collation included in canonical chain
	NotarySubsidy = new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil) // 0.001 ETH
	// Number of notaries sampled per block from the notaries pool per period per shard
	NotaryCommitSize = int64(135)
	// Number of notary votes the collation needs to get accepted to the canonical chain
	NotaryQuorumSize = int64(90)
)
