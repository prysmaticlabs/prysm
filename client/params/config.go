// Package params defines important configuration options to be used when instantiating
// objects within the sharding package. For example, it defines objects such as a
// Config that will be useful when creating new shard instances.
package params

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	// DefaultPeriodLength is the default value for period lengths in sharding.
	DefaultPeriodLength = 5
	// DefaultAttesterLockupLength is the default number of blocks to lock up
	// an attesters deposit before they can withdraw it.
	DefaultAttesterLockupLength = 16128
)

// DefaultConfig returns pointer to a Config value with same defaults.
func DefaultConfig() *Config {
	return &Config{
		SMCAddress:              common.HexToAddress("0x0"),
		PeriodLength:            DefaultPeriodLength,
		AttesterDeposit:         DefaultAttesterDeposit(),
		AttesterLockupLength:    DefaultAttesterLockupLength,
		ProposerLockupLength:    48,
		AttesterCommitteeSize:   135,
		AttesterQuorumSize:      90,
		AttesterChallengePeriod: 25,
		CollationSizeLimit:      DefaultCollationSizeLimit(),
	}
}

// DefaultAttesterDeposit required to be an attester.
func DefaultAttesterDeposit() *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil) // 1000 ETH
}

// DefaultCollationSizeLimit is the integer value representing the maximum
// number of bytes allowed in a given collation.
func DefaultCollationSizeLimit() int64 {
	return int64(math.Pow(float64(2), float64(20)))
}

// DefaultChainConfig contains default chain configs of an individual shard.
var DefaultChainConfig = &ChainConfig{}

// Config contains configs for node to participate in the sharded universe.
type Config struct {
	SMCAddress              common.Address // SMCAddress is the address of SMC in mainchain.
	PeriodLength            int64          // PeriodLength is num of blocks in period.
	AttesterDeposit         *big.Int       // AttesterDeposit is a required deposit size in wei.
	AttesterLockupLength    int64          // AttesterLockupLength to lockup attester deposit from time of deregistration.
	ProposerLockupLength    int64          // ProposerLockupLength to lockup proposer deposit from time of deregistration.
	AttesterCommitteeSize   int64          // AttesterCommitSize sampled per block from the attesters pool per period per shard.
	AttesterQuorumSize      int64          // AttesterQuorumSize votes the collation needs to get accepted to the canonical chain.
	AttesterChallengePeriod int64          // AttesterChallengePeriod is the duration a attester has to store collations for.
	CollationSizeLimit      int64          // CollationSizeLimit is the maximum size the serialized blobs in a collation can take.
}

// ChainConfig contains chain config of an individual shard. Still to be designed.
type ChainConfig struct{}
