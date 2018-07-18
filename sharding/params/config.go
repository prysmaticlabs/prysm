// Package params defines important configuration options to be used when instantiating
// objects within the sharding package. For example, it defines objects such as a
// Config that will be useful when creating new shard instances.
package params

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// DefaultConfig contains default configs for node to use in the sharded universe.
var DefaultConfig = &Config{
	SMCAddress:            common.HexToAddress("0x0"),
	PeriodLength:          5,
	AttesterDeposit:         new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil), // 1000 ETH
	AttesterLockupLength:    16128,
	ProposerLockupLength:  48,
	AttesterCommitteeSize:   135,
	AttesterQuorumSize:      90,
	AttesterChallengePeriod: 25,
}

// DefaultChainConfig contains default chain configs of an individual shard.
var DefaultChainConfig = &ChainConfig{}

// Config contains configs for node to participate in the sharded universe.
type Config struct {
	SMCAddress            common.Address // SMCAddress is the address of SMC in mainchain.
	PeriodLength          int64          // PeriodLength is num of blocks in period.
	AttesterDeposit         *big.Int       // AttesterDeposit is a required deposit size in wei.
	AttesterLockupLength    int64          // AttesterLockupLength to lockup attester deposit from time of deregistration.
	ProposerLockupLength  int64          // ProposerLockupLength to lockup proposer deposit from time of deregistration.
	AttesterCommitteeSize   int64          // AttesterCommitSize sampled per block from the attesters pool per period per shard.
	AttesterQuorumSize      int64          // AttesterQuorumSize votes the collation needs to get accepted to the canonical chain.
	AttesterChallengePeriod int64          // AttesterChallengePeriod is the duration an attester has to store collations for.
}

// ChainConfig contains chain config of an individual shard. Still to be designed.
type ChainConfig struct{}
