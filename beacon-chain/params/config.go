// Package params defines important constants that are essential to the beacon chain.
package params

import (
	"math/big"
	"time"
)

var env = "default"

// Config contains configs for node to participate in beacon chain.
type Config struct {
	ShardCount                  int       // ShardCount is a fixed number.
	DefaultBalance              *big.Int  // DefaultBalance of a validator in wei.
	MaxValidators               int       // MaxValidators in the protocol.
	Cofactor                    int       // Cofactor is used cutoff algorithm to select slot and shard cutoffs.
	BootstrappedValidatorsCount int       // BootstrappedValidatorsCount is the number of validators we seed the first crystallized state.
	EtherDenomination           int       // EtherDenomination is the denomination of ether in wei.
	CycleLength                 uint64    // CycleLength is the beacon chain cycle length in slots.
	SlotDuration                uint64    // SlotDuration in seconds.
	MinCommiteeSize             uint64    // MinDynastyLength is the slots needed before dynasty transition happens.
	DefaultEndDynasty           uint64    // DefaultEndDynasty is the upper bound of dynasty. We use it to track queued and exited validators.
	MinDynastyLength            uint64    // MinCommiteeSize is the minimal number of validator needs to be in a committee.
	BaseRewardQuotient          uint64    // BaseRewardQuotient is used to calculate validator per-slot interest rate.
	SqrtDropTime                uint64    // SqrtDropTime is a constant to reflect time it takes to cut offline validators’ deposits by 39.4%.
	GenesisTime                 time.Time // GenesisTime used by the protocol.
}

var defaultConfig = &Config{
	GenesisTime:                 time.Date(2018, 9, 0, 0, 0, 0, 0, time.UTC),
	CycleLength:                 uint64(64),
	ShardCount:                  1024,
	DefaultBalance:              new(big.Int).Div(big.NewInt(32), big.NewInt(int64(1e18))),
	MaxValidators:               4194304,
	SlotDuration:                uint64(8),
	Cofactor:                    19,
	MinCommiteeSize:             uint64(128),
	DefaultEndDynasty:           uint64(999999999999999999),
	BootstrappedValidatorsCount: 1000,
	MinDynastyLength:            uint64(256),
	EtherDenomination:           1e18,
	BaseRewardQuotient:          uint64(32768),
	SqrtDropTime:                uint64(1048576),
}

var demoConfig = &Config{
	GenesisTime:        time.Now(),
	CycleLength:        uint64(5),
	ShardCount:         3,
	DefaultBalance:     new(big.Int).Div(big.NewInt(32), big.NewInt(int64(1e18))),
	MaxValidators:      10,
	SlotDuration:       uint64(8),
	Cofactor:           19,
	MinCommiteeSize:    uint64(3),
	DefaultEndDynasty:  uint64(999999999999999999),
	MinDynastyLength:   uint64(256),
	EtherDenomination:  1e18,
	BaseRewardQuotient: uint64(32768),
	SqrtDropTime:       uint64(1048576),
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
