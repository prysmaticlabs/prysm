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
	SMCAddress:            common.HexToAddress("0x748c193E6f1aC27643F392256C0fF518d3f526E3"),
	PeriodLength:          5,
	NotaryDeposit:         new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil), // 1000 ETH
	NotaryLockupLength:    16128,
	ProposerLockupLength:  48,
	NotaryCommitteeSize:   135,
	NotaryQuorumSize:      90,
	NotaryChallengePeriod: 25,
}

// DefaultChainConfig contains default chain configs of an individual shard.
var DefaultChainConfig = &ChainConfig{}

// Config contains configs for node to participate in the sharded universe.
type Config struct {
	SMCAddress            common.Address // SMCAddress is the address of SMC in mainchain.
	PeriodLength          int64          // PeriodLength is num of blocks in period.
	NotaryDeposit         *big.Int       // NotaryDeposit is a required deposit size in wei.
	NotaryLockupLength    int64          // NotaryLockupLength to lockup notary deposit from time of deregistration.
	ProposerLockupLength  int64          // ProposerLockupLength to lockup proposer deposit from time of deregistration.
	NotaryCommitteeSize   int64          // NotaryCommitSize sampled per block from the notaries pool per period per shard.
	NotaryQuorumSize      int64          // NotaryQuorumSize votes the collation needs to get accepted to the canonical chain.
	NotaryChallengePeriod int64          // NotaryChallengePeriod is the duration a notary has to store collations for.
}

// ChainConfig contains chain config of an individual shard. Still to be designed.
type ChainConfig struct{}
