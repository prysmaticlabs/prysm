package sharding

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	// ShardCount is the number of network shards.
	ShardCount = int64(100)
	// ShardingManagerAddress is the address of the sharding manager contract.
	ShardingManagerAddress = common.HexToAddress("0x0") // TODO
	// SigGasLimit for verifying signatures.
	SigGasLimit = 40000
	// PeriodLength is num of blocks in period.
	PeriodLength = int64(5)
	// LookaheadPeriods is the number of periods ahead of current period
	// which the contract is able to return the notary of that period.
	LookaheadPeriods = 4
	// NotaryDeposit is a required deposit size in wei.
	NotaryDeposit = new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil) // 1000 ETH
	// ProposerDeposit is a required deposit size in wei.
	ProposerDeposit = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // 1 ETH
	// MinProposerBalance of proposer where bids are deducted.
	MinProposerBalance = new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil) // 0.1 ETH
	// ContractGasLimit to create contract.
	ContractGasLimit = uint64(4700000) // Max is 4712388.
	// NotaryLockupLength to lockup notary deposit from time of deregistration.
	NotaryLockupLength = int64(16128)
	// ProposerLockupLength to lockup proposer deposit from time of deregistration.
	ProposerLockupLength = int64(48)
	// NotarySubsidy is ETH awarded to notary after collation included in canonical chain.
	NotarySubsidy = new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil) // 0.001 ETH.
	// NotaryCommitSize sampled per block from the notaries pool per period per shard.
	NotaryCommitSize = int64(135)
	// NotaryQuorumSize votes the collation needs to get accepted to the canonical chain.
	NotaryQuorumSize = int64(90)
)
