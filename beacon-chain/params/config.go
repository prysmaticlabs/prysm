// Package params defines important constants that are essential to the beacon chain.
package params

import (
	"math/big"
	"time"
)

var env = "default"

// Config contains configs for node to participate in beacon chain.
type Config struct {
	ShardCount                  int       // ShardCount is the fixed number of shards in Ethereum 2.0.
	DepositSize                 *big.Int  // DepositSize is how much a validator has deposited in wei.
	BootstrappedValidatorsCount int       // BootstrappedValidatorsCount is the number of validators we seed the first crystallized state.
	MaxValidators               int       // MaxValidators is the max number of validators allowed in Ethereum 2.0.
	EtherDenomination           int       // EtherDenomination is the denomination of ether in wei.
	CycleLength                 uint64    // CycleLength is one beacon chain cycle length in slots.
	SlotDuration                uint64    // SlotDuration is how many seconds are in a single slot.
	MinCommiteeSize             uint64    // MinDynastyLength is the slots needed before dynasty transition happens.
	DefaultEndDynasty           uint64    // DefaultEndDynasty is the upper bound of dynasty. We use it to track queued and exited validators.
	MinDynastyLength            uint64    // MinCommiteeSize is the minimal number of validator needs to be in a committee.
	BaseRewardQuotient          uint64    // BaseRewardQuotient is used to calculate validator per-slot interest rate.
	SqrtEDropTime               uint64    // SqrtEDropTime is a constant to reflect time it takes to cut offline validators’ deposits by 39.4%.
	GenesisTime                 time.Time // GenesisTime used by the protocol.
	LogOutMessage               string    // This is the message a validator will send in order to log out.
	WithdrawalPeriod            uint64    // WithdrawalPeriod is the number of slots between a validator exit and validator balance being withdrawable.
	MaxValidatorChurnQuotient   uint64    // MaxValidatorChurnQuotient defines the quotient how many validators can change during each dynasty.
}

var defaultConfig = &Config{
	GenesisTime:                 time.Date(2018, 9, 0, 0, 0, 0, 0, time.UTC),
	MaxValidators:               16777216,
	CycleLength:                 uint64(64),
	ShardCount:                  1024,
	EtherDenomination:           1e18,
	DepositSize:                 new(big.Int).Div(big.NewInt(32), big.NewInt(int64(1e18))),
	SlotDuration:                uint64(8),
	MinCommiteeSize:             uint64(128),
	DefaultEndDynasty:           uint64(999999999999999999),
	BootstrappedValidatorsCount: 1000,
	MinDynastyLength:            uint64(256),
	BaseRewardQuotient:          uint64(32768),
	SqrtEDropTime:               uint64(65536),
	WithdrawalPeriod:            uint64(524288),
	MaxValidatorChurnQuotient:   uint64(32),
	LogOutMessage:               "LOGOUT",
}

var demoConfig = &Config{
	GenesisTime:               time.Now(),
	MaxValidators:             16777216,
	CycleLength:               uint64(5),
	ShardCount:                3,
	EtherDenomination:         1e18,
	DepositSize:               new(big.Int).Div(big.NewInt(32), big.NewInt(int64(1e18))),
	SlotDuration:              uint64(8),
	MinCommiteeSize:           uint64(3),
	DefaultEndDynasty:         uint64(999999999999999999),
	MinDynastyLength:          uint64(256),
	BaseRewardQuotient:        uint64(32768),
	SqrtEDropTime:             uint64(65536),
	WithdrawalPeriod:          uint64(128),
	MaxValidatorChurnQuotient: uint64(32),
	LogOutMessage:             "LOGOUT",
}

// GetConfig retrieves beacon node config.
func GetConfig() *Config {
	switch env {
	case "default":
		return defaultConfig
	case "demo":
		return demoConfig
	default:
		return defaultConfig
	}
}

// SetEnv sets which config to use.
func SetEnv(e string) {
	env = e
}
